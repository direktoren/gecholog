package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/direktoren/gecholog/internal/glconfig"
	"github.com/direktoren/gecholog/internal/validate"
	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
)

// ------------------------------- GLOBALS --------------------------------

const (
	CONFIG_NAME = "tokencounter_config"
)

var globalConfig = tokencounter_config{}
var thisBinary = "tokencounter"
var version string
var logLevel = &slog.LevelVar{} // INFO
var opts = &slog.HandlerOptions{
	AddSource: true,
	Level:     logLevel,
}
var logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))

// ------------------------------- DATA STRUCTURES --------------------------------

type serviceBusConfig struct {
	Hostname string `json:"hostname" validate:"required,hostname_port"`
	Topic    string `json:"topic" validate:"required,alphanumdot"`
	Token    string `json:"token" validate:"required,ascii"`
}

func (s serviceBusConfig) String() string {
	str := fmt.Sprintf("hostname:%s topic:%s", s.Hostname, s.Topic)
	if s.Token != "" {
		str += " token:[*****MASKED*****]"
	}
	return str

}

type routerCountFields struct {
	Router       string       `json:"router" validate:"required,router"`
	Fields       []tokenCount `json:"fields" validate:"gt=0,unique=Field,dive"`
	mappedFields map[string]*tokenCount
}

func (r routerCountFields) String() string {
	str := fmt.Sprintf("Router:%s Fields:%v", r.Router, r.Fields)
	return str
}

func (r routerCountFields) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, r)
}

type tokenCount struct {
	Field string `json:"field" validate:"required,alphanumunderscore"`
	Value int    `json:"value" validate:"min=0"`
}

type routerPatternFields struct {
	Router         string         `json:"router" validate:"required,router|eq=default"`
	Patterns       []patternField `json:"patterns" validate:"gt=0,unique=Field,dive"`
	mappedPatterns map[string]*patternField
}

func (r routerPatternFields) String() string {
	str := fmt.Sprintf("Router:%s Patterns:%v", r.Router, r.Patterns)
	return str
}

func (r routerPatternFields) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, r)
}

type patternField struct {
	Field   string `json:"field" validate:"required,alphanumunderscore"`
	Pattern string `json:"pattern" validate:"required,ascii,min=1"`
}

type tokencounter_config struct {
	Version  string `json:"version" validate:"required,semver"`
	LogLevel string `json:"log_level" validate:"required,oneof=DEBUG INFO WARN ERROR"`

	ServiceBusConfig  serviceBusConfig      `json:"service_bus" validate:"required"`
	CapPeriodSeconds  int64                 `json:"cap_period_seconds" validate:"min=1"`
	TotalTokenCaps    []routerCountFields   `json:"token_caps" validate:"unique=Router,dive"`
	UsageFieldsConfig []routerPatternFields `json:"usage_fields" validate:"gt=0,unique=Router,dive"`

	caps     map[string]*routerCountFields
	consumed map[string]*routerCountFields
	patterns map[string]*routerPatternFields

	m sync.Mutex

	sha256       string
	checksumFile string
}

func (c *tokencounter_config) String() string {
	return fmt.Sprintf("version:%s log_level:%s service_bus_config:%s cap_period_seconds:%d token_caps:%s usage_fields:%s", c.Version, c.LogLevel, c.ServiceBusConfig.String(), c.CapPeriodSeconds, c.TotalTokenCaps, c.UsageFieldsConfig)
}

func (c *tokencounter_config) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, c)
}

// ------------------------------- HELPERS --------------------------------

// Add definitions of noop and error messages here
func defaultMessages(inputRaw []byte) ([]byte, []byte) {
	return []byte{}, inputRaw // Returns defaultErrorMsg and defaultNoopMsg
}

// This is the simple structure of messages to/from gechoLog
type gechoLogProcessorMessage map[string]json.RawMessage

// ------------------------------- UPDATING STATES --------------------------------

func add(consumption routerCountFields) error {
	globalConfig.m.Lock()
	defer globalConfig.m.Unlock()

	_, exists := globalConfig.consumed[consumption.Router]
	if !exists {
		// if path does not exist, create it
		globalConfig.consumed[consumption.Router] = &routerCountFields{
			Router:       consumption.Router,
			mappedFields: make(map[string]*tokenCount),
		}
	}

	for index := range consumption.Fields {
		router, exists := (globalConfig.consumed[consumption.Router])
		if !exists {
			logger.Error(
				"router doesn't exist",
				slog.Any("router", consumption.Router),
			)

			return fmt.Errorf("error: %v", fmt.Sprintf("'%s' router does not exist.", consumption.Router))
		}
		if router.mappedFields == nil {
			router.mappedFields = make(map[string]*tokenCount)
		}
		field, exists := router.mappedFields[consumption.Fields[index].Field]
		if !exists {
			// if field does not exist, create it
			field = &tokenCount{
				Field: consumption.Fields[index].Field,
				Value: 0,
			}

			router.mappedFields[consumption.Fields[index].Field] = field
		}
		field.Value += consumption.Fields[index].Value
	}

	return nil
}

func reset() error {
	globalConfig.m.Lock()
	defer globalConfig.m.Unlock()

	globalConfig.consumed = make(map[string]*routerCountFields)

	return nil
}

func isWithinCap(glPath string) bool {

	cap, exists := globalConfig.caps[glPath]
	if !exists {
		// If path is not in cap
		return true
	}

	consumption, exists := globalConfig.consumed[glPath]
	if !exists {
		// We always let the first call through
		// If path is not in totals
		return true
	}

	// If path is in both cap and totals, we check if the values are within the cap

	for field, count := range consumption.mappedFields {
		capField, exists := cap.mappedFields[field]
		if !exists {
			// If field is not in cap
			continue
		}
		if capField.Value != 0 && count.Value >= capField.Value {
			return false
		}
	}

	return true
}

// ------------------------------- PROCESS MESSAGES --------------------------------

func processIngress(glPath string) ([]byte, error) {
	var response = make(gechoLogProcessorMessage)

	ok := isWithinCap(glPath)

	if !ok {
		errorMsg := struct {
			Error string
			Path  string
		}{
			Error: "Consumption Cap Exceeded. Try again later",
			Path:  glPath,
		}
		errorBytes, err := json.Marshal(&errorMsg)
		if err != nil {
			logger.Error(
				"problem marshalling error msg",
				slog.Any("error", err),
			)

			return []byte{}, fmt.Errorf("error: %v", err)
		}
		response["control"] = errorBytes

		// Prepare the response back to nats (should structurally happen outside of process)
		responseJson, err := json.Marshal(&response)
		if err != nil {
			return []byte{}, fmt.Errorf("error: %v", err)
		}

		return responseJson, nil
	}

	return []byte{}, nil
}

// This function includes the logic of the processor
func processEgress(outputChan chan routerCountFields, glPath string, inputData []byte) ([]byte, error) {
	var response = make(gechoLogProcessorMessage)

	// Find the patterns to use
	pattern, exists := globalConfig.patterns[glPath]
	if !exists {
		pattern, exists = globalConfig.patterns["default"]
		if !exists {
			return []byte{}, nil
		}
	}

	// Populate usage data
	usage := routerCountFields{Router: glPath}
	for _, field := range pattern.Patterns {
		val := gjson.Get(string(inputData), field.Pattern)
		if val.Type == gjson.Number {
			usage.Fields = append(usage.Fields, tokenCount{Field: field.Field, Value: int(val.Uint())})
		}
	}

	go func() {
		// Send internal message to channel
		outputChan <- usage
	}()

	// Prepare response
	type processorTokenCountBody map[string]int
	var tokenCount = processorTokenCountBody{}
	for _, field := range usage.Fields {
		tokenCount[field.Field] = field.Value
	}
	usageBytes, err := json.Marshal(&tokenCount)
	if err != nil {
		logger.Error(
			"problem marshalling usage",
			slog.Any("error", err),
		)

		return []byte{}, fmt.Errorf("error: %v", err)
	}
	response["token_count"] = usageBytes

	// Prepare the response back to nats (should structurally happen outside of process)
	responseJson, err := json.Marshal(&response)
	if err != nil {
		return []byte{}, fmt.Errorf("error: %v", err)
	}

	return responseJson, nil
}

// ------------------------------- DO --------------------------------

// Connect to nats, do basic checks and call the process function
func do(ctx context.Context, cancelTheContext context.CancelFunc) {

	// Create NATS client options
	opts := nats.GetDefaultOptions()
	opts.Url = globalConfig.ServiceBusConfig.Hostname
	opts.Token = globalConfig.ServiceBusConfig.Token
	// Set reconnection options
	opts.ReconnectWait = 3 * time.Second // Wait 3 seconds before trying to reconnect
	opts.MaxReconnect = 3                // Negative value means unlimited reconnect attempts

	opts.ClosedCB = func(_ *nats.Conn) {
		logger.Warn(
			"connection to NATS closed",
		)

		cancelTheContext()
	}
	opts.ReconnectedCB = func(nc *nats.Conn) {
		logger.Info(
			"reconnected to NATS",
		)

	}
	opts.DisconnectedErrCB = func(_ *nats.Conn, err error) {
		if err != nil {
			logger.Error(
				"disconnected from NATS",
				slog.Any("error", err),
			)
			return
		}
		logger.Warn(
			"disconnected from NATS",
		)

	}
	nc, err := opts.Connect()

	//nc, err := nats.Connect(, nats.Token(globalConfig.ServiceBusConfig.Token))
	if err != nil {
		logger.Error(
			"issue connecting to nats-server",
			slog.Any("error", err),
		)

		cancelTheContext()
		return
	}
	defer nc.Close()

	//caps := totals{}
	consumptionChan := make(chan routerCountFields)

	go func(inputChan <-chan routerCountFields, resetInterval time.Duration) {
		resetTicker := time.NewTicker(resetInterval)

		for {
			select {
			case <-ctx.Done():
				logger.Info(
					"context ended. exiting function",
				)

				return
			case consumption := <-inputChan:
				add(consumption)
				logger.Debug(
					"received",
					slog.Any("consumption", consumption),
					slog.Any("total", globalConfig.consumed),
				)

			case <-resetTicker.C:
				reset()
				logger.Debug(
					"total reset",
				)

			}
		}
	}(consumptionChan, time.Duration(globalConfig.CapPeriodSeconds)*time.Second)

	sub, err := nc.QueueSubscribe(globalConfig.ServiceBusConfig.Topic, "anything", func(msg *nats.Msg) {
		// This function subscribes to the natsSubject queue. This is where we get requests to process

		defaultErrorMsg, defaultNoopMsg := defaultMessages(msg.Data)
		logger.Debug(
			"received",
			slog.String("data", string(msg.Data)),
		)

		data := defaultNoopMsg
		defer func() {
			msg.Respond(data) // Write to response
			logger.Debug(
				"respond",
				slog.String("data", string(data)),
			)

		}()

		// Unmarshal the message to a inputMessage var
		inputMessage := gechoLogProcessorMessage{}
		err := json.Unmarshal(msg.Data, &inputMessage)
		if err != nil {
			data = defaultErrorMsg
			logger.Error(
				"input_message marshal issue",
				slog.Any("error", err),
			)

			return
		}

		// First check if gl_path exists
		glPathRaw, exists := inputMessage["gl_path"]
		if !exists {
			logger.Info(
				"field not present",
				slog.String("field", "gl_path"),
			)

			return // Typical noop
		}
		var glPath string
		_ = json.Unmarshal(glPathRaw, &glPath)

		// Is it a request message?
		_, exists = inputMessage["ingress_payload"]
		if exists {
			// Process request
			processedData, err := processIngress(glPath)
			if err != nil {
				// Problem in processing
				data = defaultErrorMsg
				logger.Error(
					"processed_data marshal issue",
					slog.Any("error", err),
				)

				return
			}
			if len(processedData) > 0 {
				data = processedData
			}
			return
		}

		// It is a response message
		processedData, err := processEgress(consumptionChan, glPath, msg.Data)
		if err != nil {
			// Problem in processing
			data = defaultErrorMsg
			logger.Error(
				"processed_data marshal issue",
				slog.Any("error", err),
			)

			return
		}
		if len(processedData) > 0 {
			data = processedData
		}

	})
	if err != nil {
		logger.Error(
			"error subscribing to subject",
			slog.Any("error", err),
		)

		cancelTheContext()
		return
	}
	defer sub.Unsubscribe()

	logger.Info(
		"listening for messages",
	)

	var newLogLevel slog.LevelVar
	err = json.Unmarshal([]byte("\""+globalConfig.LogLevel+"\""), &newLogLevel)

	if err == nil && newLogLevel != *logLevel {

		logger.Info(
			"log level changed",
			slog.String("log_level", newLogLevel.Level().String()),
		)
		logLevel.Set(newLogLevel.Level())

	}

	healthyChecksumHandler(ctx, cancelTheContext) // Write the checksum to a file, IE we are healthy

	//vLog(false, "listening for messages")
	<-ctx.Done()
}

// ------------------------------- MAIN --------------------------------

// CustomFlag to track if a flag has been set
type CustomFlag struct {
	Value string
	IsSet bool
}

// String is required by the flag.Value interface.
func (c *CustomFlag) String() string {
	return c.Value
}

// Set is required by the flag.Value interface.
func (c *CustomFlag) Set(value string) error {
	c.Value = value
	c.IsSet = true
	return nil
}

func updateConfiguration(config string, g *tokencounter_config) error {
	configVersion, err := glconfig.GetVersion(config, "version")
	if err != nil {
		return err
	}

	exceptions := map[string]struct{}{
		"0.92.1": {}, // Example
	}
	_, exception := exceptions[configVersion]
	if !exception {
		// Parse the current version
		return glconfig.SetConfWithEnvVarsFromString(config, g)
	}

	/*
		switch configVersion {
		case "", "0.92.1":
			legacy := tokencounter_config_v0921{}
			err := glconfig.SetConfWithEnvVarsFromString(config, &legacy)
			if err != nil {
				return err
			}
			err = legacy.convert(g)
			if err != nil {
				return err
			}
			//Mapping

		default:
			// Code for all other cases
			// ...
		}*/

	return nil
}

func healthyChecksumHandler(ctx context.Context, cancelTheContext context.CancelFunc) {
	// Write the checksum to a file
	err := os.WriteFile(globalConfig.checksumFile, []byte(globalConfig.sha256), 0644)
	if err != nil {
		logger.Error(
			"error writing checksum file",
			slog.String("file", globalConfig.checksumFile),
			slog.Any("error", err),
		)

		cancelTheContext()
		return
	}
	logger.Info(
		"checksum file written",
		slog.String("file", globalConfig.checksumFile),
	)

	go func() {
		select {
		case <-ctx.Done():
			err := os.Remove(globalConfig.checksumFile)
			if err != nil {
				logger.Error(
					"error removing checksum file",
					slog.String("file", globalConfig.checksumFile),
					slog.Any("error", err),
				)

				return
			}
			logger.Info(
				"checksum file removed",
				slog.String("file", globalConfig.checksumFile),
			)

			return
		}
	}()
}

func setupConfig(fs *flag.FlagSet, args []string) error {

	// Select flag options and parse
	var configFilename CustomFlag
	fs.Var(&configFilename, "o", "Set name and path to config file")

	var versionFlag bool
	fs.BoolVar(&versionFlag, "version", false, "Print version and exit")

	var validateFlag bool
	fs.BoolVar(&validateFlag, "validate", false, "Validate config file and exit")

	var serviceAlias string
	fs.StringVar(&serviceAlias, "a", thisBinary, "Set service alias")

	fs.Parse(args)

	logger = logger.With("service", serviceAlias) // To be used as default

	if versionFlag {
		// Print version and exit
		fmt.Println(version)
		os.Exit(0)
	}

	config, err := func() (string, error) {
		if (validateFlag) && !configFilename.IsSet {
			var input []byte
			input, err := io.ReadAll(os.Stdin)
			if err != nil {
				logger.Error(
					"error reading stdin",
					slog.Any("error", err),
				)

				return "", err
			}
			return string(input), nil
		}

		c, err := glconfig.ReadFile(configFilename.Value)
		if err != nil {
			logger.Error(
				"error opening file",
				slog.Any("error", err),
			)

			return "", err
		}
		return c, nil
	}()
	if err != nil {
		return err
	}
	err = updateConfiguration(config, &globalConfig)
	if err != nil {
		logger.Error(
			"error loading configuration",
			slog.Any("error", err),
		)

		return err
	}

	rejectedFields := validate.ValidationErrors{}
	validCaps := []routerCountFields{}
	for index, router := range globalConfig.TotalTokenCaps {
		e := router.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.token_caps[%d].%s", CONFIG_NAME, index, k)] = v
			}
			continue

		}
		validCaps = append(validCaps, router)
	}
	globalConfig.TotalTokenCaps = validCaps

	validPatterns := []routerPatternFields{}
	for index, router := range globalConfig.UsageFieldsConfig {
		e := router.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.usage_fields[%d].%s", CONFIG_NAME, index, k)] = v
			}
			continue

		}
		validPatterns = append(validPatterns, router)
	}
	globalConfig.UsageFieldsConfig = validPatterns

	validationErrors := globalConfig.Validate()
	if validationErrors != nil {
		if len(rejectedFields) != 0 {
			logger.Warn(
				"configuration has rejected fields",
				slog.Any("rejected_fields", rejectedFields),
			)

		}
		logger.Error(
			"configuration file validation failed",
			slog.Any("validation_errors", validationErrors),
		)

		return fmt.Errorf("error validating configuration")
	}

	logger.Info(
		"configuration file valid",
		slog.String("configuration", globalConfig.String()),
	)

	if len(rejectedFields) != 0 {
		logger.Warn(
			"configuration has rejected fields",
			slog.Any("rejected_fields", rejectedFields),
		)

	}

	if validateFlag {
		// Validation of config successful, exit
		os.Exit(0)
	}
	globalConfig.checksumFile = "/app/checksum/." + serviceAlias + "_config.sha256"
	_, err = os.Stat(globalConfig.checksumFile)
	if err == nil {
		err := os.Remove(globalConfig.checksumFile)
		if err != nil {
			logger.Error(
				"error removing checksum file",
				slog.Any("error", err),
			)

			return err
		}
		logger.Info(
			"old checksum file removed",
			slog.String("checksum_file", globalConfig.checksumFile),
		)

	}

	logger.Info(
		"system version",
		slog.String("version", version),
	)

	logger.Info("running as user", slog.Int("user", os.Geteuid()))

	globalConfig.sha256, err = glconfig.GenerateChecksum(configFilename.Value)
	if err != nil {
		logger.Error(
			"error generating checksum",
			slog.Any("error", err),
		)

		return err
	}

	logger.Info(
		"configuration file checksum calculated",
		slog.String("checksum", globalConfig.sha256),
	)

	// sha256sum config.json        # Linux
	// shasum -a 256 config.json	# Mac

	// Populate the maps
	globalConfig.caps = make(map[string]*routerCountFields)
	for index, _ := range globalConfig.TotalTokenCaps {
		router := &globalConfig.TotalTokenCaps[index]
		if router.mappedFields == nil {
			router.mappedFields = make(map[string]*tokenCount)
		}
		for fieldIndex, field := range router.Fields {
			router.mappedFields[field.Field] = &router.Fields[fieldIndex]
		}
		globalConfig.caps[globalConfig.TotalTokenCaps[index].Router] = router

	}
	globalConfig.consumed = make(map[string]*routerCountFields)
	globalConfig.patterns = make(map[string]*routerPatternFields)
	for index, _ := range globalConfig.UsageFieldsConfig {
		router := &globalConfig.UsageFieldsConfig[index]
		if router.mappedPatterns == nil {
			router.mappedPatterns = make(map[string]*patternField)
		}
		for fieldIndex, field := range router.Patterns {
			router.mappedPatterns[field.Field] = &router.Patterns[fieldIndex]
		}
		globalConfig.patterns[globalConfig.UsageFieldsConfig[index].Router] = router
	}

	return nil
}

// Create context and capture ctrl-C, and give everyone 1s to cleanup
func main() {
	fs := flag.NewFlagSet("prod", flag.ExitOnError)
	err := setupConfig(fs, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	// Create context & sync
	ctx, cancelFunction := context.WithCancel(context.Background())
	defer cancelFunction()

	go do(ctx, cancelFunction)

	// wait for ctrl-C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {

	case <-c:
		cancelFunction()
		time.Sleep(1 * time.Second) // Allow one second for everyone to cleanup
	case <-ctx.Done():

	}

}

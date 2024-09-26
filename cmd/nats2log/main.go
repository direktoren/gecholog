package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/direktoren/gecholog/internal/glconfig"
	"github.com/direktoren/gecholog/internal/validate"
	"github.com/nats-io/nats.go"
)

// ------------------------------- GLOBALS --------------------------------
const (
	CONFIG_NAME = "nats2log_config"
)

var globalConfig = nats2log_config{}
var thisBinary = "nats2log"
var version string
var logLevel = &slog.LevelVar{} // INFO
var opts = &slog.HandlerOptions{
	AddSource: true,
	Level:     logLevel,
}
var logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))

// ------------------------------- CONFIGURATION --------------------------------

type serviceBusConfig struct {
	Hostname         string `json:"hostname" validate:"required,hostname_port"`
	Topic            string `json:"topic" validate:"required,alphanumdot"`
	TopicExactLogger string `json:"topic_exact_logger" validate:"required,nefield=Topic,alphanumdot"`
	Token            string `json:"token" validate:"required,ascii"`
}

func (s serviceBusConfig) String() string {
	str := fmt.Sprintf("hostname:%s topic:%s topic_logger_exact:%s", s.Hostname, s.Topic, s.TopicExactLogger)
	if s.Token != "" {
		str += " token:[*****MASKED*****]"
	}
	return str

}

type tlsConfig struct {
	InsecureFlag       bool     `json:"insecure"`
	SystemCertPoolFlag bool     `json:"system_cert_pool"`
	CertFiles          []string `json:"cert_files" validate:"unique,dive,file"`
}

func (t tlsConfig) String() string {
	// Prints the part of the configuration that is relevant to the user.
	return fmt.Sprintf("insecure:%v system_cert_pool:%v cert_files:%v", t.InsecureFlag, t.SystemCertPoolFlag, t.CertFiles)
}

type nats2log_config struct {
	// From json config file
	Version  string `json:"version" validate:"required,semver"`
	LogLevel string `json:"log_level" validate:"required,oneof=DEBUG INFO WARN ERROR"`
	Mode     string `json:"mode" validate:"required,oneof=file_writer rest_api_writer elastic_writer azure_log_analytics_writer"`

	Retries          int              `json:"retries" validate:"min=0"`
	RetryDelay       time.Duration    `json:"retry_delay_milliseconds"  validate:"min=0"`
	ServiceBusConfig serviceBusConfig `json:"service_bus" validate:"required"`

	TlsConfig tlsConfig `json:"tls" validate:"required"`

	FileWriter              *fileWriter              `json:"file_writer" validate:"required_if=Mode file_writer"`
	RestAPIWriter           *restAPIWriter           `json:"rest_api_writer" validate:"required_if=Mode rest_api_writer"`
	ElasticWriter           *elasticWriter           `json:"elastic_writer" validate:"required_if=Mode elastic_writer"`
	AzureLogAnalyticsWriter *azureLogAnalyticsWriter `json:"azure_log_analytics_writer" validate:"required_if=Mode azure_log_analytics_writer"`

	// Internal
	bW byteWriter

	sha256       string
	checksumFile string
}

func (c *nats2log_config) String() string {
	// Prints the part of the configuration that is relevant to the user.
	str := fmt.Sprintf(
		"version:%s"+
			"mode:%s "+
			"log_level:%v "+
			"retries:%v "+
			"service_bus:%s "+
			"tls:%s ",
		c.Version,
		c.Mode,
		c.LogLevel,
		c.Retries,
		c.ServiceBusConfig.String(),
		c.TlsConfig.String())
	switch c.Mode {
	case "file_writer":
		str += c.FileWriter.String()
	case "rest_api_writer":
		str += c.RestAPIWriter.String()
	case "elastic_writer":
		str += c.ElasticWriter.String()
	case "azure_log_analytics_writer":
		str += c.AzureLogAnalyticsWriter.String()
	default:
		str += "No writer selected"
	}
	return str
}

func (c *nats2log_config) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, c)
}

// ------------------------------- GLOBALS --------------------------------

var writeModeMap = map[string]int{
	// helper map to convert string to os.OpenFile mode
	"append":    os.O_APPEND,
	"overwrite": os.O_TRUNC,
	"new":       os.O_TRUNC,
}

// ------------------------------- byteWriter interface --------------------------------

type byteWriter interface {
	write(b []byte) error
	open() error
	close()
	// validate() error
}

// ------------------------------- restAPIWriter --------------------------------

type elasticWriter struct {
	// elasticWriter is a struct that implements the byteWriter interface.

	// Public
	URL      string `json:"url" validate:"http_url"`
	Port     int    `json:"port" validate:"min=1,max=65535"`
	Index    string `json:"index" validate:"required,alphanum"`
	Username string `json:"username" validate:"required,ascii"`
	Password string `json:"password" validate:"required,ascii"`

	// internal
	restAPIWriter byteWriter
}

func (e elasticWriter) String() string {
	// Prints the part of the configuration that is relevant to the user.
	return fmt.Sprintf("url:%s port:%v index:%s username:%s password:[*****MASKED*****]", e.URL, e.Port, e.Index, e.Username)
}

func (e elasticWriter) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, e)
}

func (e *elasticWriter) open() error {
	// Open the restAPIWriter
	if e.restAPIWriter == nil {
		return fmt.Errorf("elasticWriter: restAPIWriter is nil")
	}
	return e.restAPIWriter.open()
}

func (e *elasticWriter) close() {
	// Close the restAPIWriter
	if e.restAPIWriter == nil {
		return
	}
	e.restAPIWriter.close()
}

func (e *elasticWriter) write(b []byte) error {
	// Write to elastic
	if e.restAPIWriter == nil {
		return fmt.Errorf("elasticWriter: restAPIWriter is nil")
	}

	// Write to rest api
	return e.restAPIWriter.write(b)
}

// ------------------------------- restAPIWriter --------------------------------

type restAPIWriter struct {
	// restAPIWriter is a struct that implements the byteWriter interface.

	// Public
	URL      string              `json:"url" validate:"http_url"`
	Port     int                 `json:"port" validate:"min=1,max=65535"`
	Endpoint string              `json:"endpoint" validate:"omitempty,endpoint"`
	Headers  map[string][]string `json:"headers" validate:"dive,keys,ascii,excludesall= /()<>@;:\\\"[]?=,endkeys,gt=0,dive,required,ascii"`
	//	Headers  []header `json:"headers" validate:"dive"`

	// Internal
	client *http.Client
	tls    tlsConfig
}

func (r restAPIWriter) String() string {
	// Prints the part of the configuration that is relevant to the user.
	return fmt.Sprintf("url:%s port:%v endpoint:%s headers:%v", r.URL, r.Port, r.Endpoint, r.Headers)
}

func (r restAPIWriter) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, r)
}

func (r *restAPIWriter) open() error {

	if r.client != nil {
		// We don't reopen the client
		return nil
	}

	parsedURL, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("restAPIWriter: Error parsing URL: %v", err)
	}

	if parsedURL.Scheme == "http" { // ONLY FOR TESTING
		// No tls required
		r.client = &http.Client{}
		return nil
	}

	// TLS required
	if r.tls.InsecureFlag {
		// Skip TLS verification
		r.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
		return nil
	}
	logger.Debug(
		"tls enabled",
	)

	caCertPool := x509.NewCertPool()
	if r.tls.SystemCertPoolFlag {
		// Load system cert pool
		systemCertPool, err := x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("restAPIWriter: Error loading system cert pool: %v", err)
		}
		caCertPool = systemCertPool
		logger.Debug(
			"system cert pool loaded",
		)

	}

	// Load custom CA certificates
	for _, filename := range r.tls.CertFiles {
		caCert, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("restAPIWriter: Error reading CA certificate file: %v", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
		logger.Debug(
			"CA certificate file loaded",
			slog.String("filename", filename),
		)

	}

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	r.client = &http.Client{Transport: tr}
	return nil
}

func (r *restAPIWriter) close() {
	// nothing to do
	r.client = nil
}

func (r *restAPIWriter) write(b []byte) error {
	if r.client == nil {
		return fmt.Errorf("restAPIWriter: client is nil")
	}

	// Write to rest api
	req, err := http.NewRequest("POST", r.URL+":"+fmt.Sprintf("%d", r.Port)+r.Endpoint, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("restAPIWriter: Error creating request: %v", err)
	}

	for k, v := range r.Headers {
		req.Header[k] = v
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("restAPIWriter: Error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("restAPIWriter: Server response: %v", resp.Status)
	}

	return nil
}

// ------------------------------- azureLogAnalyticsWriter --------------------------------

type azureLogAnalyticsWriter struct {
	// azureLogAnalyticsWriter is a struct that implements the byteWriter interface.
	WorkspaceID string `json:"workspace_id" validate:"required,ascii"`
	SharedKey   string `json:"shared_key" validate:"required,ascii"`
	LogType     string `json:"log_type" validate:"required,ascii"`

	// Internal
	restAPIWriter *restAPIWriter
}

func (a azureLogAnalyticsWriter) String() string {
	// Prints the part of the configuration that is relevant to the user.
	return fmt.Sprintf("workspace_id:%s shared_key:[*****MASKED*****] log_type:%s", a.WorkspaceID, a.LogType)
}

func (a azureLogAnalyticsWriter) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, a)
}

func (a *azureLogAnalyticsWriter) open() error {
	// Open the restAPIWriter
	if a.restAPIWriter == nil {
		return fmt.Errorf("azureLogAnalyticsWriter: restAPIWriter is nil")
	}
	return a.restAPIWriter.open()
}

func (a *azureLogAnalyticsWriter) close() {
	// Close the restAPIWriter
	if a.restAPIWriter == nil {
		return
	}
	a.restAPIWriter.close()
}

func (a *azureLogAnalyticsWriter) write(b []byte) error {
	// Write to azure log analytics
	if a.restAPIWriter == nil {
		return fmt.Errorf("azureLogAnalyticsWriter: restAPIWriter is nil")
	}

	buildSignature := func(message, secret string) (string, error) {
		keyBytes, err := base64.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", err
		}
		mac := hmac.New(sha256.New, keyBytes)
		mac.Write([]byte(message))
		return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
	}

	rfc1123date := time.Now().UTC().Format(time.RFC1123)
	rfc1123date = strings.ReplaceAll(rfc1123date, "UTC", "GMT") // Replace UTC with GMT

	xHeaders := "x-ms-date:" + rfc1123date
	contentLength := len(b)
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s\n%s", "POST", contentLength, "application/json", xHeaders, "/api/logs")
	signature, err := buildSignature(stringToSign, a.SharedKey)
	if err != nil {
		return fmt.Errorf("azureLogAnalyticsWriter: Error building signature: %v", err)
	}
	authorization := fmt.Sprintf("SharedKey %s:%s", a.WorkspaceID, signature)

	a.restAPIWriter.Headers["x-ms-date"] = []string{rfc1123date}
	a.restAPIWriter.Headers["Authorization"] = []string{authorization}

	// Write to rest api
	return a.restAPIWriter.write(b)
}

// ------------------------------- fileWriter --------------------------------

type fileWriter struct {
	// fileWriter is a struct that implements the byteWriter interface.
	// for file writing

	// Public
	ConfigFilename string `json:"filename" validate:"filepath,excludesall= ()<>@;:\\\"[]?="`
	WriteMode      string `json:"write_mode" validate:"oneof=append overwrite new"`

	// Interna
	writeMode       int
	file            *os.File
	currentFilename string
}

func (f fileWriter) String() string {
	// Prints the part of the configuration that is relevant to the user.
	return fmt.Sprintf("filename:%s write_mode:%v", f.ConfigFilename, f.WriteMode)
}

func (f fileWriter) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, f)
}

// write writes a byte slice to the file.
func (f *fileWriter) write(b []byte) error {
	if f.file == nil {
		return fmt.Errorf("fileWriter: file is nil")
	}
	_, err := os.Stat(f.currentFilename) // Check if file is gone. Quicker than fsync but not ideal to do every write
	if os.IsNotExist(err) {
		return fmt.Errorf("fileWriter: file is gone")
	}

	bData := bytes.Trim(b, "\n")
	_, err = f.file.Write(bData)
	if err != nil {
		return err
	}
	_, err = f.file.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

// Close closes the file.
func (f *fileWriter) close() {
	if f.file != nil {
		f.file.Close()
	}
}

// getCurrentLogFilename returns a the correct log filename for the current write mode.
func (f fileWriter) getCurrentLogFilename() string {

	filename := f.ConfigFilename
	_, err := os.Stat(filename)

	if !os.IsNotExist(err) {
		// File already exists
		if f.WriteMode == "new" {
			// We are in file mode NEW
			// For new write mode, if file exists, find the next available number to append to the filename.
			parts := strings.Split(filename, ".")
			ext := ""
			if len(parts) > 1 {
				ext = "." + parts[len(parts)-1]
				parts = parts[:len(parts)-1]
			}
			base := strings.Join(parts, ".")

			logQuantityWarning := 1000
			for i := 1; ; i++ {
				filename = fmt.Sprintf("%s_%d%s", base, i, ext)
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					break
				}

				if i%logQuantityWarning == 0 {
					logger.Warn(
						"log files already exist. Amount is not limited, but consider removing some",
						slog.Int("log_files", i),
					)

				}
			}
		}
	}
	return filename
}

// open opens the file for writing.
func (f *fileWriter) open() error {

	if f.writeMode == 0 {
		f.writeMode = writeModeMap[f.WriteMode]
	}
	f.currentFilename = f.getCurrentLogFilename()
	if f.file != nil {
		// Close the file if it is already open
		logger.Info("closing file")
		f.file.Close()
	}

	var err error
	f.file, err = os.OpenFile(f.currentFilename, f.writeMode|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.EACCES {
			// permissions error. We could handle this better, but for now just log and exit
			return err
		}
		return err
	}
	logger.Info(
		"opened file",
		slog.String("filename", f.currentFilename),
	)

	return nil
}

// ------------------------------- DO --------------------------------

// Connect to nats, do basic checks and set up processing

type message struct {
	data    []byte
	retries int
}

func processMessage(ctx context.Context, ch chan message) {
	defer func() {
		logger.Info(
			"processor: exiting",
		)

	}()
	for {
		select {
		case msg := <-ch:
			logger.Debug(
				"processor: picking up message",
				slog.String("message", string(msg.data)),
				slog.Int("attempt", msg.retries),
			)

			err := globalConfig.bW.write(msg.data)
			if err != nil {
				go func(msg message) {
					if msg.retries >= globalConfig.Retries {
						logger.Error(
							"processor: error processing message",
							slog.Any("error", err),
							slog.String("message", string(msg.data)),
							slog.Int("attempt", msg.retries),
						)

						return
					}
					if msg.retries == 1 {
						err := globalConfig.bW.open()
						if err != nil {
							logger.Error(
								"processor: error re-opening writer",
								slog.Any("error", err),
							)

						}
					}
					logger.Error(
						"processor: error processing message, retrying",
						slog.Any("error", err),
						slog.String("message", string(msg.data)),
						slog.Int("attempt", msg.retries),
					)

					time.Sleep(time.Millisecond * globalConfig.RetryDelay)

					checkContextCancelled := func(ctx context.Context) bool {
						select {
						case <-ctx.Done():
							// The context is cancelled or expired
							return true
						default:
							// The context is not yet cancelled
							return false
						}
					}
					if checkContextCancelled(ctx) {
						return
					}
					ch <- message{data: msg.data, retries: msg.retries + 1}
				}(msg)
				break
			}
			logger.Debug(
				"processor: message processed",
				slog.String("message", string(msg.data)),
				slog.Int("attempt", msg.retries),
			)

		case <-ctx.Done():
			logger.Info(
				"processor: process cancelled",
			)

			return
		}
	}
}

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

	if err != nil {
		logger.Error(
			"issue connecting to nats-server",
			slog.Any("error", err),
		)

		cancelTheContext()
		return
	}
	defer nc.Close()

	// publish to topic
	err = nc.Publish(globalConfig.ServiceBusConfig.Topic, []byte("running & connected to message bus"))
	if err != nil {
		logger.Error(
			"error publishing status to channel",
			slog.Any("error", err),
		)

		cancelTheContext()
		return
	}

	// Open the writer
	err = globalConfig.bW.open()
	if err != nil {
		logger.Error(
			"error opening writer",
			slog.Any("error", err),
		)

		cancelTheContext()
		return
	}
	defer globalConfig.bW.close()

	ch := make(chan message)

	// Setup writer process
	go processMessage(ctx, ch)

	// Subscribe to the channel.
	sub, err := nc.Subscribe(globalConfig.ServiceBusConfig.TopicExactLogger, func(msg *nats.Msg) {
		logger.Debug(
			"receiver: received message",
			slog.String("message", string(msg.Data)),
		)

		go func() {
			ch <- message{data: msg.Data, retries: 0}
		}()
	})

	if err != nil {
		logger.Error(
			"error subscribing to inbound channel",
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

	<-ctx.Done()
}

// ------------------------------ MAIN ------------------------------

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

func updateConfiguration(config string, g *nats2log_config) error {
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
			legacy := nats2log_config_v0921{}
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

	// Identify and remove rejected fields
	rejectedFields := validate.ValidationErrors{}
	if globalConfig.FileWriter != nil {
		e := globalConfig.FileWriter.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.FileWriter.%s", CONFIG_NAME, k)] = v
			}
			globalConfig.FileWriter = nil
		}
	}

	if globalConfig.RestAPIWriter != nil {
		e := globalConfig.RestAPIWriter.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.RestAPIWriter.%s", CONFIG_NAME, k)] = v
			}
			globalConfig.RestAPIWriter = nil
		}
	}

	if globalConfig.ElasticWriter != nil {
		e := globalConfig.ElasticWriter.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.ElasticWriter.%s", CONFIG_NAME, k)] = v
			}
			globalConfig.ElasticWriter = nil
		}
	}

	if globalConfig.AzureLogAnalyticsWriter != nil {
		e := globalConfig.AzureLogAnalyticsWriter.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.AzureLogAnalyticsWriter.%s", CONFIG_NAME, k)] = v
			}
			globalConfig.AzureLogAnalyticsWriter = nil
		}
	}

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

	// Selector for writer
	switch globalConfig.Mode {
	case "file_writer":
		globalConfig.bW = &fileWriter{ConfigFilename: globalConfig.FileWriter.ConfigFilename, WriteMode: globalConfig.FileWriter.WriteMode}

	case "rest_api_writer":
		globalConfig.bW = &restAPIWriter{
			URL:      globalConfig.RestAPIWriter.URL,
			Port:     globalConfig.RestAPIWriter.Port,
			Endpoint: globalConfig.RestAPIWriter.Endpoint,
			Headers:  globalConfig.RestAPIWriter.Headers,
			tls:      globalConfig.TlsConfig,
		}

	case "elastic_writer":
		globalConfig.bW = &elasticWriter{
			URL:      globalConfig.ElasticWriter.URL,
			Port:     globalConfig.ElasticWriter.Port,
			Index:    globalConfig.ElasticWriter.Index,
			Username: globalConfig.ElasticWriter.Username,
			Password: globalConfig.ElasticWriter.Password,
			restAPIWriter: &restAPIWriter{
				URL:      globalConfig.ElasticWriter.URL,
				Port:     globalConfig.ElasticWriter.Port,
				Endpoint: fmt.Sprintf("/%s/_doc", globalConfig.ElasticWriter.Index),
				Headers: map[string][]string{
					"Content-Type": {"application/json"},
					"Authorization": {"Basic " + func() string {
						return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", globalConfig.ElasticWriter.Username, globalConfig.ElasticWriter.Password)))
					}(),
					},
				},
				tls: globalConfig.TlsConfig,
			}}

	case "azure_log_analytics_writer":

		globalConfig.bW = &azureLogAnalyticsWriter{
			WorkspaceID: globalConfig.AzureLogAnalyticsWriter.WorkspaceID,
			SharedKey:   globalConfig.AzureLogAnalyticsWriter.SharedKey,
			LogType:     globalConfig.AzureLogAnalyticsWriter.LogType,
			restAPIWriter: &restAPIWriter{
				URL:      fmt.Sprintf("https://%s.ods.opinsights.azure.com", globalConfig.AzureLogAnalyticsWriter.WorkspaceID),
				Port:     443,
				Endpoint: "/api/logs?api-version=2016-04-01",
				Headers: map[string][]string{
					"Content-Type": {"application/json"},
					"Log-Type":     {globalConfig.AzureLogAnalyticsWriter.LogType},
				},
				tls: globalConfig.TlsConfig,
			},
		}

	default:
		logger.Error(
			"error validating mode. Using default file_writer",
		)

		globalConfig.bW = &fileWriter{ConfigFilename: globalConfig.FileWriter.ConfigFilename, WriteMode: globalConfig.FileWriter.WriteMode}

	}

	return nil
}

func main() {
	fs := flag.NewFlagSet("prod", flag.ExitOnError)
	err := setupConfig(fs, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	ctx, cancelFunction := context.WithCancel(context.Background())
	defer cancelFunction()

	go do(ctx, cancelFunction)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		cancelFunction()
		time.Sleep(1 * time.Second) // Allow one second for everyone to cleanup
	case <-ctx.Done():
	}
}

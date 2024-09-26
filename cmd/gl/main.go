package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/direktoren/gecholog/internal/glconfig"
	"github.com/direktoren/gecholog/internal/processorconfiguration"
	"github.com/direktoren/gecholog/internal/protectedheader"
	"github.com/direktoren/gecholog/internal/router"
	"github.com/direktoren/gecholog/internal/timer"
	"github.com/direktoren/gecholog/internal/validate"
	"github.com/nats-io/nats.go"
)

// ------------------------------- GLOBALS --------------------------------

const (
	CONFIG_NAME = "gl_config"
)

var globalConfig = gl_config{}
var thisBinary = "gl"
var version string
var logLevel = &slog.LevelVar{} // INFO
var opts = &slog.HandlerOptions{
	AddSource: true,
	Level:     logLevel,
}
var logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))

// ------------------------------- CONFIGURATION --------------------------------

type serviceBusConfig struct {
	Hostname          string `json:"hostname" validate:"required,hostname_port"`
	Topic             string `json:"topic" validate:"required,alphanumdot"`
	TopicExactIsAlive string `json:"topic_exact_isalive" validate:"required,nefield=Topic,nefield=TopicExactLogger,alphanumdot"`
	TopicExactLogger  string `json:"topic_exact_logger" validate:"required,nefield=Topic,nefield=TopicExactIsAlive,alphanumdot"`
	Token             string `json:"token" validate:"required,ascii"`
}

func (s serviceBusConfig) String() string {
	str := fmt.Sprintf("hostname:%s topic:%s topic_exact_isalive:%s topic_logger_exact:%s", s.Hostname, s.Topic, s.TopicExactIsAlive, s.TopicExactLogger)
	if s.Token != "" {
		str += " token:[*****MASKED*****]"
	}
	return str

}

type tlsUserConfig struct {
	Ingress struct {
		Enabled         bool   `json:"enabled"`
		CertificateFile string `json:"certificate_file" validate:"required_if=Enabled true|file"`
		PrivateKeyFile  string `json:"private_key_file" validate:"required_if=Enabled true|file"`
	} `json:"ingress"`
	Outbound struct {
		InsecureFlag       bool     `json:"insecure"`
		SystemCertPoolFlag bool     `json:"system_cert_pool"`
		CertFiles          []string `json:"cert_files" validate:"unique,dive,file"`
	} `json:"outbound"`
}

func (t tlsUserConfig) String() string {
	s := fmt.Sprintf("ingress:{%s} ", fmt.Sprintf("enabled:%v certificate_file:%s private_key_file:%s ", t.Ingress.Enabled, t.Ingress.CertificateFile, t.Ingress.PrivateKeyFile))
	s += fmt.Sprintf("outbound:{%s}", fmt.Sprintf("insecure:%v system_cert_pool:%v cert_files:%v", t.Outbound.InsecureFlag, t.Outbound.SystemCertPoolFlag, t.Outbound.CertFiles))
	return s
}

type isAliveBody struct {
	LastTransactionID string
	LastIngressTime   string

	LastError     string
	LastErrorTime string
}

type processorsMatrix struct {
	Processors []([]processorconfiguration.ProcessorConfiguration) `json:"processors" validate:"dive,unique=Name,unique=ServiceBusTopic,dive"`
}

type filter struct {
	FieldsInclude []string `json:"fields_include" validate:"unique,dive,alphanumunderscore"`
	FieldsExclude []string `json:"fields_exclude" validate:"unique,dive,alphanumunderscore"`
}

type finalLogger struct {
	Request  filter `json:"request"`
	Response filter `json:"response"`
}

// ----------- Configuration object -----------------

type gl_config struct {
	GatewayID string `json:"gateway_id" validate:"required,alphanum,uppercase,len=8"`
	Version   string `json:"version" validate:"required,semver"`
	LogLevel  string `json:"log_level" validate:"required,oneof=DEBUG INFO WARN ERROR"`

	TlsUserConfig tlsUserConfig `json:"tls" validate:"required"`

	ServiceBusConfig serviceBusConfig `json:"service_bus" validate:"required"`

	Port            int    `json:"gl_port" validate:"min=1,max=65535"`
	SessionIDHeader string `json:"session_id_header" validate:"required,ascii,excludesall= /()<>@;:\\\"[]?="`

	MaskedHeaders    []string `json:"masked_headers" validate:"unique,dive,ascii,excludesall= /()<>@;:\\\"[]?="`
	maskedHeadersMap map[string]struct{}

	RemoveHeaders    []string `json:"remove_headers" validate:"unique,dive,ascii,excludesall= /()<>@;:\\\"[]?="`
	removeHeadersMap map[string]struct{}

	LogUnauthorized bool            `json:"log_unauthorized"`
	Routers         []router.Router `json:"routers" validate:"unique=Path,gt=0,dive"`

	RequestProcessors  processorsMatrix `json:"request"`
	ResponseProcessors processorsMatrix `json:"response"`
	Logger             finalLogger      `json:"logger"`

	IsAliveData isAliveBody
	//	performanceLog *logrus.Logger
	client    *http.Client
	tlsConfig *tls.Config
	m         sync.Mutex

	sha256       string
	checksumFile string
}

// Configuration stringer
func (c *gl_config) String() string {
	s := fmt.Sprintf("gateway_id:%s ", c.GatewayID)
	s += fmt.Sprintf("version:%s ", c.Version)
	s += fmt.Sprintf("log_level:%s ", c.LogLevel)
	s += fmt.Sprintf("tls:%s ", c.TlsUserConfig.String())
	s += fmt.Sprintf("service_bus:%s ", c.ServiceBusConfig.String())
	s += fmt.Sprintf("gl_port:%d ", c.Port)
	s += fmt.Sprintf("session_id_header:%s ", c.SessionIDHeader)
	s += fmt.Sprintf("masked_headers:%v ", c.MaskedHeaders)
	s += fmt.Sprintf("remove_headers:%v ", c.RemoveHeaders)
	s += fmt.Sprintf("log_unauthorized:%v ", c.LogUnauthorized)
	for i, r := range c.Routers {
		s += fmt.Sprintf("router %d:{%s} ", i, r.String())
	}
	s += fmt.Sprintf("request:%v ", c.RequestProcessors)
	s += fmt.Sprintf("response:%v ", c.ResponseProcessors)
	s += fmt.Sprintf("logger:%v ", c.Logger)
	return s
}

func (c *gl_config) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, c)
}

func (g *gl_config) setLastError(err error, t time.Time) {
	g.m.Lock()
	defer g.m.Unlock()
	g.IsAliveData.LastError = err.Error()
	g.IsAliveData.LastErrorTime = t.String()[:19]
}

func (g *gl_config) setLastTransactionID(tid string) {
	g.m.Lock()
	defer g.m.Unlock()
	g.IsAliveData.LastTransactionID = tid
}

// ----------- Processor helpers -----------------

type processorLog struct {
	Required  bool        `json:"required"`
	Completed bool        `json:"completed"`
	Timestamp timer.Timer `json:"timestamp"`
}

func noTrailingSlashHandlerFunc(path string) (string, func(w http.ResponseWriter, r *http.Request)) {
	return path[:len(path)-1], func(w http.ResponseWriter, r *http.Request) {
		// Respond with a custom message and 400 Bad Request status
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Unsupported path without '/'")
	}
}

type CustomWriter struct {
	logger *slog.Logger

	request uint64
	supress uint64
}

// Write implements the io.Writer interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	// Check if the log message contains the unwanted error message
	if strings.HasPrefix(string(p), "http: TLS handshake error") {
		//		if strings.HasSuffix(string(p), " timeout\n") {
		cw.request++
		if cw.request >= cw.supress {
			cw.logger.Warn(string(p), slog.Uint64("request", cw.request), slog.String("layer", "httpServer"))
			cw.supress = cw.supress * 10
		}
		return len(p), nil
	}
	// Log the message using slog
	cw.logger.Info(string(p), slog.String("layer", "httpServer"))
	return len(p), nil
}

// ----------- do function - Sets up service bus, isalive, fires of http(s) server  -----------------

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

		//cancelTheContext()
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

	logger.Info(
		"service bus initialized",
	)
	logger.Info(
		"starting the service",
	)

	buildProcessorsMiddleware := func(async bool, pm processorsMiddlewareFunc, processors [][]processorconfiguration.ProcessorConfiguration, s *state) (func(http.Handler) http.Handler, int) {
		m := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Do nothing
				next.ServeHTTP(w, r)
			})
		}
		count := 0
		for _, processorRow := range processors {
			filteredRow := []processorconfiguration.ProcessorConfiguration{}
			for _, p := range processorRow {
				if p.Async == async {
					filteredRow = append(filteredRow, p)
				}
			}
			if len(filteredRow) == 0 {
				continue
			}
			count++
			layer := pm(ctx, nc, filteredRow, s)

			previousMiddleware := m
			m = func(next http.Handler) http.Handler {
				return previousMiddleware(layer(next))
			}
		}
		return m, count
	}

	s := state{m: &sync.Mutex{}}

	// Build the SYNC requestProcessorMiddleware
	requestProcessorsMiddleware, countRequestSync := buildProcessorsMiddleware(false, requestProcessorMiddlewareFunc, globalConfig.RequestProcessors.Processors, &s)
	logger.Info("adding request processor layer", slog.Int("layer", countRequestSync))

	// Build the ASYNC requestProcessorMiddleware
	requestProcessorsMiddlewareAsync, countRequestAsync := buildProcessorsMiddleware(true, requestProcessorMiddlewareFunc, globalConfig.RequestProcessors.Processors, &s)
	logger.Info("adding async request processor layer", slog.Int("layer", countRequestAsync))

	// Build the SYNC responseProcessorMiddleware
	responseProcessorsMiddleware, countResponseSync := buildProcessorsMiddleware(false, responseProcessorMiddlewareFunc, globalConfig.ResponseProcessors.Processors, &s)
	logger.Info("adding response processor layer", slog.Int("layer", countResponseSync))

	// Build the ASYNC responseProcessorMiddleware
	responseProcessorsMiddlewareAsync, countResponseAsync := buildProcessorsMiddleware(true, responseProcessorMiddlewareFunc, globalConfig.ResponseProcessors.Processors, &s)
	logger.Info("adding async response processor layer", slog.Int("layer", countResponseAsync))

	loggingPostProcessorFunction := loggingPostProcessorFunc(
		nc,
		globalConfig.ServiceBusConfig.TopicExactLogger,
		globalConfig.setLastError,
		globalConfig.setLastTransactionID,
		requestProcessorsMiddlewareAsync(
			responseProcessorsMiddlewareAsync(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), // noop
			),
		),
		globalConfig.Logger,
		&s,
	)
	if loggingPostProcessorFunction == nil {
		logger.Error("error creating logging post processor")
		cancelTheContext()
		return
	}
	loggingMiddleware := loggingMiddlewareFunc(loggingPostProcessorFunction, globalConfig.LogUnauthorized)
	if loggingMiddleware == nil {
		logger.Error("error creating logging middleware")
		cancelTheContext()
		return
	}
	sessionMiddleware := sessionMiddlewareFunc(globalConfig.GatewayID, globalConfig.SessionIDHeader, &s)
	if sessionMiddleware == nil {
		logger.Error("error creating session middleware")
		cancelTheContext()
		return
	}
	standardRequestHandler := standardRequestFunc(globalConfig.client)
	if standardRequestHandler == nil {
		logger.Error("error creating request handler")
		cancelTheContext()
		return
	}
	echoRequestHandler := echoRequestFunc()

	// Start web service
	mux := http.NewServeMux()

	for _, currentRouter := range globalConfig.Routers {

		requestHandler := standardRequestHandler
		if currentRouter.Path == "/echo" || strings.HasPrefix(currentRouter.Path, "/echo/") {
			// Special case path
			requestHandler = echoRequestHandler
		}
		ingressEgressHeaderMiddleware := ingressEgressHeaderMiddlewareFunc(currentRouter.Ingress.Headers, globalConfig.removeHeadersMap, globalConfig.maskedHeadersMap, globalConfig.SessionIDHeader, &s)
		outboundQueryParametersMiddleware := outboundQueryParametersMiddlewareFunc(currentRouter.Outbound, &s)
		outboundInboundHeaderMiddleware := outboundInboundHeaderMiddlewareFunc(currentRouter.Outbound.Headers, globalConfig.removeHeadersMap, globalConfig.maskedHeadersMap, globalConfig.SessionIDHeader, &s)
		ingressPathMiddleware := ingressPathMiddlewareFunc(currentRouter, &s)
		outboundInboundPathMiddleware := outboundInboundPathMiddlewareFunc(currentRouter, globalConfig.Routers, &s)

		logger.Info("building router", slog.String("path", currentRouter.Path))
		// Build the handler
		handler := loggingMiddleware(
			sessionMiddleware(
				egressResponseMiddleware(
					ingressEgressPayloadMiddleware(
						ingressPathMiddleware(
							ingressEgressHeaderMiddleware(
								ingressQueryParametersMiddleware(
									requestProcessorsMiddleware(
										responseProcessorsMiddleware(
											outboundInboundPathMiddleware(
												outboundQueryParametersMiddleware(
													outboundInboundHeaderMiddleware(
														outboundInboundPayloadMiddleware(
															controlFieldMiddleware(
																requestHandler,
															),
														),
													),
												),
											),
										),
									),
								),
							),
						),
					),
				),
			),
		)

		mux.Handle(currentRouter.Path, handler)
		// If the path is /test/it/ then /test/it will return something bad request ish
		noSlash, noTrailingSlashHandler := noTrailingSlashHandlerFunc(currentRouter.Path)
		mux.HandleFunc(noSlash, noTrailingSlashHandler)

		logger.Info(
			"adding router",
			slog.Group("router",
				slog.String("path", currentRouter.Path),
				slog.String("ingress", currentRouter.Ingress.String()),
				slog.String("egress", currentRouter.Outbound.String())),
		)
	}

	// Create a custom writer using the slog logger
	customWriter := &CustomWriter{logger: logger, supress: 1}

	// Create a custom log.Logger using the custom writer
	customLogger := log.New(customWriter, "", 0)

	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", globalConfig.Port),
		Handler:        mux,
		ErrorLog:       customLogger,
		ReadTimeout:    200 * time.Second,
		WriteTimeout:   200 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {

		if !globalConfig.TlsUserConfig.Ingress.Enabled {
			err := httpServer.ListenAndServe()
			if err != nil {
				logger.Error("error starting http server", slog.Any("error", err))
				cancelTheContext()
			}
			return
		}
		err := httpServer.ListenAndServeTLS(globalConfig.TlsUserConfig.Ingress.CertificateFile, globalConfig.TlsUserConfig.Ingress.PrivateKeyFile)
		if err != nil {
			logger.Error("error starting https server", slog.Any("error", err))
			cancelTheContext()
		}

	}()

	logger.Info(
		"http(s) server initialized",
	)

	_ = nc.Publish(globalConfig.ServiceBusConfig.Topic, []byte("HTTP(s) server listening"))

	var newLogLevel slog.LevelVar
	err = json.Unmarshal([]byte("\""+globalConfig.LogLevel+"\""), &newLogLevel)

	if err == nil && newLogLevel != *logLevel {

		logger.Info(
			"log level changed",
			slog.String("log_level", newLogLevel.Level().String()),
		)
		logLevel.Set(newLogLevel.Level())

	}
	// Start Isalive service
	sub, err := nc.Subscribe(globalConfig.ServiceBusConfig.TopicExactIsAlive, func(msg *nats.Msg) {

		response := struct {
			Status int
			Body   isAliveBody
			Error  error
		}{
			Status: http.StatusOK,
			Body:   globalConfig.IsAliveData,
			Error:  nil,
		}

		byteString, _ := json.Marshal(&response)
		nc.Publish(msg.Reply, byteString)
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

	healthyChecksumHandler(ctx, cancelTheContext) // Write the checksum to a file, IE we are healthy

	<-ctx.Done()

}

// ----------- Populates configuration, basic checks -----------------

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

func updateConfiguration(config string, g *gl_config) error {
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
			legacy := gl_config_v0921{}
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
		<-ctx.Done()
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

	/*logger.Service = thisBinary
	if serviceAlias != thisBinary {
		logger.Service = serviceAlias
	}*/
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

	validRouters := []router.Router{}
	rejectedFields := validate.ValidationErrors{}
	for index, candidateRouter := range globalConfig.Routers {
		e := candidateRouter.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.Routers[%d].%s", CONFIG_NAME, index, k)] = v
			}

			continue
		}
		validRouters = append(validRouters, candidateRouter)
	}
	globalConfig.Routers = validRouters

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

	// Populate maskedHeadersMap
	globalConfig.maskedHeadersMap = map[string]struct{}{}
	for _, header := range globalConfig.MaskedHeaders {
		globalConfig.maskedHeadersMap[header] = struct{}{}
	}

	// Populate removeHeadersMap
	globalConfig.removeHeadersMap = map[string]struct{}{}
	for _, header := range globalConfig.RemoveHeaders {
		globalConfig.removeHeadersMap[header] = struct{}{}
	}
	protectedheader.AddMaskedHeadersOnce(globalConfig.MaskedHeaders)

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

	// Binary specific setup
	// ...

	err = func() error {
		// TLS outbound setup
		if globalConfig.TlsUserConfig.Outbound.InsecureFlag {
			// Skip TLS verification
			globalConfig.client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}}
			return nil
		}

		caCertPool := x509.NewCertPool()
		if globalConfig.TlsUserConfig.Outbound.SystemCertPoolFlag {
			// Load system cert pool
			systemCertPool, err := x509.SystemCertPool()
			if err != nil {
				return fmt.Errorf("config: Error loading system cert pool: %v", err)
			}
			caCertPool = systemCertPool
		}

		// Load custom CA certificates
		for _, filename := range globalConfig.TlsUserConfig.Outbound.CertFiles {
			caCert, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("config: Error reading CA certificate file: %v", err)
			}
			caCertPool.AppendCertsFromPEM(caCert)
		}

		globalConfig.tlsConfig = &tls.Config{
			RootCAs: caCertPool,
		}

		tr := &http.Transport{
			TLSClientConfig: globalConfig.tlsConfig,
		}
		globalConfig.client = &http.Client{Transport: tr}
		return nil
	}()
	if err != nil {
		logger.Error(
			"error setting up TLS outbound",
			slog.Any("error", err),
		)

		return fmt.Errorf("config: Error setting up TLS: %v", err)
	}

	// tls inbound setup
	if globalConfig.TlsUserConfig.Ingress.Enabled {
		// Check if the certificate and key files exist and are readable
		file, err := os.Open(globalConfig.TlsUserConfig.Ingress.CertificateFile)
		if err != nil {
			logger.Error("cannot read certificate file", slog.Any("error", err), slog.String("file", globalConfig.TlsUserConfig.Ingress.CertificateFile))
			return fmt.Errorf("cannot read certificate file: %v", err)
		}
		file.Close()
		file, err = os.Open(globalConfig.TlsUserConfig.Ingress.PrivateKeyFile)
		if err != nil {
			logger.Error("cannot read private key file", slog.Any("error", err), slog.String("file", globalConfig.TlsUserConfig.Ingress.PrivateKeyFile))
			return fmt.Errorf("cannot read private key file: %v", err)
		}
		file.Close()
	}

	// Find if we have any required processors
	requiredProcessorFlag := false
	for _, list := range globalConfig.RequestProcessors.Processors {
		for _, p := range list {
			if p.Required {
				requiredProcessorFlag = true
			}
		}
	}
	for _, list := range globalConfig.ResponseProcessors.Processors {
		for _, p := range list {
			if p.Required {
				requiredProcessorFlag = true
			}
		}
	}

	if requiredProcessorFlag {
		logger.Warn(
			"required processor(s) found. If not completed, log will be discarded",
		)

	}

	return nil
}

// ----------- Thin main. Setups context, catches crtl-c  -----------------

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

package main

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/direktoren/gecholog/internal/glconfig"
	"github.com/direktoren/gecholog/internal/validate"
	"github.com/nats-io/nats.go"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	sloggin "github.com/samber/slog-gin"
)

// ------------------------------- GLOBALS --------------------------------

const (
	CONFIG_NAME = "gui_config"
)

var globalConfig = gui_config{}
var thisBinary = "gui"
var version string
var logLevel = &slog.LevelVar{} // INFO
var opts = &slog.HandlerOptions{
	AddSource: true,
	Level:     logLevel,
}
var logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))

// ------------------------------- GUI CONFIGURATION --------------------------------

type configuration struct {
	Name                   string   `json:"name" validate:"required,ascii,min=2"`
	ProductionFile         string   `json:"production_file" validate:"required,file"`
	ProductionChecksumFile string   `json:"production_checksum_file" validate:"required"`
	TemplateFile           string   `json:"template_file" validate:"required,file"`
	ValidationExecutable   string   `json:"validation_executable" validate:"required,file"`
	ConfigCommand          string   `json:"config_command" validate:"required,ascii,min=1"`
	ValidateCommand        string   `json:"validate_command" validate:"required,ascii,min=1"`
	AdditionalArguments    []string `json:"additional_arguments" validate:"dive,ascii,min=1"`
}

func (p configuration) String() string {
	return fmt.Sprintf("name:%s production_file:%s default_file:%s validation_executable:%s config_command:%s validate_command:%s additional_arguments:%v", p.Name, p.ProductionFile, p.TemplateFile, p.ValidationExecutable, p.ConfigCommand, p.ValidateCommand, p.AdditionalArguments)
}

type tlsConfig struct {
	Ingress struct {
		Enabled         bool   `json:"enabled"`
		CertificateFile string `json:"certificate_file" validate:"required_if=Enabled true|file"`
		PrivateKeyFile  string `json:"private_key_file" validate:"required_if=Enabled true|file"`
	} `json:"ingress"`
}

func (t tlsConfig) String() string {
	s := fmt.Sprintf("ingress:{%s} ", fmt.Sprintf("enabled:%v certificate_file:%s private_key_file:%s ", t.Ingress.Enabled, t.Ingress.CertificateFile, t.Ingress.PrivateKeyFile))
	return s
}

type serviceBusConfig struct {
	Hostname         string `json:"hostname" validate:"required,hostname_port"`
	TopicExactLogger string `json:"topic_exact_logger" validate:"required,nefield=Topic,alphanumdot"`
	Token            string `json:"token" validate:"required,ascii"`
}

func (s serviceBusConfig) String() string {
	str := fmt.Sprintf("hostname:%s topic_logger_exact:%s", s.Hostname, s.TopicExactLogger)
	if s.Token != "" {
		str += " token:[*****MASKED*****]"
	}
	return str

}

type gui_config struct {
	// From json config file
	Version  string `json:"version" validate:"required,semver"`
	LogLevel string `json:"log_level" validate:"required,oneof=DEBUG INFO WARN ERROR"`

	Secret                    string `json:"secret" validate:"required,ascii,min=6"`
	FailedAuthenticationLimit int    `json:"failed_authentication_limit" validate:"min=0"`

	TlsConfig tlsConfig `json:"tls"`

	Port int `json:"gui_port" validate:"min=1,max=65535"`

	WorkingDirectory string `json:"working_directory" validate:"required,dir"`
	ArchiveDirectory string `json:"archive_directory" validate:"required,dir"`

	ServiceBusConfig serviceBusConfig `json:"service_bus" validate:"required"`

	GlConfiguration        configuration `json:"gl" validate:"required"`
	Nats2LogConfiguration  configuration `json:"nats2log" validate:"required"`
	Nats2FileConfiguration configuration `json:"nats2file" validate:"required"`
	//	TokenCounterConfiguration configuration `json:"token_counter"`

	sha256       string
	checksumFile string
}

func (c *gui_config) String() string {
	// Prints the part of the configuration that is relevant to the user.
	str := fmt.Sprintf(
		"version:%s "+
			"log_level:%v "+
			"secret:************"+
			"tls:%s "+
			"gui_port:%d "+
			"working_directory:%s "+
			"archive_directory:%s "+
			"service_bus:%s "+
			"gl:%s "+
			"nats2log:%s "+
			"nats2file:%s ",
		c.Version,
		c.LogLevel,
		c.TlsConfig.String(),
		c.Port,
		c.WorkingDirectory,
		c.ArchiveDirectory,
		c.ServiceBusConfig.String(),
		c.GlConfiguration.String(),
		c.Nats2LogConfiguration.String(),
		c.Nats2FileConfiguration.String())
	return str
}

func (c *gui_config) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, c)
}

// ------------------------------ app State  ------------------------------

type appState struct {
	current string
	m       sync.Mutex

	// Session
	activeSessionID              string
	basePath                     string
	lastActivity                 time.Time
	failedAuthenticationRequests int
}

const (
	GL_CONFIG        = "gl"
	NATS2LOG_CONFIG  = "nats2log"
	NATS2FILE_CONFIG = "nats2file"
	NONE             = ""
)

var state = appState{current: NONE}

func (a *appState) set(s string) {
	a.m.Lock()
	defer a.m.Unlock()
	a.current = s
}

func (a *appState) get() string {
	a.m.Lock()
	defer a.m.Unlock()
	return a.current
}

// ------------------------------ Gui State  ------------------------------

type validationErrors struct {
	Service          string            `json:"service"`
	RejectedFields   map[string]string `json:"rejected_fields"`
	ValidationErrors map[string]string `json:"validation_errors"`
}

// Area, Object, Field, Value

type area struct {
	Headline string
	Key      string
	Redirect string
	Form     string
	Objects  []inputObject

	ErrorMsg            string
	ErrorMsgTooltipText string
}

type validationStrings struct {
	executable          string
	configCommand       string
	validateCommand     string
	additionalArguments []string
}

type misc struct {
	productionFile         string
	productionChecksumFile string
	prefix                 string
	templateFile           string
	headline               string
	filePattern            string
}

type validationFunction func(ctx context.Context) error

type webbAppState struct {
	config anyGechologConfig

	webMisc misc

	workingFile         string
	workingFileChecksum string

	validation validationStrings

	systemStatus string

	saved bool

	valid          bool
	rejectedFields bool

	validationErrors map[string]string
	validate         validationFunction

	areas []area

	tutorials tutorial
}

func validationFunc(w *webbAppState) validationFunction {
	return func(ctx context.Context) error {
		if w == nil {
			return fmt.Errorf("webbAppState is nil")
		}

		if w.config == nil {
			return fmt.Errorf("config is nil")
		}

		var err error
		if !w.saved {
			w.workingFileChecksum, err = w.config.writeConfigFile(w.workingFile)
			if err != nil {
				return err
			}
			w.saved = true
		}

		arguments := []string{w.validation.configCommand, w.workingFile, w.validation.validateCommand}
		arguments = append(arguments, w.validation.additionalArguments...)

		validateCmd := exec.Command(w.validation.executable, arguments...)

		// Create a buffer to capture both stdout and stderr
		// We use output buffer since the validation command finishes immediately
		var outputBuffer bytes.Buffer
		validateCmd.Stdout = &outputBuffer
		validateCmd.Stderr = &outputBuffer

		err = validateCmd.Start()
		if err != nil {
			return err
		}

		// Create a channel to signal when the command finishes
		done := make(chan error, 1)
		go func() {
			done <- validateCmd.Wait()
		}()

		validationErrorsRaw := []validationErrors{}

		select {
		case <-ctx.Done():
			w.valid = false
			w.systemStatus = "validate: context cancelled"
			return fmt.Errorf("context cancelled")
		case <-time.After(1 * time.Second):
			logger.Warn("stuck during configuration validation")
			err := validateCmd.Process.Kill()
			if err != nil {
				logger.Error("error killing validation command", slog.Any("error", err))
			}
			w.valid = false
			w.systemStatus = "validate: timeout"
			return fmt.Errorf("timeout")
		case cmdErr := <-done:
			line := outputBuffer.Bytes()
			jsonStrings := strings.Split(string(line), "\n")

			for _, jsonStr := range jsonStrings {
				if jsonStr == "" {
					continue
				}

				valError := validationErrors{}

				err = json.Unmarshal([]byte(jsonStr), &valError)
				if err != nil {
					w.valid = false
					w.systemStatus = "validate: internal error"
					return err
				}
				validationErrorsRaw = append(validationErrorsRaw, valError)
			}
			//logger.Debug("err from command", slog.Any("error", cmdErr))
			w.valid = true
			if cmdErr != nil {
				_, ok := cmdErr.(*exec.ExitError)
				//logger.Debug("*exec.ExitError", slog.Any("ok", ok))
				if !ok {
					w.valid = false
					w.systemStatus = "validate: cmd error"
					return cmdErr
				}
				w.valid = false
			}
		}

		w.rejectedFields = false
		w.validationErrors = make(map[string]string, 20)
		for _, validationError := range validationErrorsRaw {
			for key, value := range validationError.RejectedFields {
				w.validationErrors[key] = value
				w.rejectedFields = true
			}
			for key, value := range validationError.ValidationErrors {
				w.validationErrors[key] = value
			}
		}

		w.systemStatus = ""
		return nil
	}
}

func new(ctx context.Context, m misc, workingFile string, c anyGechologConfig, v validationStrings) (*webbAppState, error) {
	if workingFile == "" {
		return nil, fmt.Errorf("working file is empty")
	}
	if c == nil {
		return nil, fmt.Errorf("config is nil")
	}

	w := &webbAppState{
		config:      c,
		webMisc:     m,
		workingFile: workingFile,
		validation:  v,
		saved:       false,
		tutorials: tutorial{
			Steps: make(map[string]tutorialStep),
		},
	}

	var err error
	_, err = w.config.update()
	if err != nil {
		return nil, err
	}

	w.areas, err = w.config.createAreas(map[string]string{})
	if err != nil {
		return nil, err
	}

	w.workingFileChecksum, err = w.config.writeConfigFile(w.workingFile)
	if err != nil {
		return nil, err
	}

	// We do this to decouple the validation function from the webbAppState
	// This enable us to test the functions dependent of the webbAppState without need to run the real validate function
	w.validate = validationFunc(w)

	return w, nil
}

// ------------------------------ menu handler  ------------------------------

func menuFunc(ctx context.Context, w *webbAppState, menuIndex int) gin.HandlerFunc {
	return func(c *gin.Context) {

		if w == nil {
			logger.Error("w is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.webMisc.headline == "" {
			logger.Error("headline is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.webMisc.productionFile == "" {
			logger.Error("productionFile is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.validate == nil {
			logger.Error("validate is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.config == nil {
			logger.Error("config is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		if w.areas == nil {
			logger.Error("areas is nil")
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		tutorialIndex := c.Query("tutorial")
		tutorial := func() tutorialStep {
			if tutorialIndex == "" {
				return tutorialStep{}
			}

			t, err := w.tutorials.findStep(tutorialIndex, menuIndex, "menu.html")
			if err != nil {
				logger.Warn("error getting tutorial", slog.String("tutorialIndex", tutorialIndex), slog.Any("error", err), slog.String("form", "menu.html"))
				return tutorialStep{}
			}
			logger.Debug("tutorialIndex", slog.String("tutorialIndex", tutorialIndex), slog.Any("tutorial", t), slog.String("form", "menu.html"))
			return t
		}()

		status := map[string]string{}
		status["Headline"] = w.webMisc.headline
		status["ProductionFile"] = w.webMisc.productionFile

		err := w.validate(ctx)
		if err != nil {
			logger.Error("error validating", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		err = w.config.updateAreasFromConfig(w.validationErrors, w.areas)
		if err != nil {
			logger.Error("error updating areas", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}

		productionChecksum, err := glconfig.ReadFile(w.webMisc.productionChecksumFile)
		if err != nil {
			logger.Debug("error generating checksum for production file", slog.String("file", status["ProductionFile"]), slog.Any("error", err))
			productionChecksum = "N/A"
		}

		deployedChecksum, err := glconfig.GenerateChecksum(w.webMisc.productionFile)
		if err != nil {
			logger.Debug("error generating checksum for production file", slog.String("file", status["ProductionFile"]), slog.Any("error", err))
			deployedChecksum = "N/A"
		}

		status["ProductionChecksum"] = productionChecksum
		status["DeployedChecksum"] = deployedChecksum
		status["Deployed"] = "Match"
		status["DeployedChecksumFormat"] = "valid"
		if status["ProductionChecksum"] != status["DeployedChecksum"] {
			status["DeployedChecksumFormat"] = "error"
			status["Deployed"] = "Mismatch"
		}

		status["WorkingFile"] = w.workingFile
		status["WorkingChecksum"] = w.workingFileChecksum
		status["Staged"] = "Staged"
		status["StagedFormat"] = ""
		switch status["WorkingChecksum"] {
		case status["ProductionChecksum"]:
			if status["WorkingChecksum"] != status["DeployedChecksum"] {
				status["Staged"] = "In Production but not Deployed"
				status["StagedFormat"] = "error"
				break
			}
			status["Staged"] = "Deployed and in Production"
			status["StagedFormat"] = "valid"

		case status["DeployedChecksum"]:
			status["Staged"] = "Deployed but not in Production"
			status["StagedFormat"] = "error"

		}
		status["ExitCode"] = "Valid"
		status["ExitCodeFormat"] = "valid"
		if !w.valid {
			status["ExitCode"] = "Invalid"
			status["ExitCodeFormat"] = "error"
		}

		status["RejectedFields"] = "None"
		status["RejectedFieldsFormat"] = "valid"
		if w.rejectedFields {
			status["RejectedFields"] = "Rejected Fields exist"
			status["RejectedFieldsFormat"] = "error"
		}

		status["DeployButton"] = "disabled"
		if status["WorkingChecksum"] != status["DeployedChecksum"] && w.valid {
			status["DeployButton"] = "edit"
		}

		c.HTML(http.StatusOK, "menu.html", gin.H{
			"Status":   status,
			"Areas":    w.areas,
			"Tutorial": tutorial,
		})
	}
}

// ------------------------------ do  ------------------------------

var store = sessions.NewCookieStore([]byte("secret"))

func appStateMiddleware(appName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if state.get() != appName {
			c.String(http.StatusBadRequest, "Access denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

func staticCacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "static/favicon.ico" || c.Request.URL.Path == "static/logo_black_t.png" {
			c.Header("Cache-Control", "public, max-age=31536000")
		}
		c.Next()
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
		//	if strings.HasSuffix(string(p), " timeout\n") {
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

func findEndVarInString(s string) string {
	if strings.HasPrefix(s, "${") {
		if strings.HasSuffix(s, "}") {
			return s[2 : len(s)-1]
		}
	}
	return ""
}

func do(ctx context.Context, cancelTheContext context.CancelFunc) {

	// Create NATS client options
	natsEnabled := false
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
		natsEnabled = false
	}
	opts.ReconnectedCB = func(nc *nats.Conn) {
		logger.Info(
			"reconnected to NATS",
		)
		natsEnabled = true
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
		natsEnabled = false
		//jsonlogger.SystemLog(logger, jsonlogger.WARNING, str)
		//cancelTheContext()
	}
	nc, err := opts.Connect()
	natsEnabled = true

	//nc, err := nats.Connect(, nats.Token(globalConfig.ServiceBusConfig.Token))
	if err != nil {
		logger.Error(
			"issue connecting to nats-server",
			slog.Any("error", err),
		)
		//jsonlogger.SystemLog(logger, jsonlogger.ERROR, fmt.Sprintf("Error connecting to NATS: %v", err))
		//vLog(false, "Error connecting to NATS: %v", err)
		natsEnabled = false
	}

	logList := threadLogList{
		listOfLogs:  list.New(),
		m:           &sync.Mutex{},
		visibleLogs: []logRecord{},
		length:      10,
	}

	defer nc.Close()

	// Create & connect loggerSubrscriber
	loggerSubscriber, err := loggerSubscriberFunc(&logList, &natsEnabled)
	if err != nil {
		logger.Error(
			"error creating logger subscriber",
			slog.Any("error", err),
		)
		natsEnabled = false
	}
	if natsEnabled && loggerSubscriber != nil {
		logger.Info("enabling nats")
		sub, err := nc.Subscribe(globalConfig.ServiceBusConfig.TopicExactLogger, loggerSubscriber)
		if err != nil {
			logger.Error(
				"error subscribing to logger channel",
				slog.Any("error", err),
			)
			natsEnabled = false
		}
		defer sub.Unsubscribe()
	}

	Sessions := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			session, _ := store.Get(c.Request, name)
			c.Set("session", session)
			c.Next()
		}
	}

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   600,  // 600 seconds = 10 minutes
		HttpOnly: true, // Recommended for security to make the cookie HTTP-only
	}

	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()

	configSloggin := sloggin.Config{
		// We could adjust this logic based on the overall log level
		ClientErrorLevel: slog.LevelDebug,
		DefaultLevel:     slog.LevelDebug,
		ServerErrorLevel: slog.LevelDebug,
	}

	ginRouter.Use(sloggin.NewWithConfig(logger, configSloggin))
	ginRouter.Use(gin.Recovery())
	ginRouter.Use(Sessions("mysession"))

	ginRouter.SetFuncMap(template.FuncMap{
		"findEndVarInString": findEndVarInString,
	})

	ginRouter.LoadHTMLGlob("templates/*")
	ginRouter.Static("/static", "./static")

	ginRouter.Use(staticCacheControl())

	ginRouter.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "login")
	})

	ginRouter.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil) // Render a login form
	})

	ginRouter.POST("/login", loginPOSTHandlerFunc(ctx, cancelTheContext, &state))

	// Protected routes
	authorized := ginRouter.Group("/")
	authorized.Use(authRequiredFunc(&state))

	authorized.GET("/logout", logoutHandlerFunc(&state))

	authorized.GET("/mainmenu", func(c *gin.Context) {
		c.HTML(http.StatusOK, "mainmenu.html", gin.H{
			"NatsConnected": natsEnabled,
		})
	})

	authorized.GET("/logs", logsListenerPageFunc(&logList, &natsEnabled))

	// ------------------------------ gl init  ------------------------------

	workingFile := globalConfig.WorkingDirectory + "gl_config_tmp.json"
	GLConfig := Gl_config_v1001{}
	err = GLConfig.loadConfigFile(workingFile)
	logger.Info("trying to load working file", slog.String("file", workingFile))

	if err != nil {
		// Try with the production file
		logger.Info("trying to load production file", slog.String("file", globalConfig.GlConfiguration.ProductionFile))
		err = GLConfig.loadConfigFile(globalConfig.GlConfiguration.ProductionFile)
		if err != nil {
			// Try with the template file
			logger.Info("trying to load template file", slog.String("file", globalConfig.GlConfiguration.TemplateFile))
			err = GLConfig.loadConfigFile(globalConfig.GlConfiguration.TemplateFile)
			if err != nil {
				logger.Error("error loading gl config", slog.Any("error", err))
				cancelTheContext()
				return
			}
		}
	}

	v := validationStrings{
		executable:          globalConfig.GlConfiguration.ValidationExecutable,
		configCommand:       globalConfig.GlConfiguration.ConfigCommand,
		validateCommand:     globalConfig.GlConfiguration.ValidateCommand,
		additionalArguments: globalConfig.GlConfiguration.AdditionalArguments,
	}

	m := misc{

		productionFile:         globalConfig.GlConfiguration.ProductionFile,
		productionChecksumFile: globalConfig.GlConfiguration.ProductionChecksumFile,
		prefix:                 "gl_config_",
		templateFile:           globalConfig.GlConfiguration.TemplateFile,
		headline:               "Main Configuration - gl_config.json",
		filePattern:            globalConfig.ArchiveDirectory + "gl_config_*.json",
	}
	w_gl, err := new(ctx, m, workingFile, &GLConfig, v)

	if err != nil {
		logger.Error("error creating webbAppState", slog.Any("error", err))
		cancelTheContext()
		return
	}

	// Add tutorials
	err = addRouterTutorialGL(w_gl.tutorials)
	if err != nil {
		logger.Error("error adding router tutorial", slog.Any("error", err))
	}

	postFix_gl, err := getUniquePostfixFunc(w_gl.webMisc.filePattern, w_gl.webMisc.prefix)
	if err != nil {
		logger.Error("error getting unique postfix", slog.Any("error", err))
		cancelTheContext()
		return
	}

	// ------------------------------ nats2log init  ------------------------------

	workingFile = globalConfig.WorkingDirectory + "nats2log_config_tmp.json"
	NATS2LOGConfig := nats2log_config_v1001{}
	err = NATS2LOGConfig.loadConfigFile(workingFile)
	logger.Info("trying to load working file", slog.String("file", workingFile))

	if err != nil {
		// Try with the production file
		logger.Info("trying to load production file", slog.String("file", globalConfig.Nats2LogConfiguration.ProductionFile))
		err = NATS2LOGConfig.loadConfigFile(globalConfig.Nats2LogConfiguration.ProductionFile)
		if err != nil {
			// Try with the template file
			logger.Info("trying to load template file", slog.String("file", globalConfig.Nats2LogConfiguration.TemplateFile))
			err = NATS2LOGConfig.loadConfigFile(globalConfig.Nats2LogConfiguration.TemplateFile)
			if err != nil {
				logger.Error("error loading nats2log config", slog.Any("error", err))
				cancelTheContext()
				return
			}
		}
	}

	v = validationStrings{
		executable:          globalConfig.Nats2LogConfiguration.ValidationExecutable,
		configCommand:       globalConfig.Nats2LogConfiguration.ConfigCommand,
		validateCommand:     globalConfig.Nats2LogConfiguration.ValidateCommand,
		additionalArguments: globalConfig.Nats2LogConfiguration.AdditionalArguments,
	}

	m = misc{

		productionFile:         globalConfig.Nats2LogConfiguration.ProductionFile,
		productionChecksumFile: globalConfig.Nats2LogConfiguration.ProductionChecksumFile,
		prefix:                 "nats2log_config_",
		templateFile:           globalConfig.Nats2LogConfiguration.TemplateFile,
		headline:               "Logger - nats2log_config.json",
		filePattern:            globalConfig.ArchiveDirectory + "nats2log_config_*.json",
	}
	w_nats2log, err := new(ctx, m, workingFile, &NATS2LOGConfig, v)

	if err != nil {
		logger.Error("error creating webbAppState", slog.Any("error", err))
		cancelTheContext()
		return
	}

	postFix_nats2log, err := getUniquePostfixFunc(w_nats2log.webMisc.filePattern, w_nats2log.webMisc.prefix)
	if err != nil {
		logger.Error("error getting unique postfix", slog.Any("error", err))
		cancelTheContext()
		return
	}

	// ------------------------------ nats2file init  ------------------------------

	workingFile = globalConfig.WorkingDirectory + "nats2file_config_tmp.json"
	NATS2FILEConfig := nats2log_config_v1001{}
	err = NATS2FILEConfig.loadConfigFile(workingFile)
	logger.Info("trying to load working file", slog.String("file", workingFile))

	if err != nil {
		// Try with the production file
		logger.Info("trying to load production file", slog.String("file", globalConfig.Nats2FileConfiguration.ProductionFile))
		err = NATS2FILEConfig.loadConfigFile(globalConfig.Nats2FileConfiguration.ProductionFile)
		if err != nil {
			// Try with the template file
			logger.Info("trying to load template file", slog.String("file", globalConfig.Nats2FileConfiguration.TemplateFile))
			err = NATS2FILEConfig.loadConfigFile(globalConfig.Nats2FileConfiguration.TemplateFile)
			if err != nil {
				logger.Error("error loading nats2log config", slog.Any("error", err))
				cancelTheContext()
				return
			}
		}
	}

	v = validationStrings{
		executable:          globalConfig.Nats2FileConfiguration.ValidationExecutable,
		configCommand:       globalConfig.Nats2FileConfiguration.ConfigCommand,
		validateCommand:     globalConfig.Nats2FileConfiguration.ValidateCommand,
		additionalArguments: globalConfig.Nats2FileConfiguration.AdditionalArguments,
	}

	m = misc{

		productionFile:         globalConfig.Nats2FileConfiguration.ProductionFile,
		productionChecksumFile: globalConfig.Nats2FileConfiguration.ProductionChecksumFile,
		prefix:                 "nats2file_config_",
		templateFile:           globalConfig.Nats2FileConfiguration.TemplateFile,
		headline:               "Logger - nats2file_config.json",
		filePattern:            globalConfig.ArchiveDirectory + "nats2file_config_*.json",
	}
	w_nats2file, err := new(ctx, m, workingFile, &NATS2FILEConfig, v)

	if err != nil {
		logger.Error("error creating webbAppState", slog.Any("error", err))
		cancelTheContext()
		return
	}

	// Add tutorials
	err = addRouterTutorialNats2file(w_nats2file.tutorials)
	if err != nil {
		logger.Error("error adding router tutorial", slog.Any("error", err))
	}

	postFix_nats2file, err := getUniquePostfixFunc(w_nats2file.webMisc.filePattern, w_nats2file.webMisc.prefix)
	if err != nil {
		logger.Error("error getting unique postfix", slog.Any("error", err))
		cancelTheContext()
		return
	}

	// Endpoint to switch apps
	authorized.POST("/select", func(c *gin.Context) {
		app := c.PostForm("app")
		path := c.PostForm("path")
		tutorial := c.PostForm("tutorial")
		logger.Debug("select app", slog.String("app", app), slog.String("path", path), slog.String("tutorial", tutorial))
		switch app {
		case GL_CONFIG:

			state.set(GL_CONFIG)

		case NATS2LOG_CONFIG:

			state.set(NATS2LOG_CONFIG)

		case NATS2FILE_CONFIG:

			state.set(NATS2FILE_CONFIG)

		default:
			state.set(NONE)
			c.String(http.StatusBadRequest, "Bad request")
			return
		}
		if path != "" {
			u, err := url.Parse(state.basePath + "/" + app + "/" + path)
			if err != nil {
				logger.Error("error parsing URL", slog.String("path", path), slog.Any("error", err), slog.String("basePath", state.basePath))
				c.String(http.StatusBadRequest, "Bad request")
				return
			}
			if tutorial != "" {
				q := u.Query()
				q.Set("tutorial", tutorial)
				u.RawQuery = q.Encode()
			}
			c.Redirect(http.StatusFound, u.String())
			return
		}

		c.Redirect(http.StatusFound, state.basePath+"/"+app+"/menu")
	})

	// ------------------------------ submit & form handlers ------------------------------

	gl := authorized.Group(GL_CONFIG)
	gl.Use(appStateMiddleware(GL_CONFIG))
	{
		gl.GET(
			"settings-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_SETTINGS,
				"form.html",
				"settings-submit",
				"settings-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.POST(
			"settings-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_SETTINGS,
				"settings-form",
				"settings-form",
				updateObjectsSubmitHandler,
			),
		)

		gl.GET(
			"tls-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_TLS,
				"form.html",
				"tls-submit",
				"tls-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.POST(
			"tls-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_TLS,
				"tls-form",
				"tls-form",
				updateObjectsSubmitHandler,
			),
		)

		gl.GET(
			"servicebus-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_SERVICEBUSCONFIG,
				"form.html",
				"servicebus-submit",
				"servicebus-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.POST(
			"servicebus-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_SERVICEBUSCONFIG,
				"servicebus-form",
				"servicebus-form",
				updateObjectsSubmitHandler,
			),
		)

		gl.GET(
			"routers",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_ROUTERS,
				"routers.html",
				"routers",
				"routers",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.POST(
			"allrouters-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_ROUTERS,
				"routers",
				"routers",
				updateAllRouterPathsSubmitHandler,
			),
		)

		gl.GET(
			"routers-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_ROUTERS,
				"form.html",
				"routers-submit",
				"routers-form",
				"routers",
				populateRouterObjectsFormHandler,
			),
		)

		gl.POST(
			"routers-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_ROUTERS,
				"routers-form",
				"routers",
				updateRouterObjectSubmitHandler,
			),
		)

		gl.GET(
			"requestprocessors",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_REQUESTPROCESSORS,
				"processors.html",
				"requestprocessors",
				"requestprocessors",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.GET(
			"/requestprocessors-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_REQUESTPROCESSORS,
				"form.html",
				"requestprocessors-submit",
				"requestprocessors-form",
				"requestprocessors",
				populateProcessorObjectsFormHandler,
			),
		)

		gl.POST(
			"requestprocessors-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_REQUESTPROCESSORS,
				"requestprocessors-form",
				"requestprocessors",
				updateProcessorObjectSubmitHandler,
			),
		)

		gl.GET(
			"responseprocessors",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_RESPONSEPROCESSORS,
				"processors.html",
				"responseprocessors",
				"responseprocessors",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.GET(
			"responseprocessors-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_RESPONSEPROCESSORS,
				"form.html",
				"responseprocessors-submit",
				"responseprocessors-form",
				"responseprocessors",
				populateProcessorObjectsFormHandler,
			),
		)

		gl.POST(
			"responseprocessors-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_RESPONSEPROCESSORS,
				"responseprocessors-form",
				"responseprocessors",
				updateProcessorObjectSubmitHandler,
			),
		)

		gl.GET(
			"logger-form",
			wrapFormHandlerFunc(
				ctx,
				w_gl,
				GL_CONFIG_LOGGER,
				"form.html",
				"logger-submit",
				"logger-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		gl.POST(
			"logger-submit",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_LOGGER,
				"logger-form",
				"logger-form",
				updateObjectsSubmitHandler,
			),
		)

		gl.GET(
			"routers-new",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_ROUTERS,
				"routers",
				"routers",
				addNewRouterSubmitHandler,
			),
		)

		gl.GET(
			"routers-copy",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_ROUTERS,
				"routers",
				"routers",
				copyRouterSubmitHandler,
			),
		)

		gl.GET(
			"routers-delete",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_ROUTERS,
				"routers",
				"routers",
				deleteRouterSubmitHandler,
			),
		)

		gl.GET(
			"requestprocessors-newparallel",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_REQUESTPROCESSORS,
				"requestprocessors",
				"requestprocessors",
				newParallelProcessorSubmitHandler,
			),
		)

		gl.GET(
			"requestprocessors-delete",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_REQUESTPROCESSORS,
				"requestprocessors",
				"requestprocessors",
				deleteProcessorSubmitHandler,
			),
		)

		gl.GET(
			"requestprocessors-newsequence",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_REQUESTPROCESSORS,
				"requestprocessors",
				"requestprocessors",
				newSequenceProcessorSubmitHandler,
			),
		)

		gl.GET(
			"responseprocessors-newparallel",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_RESPONSEPROCESSORS,
				"responseprocessors",
				"responseprocessors",
				newParallelProcessorSubmitHandler,
			),
		)

		gl.GET(
			"responseprocessors-delete",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_RESPONSEPROCESSORS,
				"responseprocessors",
				"responseprocessors",
				deleteProcessorSubmitHandler,
			),
		)

		gl.GET(
			"responseprocessors-newsequence",
			wrapSubmitHandlerFunc(
				w_gl,
				GL_CONFIG_RESPONSEPROCESSORS,
				"responseprocessors",
				"responseprocessors",
				newSequenceProcessorSubmitHandler,
			),
		)

		// ------------------------------ file handlers ------------------------------

		gl.GET(
			"archive-workingfile",
			archiveFormHandler(
				w_gl,
				"write-workingfile",
				"Archive Working File",
				"menu",
				postFix_gl,
				MENU_GL_INDEX,
			),
		)

		gl.POST(
			"write-workingfile",
			fileWriteHandlerFunc(
				w_gl,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return globalConfig.ArchiveDirectory + w_gl.webMisc.prefix + filename + ".json", nil
				},
			),
		)

		gl.GET(
			"select-file-to-open",
			wrapFileListHandlerFunc(
				w_gl,
				"open.html",
				"select-file-to-open",
				"menu",
				selectFileToOpenMetaHandler,
				MENU_GL_INDEX,
			),
		)

		gl.POST(
			"read-file",
			readFileHandlerFunc(
				w_gl,
				cancelTheContext,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return filename, nil
				},
			),
		)

		gl.GET(
			"publish",
			wrapFileListHandlerFunc(
				w_gl,
				"publish.html",
				"publish",
				"menu",
				checkIfProductionFileIsArchivedHandler,
				MENU_GL_INDEX,
			),
		)

		gl.GET(
			"archive-productionfile",
			archiveFormHandler(
				w_gl,
				"copy-productionfile",
				"Archive Production File",
				"publish",
				postFix_gl,
				MENU_GL_INDEX,
			),
		)

		gl.POST(
			"copy-productionfile",
			copyFileHandlerFunc(
				w_gl,
				"publish",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return globalConfig.ArchiveDirectory + w_gl.webMisc.prefix + filename + ".json", nil
				},
				fileCopy,
			),
		)

		gl.POST(
			"write-to-production",
			fileWriteHandlerFunc(
				w_gl,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					if filename != w_gl.webMisc.productionFile {
						return "", fmt.Errorf("filename does not match production file: %s != %s", filename, w_gl.webMisc.productionFile)
					}
					return filename, nil
				},
			),
		)

		// ------------------------------ menu ------------------------------

		gl.GET("menu", menuFunc(ctx, w_gl, MENU_GL_INDEX))

	}

	// ------------------------------ nats2log handlers ------------------------------

	nats2log := authorized.Group(NATS2LOG_CONFIG)
	nats2log.Use(appStateMiddleware(NATS2LOG_CONFIG))
	{
		nats2log.GET(
			"settings-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_SETTINGS,
				"form.html",
				"settings-submit",
				"settings-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"settings-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_SETTINGS,
				"settings-form",
				"settings-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2log.GET(
			"tls-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_TLS,
				"form.html",
				"tls-submit",
				"tls-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"tls-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_TLS,
				"tls-form",
				"tls-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2log.GET(
			"servicebus-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_SERVICEBUSCONFIG,
				"form.html",
				"servicebus-submit",
				"servicebus-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"servicebus-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_SERVICEBUSCONFIG,
				"servicebus-form",
				"servicebus-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2log.GET(
			"filewriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_FILEWRITER,
				"form.html",
				"filewriter-submit",
				"filewriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"filewriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_FILEWRITER,
				"filewriter-form",
				"filewriter-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2log.GET(
			"elasticwriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_ELASTICWRITER,
				"form.html",
				"elasticwriter-submit",
				"elasticwriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"elasticwriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_ELASTICWRITER,
				"elasticwriter-form",
				"elasticwriter-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2log.GET(
			"azureloganalyticswriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER,
				"form.html",
				"azureloganalyticswriter-submit",
				"azureloganalyticswriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"azureloganalyticswriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER,
				"azureloganalyticswriter-form",
				"azureloganalyticswriter-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2log.GET(
			"restapiwriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2log,
				NATS2LOG_CONFIG_RESTAPIWRITER,
				"form.html",
				"restapiwriter-submit",
				"restapiwriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2log.POST(
			"restapiwriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2log,
				NATS2LOG_CONFIG_RESTAPIWRITER,
				"restapiwriter-form",
				"restapiwriter-form",
				updateObjectsSubmitHandler,
			),
		)

		// ------------------------------ file handlers ------------------------------

		nats2log.GET(
			"archive-workingfile",
			archiveFormHandler(
				w_nats2log,
				"write-workingfile",
				"Archive Working File",
				"menu",
				postFix_nats2log,
				MENU_NATS2LOG_INDEX,
			),
		)

		nats2log.POST(
			"write-workingfile",
			fileWriteHandlerFunc(
				w_nats2log,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return globalConfig.ArchiveDirectory + w_nats2log.webMisc.prefix + filename + ".json", nil
				},
			),
		)

		nats2log.GET(
			"select-file-to-open",
			wrapFileListHandlerFunc(
				w_nats2log,
				"open.html",
				"select-file-to-open",
				"menu",
				selectFileToOpenMetaHandler,
				MENU_NATS2LOG_INDEX,
			),
		)

		nats2log.POST(
			"read-file",
			readFileHandlerFunc(
				w_nats2log,
				cancelTheContext,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return filename, nil
				},
			),
		)

		nats2log.GET(
			"publish",
			wrapFileListHandlerFunc(
				w_nats2log,
				"publish.html",
				"publish",
				"menu",
				checkIfProductionFileIsArchivedHandler,
				MENU_NATS2LOG_INDEX,
			),
		)

		nats2log.GET(
			"archive-productionfile",
			archiveFormHandler(
				w_nats2log,
				"copy-productionfile",
				"Archive Production File",
				"publish",
				postFix_nats2log,
				MENU_NATS2LOG_INDEX,
			),
		)

		nats2log.POST(
			"copy-productionfile",
			copyFileHandlerFunc(
				w_nats2log,
				"publish",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return globalConfig.ArchiveDirectory + w_nats2log.webMisc.prefix + filename + ".json", nil
				},
				fileCopy,
			),
		)

		nats2log.POST(
			"write-to-production",
			fileWriteHandlerFunc(
				w_nats2log,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					if filename != w_nats2log.webMisc.productionFile {
						return "", fmt.Errorf("filename does not match production file: %s != %s", filename, w_nats2log.webMisc.productionFile)
					}
					return filename, nil
				},
			),
		)

		nats2log.GET("menu", menuFunc(ctx, w_nats2log, MENU_NATS2LOG_INDEX))
	}

	// ------------------------------ nats2file handlers ------------------------------

	nats2file := authorized.Group(NATS2FILE_CONFIG)
	nats2file.Use(appStateMiddleware(NATS2FILE_CONFIG))
	{
		nats2file.GET(
			"settings-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_SETTINGS,
				"form.html",
				"settings-submit",
				"settings-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"settings-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_SETTINGS,
				"settings-form",
				"settings-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2file.GET(
			"tls-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_TLS,
				"form.html",
				"tls-submit",
				"tls-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"tls-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_TLS,
				"tls-form",
				"tls-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2file.GET(
			"servicebus-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_SERVICEBUSCONFIG,
				"form.html",
				"servicebus-submit",
				"servicebus-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"servicebus-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_SERVICEBUSCONFIG,
				"servicebus-form",
				"servicebus-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2file.GET(
			"filewriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_FILEWRITER,
				"form.html",
				"filewriter-submit",
				"filewriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"filewriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_FILEWRITER,
				"filewriter-form",
				"filewriter-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2file.GET(
			"elasticwriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_ELASTICWRITER,
				"form.html",
				"elasticwriter-submit",
				"elasticwriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"elasticwriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_ELASTICWRITER,
				"elasticwriter-form",
				"elasticwriter-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2file.GET(
			"azureloganalyticswriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER,
				"form.html",
				"azureloganalyticswriter-submit",
				"azureloganalyticswriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"azureloganalyticswriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER,
				"azureloganalyticswriter-form",
				"azureloganalyticswriter-form",
				updateObjectsSubmitHandler,
			),
		)

		nats2file.GET(
			"restapiwriter-form",
			wrapFormHandlerFunc(
				ctx,
				w_nats2file,
				NATS2LOG_CONFIG_RESTAPIWRITER,
				"form.html",
				"restapiwriter-submit",
				"restapiwriter-form",
				"menu",
				func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
					// No filtering or selection needed
					return http.StatusOK, "", (*objectList), nil
				},
			),
		)

		nats2file.POST(
			"restapiwriter-submit",
			wrapSubmitHandlerFunc(
				w_nats2file,
				NATS2LOG_CONFIG_RESTAPIWRITER,
				"restapiwriter-form",
				"restapiwriter-form",
				updateObjectsSubmitHandler,
			),
		)

		// ------------------------------ file handlers ------------------------------

		nats2file.GET(
			"archive-workingfile",
			archiveFormHandler(
				w_nats2file,
				"write-workingfile",
				"Archive Working File",
				"menu",
				postFix_nats2file,
				MENU_NATS2FILE_INDEX,
			),
		)

		nats2file.POST(
			"write-workingfile",
			fileWriteHandlerFunc(
				w_nats2file,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return globalConfig.ArchiveDirectory + w_nats2file.webMisc.prefix + filename + ".json", nil
				},
			),
		)

		nats2file.GET(
			"select-file-to-open",
			wrapFileListHandlerFunc(
				w_nats2file,
				"open.html",
				"select-file-to-open",
				"menu",
				selectFileToOpenMetaHandler,
				MENU_NATS2FILE_INDEX,
			),
		)

		nats2file.POST(
			"read-file",
			readFileHandlerFunc(
				w_nats2file,
				cancelTheContext,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return filename, nil
				},
			),
		)

		nats2file.GET(
			"publish",
			wrapFileListHandlerFunc(
				w_nats2file,
				"publish.html",
				"publish",
				"menu",
				checkIfProductionFileIsArchivedHandler,
				MENU_NATS2FILE_INDEX,
			),
		)

		nats2file.GET(
			"archive-productionfile",
			archiveFormHandler(
				w_nats2file,
				"copy-productionfile",
				"Archive Production File",
				"publish",
				postFix_nats2file,
				MENU_NATS2FILE_INDEX,
			),
		)

		nats2file.POST(
			"copy-productionfile",
			copyFileHandlerFunc(
				w_nats2file,
				"publish",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					return globalConfig.ArchiveDirectory + w_nats2file.webMisc.prefix + filename + ".json", nil
				},
				fileCopy,
			),
		)

		nats2file.POST(
			"write-to-production",
			fileWriteHandlerFunc(
				w_nats2file,
				"menu",
				func(filename string) (string, error) {
					if filename == "" {
						return "", fmt.Errorf("filename is empty")
					}
					if filename != w_nats2file.webMisc.productionFile {
						return "", fmt.Errorf("filename does not match production file: %s != %s", filename, w_nats2file.webMisc.productionFile)
					}
					return filename, nil
				},
			),
		)

		nats2file.GET("menu", menuFunc(ctx, w_nats2file, MENU_NATS2FILE_INDEX))
	}

	// Create a custom writer using the slog logger
	customWriter := &CustomWriter{logger: logger, supress: 1}

	// Create a custom log.Logger using the custom writer
	customLogger := log.New(customWriter, "", 0)

	httpServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", globalConfig.Port),
		Handler:        ginRouter,
		ErrorLog:       customLogger,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {

		if !globalConfig.TlsConfig.Ingress.Enabled {
			err := httpServer.ListenAndServe()
			if err != nil {
				logger.Error("error starting web server", slog.Any("error", err))
				cancelTheContext()
			}
			return
		}

		err := httpServer.ListenAndServeTLS(globalConfig.TlsConfig.Ingress.CertificateFile, globalConfig.TlsConfig.Ingress.PrivateKeyFile)
		if err != nil {
			logger.Error("error starting web server", slog.Any("error", err))
			cancelTheContext()
		}

	}()
	logger.Info(
		"web server up and running",
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

	logger.Debug("gl tutorials", slog.Any("tutorials", w_gl.tutorials))
	logger.Debug("nats2file tutorials", slog.Any("tutorials", w_nats2file.tutorials))
	logger.Debug("nats2log tutorials", slog.Any("tutorials", w_nats2log.tutorials))

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

func updateConfiguration(config string, g *gui_config) error {
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
			legacy := gui_config_v0921{}
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

	validationErrors := globalConfig.Validate()
	if validationErrors != nil {
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

	if validateFlag {
		// Validation of config successful, exit
		os.Exit(0)
	}

	globalConfig.checksumFile = "/app/checksum/." + serviceAlias + "_config.sha256"
	//globalConfig.checksumFile = serviceAlias + "_config.sha256"
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
	//jsonlogger.SystemLog(logger, jsonlogger.INFO, "Computed checksum for configuration file: "+globalConfig.sha256)
	// sha256sum config.json        # Linux
	// shasum -a 256 config.json	# Mac

	// tls inbound setup
	if globalConfig.TlsConfig.Ingress.Enabled {
		// Check if the certificate and key files exist and are readable
		file, err := os.Open(globalConfig.TlsConfig.Ingress.CertificateFile)
		if err != nil {
			logger.Error("cannot read certificate file", slog.Any("error", err), slog.String("file", globalConfig.TlsConfig.Ingress.CertificateFile))
			return fmt.Errorf("cannot read certificate file: %v", err)
		}
		file.Close()
		file, err = os.Open(globalConfig.TlsConfig.Ingress.PrivateKeyFile)
		if err != nil {
			logger.Error("cannot read private key file", slog.Any("error", err), slog.String("file", globalConfig.TlsConfig.Ingress.PrivateKeyFile))
			return fmt.Errorf("cannot read private key file: %v", err)
		}
		file.Close()
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

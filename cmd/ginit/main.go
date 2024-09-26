package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/direktoren/gecholog/internal/glconfig"
	"github.com/direktoren/gecholog/internal/validate"
)

// ------------------------------- GLOBALS --------------------------------

const (
	CONFIG_NAME = "ginit_config"
)

var globalConfig = ginit_config{}
var thisBinary = "ginit"
var version string
var outWriter = NewSafeWriter(os.Stdout)
var logLevel = &slog.LevelVar{} // INFO
var opts = &slog.HandlerOptions{
	AddSource: true,
	Level:     logLevel,
}
var outLogger = slog.New(slog.NewJSONHandler(outWriter, opts))
var logger *slog.Logger

// ------------------------------- DATA STRUCTURES --------------------------------

type serviceState int

const (
	NOTSTARTED serviceState = iota
	PENDING
	IDLE
	HEALTHY
)

type executable struct {
	Name                string   `json:"name" validate:"required,ascii,min=2"`
	File                string   `json:"file" validate:"required,file"`
	ConfigCommand       string   `json:"config_command" validate:"required,ascii,min=1"`
	ConfigurationFile   string   `json:"configuration_file" validate:"required,file"`
	AdditionalArguments []string `json:"additional_arguments" validate:"dive,ascii,min=1"`

	ValidateCommand string `json:"validate_command" validate:"required,ascii,min=1"`

	HealthyOutput string `json:"healthy_output" validate:"required,ascii,min=1"`

	DisableConfigFileMonitoring bool `json:"disable_config_file_monitoring"`
	DiePromise                  bool `json:"die_promise"`

	state  serviceState
	starts int
	cmd    *exec.Cmd
	//configFileModTime time.Time
	configFileLastSha256 string
	lastError            error
	lastModified         time.Time
	lastSize             int64
}

func (p executable) String() string {
	return fmt.Sprintf("name:%s file:%s config_command:%s configuration_file:%s additional_arguments:%v validate_command:%s healthy_output:%s disable_config_file_monitoring:%v die_promise:%v ", p.Name, p.File, p.ConfigCommand, p.ConfigurationFile, p.AdditionalArguments, p.ValidateCommand, p.HealthyOutput, p.DisableConfigFileMonitoring, p.DiePromise)
}

func (e executable) Validate() validate.ValidationErrors {
	v := validate.New()
	return validate.ValidateStruct(v, e)
}

type ginit_config struct {
	Version string `json:"version" validate:"required,semver"`

	LogLevel  string `json:"log_level" validate:"required,oneof=DEBUG INFO WARN ERROR"`
	MaxStarts int    `json:"max_starts" validate:"min=0"`

	Services []executable `json:"services" validate:"gt=0,unique=Name,unique=ConfigurationFile,dive"`

	sha256       string
	checksumFile string
}

func (c *ginit_config) Validate() validate.ValidationErrors {
	v := validate.New()
	return validate.ValidateStruct(v, c)
}

func (c *ginit_config) String() string {
	str := fmt.Sprintf("version:%s log_level:%v max_restarts:%d, services:[", c.Version, c.LogLevel, c.MaxStarts)
	for _, s := range c.Services {
		str += s.String()
	}
	return str + "]"
}

// SafeWriter wraps an io.Writer with a mutex to ensure thread-safe, concurrent writes.
type SafeWriter struct {
	w  io.Writer  // Embed io.Writer to write to
	mu sync.Mutex // Mutex to ensure exclusive access
}

// NewSafeWriter creates a new SafeWriter wrapping the given io.Writer.
func NewSafeWriter(w io.Writer) *SafeWriter {
	return &SafeWriter{w: w}
}

// Write writes data to the embedded io.Writer, ensuring thread safety using a mutex.
func (sw *SafeWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()         // Lock the mutex before writing
	defer sw.mu.Unlock() // Unlock the mutex after writing is done
	return sw.w.Write(p) // Write to the underlying io.Writer
}

func forwardLog(logLevel slog.Level, msg []byte, service string) error {
	if json.Valid(msg) {
		trimmed := bytes.TrimRight(msg, "\n")
		_, err := outWriter.Write(append(trimmed, '\n'))
		return err
	}
	fields := strings.Fields(string(msg))
	str := strings.Join(fields, " ")
	if str == "" {
		return nil
	}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	outLogger.Log(context.TODO(), logLevel, str, slog.String("service", service), slog.Group("source", slog.String("function", "forwardLog"), slog.String("file", file), slog.Int("line", line)))

	return nil
}

// ------------------------------- DO --------------------------------

// Do the checks, and spin up the processes
func do(ctx context.Context, cancelTheContext context.CancelFunc) {

	defer func() {
		// Terminate all services when the context is cancelled
		logger.Info(
			"terminating service...",
		)

		for index := range globalConfig.Services {
			service := &(globalConfig.Services[index])

			if service.state != HEALTHY {
				continue
			}
			if service.cmd.Process == nil || (service.cmd.ProcessState != nil && service.cmd.ProcessState.Exited()) {
				continue
			}
			if service.cmd != nil {
				service.cmd.Process.Kill()
			}
			if err := service.cmd.Process.Signal(os.Interrupt); err != nil {
				service.state = PENDING
				logger.Error(
					"problem sending SIGINT",
					slog.String("child", service.Name),
					slog.Any("error", err),
				)

				continue
			}
			service.cmd.Wait()
			logger.Info(
				"shutdown completed",
				slog.String("child", service.Name),
			)

		}
	}()

	// --------------------------------- SIGNALS ---------------------------------

	type event interface{}

	type programExited struct {
		index int
		pid   int
	}
	type programStart struct {
		index int
	}
	type gracefulShutdown struct {
		index   int
		restart bool
	}
	type configurationFileChanged struct {
		index int
	}

	type healthy struct {
		index int
	}

	type tick struct{}

	// Create event channel
	eventChan := make(chan event)

	// Function to start a process and return its output reader
	startProcess := func(name string, args ...string) (*exec.Cmd, io.ReadCloser, error) {
		cmd := exec.Command(name, args...)
		cmd.Env = os.Environ()

		// Create a pipe for combined stdout and stderr
		stdoutStderr, err := cmd.StdoutPipe()
		if err != nil {
			logger.Error(
				"error creating stdout pipe",
				slog.Any("error", err),
			)

			return nil, nil, err
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		return cmd, stdoutStderr, nil
	}

	// Function to handle the output of a process
	handleOutput := func(reader io.ReadCloser, cmd *exec.Cmd, processName string, eChan chan<- event, index int) {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Bytes()
			fields := bytes.Split(line, []byte("\n"))

			for _, field := range fields {
				forwardLog(slog.LevelInfo, field, processName)

			}
			healthyFlag := strings.Contains(string(line), globalConfig.Services[index].HealthyOutput)
			if healthyFlag {
				go func(i int) {
					eChan <- healthy{index: i}
				}(index)
			}

		}
		if err := scanner.Err(); err != nil {
			if cmd.Process != nil && cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
				logger.Error(
					"standard output stream error",
					slog.String("child", processName),
					slog.Any("error", err),
				)

			}

		}
		logger.Warn(
			"standard output stream closed",
			slog.String("child", processName),
		)

	}

	// Bootstrap the services
	for index := range globalConfig.Services {
		service := &(globalConfig.Services[index])

		// ------- Validate configuration file ------
		arguments := []string{service.ConfigCommand, service.ConfigurationFile, service.ValidateCommand}
		arguments = append(arguments, service.AdditionalArguments...)

		validateCmd := exec.Command(service.File, arguments...)
		validateCmd.Env = os.Environ()

		// Create a buffer to capture both stdout and stderr
		// We use output buffer since the validation command finishes immediately
		var outputBuffer bytes.Buffer
		validateCmd.Stdout = &outputBuffer
		validateCmd.Stderr = &outputBuffer

		// Run the validation command
		logger.Info(
			"validating configuration file",
			slog.String("child", service.Name),
		)

		// Store the initial checksum
		var err error
		service.configFileLastSha256, err = glconfig.GenerateChecksum(service.ConfigurationFile)
		if err != nil {
			logger.Error(
				"error reading checksum file",
				slog.String("child", service.Name),
				slog.Any("error", err),
			)

			cancelTheContext()
			return
		}

		// Info for polling
		fileInfo, err := os.Stat(service.ConfigurationFile)
		if err != nil {
			logger.Error(
				"error reading file info",
				slog.String("child", service.Name),
				slog.Any("error", err),
			)
			cancelTheContext()
			return
		}
		service.lastModified = fileInfo.ModTime()
		service.lastSize = fileInfo.Size()

		err = validateCmd.Start()
		if err != nil {
			logger.Error(
				"failed to start",
				slog.String("child", service.Name),
				slog.Any("error", err),
			)
			logger.Info(
				"idle",
				slog.String("child", service.Name),
			)

			service.state = IDLE
			continue
		}
		// Create a channel to signal when the command finishes
		done := make(chan error, 1)
		go func() {
			done <- validateCmd.Wait()
		}()

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			logger.Warn(
				"stuck during configuration validation",
				slog.String("child", service.Name),
			)

			err := validateCmd.Process.Kill()
			if err != nil {
				logger.Error(
					"failed to kill",
					slog.String("child", service.Name),
					slog.Any("error", err),
				)

			}
			logger.Info(
				"idle",
				slog.String("child", service.Name),
			)

			service.state = IDLE
			continue
		case err := <-done:
			if err != nil {
				_, ok := err.(*exec.ExitError)
				if ok {
					line := outputBuffer.Bytes()
					fields := bytes.Split(line, []byte("\n"))
					for _, field := range fields {
						forwardLog(slog.LevelInfo, field, service.Name)

					}
				}
				logger.Warn(
					"configuration file is invalid",
					slog.String("child", service.Name),
					slog.Any("error", err),
				)
				logger.Info(
					"idle",
					slog.String("child", service.Name),
				)

				service.state = IDLE
				continue
			}
		}

		// Configuration file is valid
		// Let's capture the output from the service
		line := outputBuffer.Bytes()
		fields := bytes.Split(line, []byte("\n"))
		for _, field := range fields {
			forwardLog(slog.LevelInfo, field, service.Name)

		}
		logger.Info(
			"configuration file is valid",
			slog.String("child", service.Name),
		)

		// ------- Start the service ------
		logger.Info(
			"initiating",
			slog.String("child", service.Name),
		)

		arguments = []string{service.ConfigCommand, service.ConfigurationFile}
		arguments = append(arguments, service.AdditionalArguments...)
		logger.Debug(
			"command arguments",
			slog.String("child", service.Name),
			slog.String("arguments", strings.Join(arguments, " ")),
		)

		cmd, stdOutstdErr, err := startProcess(service.File, arguments...)

		if err != nil {
			logger.Error(
				"failed to start",
				slog.String("child", service.Name),
				slog.Any("error", err),
			)

			service.state = IDLE
			continue
		}

		service.cmd = cmd
		service.starts = 1

		// Start the stdout pipe service in a goroutine
		go handleOutput(stdOutstdErr, cmd, service.Name, eventChan, index)

		go func(i int, cmdLocal *exec.Cmd) {
			// Monitor the process and send event if it dies
			cmdLocal.Wait()
			eventChan <- programExited{index: i, pid: cmdLocal.Process.Pid}
		}(index, cmd)

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			logger.Warn(
				"stuck in pending",
				slog.String("child", service.Name),
			)

			service.state = PENDING
			// Stop the service, and kill if there is a die promise

		case e := <-eventChan:
			switch e.(type) {
			case programExited:
				logger.Error(
					"exited",
					slog.String("child", service.Name),
				)
				logger.Info(
					"idle",
					slog.String("child", service.Name),
				)

				service.state = IDLE

				// stop everything if there is a die promise
			case healthy:
				service.state = HEALTHY
				logger.Info(
					"healthy",
					slog.String("child", service.Name),
				)

				continue
			}
		}

	}

	pollInterval := 2 * time.Second
	go func() {
		defer func() {
			logger.Warn("file poller closed")
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollInterval):
				for i, service := range globalConfig.Services {
					if service.DisableConfigFileMonitoring {
						continue
					}
					// Check if the file has changed
					fileInfo, err := os.Stat(service.ConfigurationFile)
					if err != nil {
						if service.lastError == nil {
							globalConfig.Services[i].lastError = err
							if os.IsNotExist(err) {
								logger.Warn(
									"file doesn't exist",
									slog.String("child", service.Name),
									slog.String("file", service.ConfigurationFile),
									slog.Any("error", err),
								)
								continue
							}
							logger.Error(
								"error reading file info",
								slog.String("child", service.Name),
								slog.String("file", service.ConfigurationFile),
								slog.Any("error", err),
							)
						}

						continue
					}
					globalConfig.Services[i].lastError = nil
					if !fileInfo.ModTime().Equal(service.lastModified) || fileInfo.Size() != service.lastSize {
						// file has changed
						logger.Info(
							"file event",
							slog.String("child", service.Name),
							slog.String("file", service.ConfigurationFile),
						)

						globalConfig.Services[i].lastModified = fileInfo.ModTime()
						globalConfig.Services[i].lastSize = fileInfo.Size()

						sha256, err := glconfig.GenerateChecksum(service.ConfigurationFile)
						if err != nil {
							logger.Error("error reading checksum", slog.String("file", service.ConfigurationFile), slog.Any("error", err))
							continue
						}
						if sha256 != service.configFileLastSha256 {
							// file has changed
							logger.Info("file change detected",
								slog.String("child", service.Name),
								slog.String("file", service.ConfigurationFile),
							)
							globalConfig.Services[i].configFileLastSha256 = sha256
							go func(i int) {
								eventChan <- configurationFileChanged{index: i}
							}(i)
						}

					}
				}
			}
		}

	}()
	logger.Info("file poller started")

	healthyChecksumHandler(ctx, cancelTheContext) // Write the checksum to a file, IE we are healthy

	go func() {
		// bootstrap the tick
		eventChan <- tick{}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case e := <-eventChan:
			switch newEvent := e.(type) {

			case programExited:
				// ------- Make sure we have a valid index ------
				index := newEvent.index
				if index < 0 || index >= len(globalConfig.Services) {
					logger.Error(
						"fatal, index out of range",
						slog.Int("index", index),
						slog.Int("len_service", len(globalConfig.Services)),
					)

					cancelTheContext()
					return
				}
				service := &(globalConfig.Services[index])

				service.cmd = nil
				logger.Warn(
					"exited",
					slog.String("child", service.Name),
				)
				logger.Info(
					"idle",
					slog.String("child", service.Name),
				)

				service.state = IDLE

				queuedChan := make(chan struct{})
				go func(i int) {
					queuedChan <- struct{}{}
					eventChan <- programStart{index: i}
				}(index)
				<-queuedChan

			case programStart:
				// ------- Make sure we have a valid index ------
				index := newEvent.index
				if index < 0 || index >= len(globalConfig.Services) {
					logger.Error(
						"fatal, index out of range",
						slog.Int("index", index),
						slog.Int("len_service", len(globalConfig.Services)),
					)
					cancelTheContext()
					return
				}
				service := &(globalConfig.Services[index])
				logger.Info(
					"number of starts",
					slog.String("child", service.Name),
					slog.Int("starts", service.starts),
				)

				// ------- Check if we are allowed to start/restart it ------
				service.cmd = nil
				if globalConfig.MaxStarts != 0 && service.starts > globalConfig.MaxStarts {
					logger.Warn(
						"too many starts",
						slog.String("child", service.Name),
						slog.Int("starts", service.starts),
					)
					logger.Info(
						"idle",
						slog.String("child", service.Name),
					)

					service.state = IDLE

					go func() {
						time.Sleep(1 * time.Second)
						eventChan <- tick{}
					}()
					continue
				}

				// ------- Start the service ------
				logger.Info(
					"initiating",
					slog.String("child", service.Name),
				)

				arguments := []string{service.ConfigCommand, service.ConfigurationFile}
				arguments = append(arguments, service.AdditionalArguments...)
				logger.Debug(
					"command arguments",
					slog.String("child", service.Name),
					slog.String("arguments", strings.Join(arguments, " ")),
				)

				cmd, stdOutstdErr, err := startProcess(service.File, arguments...)

				if err != nil {
					logger.Error(
						"failed to start",
						slog.String("child", service.Name),
						slog.Any("error", err),
					)
					logger.Info(
						"idle",
						slog.String("child", service.Name),
					)

					service.state = IDLE
					continue
				}

				service.cmd = cmd
				service.starts++

				// Start the stdout pipe service in a goroutine
				go handleOutput(stdOutstdErr, cmd, service.Name, eventChan, index)

				go func(i int, cmdLocal *exec.Cmd) {
					// Monitor the process and send event if it dies
					cmdLocal.Wait()
					eventChan <- programExited{index: i, pid: cmdLocal.Process.Pid}
				}(index, cmd)

				eventQueue := make([]event, 0)
				func() {
					for {
						logger.Debug(
							"waiting for events",
							slog.String("child", service.Name),
						)

						select {
						case <-ctx.Done():
							return
						case <-time.After(5 * time.Second):
							logger.Warn(
								"stuck in pending",
								slog.String("child", service.Name),
							)

							service.state = PENDING
							return
							// Stop the service, and kill if there is a die promise

						case e := <-eventChan:
							switch anotherEvent := e.(type) {
							case programExited:
								if anotherEvent.index == index {
									if anotherEvent.pid == cmd.Process.Pid {
										logger.Warn(
											"exited",
											slog.String("child", service.Name),
										)
										logger.Info(
											"idle",
											slog.String("child", service.Name),
										)

										service.state = IDLE
										return
									}
									logger.Debug(
										"waiting for events",
										slog.String("child", service.Name),
									)
									logger.Debug(
										"flushing event (old pid)",
										slog.String("event", "programExited"),
									)

									continue
								}
								logger.Debug(
									"buffering event",
									slog.String("event", "programExited"),
								)

								eventQueue = append(eventQueue, e)
								continue

							case healthy:
								if anotherEvent.index == index {
									service.state = HEALTHY
									logger.Info(
										"healthy",
										slog.String("child", service.Name),
									)

									service.configFileLastSha256, err = glconfig.GenerateChecksum(service.ConfigurationFile)
									if err != nil {

										logger.Error(
											"error reading checksum",
											slog.String("child", service.Name),
											slog.Any("error", err),
										)
									}
									return
								}
								logger.Debug(
									"buffering event",
									slog.String("event", "healthy"),
								)

								eventQueue = append(eventQueue, e)

							case programStart:
								if anotherEvent.index == index {
									logger.Debug(
										"flushing event",
										slog.String("event", "programStart"),
									)

									continue
								}
								logger.Debug(
									"buffering event",
									slog.String("event", "programStart"),
								)

								eventQueue = append(eventQueue, e)

							case gracefulShutdown:
								if anotherEvent.index == index {
									logger.Debug(
										"flushing event",
										slog.String("event", "gracefulShutdown"),
									)

									continue
								}
								logger.Debug(
									"buffering event",
									slog.String("event", "gracefulShutdown"),
								)

								eventQueue = append(eventQueue, e)

							case configurationFileChanged:
								if anotherEvent.index == index {
									logger.Debug(
										"flushing event",
										slog.String("event", "configurationFileChanged"),
									)

									continue
								}
								logger.Debug(
									"buffering event",
									slog.String("event", "configurationFileChanged"),
								)

								eventQueue = append(eventQueue, e)

							case tick:
								logger.Debug(
									"flushing event",
									slog.String("event", "tick"),
								)

								continue

							default:
								logger.Error(
									"buffering event",
									slog.String("event", reflect.TypeOf(e).String()),
								)

								eventQueue = append(eventQueue, e)
							}
						}

					}
				}()
				if len(eventQueue) > 0 {
					queuedChan := make(chan struct{})
					go func() {
						queuedChan <- struct{}{}
						for _, e := range eventQueue {
							eventChan <- e
						}
					}()
					<-queuedChan
					continue
				}
				go func() {
					time.Sleep(1 * time.Second)
					eventChan <- tick{}
				}()

			case gracefulShutdown:

				// ------- Make sure we have a valid index ------
				index := newEvent.index
				if index < 0 || index >= len(globalConfig.Services) {
					logger.Error(
						"fatal, index out of range",
						slog.Int("index", index),
						slog.Int("len_service", len(globalConfig.Services)),
					)

					cancelTheContext()
					return
				}
				service := &(globalConfig.Services[index])
				service.starts = 0 // Reset

				logger.Info(
					"initiating controlled shutdown",
					slog.String("child", service.Name),
				)

				if service.cmd == nil {
					// It's not running, try to start it
					queueChan := make(chan struct{})
					go func(i int) {
						queueChan <- struct{}{}
						eventChan <- programStart{index: i}
					}(index)
					<-queueChan
					continue
				}
				processCmd := service.cmd
				if processCmd.Process == nil || (processCmd.ProcessState != nil && processCmd.ProcessState.Exited()) {
					// It's not running, start it
					queueChan := make(chan struct{})
					go func(i int) {
						queueChan <- struct{}{}
						eventChan <- programStart{index: i}
					}(index)
					<-queueChan
					continue
				}

				if err := processCmd.Process.Signal(os.Interrupt); err != nil {
					service.state = PENDING
					logger.Error(
						"problem sending SIGINT",
						slog.String("child", service.Name),
						slog.Any("error", err),
					)

					go func() {
						time.Sleep(1 * time.Second)
						eventChan <- tick{}
					}()
					continue
				}
				processCmd.Wait()

				logger.Info(
					"graceful shutdown completed",
					slog.String("child", service.Name),
				)
				logger.Info(
					"idle",
					slog.String("child", service.Name),
				)

				service.state = IDLE

				if newEvent.restart {
					// Send a controlled restart event
					queuedChan := make(chan struct{})
					go func(i int) {
						queuedChan <- struct{}{}
						eventChan <- programStart{index: i}
					}(index)
					<-queuedChan
					continue
				}
				go func() {
					time.Sleep(1 * time.Second)
					eventChan <- tick{}
				}()

			case configurationFileChanged:
				// ------- Make sure we have a valid index ------
				index := newEvent.index
				if index < 0 || index >= len(globalConfig.Services) {
					logger.Error(
						"fatal, index out of range",
						slog.Int("index", index),
						slog.Int("len_service", len(globalConfig.Services)),
					)

					cancelTheContext()
					return
				}
				service := &(globalConfig.Services[index])

				// ------- We need to check that the config file is valid ------
				logger.Info(
					"validating configuration",
					slog.String("child", service.Name),
				)

				arguments := []string{service.ConfigCommand, service.ConfigurationFile, service.ValidateCommand}
				arguments = append(arguments, service.AdditionalArguments...)
				cmd := exec.Command(service.File, arguments...)
				cmd.Env = os.Environ()

				// Create a buffer to capture both stdout and stderr
				var outputBuffer bytes.Buffer
				cmd.Stdout = &outputBuffer
				cmd.Stderr = &outputBuffer

				// Run the validation command
				err := cmd.Run()

				// Handle errors
				if err != nil {
					_, doWeHaveExitCode := err.(*exec.ExitError)
					func() {
						// Log the error
						if doWeHaveExitCode {
							// Capture the output from the service
							line := outputBuffer.Bytes()
							fields := bytes.Split(line, []byte("\n"))
							for _, field := range fields {
								forwardLog(slog.LevelInfo, field, service.Name)

							}
							return
						}
						logger.Error(
							"execution of conf validation command failed",
							slog.String("child", service.Name),
							slog.Any("error", err),
						)

					}()
					logger.Warn(
						"configuration validation failed. Ignoring change. No restart.",
						slog.String("child", service.Name),
						slog.Any("error", err),
					)

					go func() {
						time.Sleep(1 * time.Second)
						eventChan <- tick{}
					}()
					continue
				}
				// Log the validation command output from the service
				line := outputBuffer.Bytes()
				fields := bytes.Split(line, []byte("\n"))
				for _, field := range fields {
					forwardLog(slog.LevelInfo, field, service.Name)

				}

				// Begin the controlled restart by sending a graceful shutdown event
				queuedChan := make(chan struct{})
				go func(i int) {
					queuedChan <- struct{}{}
					eventChan <- gracefulShutdown{index: i, restart: true}
				}(index)
				<-queuedChan

			case tick:

				select {
				case nextEvent := <-eventChan:
					switch nextEvent.(type) {
					case tick:
						logger.Debug(
							"flushing event",
							slog.String("event", "tick"),
						)

						continue
					default:
						queueChan := make(chan struct{})
						go func() {
							// put back the event
							logger.Debug(
								"next event",
								slog.String("event", reflect.TypeOf(nextEvent).String()),
							)

							queueChan <- struct{}{}
							eventChan <- nextEvent
						}()
						<-queueChan
						continue
					}
				default:
					// No event in queue, continue
				}

				notick := false
				for _, service := range globalConfig.Services {

					// Review the state of the services
					if service.state != HEALTHY {
						if service.DiePromise {
							logger.Error(
								"cannot proceed: child has die promise and is not healthy",
								slog.String("child", service.Name),
							)

							cancelTheContext()
							return
						}
					}

				}

				if notick {
					continue
				}
				go func() {
					time.Sleep(1 * time.Second)
					eventChan <- tick{}
				}()

			}
		}
	}
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

func updateConfiguration(config string, g *ginit_config) error {
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
			legacy := ginit_config_v0921{}
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
			"issue writing checksum file",
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

	logger = outLogger.With("service", serviceAlias) // To be used as default

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
	validServices := []executable{}
	for index, s := range globalConfig.Services {
		e := s.Validate()
		if e != nil {
			for k, v := range e {
				rejectedFields[fmt.Sprintf("%s.executable[%d].%s", CONFIG_NAME, index, k)] = v
			}
			continue
		}
		validServices = append(validServices, s)
	}
	globalConfig.Services = validServices

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

	var newLogLevel slog.LevelVar
	err = json.Unmarshal([]byte("\""+globalConfig.LogLevel+"\""), &newLogLevel)

	if err == nil && newLogLevel != *logLevel {

		logger.Info(
			"log level changed",
			slog.String("log_level", newLogLevel.Level().String()),
		)
		logLevel.Set(newLogLevel.Level())

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
		time.Sleep(2 * time.Second) // Allow one second for everyone to cleanup
	case <-ctx.Done():

	}

}

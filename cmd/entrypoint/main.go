package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ------------------------------- GLOBALS --------------------------------

var thisBinary = "entrypoint"
var version string
var logLevel = &slog.LevelVar{} // INFO
var opts = &slog.HandlerOptions{
	AddSource: true,
	Level:     logLevel,
}
var logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	s := make([]rune, n)
	for i := range s {
		s[i] = rune(letters[rand.IntN(len(letters))])
	}
	return string(s)
}

func main() {

	logger = logger.With("service", thisBinary) // To be used as default
	//logLevel.Set(slog.LevelDebug)

	// Define flags
	var versionFlag bool
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")

	var targetFolder string
	flag.StringVar(&targetFolder, "t", "", "Specify the target folder")

	var sourceFolder string
	flag.StringVar(&sourceFolder, "s", "", "Specify the source folder")

	var filesString string
	flag.StringVar(&filesString, "f", "", "Specify the files to process in format \"file1:file2:file3\"")

	var envVarString string
	flag.StringVar(&envVarString, "e", "", "Specify the environment variables in format \"ENV1:ENV2\"")

	var childProcessString string
	flag.StringVar(&childProcessString, "c", "", "Specify the child process to spawn format \"command:arg1:arg2\"")

	// Parse the flags
	flag.Parse()

	if versionFlag {
		// Print version and exit
		fmt.Println(version)
		os.Exit(0)
	}

	if childProcessString == "" {
		logger.Error("child process not specified")
		os.Exit(1)
	}

	// Split the child process string into command and arguments
	childProcessArgs := strings.Split(childProcessString, ":")
	if len(childProcessArgs) < 1 {
		logger.Error("child process command not specified")
		os.Exit(1)
	}

	defer func() {
		cmd := exec.Command(childProcessArgs[0], childProcessArgs[1:]...)

		// Set environment variables for the child process
		cmd.Env = os.Environ()

		// Set up standard input/output for the child process
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Start the child process
		if err := cmd.Run(); err != nil {
			logger.Error("unable to spawn process", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	logger.Info(
		"system version",
		slog.String("version", version),
	)

	logger.Info("running as user", slog.Int("user", os.Geteuid()))

	if targetFolder == "" {
		logger.Warn("target folder not specified")
		return
	}

	if sourceFolder == "" {
		logger.Warn("source folder not specified")
		return
	}

	if filesString == "" {
		logger.Warn("files not specified")
		return
	}

	// Split the files string into individual files
	files := strings.Split(filesString, ":")
	if len(files) < 1 {
		logger.Warn("no files specified")
		return
	}

	for _, file := range files {
		fileInfo, err := os.Stat(targetFolder + file)

		if os.IsNotExist(err) {
			// copy the default file to to the config file
			bytes, err := os.ReadFile(sourceFolder + file)
			if err != nil {
				logger.Error(
					"error reading file",
					slog.String("file", sourceFolder+file),
					slog.Any("error", err),
				)
				return
			}

			err = os.WriteFile(targetFolder+file, bytes, 0666)
			if err != nil {
				logger.Error(
					"error copying file",
					slog.String("file", targetFolder+file),
					slog.Any("error", err),
				)

				fileInfo, err := os.Stat(targetFolder)
				if err != nil {
					logger.Error("target folder issue", slog.Any("error", err))
					continue
				}

				permissions := fileInfo.Mode()
				logger.Info("folder permissions", slog.String("folder", targetFolder), slog.String("permissions", fmt.Sprintf("%#o", permissions.Perm())))

				// Attempt to get the file owner on Unix-like systems
				if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
					logger.Info("folder owner", slog.String("folder", targetFolder), slog.Uint64("owner", uint64(stat.Uid)), slog.Uint64("group", uint64(stat.Gid)))

				} else {
					logger.Warn("could not assert fileInfo.Sys() to *syscall.Stat_t", slog.String("folder", targetFolder))
				}

				continue
			}

			logger.Info("file copied", slog.String("file", targetFolder+file))

			continue
		}

		if err != nil {
			// There was an issue getting the file info
			logger.Error("file info", slog.String("file", targetFolder+file), slog.Any("error", err))

			if fileInfo != nil {
				permissions := fileInfo.Mode()
				logger.Info("file permissions",
					slog.String("file", targetFolder+file),
					slog.String("permissions",
						fmt.Sprintf("%#o", permissions.Perm())),
				)

				// Attempt to get the file owner on Unix-like systems
				if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
					logger.Info("file owner", slog.String("file", targetFolder+file), slog.Uint64("owner", uint64(stat.Uid)), slog.Uint64("group", uint64(stat.Gid)))
				} else {
					logger.Warn("could not assert fileInfo.Sys() to *syscall.Stat_t", slog.String("file", targetFolder+file))
				}
			}

			folderInfo, err := os.Stat(targetFolder)
			if err != nil {
				logger.Error("error getting folder info",
					slog.String("folder", targetFolder),
					slog.Any("error", err),
				)
				continue
			}

			permissions := folderInfo.Mode()
			logger.Info("folder permissions",
				slog.String("folder", targetFolder),
				slog.String("permissions",
					fmt.Sprintf("%#o", permissions.Perm())),
			)

			// Attempt to get the file owner on Unix-like systems
			if stat, ok := folderInfo.Sys().(*syscall.Stat_t); ok {
				logger.Info("folder owner", slog.String("folder", targetFolder), slog.Uint64("owner", uint64(stat.Uid)), slog.Uint64("group", uint64(stat.Gid)))
			} else {
				logger.Warn("could not assert fileInfo.Sys() to *syscall.Stat_t", slog.String("folder", targetFolder))
			}

			logger.Error("file ignored", slog.String("file", targetFolder+file))
			continue
		}

		logger.Info(
			"file already exists",
			slog.String("file", targetFolder+file))
	}

	if envVarString == "" {
		return
	}
	// Split the environment variables string into individual variables
	envVars := strings.Split(envVarString, ":")
	for _, envVar := range envVars {
		currentValue := os.Getenv(envVar)
		if currentValue != "" && currentValue != "not_set" {
			logger.Info("environment variable is already set", slog.String("variable", envVar))
			continue
		}
		os.Setenv(envVar, randomString(32))
		logger.Info("setting environment variable", slog.String("variable", envVar))
	}

}

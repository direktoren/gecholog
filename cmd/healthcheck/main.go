package main

import (
	"flag"
	"os"
)

var checksumMap = map[string]string{
	"":             "/app/checksum/.ginit_config.sha256",
	"ginit":        "/app/checksum/.ginit_config.sha256",
	"gl":           "/app/checksum/.gl_config.sha256",
	"gui":          "/app/checksum/.gui_config.sha256",
	"nats2file":    "/app/checksum/.nats2file_config.sha256",
	"nats2log":     "/app/checksum/.nats2log_config.sha256",
	"tokencounter": "/app/checksum/.tokencounter_config.sha256",
}

var version string

func main() {
	fs := flag.NewFlagSet("prod", flag.ExitOnError)

	var versionFlag bool
	fs.BoolVar(&versionFlag, "version", false, "Print version and exit")

	var printFlag bool
	fs.BoolVar(&printFlag, "p", false, "Print the checksum")

	var service string
	fs.StringVar(&service, "s", "ginit", "The service to check")

	fs.Parse(os.Args[1:])

	if versionFlag {
		// Print version and exit
		os.Stdout.Write([]byte(version))
		os.Exit(0)
	}

	_, exists := checksumMap[service]
	if !exists {
		os.Exit(1) // Service does not exist
	}

	if printFlag {
		if _, err := os.Stat(checksumMap[service]); os.IsNotExist(err) {
			os.Exit(1) // File does not exist
		}
		bytes, err := os.ReadFile(checksumMap[service])
		if err != nil {
			os.Exit(1) // Error reading file
		}
		os.Stdout.Write(bytes)
		os.Exit(0)
	}

	if _, err := os.Stat(checksumMap[service]); os.IsNotExist(err) {
		os.Exit(1) // File does not exist
	}
	os.Exit(0) // File exists
}

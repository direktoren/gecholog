package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/direktoren/gecholog/internal/glconfig"
)

const (
	NATS2LOG_CONFIG_SETTINGS                = iota // 0
	NATS2LOG_CONFIG_TLS                            // 1
	NATS2LOG_CONFIG_SERVICEBUSCONFIG               // 2
	NATS2LOG_CONFIG_FILEWRITER                     // 3
	NATS2LOG_CONFIG_ELASTICWRITER                  // 4
	NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER        // 5
	NATS2LOG_CONFIG_RESTAPIWRITER                  // 6
)

const (
	NATS2LOG_CONFIG_NAME = "nats2log_config"
)

type nats2log_serviceBusConfig struct {
	Hostname         string `json:"hostname"`
	Topic            string `json:"topic"`
	TopicExactLogger string `json:"topic_exact_logger"`
	Token            string `json:"token"`
}

type nats2log_tlsConfig struct {
	InsecureFlag       bool     `json:"insecure"`
	SystemCertPoolFlag bool     `json:"system_cert_pool"`
	CertFiles          []string `json:"cert_files"`
}

type nats2log_config_v1001 struct {
	// From json config file
	Version  string `json:"version"`
	LogLevel string `json:"log_level"`
	Mode     string `json:"mode"`

	Retries    int `json:"retries"`
	RetryDelay int `json:"retry_delay_milliseconds"`

	TlsConfig nats2log_tlsConfig `json:"tls"`

	ServiceBusConfig nats2log_serviceBusConfig `json:"service_bus"`

	FileWriter              *fileWriter              `json:"file_writer"`
	ElasticWriter           *elasticWriter           `json:"elastic_writer"`
	AzureLogAnalyticsWriter *azureLogAnalyticsWriter `json:"azure_log_analytics_writer"`
	RestAPIWriter           *restAPIWriter           `json:"rest_api_writer"`
}

type elasticWriter struct {
	// elasticWriter is a struct that implements the byteWriter interface.

	// Public
	URL      string `json:"url"`
	Port     int    `json:"port"`
	Index    string `json:"index"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type restAPIWriter struct {
	// restAPIWriter is a struct that implements the byteWriter interface.

	// Public
	URL      string              `json:"url"`
	Port     int                 `json:"port"`
	Endpoint string              `json:"endpoint"`
	Headers  map[string][]string `json:"headers"`
	//	Headers  []header `json:"headers" validate:"dive"`

}

type azureLogAnalyticsWriter struct {
	// azureLogAnalyticsWriter is a struct that implements the byteWriter interface.
	WorkspaceID string `json:"workspace_id"`
	SharedKey   string `json:"shared_key"`
	LogType     string `json:"log_type"`
}

type fileWriter struct {
	// fileWriter is a struct that implements the byteWriter interface.
	// for file writing

	// Public
	ConfigFilename string `json:"filename"`
	WriteMode      string `json:"write_mode"`
}

func (n *nats2log_config_v1001) loadConfigFile(file string) error {
	unparsedJSON, err := glconfig.ReadFile(file)
	if err != nil {
		return err
	}
	logger.Info("file read", slog.String("file", file))
	n.ServiceBusConfig = nats2log_serviceBusConfig{}
	n.TlsConfig = nats2log_tlsConfig{}
	n.FileWriter = &fileWriter{}
	n.RestAPIWriter = &restAPIWriter{
		Headers: make(map[string][]string),
	}
	n.ElasticWriter = &elasticWriter{}
	n.AzureLogAnalyticsWriter = &azureLogAnalyticsWriter{}

	err = json.Unmarshal([]byte(unparsedJSON), n)
	if err != nil {
		return err
	}
	return nil

}

func (n *nats2log_config_v1001) writeConfigFile(file string) (string, error) {
	marshalledBytes, err := json.MarshalIndent(n, "", "   ")
	if err != nil {
		return "", err
	}

	// Convert byte array to string and append a newline
	marshalledString := string(marshalledBytes) + "\n"

	// Convert the string back to a byte array
	marshalledBytes = []byte(marshalledString)

	err = os.WriteFile(file, marshalledBytes, 0644)
	if err != nil {
		return "", err
	}
	logger.Debug("file saved", slog.String("file", file))

	sha256, err := glconfig.GenerateChecksum(file)
	if err != nil {
		return "", err
	}
	return sha256, nil
}

func (n *nats2log_config_v1001) setConfigFromAreas(areas []area) error {
	if len(areas) != 7 {
		return fmt.Errorf("invalid number of areas")
	}
	if areas[NATS2LOG_CONFIG_SETTINGS].Key != "settings" {
		return fmt.Errorf("settings: invalid area key settings")
	}
	if areas[NATS2LOG_CONFIG_TLS].Key != "tls" {
		return fmt.Errorf("tls: invalid area key tls")
	}
	if areas[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Key != "servicebus" {
		return fmt.Errorf("servicebus: invalid area key servicebus")
	}
	if areas[NATS2LOG_CONFIG_ELASTICWRITER].Key != "elasticwriter" {
		return fmt.Errorf("elasticwriter: invalid area key")
	}
	if areas[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Key != "azureloganalyticswriter" {
		return fmt.Errorf("azureloganalyticswriter: invalid area key")
	}
	if areas[NATS2LOG_CONFIG_RESTAPIWRITER].Key != "restapiwriter" {
		return fmt.Errorf("restapiwriter: invalid area key")
	}

	if areas[NATS2LOG_CONFIG_FILEWRITER].Key != "filewriter" {
		return fmt.Errorf("filewriter: invalid area key")
	}

	if len(areas[NATS2LOG_CONFIG_SETTINGS].Objects) != 4 {
		return fmt.Errorf("settings: invalid number of settings objects")
	}

	if len(areas[NATS2LOG_CONFIG_TLS].Objects) != 3 {
		return fmt.Errorf("tls: invalid number of tls objects")
	}

	if len(areas[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects) != 4 {
		return fmt.Errorf("servicebus: invalid number of servicebus objects")
	}

	if len(areas[NATS2LOG_CONFIG_FILEWRITER].Objects) != 2 {
		return fmt.Errorf("filewriter: invalid number of filewriter objects")
	}

	if len(areas[NATS2LOG_CONFIG_ELASTICWRITER].Objects) != 5 {
		return fmt.Errorf("elasticwriter: invalid number of elasticwriter objects")
	}

	if len(areas[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects) != 3 {
		return fmt.Errorf("azureloganalyticswriter: invalid number of azureloganalyticswriter objects")
	}

	if len(areas[NATS2LOG_CONFIG_RESTAPIWRITER].Objects) != 4 {
		return fmt.Errorf("restapiwriter: invalid number of restapiwriter objects")
	}

	// Settings

	n.LogLevel = areas[NATS2LOG_CONFIG_SETTINGS].Objects[0].Fields.(*radioField).Value
	n.Mode = areas[NATS2LOG_CONFIG_SETTINGS].Objects[1].Fields.(*radioField).Value
	n.Retries, _ = strconv.Atoi(areas[NATS2LOG_CONFIG_SETTINGS].Objects[2].Fields.(*textField).Value)
	n.RetryDelay, _ = strconv.Atoi(areas[NATS2LOG_CONFIG_SETTINGS].Objects[3].Fields.(*textField).Value)
	//n.RetryDelay = int64(intRetryDelay) * time.Millisecond
	//n.RetryDelay, _ = time.ParseDuration(areas[NATS2LOG_CONFIG_SETTINGS].Objects[3].Fields.(*textField).Value)

	// TLS

	n.TlsConfig.InsecureFlag, _ = strconv.ParseBool(areas[NATS2LOG_CONFIG_TLS].Objects[0].Fields.(*radioField).Value)
	n.TlsConfig.SystemCertPoolFlag, _ = strconv.ParseBool(areas[NATS2LOG_CONFIG_TLS].Objects[1].Fields.(*radioField).Value)
	n.TlsConfig.CertFiles = make([]string, len(areas[NATS2LOG_CONFIG_TLS].Objects[2].Fields.(*arrayField).Values))
	for index, v := range areas[NATS2LOG_CONFIG_TLS].Objects[2].Fields.(*arrayField).Values {
		n.TlsConfig.CertFiles[index] = v.Value
	}

	// ServiceBusConfig

	n.ServiceBusConfig.Hostname = areas[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[0].Fields.(*textField).Value
	n.ServiceBusConfig.Topic = areas[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[1].Fields.(*textField).Value
	n.ServiceBusConfig.TopicExactLogger = areas[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[2].Fields.(*textField).Value
	n.ServiceBusConfig.Token = areas[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[3].Fields.(*textField).Value

	// FileWriter

	n.FileWriter.ConfigFilename = areas[NATS2LOG_CONFIG_FILEWRITER].Objects[0].Fields.(*textField).Value
	n.FileWriter.WriteMode = areas[NATS2LOG_CONFIG_FILEWRITER].Objects[1].Fields.(*radioField).Value

	// ElasticWriter

	n.ElasticWriter.URL = areas[NATS2LOG_CONFIG_ELASTICWRITER].Objects[0].Fields.(*textField).Value
	n.ElasticWriter.Port, _ = strconv.Atoi(areas[NATS2LOG_CONFIG_ELASTICWRITER].Objects[1].Fields.(*textField).Value)
	n.ElasticWriter.Index = areas[NATS2LOG_CONFIG_ELASTICWRITER].Objects[2].Fields.(*textField).Value
	n.ElasticWriter.Username = areas[NATS2LOG_CONFIG_ELASTICWRITER].Objects[3].Fields.(*textField).Value
	n.ElasticWriter.Password = areas[NATS2LOG_CONFIG_ELASTICWRITER].Objects[4].Fields.(*textField).Value

	// AzureLogAnalyticsWriter

	n.AzureLogAnalyticsWriter.WorkspaceID = areas[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[0].Fields.(*textField).Value
	n.AzureLogAnalyticsWriter.SharedKey = areas[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[1].Fields.(*textField).Value
	n.AzureLogAnalyticsWriter.LogType = areas[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[2].Fields.(*textField).Value

	// RestAPIWriter

	n.RestAPIWriter.URL = areas[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[0].Fields.(*textField).Value
	n.RestAPIWriter.Port, _ = strconv.Atoi(areas[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[1].Fields.(*textField).Value)
	n.RestAPIWriter.Endpoint = areas[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[2].Fields.(*textField).Value
	n.RestAPIWriter.Headers = make(map[string][]string)
	for _, header := range areas[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[3].Fields.(*headerField).Headers {
		values := []string{}
		for _, value := range header.Values {
			values = append(values, value.Value)
			logger.Debug("header", slog.String("header", header.Header), slog.String("value", value.Value))
		}
		n.RestAPIWriter.Headers[header.Header] = values
	}

	return nil
}

func (n *nats2log_config_v1001) update() (map[string]string, error) {

	// Update TLS

	sort.SliceStable(n.TlsConfig.CertFiles, func(i, j int) bool {
		return n.TlsConfig.CertFiles[i] < n.TlsConfig.CertFiles[j]
	})

	// Sort headers
	for key, _ := range n.RestAPIWriter.Headers {
		sort.Strings(n.RestAPIWriter.Headers[key])
	}

	return map[string]string{}, nil
}

func (n *nats2log_config_v1001) findError(v map[string]string, key string, exact bool) string {
	if v == nil {
		return "no match"
	}
	if exact {
		errorMsg, found := v[key]
		if found {
			return errorMsg
		}
		return "valid"
	}

	switch key {

	case "settings":

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".TlsConfig",
			NATS2LOG_CONFIG_NAME + ".ServiceBusConfig",
			NATS2LOG_CONFIG_NAME + ".FileWriter",
			NATS2LOG_CONFIG_NAME + ".ElasticWriter",
			NATS2LOG_CONFIG_NAME + ".AzureLogAnalyticsWriter",
			NATS2LOG_CONFIG_NAME + ".RestAPIWriter",
		}
		count := 0
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					count++
					break
				}
			}

		}
		if count != len(v) {
			logger.Debug("v", slog.Any("v", v))
			return "invalid"
		}

	case "tls":

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".TlsConfig",
		}
		found := false
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}

		}
		if found {
			return "invalid"
		}

	case "servicebus":

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".ServiceBusConfig",
		}
		found := false
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}

		}
		if found {
			return "invalid"
		}

	case "filewriter":

		_, exists := v[NATS2LOG_CONFIG_NAME+".FileWriter"]
		if exists {
			return "invalid"
		}

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".FileWriter",
		}
		found := false
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}

		}
		if found {
			return "rejected"
		}

	case "elasticwriter":

		_, exists := v[NATS2LOG_CONFIG_NAME+".ElasticWriter"]
		if exists {
			return "invalid"
		}

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".ElasticWriter",
		}
		found := false
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}

		}
		if found {
			return "rejected"
		}

	case "restapiwriter":

		_, exists := v[NATS2LOG_CONFIG_NAME+".RestAPIWriter"]
		if exists {
			return "invalid"
		}

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".RestAPIWriter",
		}
		found := false
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}

		}
		if found {
			return "rejected"
		}

	case "azureloganalyticswriter":

		_, exists := v[NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter"]
		if exists {
			return "invalid"
		}

		prefixes := []string{
			NATS2LOG_CONFIG_NAME + ".AzureLogAnalyticsWriter",
		}
		found := false
		for key, _ := range v {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}

		}
		if found {
			return "rejected"
		}
	}

	return "valid"
}

func (n *nats2log_config_v1001) updateAreasFromConfig(v map[string]string, a []area) error {
	if len(a) != 7 {
		return fmt.Errorf("invalid number of areas")
	}
	if a[NATS2LOG_CONFIG_SETTINGS].Key != "settings" {
		return fmt.Errorf("settings: invalid area key settings")
	}
	if a[NATS2LOG_CONFIG_TLS].Key != "tls" {
		return fmt.Errorf("tls: invalid area key tls")
	}
	if a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Key != "servicebus" {
		return fmt.Errorf("servicebus: invalid area key servicebus")
	}
	if a[NATS2LOG_CONFIG_ELASTICWRITER].Key != "elasticwriter" {
		return fmt.Errorf("elasticwriter: invalid area key")
	}
	if a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Key != "azureloganalyticswriter" {
		return fmt.Errorf("azureloganalyticswriter: invalid area key")
	}
	if a[NATS2LOG_CONFIG_RESTAPIWRITER].Key != "restapiwriter" {
		return fmt.Errorf("restapiwriter: invalid area key")
	}
	if a[NATS2LOG_CONFIG_FILEWRITER].Key != "filewriter" {
		return fmt.Errorf("filewriter: invalid area key")
	}

	if len(a[NATS2LOG_CONFIG_SETTINGS].Objects) != 4 {
		return fmt.Errorf("settings: invalid number of settings objects")
	}

	if len(a[NATS2LOG_CONFIG_TLS].Objects) != 3 {
		return fmt.Errorf("tls: invalid number of tls objects")
	}

	if len(a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects) != 4 {
		return fmt.Errorf("servicebus: invalid number of servicebus objects")
	}

	if len(a[NATS2LOG_CONFIG_FILEWRITER].Objects) != 2 {
		return fmt.Errorf("filewriter: invalid number of filewriter objects")
	}

	if len(a[NATS2LOG_CONFIG_ELASTICWRITER].Objects) != 5 {
		return fmt.Errorf("elasticwriter: invalid number of elasticwriter objects")
	}

	if len(a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects) != 3 {
		return fmt.Errorf("azureloganalyticswriter: invalid number of azureloganalyticswriter objects")
	}

	if len(a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects) != 4 {
		return fmt.Errorf("restapiwriter: invalid number of restapiwriter objects")
	}

	// Settings
	a[NATS2LOG_CONFIG_SETTINGS].ErrorMsg = n.findError(v, "settings", false)
	a[NATS2LOG_CONFIG_SETTINGS].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_SETTINGS].ErrorMsg)

	a[NATS2LOG_CONFIG_SETTINGS].Objects[0].Type = RADIO
	a[NATS2LOG_CONFIG_SETTINGS].Objects[0].Headline = "Log Level"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[0].Key = "loglevel"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[0].Fields = &radioField{
		Value:               n.LogLevel,
		Options:             []string{"DEBUG", "INFO", "WARN", "ERROR"},
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".LogLevel", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".LogLevel", true)),
	}

	a[NATS2LOG_CONFIG_SETTINGS].Objects[1].Type = RADIO
	a[NATS2LOG_CONFIG_SETTINGS].Objects[1].Headline = "Mode"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[1].Key = "mode"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[1].Fields = &radioField{
		Value:               n.Mode,
		Options:             []string{"file_writer", "elastic_writer", "azure_log_analytics_writer", "rest_api_writer"},
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".Mode", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".Mode", true)),
	}

	a[NATS2LOG_CONFIG_SETTINGS].Objects[2].Type = INPUT
	a[NATS2LOG_CONFIG_SETTINGS].Objects[2].Headline = "Retries"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[2].Key = "retries"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[2].Fields = &textField{
		Value:               fmt.Sprintf("%d", n.Retries),
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".Retries", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".Retries", true)),
		Placeholder:         "3",
	}

	a[NATS2LOG_CONFIG_SETTINGS].Objects[3].Type = INPUT
	a[NATS2LOG_CONFIG_SETTINGS].Objects[3].Headline = "Retry Delay Milliseconds"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[3].Key = "retrydelay"
	a[NATS2LOG_CONFIG_SETTINGS].Objects[3].Fields = &textField{
		Value:               fmt.Sprintf("%d", n.RetryDelay),
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".RetryDelay", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".RetryDelay", true)),
		Placeholder:         "250",
	}

	// TLS

	a[NATS2LOG_CONFIG_TLS].ErrorMsg = n.findError(v, "tls", false)
	a[NATS2LOG_CONFIG_TLS].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_TLS].ErrorMsg)

	a[NATS2LOG_CONFIG_TLS].Objects[0].Type = RADIO
	a[NATS2LOG_CONFIG_TLS].Objects[0].Headline = "Insecure Flag"
	a[NATS2LOG_CONFIG_TLS].Objects[0].Key = "insecureflag"
	a[NATS2LOG_CONFIG_TLS].Objects[0].Fields = &radioField{
		Value:               fmt.Sprintf("%t", n.TlsConfig.InsecureFlag),
		Options:             []string{"true", "false"},
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".TlsConfig.InsecureFlag", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".TlsConfig.Tls.InsecureFlag", true)),
	}

	a[NATS2LOG_CONFIG_TLS].Objects[1].Type = RADIO
	a[NATS2LOG_CONFIG_TLS].Objects[1].Headline = "System Cert Pool Flag"
	a[NATS2LOG_CONFIG_TLS].Objects[1].Key = "systemcertpoolflag"
	a[NATS2LOG_CONFIG_TLS].Objects[1].Fields = &radioField{
		Value:               fmt.Sprintf("%t", n.TlsConfig.SystemCertPoolFlag),
		Options:             []string{"true", "false"},
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".TlsConfig.SystemCertPoolFlag", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".TlsConfig.SystemCertPoolFlag", true)),
	}

	a[NATS2LOG_CONFIG_TLS].Objects[2].Type = ARRAY
	a[NATS2LOG_CONFIG_TLS].Objects[2].Headline = "CA Certificate Bundle Files"
	a[NATS2LOG_CONFIG_TLS].Objects[2].Key = "certfiles"
	certfiles := make([]pair, len(n.TlsConfig.CertFiles))
	for index, _ := range n.TlsConfig.CertFiles {
		certfiles[index] = pair{
			Value:               n.TlsConfig.CertFiles[index],
			ErrorMsg:            n.findError(v, fmt.Sprintf(NATS2LOG_CONFIG_NAME+".TlsConfig.CertFiles[%d]", index), true),
			ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, fmt.Sprintf(NATS2LOG_CONFIG_NAME+".TlsConfig.Tls.CertFiles[%d]", index), true)),
		}
	}
	sort.SliceStable(certfiles, func(i, j int) bool {
		return certfiles[i].Value < certfiles[j].Value
	})
	a[NATS2LOG_CONFIG_TLS].Objects[2].Fields = &arrayField{
		Values:      certfiles,
		Placeholder: "/app/conf/ca.crt",
	}

	// ServiceBusConfig

	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].ErrorMsg = n.findError(v, "servicebus", false)
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].ErrorMsg)

	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[0].Type = INPUT
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[0].Headline = "Hostname"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[0].Key = "hostname"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[0].Fields = &textField{
		Value:               n.ServiceBusConfig.Hostname,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.Hostname", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.Hostname", true)),
		Placeholder:         "localhost:4222",
	}

	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[1].Type = INPUT
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[1].Headline = "Topic"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[1].Key = "topic"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[1].Fields = &textField{
		Value:               n.ServiceBusConfig.Topic,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.Topic", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.Topic", true)),
		Placeholder:         "coburn.gl.gecholog",
	}

	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[2].Type = INPUT
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[2].Headline = "Topic Exact Logger"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[2].Key = "topicexactlogger"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[2].Fields = &textField{
		Value:               n.ServiceBusConfig.TopicExactLogger,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.TopicExactLogger", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.TopicExactLogger", true)),
		Placeholder:         "coburn.gl.logger",
	}

	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[3].Type = INPUT
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[3].Headline = "Token"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[3].Key = "token"
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG].Objects[3].Fields = &textField{
		Value:               n.ServiceBusConfig.Token,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.Token", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ServiceBusConfig.Token", true)),
		Placeholder:         "changeme",
	}

	// FileWriter
	a[NATS2LOG_CONFIG_FILEWRITER].ErrorMsg = n.findError(v, "filewriter", false)
	a[NATS2LOG_CONFIG_FILEWRITER].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_FILEWRITER].ErrorMsg)

	a[NATS2LOG_CONFIG_FILEWRITER].Objects[0].Type = INPUT
	a[NATS2LOG_CONFIG_FILEWRITER].Objects[0].Headline = "Filename"
	a[NATS2LOG_CONFIG_FILEWRITER].Objects[0].Key = "filename"
	a[NATS2LOG_CONFIG_FILEWRITER].Objects[0].Fields = &textField{
		Value:               n.FileWriter.ConfigFilename,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".FileWriter.fileWriter.ConfigFilename", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".FileWriter.fileWriter.ConfigFilename", true)),
		Placeholder:         "/app/log/gecholog.jsonl",
	}

	a[NATS2LOG_CONFIG_FILEWRITER].Objects[1].Type = RADIO
	a[NATS2LOG_CONFIG_FILEWRITER].Objects[1].Headline = "Write Mode"
	a[NATS2LOG_CONFIG_FILEWRITER].Objects[1].Key = "writemode"
	a[NATS2LOG_CONFIG_FILEWRITER].Objects[1].Fields = &radioField{
		Value:               n.FileWriter.WriteMode,
		Options:             []string{"append", "overwrite", "new"},
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".FileWriter.fileWriter.WriteMode", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".FileWriter.fileWriter.WriteMode", true)),
	}

	// ElasticWriter
	a[NATS2LOG_CONFIG_ELASTICWRITER].ErrorMsg = n.findError(v, "elasticwriter", false)
	a[NATS2LOG_CONFIG_ELASTICWRITER].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_ELASTICWRITER].ErrorMsg)

	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[0].Type = INPUT
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[0].Headline = "URL"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[0].Key = "url"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[0].Fields = &textField{
		Value:               n.ElasticWriter.URL,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.URL", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.URL", true)),
		Placeholder:         "https://localhost",
	}

	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[1].Type = INPUT
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[1].Headline = "Port"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[1].Key = "port"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[1].Fields = &textField{
		Value:               fmt.Sprintf("%d", n.ElasticWriter.Port),
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Port", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Port", true)),
		Placeholder:         "9200",
	}

	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[2].Type = INPUT
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[2].Headline = "Index"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[2].Key = "index"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[2].Fields = &textField{
		Value:               n.ElasticWriter.Index,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Index", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Index", true)),
		Placeholder:         "gecholog",
	}

	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[3].Type = INPUT
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[3].Headline = "Username"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[3].Key = "username"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[3].Fields = &textField{
		Value:               n.ElasticWriter.Username,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Username", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Username", true)),
		Placeholder:         "elastic",
	}

	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[4].Type = INPUT
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[4].Headline = "Password"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[4].Key = "password"
	a[NATS2LOG_CONFIG_ELASTICWRITER].Objects[4].Fields = &textField{
		Value:               n.ElasticWriter.Password,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Password", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".ElasticWriter.elasticWriter.Password", true)),
		Placeholder:         "changeme",
	}

	// AzureLogAnalyticsWriter

	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].ErrorMsg = n.findError(v, "azureloganalyticswriter", false)
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].ErrorMsg)

	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[0].Type = INPUT
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[0].Headline = "Workspace ID"
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[0].Key = "workspaceid"
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[0].Fields = &textField{
		Value:               n.AzureLogAnalyticsWriter.WorkspaceID,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter.azureLogAnalyticsWriter.WorkspaceID", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter.azureLogAnalyticsWriter.WorkspaceID", true)),
		Placeholder:         "12345-12345-12345-12345",
	}

	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[1].Type = INPUT
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[1].Headline = "Shared Key"
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[1].Key = "sharedkey"
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[1].Fields = &textField{
		Value:               n.AzureLogAnalyticsWriter.SharedKey,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter.azureLogAnalyticsWriter.SharedKey", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter.azureLogAnalyticsWriter.SharedKey", true)),
		Placeholder:         "changeme",
	}

	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[2].Type = INPUT
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[2].Headline = "Log Type"
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[2].Key = "logtype"
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER].Objects[2].Fields = &textField{
		Value:               n.AzureLogAnalyticsWriter.LogType,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter.azureLogAnalyticsWriter.LogType", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".AzureLogAnalyticsWriter.azureLogAnalyticsWriter.LogType", true)),
		Placeholder:         "coburn1",
	}

	// RestAPIWriter
	a[NATS2LOG_CONFIG_RESTAPIWRITER].ErrorMsg = n.findError(v, "restapiwriter", false)
	a[NATS2LOG_CONFIG_RESTAPIWRITER].ErrorMsgTooltipText = errorMsgToTooltipText(a[NATS2LOG_CONFIG_RESTAPIWRITER].ErrorMsg)

	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[0].Type = INPUT
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[0].Headline = "URL"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[0].Key = "url"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[0].Fields = &textField{
		Value:               n.RestAPIWriter.URL,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.URL", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.URL", true)),
		Placeholder:         "https://localhost",
	}

	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[1].Type = INPUT
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[1].Headline = "Port"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[1].Key = "port"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[1].Fields = &textField{
		Value:               fmt.Sprintf("%d", n.RestAPIWriter.Port),
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Port", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Port", true)),
		Placeholder:         "443",
	}

	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[2].Type = INPUT
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[2].Headline = "Endpoint"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[2].Key = "endpoint"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[2].Fields = &textField{
		Value:               n.RestAPIWriter.Endpoint,
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Endpoint", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Endpoint", true)),
		Placeholder:         "/api/v1/log",
	}

	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[3].Type = HEADER
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[3].Headline = "Headers"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[3].Key = "headers"
	a[NATS2LOG_CONFIG_RESTAPIWRITER].Objects[3].Fields = &headerField{
		Headers: func() []webHeader {
			headers := []webHeader{}
			for header, values := range n.RestAPIWriter.Headers {
				newWebHeader := webHeader{
					Header:              header,
					ErrorMsg:            n.findError(v, fmt.Sprintf(NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Headers[%s]", header), true),
					ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, fmt.Sprintf(NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Headers[%s]", header), true)),
					Values:              []webHeaderValue{},
				}
				for _, value := range values {
					newWebHeader.Values = append(newWebHeader.Values, webHeaderValue{
						Value:               value,
						ErrorMsg:            n.findError(v, fmt.Sprintf(NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Headers[%s]", value), true),
						ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, fmt.Sprintf(NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Headers[%s]", value), true)),
					})
				}
				headers = append(headers, newWebHeader)
				logger.Debug("webheader", slog.Any("header", newWebHeader))
			}
			sort.SliceStable(headers, func(i, j int) bool {
				return headers[i].Header < headers[j].Header
			})
			return headers
		}(),
		ErrorMsg:            n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Headers", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(n.findError(v, NATS2LOG_CONFIG_NAME+".RestAPIWriter.restAPIWriter.Headers", true)),
	}

	return nil
}

func (n *nats2log_config_v1001) createAreas(v map[string]string) ([]area, error) {
	a := make([]area, 7)

	// Settings

	settingsObjects := make([]inputObject, 4)
	for i := range settingsObjects {
		settingsObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_SETTINGS] = area{
		Headline: "Settings",
		Key:      "settings",
		Redirect: "menu",
		Form:     "settings-form",
		Objects:  settingsObjects,
	}

	// TLS

	tlsObjects := make([]inputObject, 3)
	for i := range tlsObjects {
		tlsObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_TLS] = area{
		Headline: "TLS",
		Key:      "tls",
		Redirect: "menu",
		Form:     "tls-form",
		Objects:  tlsObjects,
	}

	// ServiceBusConfig

	serviceBusConfigObjects := make([]inputObject, 4)
	for i := range serviceBusConfigObjects {
		serviceBusConfigObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_SERVICEBUSCONFIG] = area{
		Headline: "Service Bus Config",
		Key:      "servicebus",
		Redirect: "menu",
		Form:     "servicebus-form",
		Objects:  serviceBusConfigObjects,
	}

	// FileWriter

	fileWriterObjects := make([]inputObject, 2)
	for i := range fileWriterObjects {
		fileWriterObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_FILEWRITER] = area{
		Headline: "File Writer",
		Key:      "filewriter",
		Redirect: "menu",
		Form:     "filewriter-form",
		Objects:  fileWriterObjects,
	}

	// ElasticWriter

	elasticWriterObjects := make([]inputObject, 5)
	for i := range elasticWriterObjects {
		elasticWriterObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_ELASTICWRITER] = area{
		Headline: "Elastic Writer",
		Key:      "elasticwriter",
		Redirect: "menu",
		Form:     "elasticwriter-form",
		Objects:  elasticWriterObjects,
	}

	// AzureLogAnalyticsWriter

	azureLogAnalyticsWriterObjects := make([]inputObject, 3)
	for i := range azureLogAnalyticsWriterObjects {
		azureLogAnalyticsWriterObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_AZURELOGANALYTICSWRITER] = area{
		Headline: "Azure Log Analytics Writer",
		Key:      "azureloganalyticswriter",
		Redirect: "menu",
		Form:     "azureloganalyticswriter-form",
		Objects:  azureLogAnalyticsWriterObjects,
	}

	// RestAPIWriter

	restAPIWriterObjects := make([]inputObject, 4)
	for i := range restAPIWriterObjects {
		restAPIWriterObjects[i] = inputObject{}
	}
	a[NATS2LOG_CONFIG_RESTAPIWRITER] = area{
		Headline: "Rest API Writer",
		Key:      "restapiwriter",
		Redirect: "menu",
		Form:     "restapiwriter-form",
		Objects:  restAPIWriterObjects,
	}

	err := n.updateAreasFromConfig(v, a)
	return a, err
}

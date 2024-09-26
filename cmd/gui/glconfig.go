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
	"github.com/direktoren/gecholog/internal/processorconfiguration"
	"github.com/direktoren/gecholog/internal/protectedheader"
	"github.com/direktoren/gecholog/internal/router"
)

const (
	MENU_GL_INDEX        = iota // 0
	MENU_NATS2FILE_INDEX        // 1
	MENU_NATS2LOG_INDEX         // 2
)

// ------------------------------ gl  ------------------------------

const (
	GL_CONFIG_SETTINGS           = iota // 0
	GL_CONFIG_TLS                       // 1
	GL_CONFIG_SERVICEBUSCONFIG          // 2
	GL_CONFIG_ROUTERS                   // 3
	GL_CONFIG_REQUESTPROCESSORS         // 4
	GL_CONFIG_RESPONSEPROCESSORS        // 5
	GL_CONFIG_LOGGER                    // 6
)

const (
	GL_CONFIG_NAME = "gl_config"
)

type gl_serviceBusConfig struct {
	Hostname          string `json:"hostname"`
	Topic             string `json:"topic"`
	TopicExactIsAlive string `json:"topic_exact_isalive"`
	TopicExactLogger  string `json:"topic_exact_logger"`
	Token             string `json:"token"`
}

type gl_tlsUserConfig struct {
	Ingress struct {
		Enabled         bool   `json:"enabled"`
		CertificateFile string `json:"certificate_file"`
		PrivateKeyFile  string `json:"private_key_file"`
	} `json:"ingress"`
	Outbound struct {
		InsecureFlag       bool     `json:"insecure"`
		SystemCertPoolFlag bool     `json:"system_cert_pool"`
		CertFiles          []string `json:"cert_files"`
	} `json:"outbound"`
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

type Gl_config_v1001 struct {
	GatewayID string `json:"gateway_id"`
	Version   string `json:"version"`
	LogLevel  string `json:"log_level"`

	TlsUserConfig gl_tlsUserConfig `json:"tls"`

	ServiceBusConfig gl_serviceBusConfig `json:"service_bus"`

	Port            int    `json:"gl_port"`
	SessionIDHeader string `json:"session_id_header"`

	MaskedHeaders []string `json:"masked_headers"`

	RemoveHeaders []string `json:"remove_headers"`

	LogUnauthorized bool `json:"log_unauthorized"`

	Routers []router.Router `json:"routers"`

	RequestProcessors  processorsMatrix `json:"request"`
	ResponseProcessors processorsMatrix `json:"response"`
	Logger             finalLogger      `json:"logger"`
}

func (g *Gl_config_v1001) loadConfigFile(file string) error {
	unparsedJSON, err := glconfig.ReadFile(file)
	if err != nil {
		return err
	}

	logger.Info("file read", slog.String("file", file))
	g.TlsUserConfig = gl_tlsUserConfig{}
	g.ServiceBusConfig = gl_serviceBusConfig{}
	g.MaskedHeaders = []string{}
	g.RemoveHeaders = []string{}
	g.Routers = []router.Router{}
	g.ResponseProcessors = processorsMatrix{}
	g.RequestProcessors = processorsMatrix{}
	g.Logger = finalLogger{}

	err = json.Unmarshal([]byte(unparsedJSON), g)
	if err != nil {
		return err
	}
	return nil
}

func getUrls(routers []router.Router) []string {

	if len(routers) == 0 {
		return []string{}
	}
	last := routers[0].Outbound.Url
	urls := []string{last}

	for _, router := range routers {
		if router.Outbound.Url != last {
			urls = append(urls, router.Outbound.Url)
			last = router.Outbound.Url
		}
	}
	return urls

}

func (g *Gl_config_v1001) writeConfigFile(file string) (string, error) {
	marshalledBytes, err := json.MarshalIndent(g, "", "   ")
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

func (g *Gl_config_v1001) setConfigFromAreas(areas []area) error {
	if len(areas) != 7 {
		return fmt.Errorf("invalid number of areas")
	}
	if areas[GL_CONFIG_SETTINGS].Key != "settings" {
		return fmt.Errorf("settings: invalid area key settings")
	}
	if areas[GL_CONFIG_TLS].Key != "tls" {
		return fmt.Errorf("tls: invalid area key tls")
	}
	if areas[GL_CONFIG_SERVICEBUSCONFIG].Key != "servicebus" {
		return fmt.Errorf("servicebus: invalid area key servicebus")
	}
	if areas[GL_CONFIG_ROUTERS].Key != "routers" {
		return fmt.Errorf("routers: invalid area key routers")
	}
	if areas[GL_CONFIG_REQUESTPROCESSORS].Key != "requestprocessors" {
		return fmt.Errorf("requestprocessors: invalid area key requestprocessors")
	}
	if areas[GL_CONFIG_RESPONSEPROCESSORS].Key != "responseprocessors" {
		return fmt.Errorf("responseprocessors: invalid area key responseprocessors")
	}
	if areas[GL_CONFIG_LOGGER].Key != "logger" {
		return fmt.Errorf("logger: invalid area key logger")
	}

	if len(areas[GL_CONFIG_SETTINGS].Objects) != 7 {
		return fmt.Errorf("settings: invalid number of settings objects")
	}
	if len(areas[GL_CONFIG_TLS].Objects) != 8 {
		return fmt.Errorf("settings: invalid number of tls objects")
	}
	if len(areas[GL_CONFIG_SERVICEBUSCONFIG].Objects) != 5 {
		return fmt.Errorf("settings: invalid number of servicebusconfig objects")
	}
	if len(areas[GL_CONFIG_LOGGER].Objects) != 6 {
		return fmt.Errorf("settings: invalid number of logger objects")
	}

	// Settings

	g.GatewayID = areas[GL_CONFIG_SETTINGS].Objects[0].Fields.(*textField).Value
	g.Port, _ = strconv.Atoi(areas[GL_CONFIG_SETTINGS].Objects[1].Fields.(*textField).Value)
	g.SessionIDHeader = areas[GL_CONFIG_SETTINGS].Objects[2].Fields.(*textField).Value
	g.LogLevel = areas[GL_CONFIG_SETTINGS].Objects[3].Fields.(*radioField).Value
	//g.Version = version
	//g.Version = areas[GL_CONFIG_SETTINGS].Objects[4].Fields.(*textField).Value
	g.MaskedHeaders = make([]string, len(areas[0].Objects[4].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_SETTINGS].Objects[4].Fields.(*arrayField).Values {
		g.MaskedHeaders[index] = v.Value
	}
	g.RemoveHeaders = make([]string, len(areas[GL_CONFIG_SETTINGS].Objects[5].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_SETTINGS].Objects[5].Fields.(*arrayField).Values {
		g.RemoveHeaders[index] = v.Value
	}
	g.LogUnauthorized, _ = strconv.ParseBool(areas[GL_CONFIG_SETTINGS].Objects[6].Fields.(*radioField).Value)

	// TLS

	g.TlsUserConfig.Ingress.Enabled, _ = strconv.ParseBool(areas[GL_CONFIG_TLS].Objects[1].Fields.(*radioField).Value)
	g.TlsUserConfig.Ingress.CertificateFile = areas[GL_CONFIG_TLS].Objects[2].Fields.(*textField).Value
	g.TlsUserConfig.Ingress.PrivateKeyFile = areas[GL_CONFIG_TLS].Objects[3].Fields.(*textField).Value
	g.TlsUserConfig.Outbound.InsecureFlag, _ = strconv.ParseBool(areas[GL_CONFIG_TLS].Objects[5].Fields.(*radioField).Value)
	g.TlsUserConfig.Outbound.SystemCertPoolFlag, _ = strconv.ParseBool(areas[GL_CONFIG_TLS].Objects[6].Fields.(*radioField).Value)
	g.TlsUserConfig.Outbound.CertFiles = make([]string, len(areas[GL_CONFIG_TLS].Objects[7].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_TLS].Objects[7].Fields.(*arrayField).Values {
		g.TlsUserConfig.Outbound.CertFiles[index] = v.Value
	}

	// Service Bus Config
	g.ServiceBusConfig.Hostname = areas[GL_CONFIG_SERVICEBUSCONFIG].Objects[0].Fields.(*textField).Value
	g.ServiceBusConfig.Topic = areas[GL_CONFIG_SERVICEBUSCONFIG].Objects[1].Fields.(*textField).Value
	g.ServiceBusConfig.TopicExactIsAlive = areas[GL_CONFIG_SERVICEBUSCONFIG].Objects[2].Fields.(*textField).Value
	g.ServiceBusConfig.TopicExactLogger = areas[GL_CONFIG_SERVICEBUSCONFIG].Objects[3].Fields.(*textField).Value
	g.ServiceBusConfig.Token = areas[GL_CONFIG_SERVICEBUSCONFIG].Objects[4].Fields.(*textField).Value

	// Routers

	// Check if any have been removed or added
	countRouters := 0
	for index, _ := range areas[GL_CONFIG_ROUTERS].Objects {
		if areas[GL_CONFIG_ROUTERS].Objects[index].Type == ROUTERS {
			countRouters++
		}
	}
	if countRouters != len(g.Routers) {
		g.Routers = make([]router.Router, countRouters)
		for i := range g.Routers {
			g.Routers[i] = router.Router{}
		}
	}

	// The routers are in the same order, but headlines inbetween
	routerIndex := 0
	for index, _ := range areas[GL_CONFIG_ROUTERS].Objects {
		if areas[GL_CONFIG_ROUTERS].Objects[index].Type == ROUTERS {
			if routerIndex >= len(g.Routers) {
				logger.Error("router out of index", slog.Int("index", routerIndex), slog.Int("len", len(g.Routers)))
				return fmt.Errorf("router out of index")
			}
			g.Routers[routerIndex].Path = areas[GL_CONFIG_ROUTERS].Objects[index].Fields.(*routerField).Fields[0].Fields.(*textField).Value
			g.Routers[routerIndex].Ingress.Headers = func() protectedheader.ProtectedHeader {
				p := protectedheader.ProtectedHeader{}
				for _, header := range areas[GL_CONFIG_ROUTERS].Objects[index].Fields.(*routerField).Fields[2].Fields.(*headerField).Headers {
					values := []string{}
					for _, value := range header.Values {
						values = append(values, value.Value)
					}
					p[header.Header] = values
				}
				return p
			}()
			g.Routers[routerIndex].Outbound.Url = areas[GL_CONFIG_ROUTERS].Objects[index].Fields.(*routerField).Fields[4].Fields.(*textField).Value
			g.Routers[routerIndex].Outbound.Endpoint = areas[GL_CONFIG_ROUTERS].Objects[index].Fields.(*routerField).Fields[5].Fields.(*textField).Value
			g.Routers[routerIndex].Outbound.Headers = func() protectedheader.ProtectedHeader {
				p := protectedheader.ProtectedHeader{}
				for _, header := range areas[GL_CONFIG_ROUTERS].Objects[index].Fields.(*routerField).Fields[6].Fields.(*headerField).Headers {
					values := []string{}
					for _, value := range header.Values {
						values = append(values, value.Value)
					}
					p[header.Header] = values
				}
				return p
			}()
			routerIndex++
		}
	}

	// Request Processors
	// Its easier to just recreate the matrix
	countRequestProcessorsRows := 0
	for index, _ := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects {
		if areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Type == PROCESSORS {
			countRequestProcessorsRows++
		}
	}

	g.RequestProcessors.Processors = make([][]processorconfiguration.ProcessorConfiguration, countRequestProcessorsRows)
	for i := range g.RequestProcessors.Processors {
		// Count processors per row
		countProcessors := 0
		for index, _ := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects[i].Fields.(*processorField).Objects {
			if areas[GL_CONFIG_REQUESTPROCESSORS].Objects[i].Fields.(*processorField).Objects[index].Type == PROCESSORS {
				countProcessors++
			}
		}
		g.RequestProcessors.Processors[i] = make([]processorconfiguration.ProcessorConfiguration, countProcessors)
	}

	rowIndex := 0
	for index, _ := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects {
		if rowIndex >= len(g.RequestProcessors.Processors) {
			logger.Error("row out of index", slog.Int("index", rowIndex), slog.Int("len", len(g.RequestProcessors.Processors)))
			return fmt.Errorf("rowIndex out of index")
		}
		if areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Type == PROCESSORS {
			columnIndex := 0
			for i, _ := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects {
				if columnIndex >= len(g.RequestProcessors.Processors[rowIndex]) {
					logger.Error("column out of index", slog.Int("index", columnIndex), slog.Int("len", len(g.RequestProcessors.Processors[rowIndex])))
					return fmt.Errorf("column out of index")
				}
				if areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Type == PROCESSORS {
					g.RequestProcessors.Processors[rowIndex][columnIndex].Name = areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[0].Fields.(*textField).Value
					g.RequestProcessors.Processors[rowIndex][columnIndex].Modifier, _ = strconv.ParseBool(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[1].Fields.(*radioField).Value)
					g.RequestProcessors.Processors[rowIndex][columnIndex].Required, _ = strconv.ParseBool(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[2].Fields.(*radioField).Value)
					g.RequestProcessors.Processors[rowIndex][columnIndex].Async, _ = strconv.ParseBool(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[3].Fields.(*radioField).Value)
					g.RequestProcessors.Processors[rowIndex][columnIndex].InputFieldsInclude = make([]string, len(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[4].Fields.(*arrayField).Values))
					for j, v := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[4].Fields.(*arrayField).Values {
						g.RequestProcessors.Processors[rowIndex][columnIndex].InputFieldsInclude[j] = v.Value
					}
					g.RequestProcessors.Processors[rowIndex][columnIndex].InputFieldsExclude = make([]string, len(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[5].Fields.(*arrayField).Values))
					for j, v := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[5].Fields.(*arrayField).Values {
						g.RequestProcessors.Processors[rowIndex][columnIndex].InputFieldsExclude[j] = v.Value
					}
					g.RequestProcessors.Processors[rowIndex][columnIndex].OutputFieldsWrite = make([]string, len(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[6].Fields.(*arrayField).Values))
					for j, v := range areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[6].Fields.(*arrayField).Values {
						g.RequestProcessors.Processors[rowIndex][columnIndex].OutputFieldsWrite[j] = v.Value
					}
					g.RequestProcessors.Processors[rowIndex][columnIndex].ServiceBusTopic = areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[7].Fields.(*textField).Value
					g.RequestProcessors.Processors[rowIndex][columnIndex].Timeout, _ = strconv.Atoi(areas[GL_CONFIG_REQUESTPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[8].Fields.(*textField).Value)

					columnIndex++
				}
			}
			rowIndex++
		}
	}

	// Response Processors
	// Its easier to just recreate the matrix
	countResponseProcessorsRows := 0
	for index, _ := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects {
		if areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Type == PROCESSORS {
			countResponseProcessorsRows++
		}
	}

	g.ResponseProcessors.Processors = make([][]processorconfiguration.ProcessorConfiguration, countResponseProcessorsRows)
	for i := range g.ResponseProcessors.Processors {
		// Count processors per row
		countProcessors := 0
		for index, _ := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[i].Fields.(*processorField).Objects {
			if areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[i].Fields.(*processorField).Objects[index].Type == PROCESSORS {
				countProcessors++
			}
		}
		g.ResponseProcessors.Processors[i] = make([]processorconfiguration.ProcessorConfiguration, countProcessors)
	}

	rowIndex = 0
	for index, _ := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects {
		if rowIndex >= len(g.ResponseProcessors.Processors) {
			logger.Error("row out of index", slog.Int("index", rowIndex), slog.Int("len", len(g.ResponseProcessors.Processors)))
			return fmt.Errorf("rowIndex out of index")
		}
		if areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Type == PROCESSORS {
			columnIndex := 0
			for i, _ := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects {
				if columnIndex >= len(g.ResponseProcessors.Processors[rowIndex]) {
					logger.Error("column out of index", slog.Int("index", columnIndex), slog.Int("len", len(g.ResponseProcessors.Processors[rowIndex])))
					return fmt.Errorf("column out of index")
				}
				if areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Type == PROCESSORS {
					g.ResponseProcessors.Processors[rowIndex][columnIndex].Name = areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[0].Fields.(*textField).Value
					g.ResponseProcessors.Processors[rowIndex][columnIndex].Modifier, _ = strconv.ParseBool(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[1].Fields.(*radioField).Value)
					g.ResponseProcessors.Processors[rowIndex][columnIndex].Required, _ = strconv.ParseBool(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[2].Fields.(*radioField).Value)
					g.ResponseProcessors.Processors[rowIndex][columnIndex].Async, _ = strconv.ParseBool(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[3].Fields.(*radioField).Value)
					g.ResponseProcessors.Processors[rowIndex][columnIndex].InputFieldsInclude = make([]string, len(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[4].Fields.(*arrayField).Values))
					for j, v := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[4].Fields.(*arrayField).Values {
						g.ResponseProcessors.Processors[rowIndex][columnIndex].InputFieldsInclude[j] = v.Value
					}
					g.ResponseProcessors.Processors[rowIndex][columnIndex].InputFieldsExclude = make([]string, len(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[5].Fields.(*arrayField).Values))
					for j, v := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[5].Fields.(*arrayField).Values {
						g.ResponseProcessors.Processors[rowIndex][columnIndex].InputFieldsExclude[j] = v.Value
					}
					g.ResponseProcessors.Processors[rowIndex][columnIndex].OutputFieldsWrite = make([]string, len(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[6].Fields.(*arrayField).Values))
					for j, v := range areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[6].Fields.(*arrayField).Values {
						g.ResponseProcessors.Processors[rowIndex][columnIndex].OutputFieldsWrite[j] = v.Value
					}
					g.ResponseProcessors.Processors[rowIndex][columnIndex].ServiceBusTopic = areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[7].Fields.(*textField).Value
					g.ResponseProcessors.Processors[rowIndex][columnIndex].Timeout, _ = strconv.Atoi(areas[GL_CONFIG_RESPONSEPROCESSORS].Objects[index].Fields.(*processorField).Objects[i].Fields.(*processorField).Objects[8].Fields.(*textField).Value)

					columnIndex++
				}
			}
			rowIndex++
		}
	}

	// Logger
	g.Logger.Request.FieldsInclude = make([]string, len(areas[GL_CONFIG_LOGGER].Objects[1].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_LOGGER].Objects[1].Fields.(*arrayField).Values {
		g.Logger.Request.FieldsInclude[index] = v.Value
	}
	g.Logger.Request.FieldsExclude = make([]string, len(areas[GL_CONFIG_LOGGER].Objects[2].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_LOGGER].Objects[2].Fields.(*arrayField).Values {
		g.Logger.Request.FieldsExclude[index] = v.Value
	}
	g.Logger.Response.FieldsInclude = make([]string, len(areas[GL_CONFIG_LOGGER].Objects[4].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_LOGGER].Objects[4].Fields.(*arrayField).Values {
		g.Logger.Response.FieldsInclude[index] = v.Value
	}
	g.Logger.Response.FieldsExclude = make([]string, len(areas[GL_CONFIG_LOGGER].Objects[5].Fields.(*arrayField).Values))
	for index, v := range areas[GL_CONFIG_LOGGER].Objects[5].Fields.(*arrayField).Values {
		g.Logger.Response.FieldsExclude[index] = v.Value
	}

	return nil
}

// returns keys that are changed + error
func (g *Gl_config_v1001) update() (map[string]string, error) {

	// Update settings
	sort.SliceStable(g.MaskedHeaders, func(i, j int) bool {
		return g.MaskedHeaders[i] < g.MaskedHeaders[j]
	})
	sort.SliceStable(g.RemoveHeaders, func(i, j int) bool {
		return g.RemoveHeaders[i] < g.RemoveHeaders[j]
	})

	// Update TLS
	sort.SliceStable(g.TlsUserConfig.Outbound.CertFiles, func(i, j int) bool {
		return g.TlsUserConfig.Outbound.CertFiles[i] < g.TlsUserConfig.Outbound.CertFiles[j]
	})

	// Update Request Processors
	// TODO: add changed keys support
	for a := range g.RequestProcessors.Processors {
		sort.SliceStable(g.RequestProcessors.Processors[a], func(i, j int) bool {
			if g.RequestProcessors.Processors[a][i].Name == "" {
				return false // We want empty names last
			}
			return g.RequestProcessors.Processors[a][i].Name < g.RequestProcessors.Processors[a][j].Name
		})
		for b := range g.RequestProcessors.Processors[a] {
			sort.SliceStable(g.RequestProcessors.Processors[a][b].InputFieldsInclude, func(i, j int) bool {
				return g.RequestProcessors.Processors[a][b].InputFieldsInclude[i] < g.RequestProcessors.Processors[a][b].InputFieldsInclude[j]
			})
			sort.SliceStable(g.RequestProcessors.Processors[a][b].InputFieldsExclude, func(i, j int) bool {
				return g.RequestProcessors.Processors[a][b].InputFieldsExclude[i] < g.RequestProcessors.Processors[a][b].InputFieldsExclude[j]
			})
			sort.SliceStable(g.RequestProcessors.Processors[a][b].OutputFieldsWrite, func(i, j int) bool {
				return g.RequestProcessors.Processors[a][b].OutputFieldsWrite[i] < g.RequestProcessors.Processors[a][b].OutputFieldsWrite[j]
			})
		}
	}

	// Update Response Processors
	// TODO: add changed keys support
	for a := range g.ResponseProcessors.Processors {
		sort.SliceStable(g.ResponseProcessors.Processors[a], func(i, j int) bool {
			if g.ResponseProcessors.Processors[a][i].Name == "" {
				return false // We want empty names last
			}
			return g.ResponseProcessors.Processors[a][i].Name < g.ResponseProcessors.Processors[a][j].Name
		})
		for b := range g.ResponseProcessors.Processors[a] {
			sort.SliceStable(g.ResponseProcessors.Processors[a][b].InputFieldsInclude, func(i, j int) bool {
				return g.ResponseProcessors.Processors[a][b].InputFieldsInclude[i] < g.ResponseProcessors.Processors[a][b].InputFieldsInclude[j]
			})
			sort.SliceStable(g.ResponseProcessors.Processors[a][b].InputFieldsExclude, func(i, j int) bool {
				return g.ResponseProcessors.Processors[a][b].InputFieldsExclude[i] < g.ResponseProcessors.Processors[a][b].InputFieldsExclude[j]
			})
			sort.SliceStable(g.ResponseProcessors.Processors[a][b].OutputFieldsWrite, func(i, j int) bool {
				return g.ResponseProcessors.Processors[a][b].OutputFieldsWrite[i] < g.ResponseProcessors.Processors[a][b].OutputFieldsWrite[j]
			})
		}
	}

	// Update routers
	originalIndices := make([]int, len(g.Routers))
	for i := range g.Routers {
		originalIndices[i] = i
	}

	sort.SliceStable(g.Routers, func(i, j int) bool {
		if g.Routers[i].Outbound.Url < g.Routers[j].Outbound.Url {
			originalIndices[i], originalIndices[j] = originalIndices[j], originalIndices[i]
			return true
		}
		if g.Routers[i].Outbound.Url == g.Routers[j].Outbound.Url {
			if g.Routers[i].Path < g.Routers[j].Path {
				originalIndices[i], originalIndices[j] = originalIndices[j], originalIndices[i]
				return true
			}
		}
		return false
	})

	changedKeys := make(map[string]string)
	for newIndex, originalIndex := range originalIndices {
		if newIndex != originalIndex {
			changedKeys[fmt.Sprintf("router%d", originalIndex)] = fmt.Sprintf("router%d", newIndex)
			//changedKeys = append(changedKeys, fmt.Sprintf("router%d", newIndex))
			logger.Debug("router changed", slog.Int("originalIndex", originalIndex), slog.Int("newIndex", newIndex))
		}
	}

	return changedKeys, nil
}

func (g *Gl_config_v1001) findError(v map[string]string, key string, exact bool) string {
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

	// Non-exact match check
	switch key {

	case "settings":

		prefixes := []string{
			GL_CONFIG_NAME + ".Routers",
			GL_CONFIG_NAME + ".ServiceBusConfig",
			GL_CONFIG_NAME + ".TlsUserConfig",
			GL_CONFIG_NAME + ".RequestProcessors",
			GL_CONFIG_NAME + ".ResponseProcessors",
			GL_CONFIG_NAME + ".Logger",
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
			GL_CONFIG_NAME + ".TlsUserConfig",
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
			GL_CONFIG_NAME + ".ServiceBusConfig",
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

	case "routers":

		_, exists := v[GL_CONFIG_NAME+".Routers"]
		if exists {
			return "invalid"
		}

		prefixes := []string{
			GL_CONFIG_NAME + ".Routers[",
		}

		for key, _ := range v {
			found := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}
			if found {
				return "has rejected"
			}
		}

	case "requestprocessors":

		prefixes := []string{
			GL_CONFIG_NAME + ".RequestProcessors.Processors",
		}

		for key, _ := range v {
			found := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}
			if found {
				return "invalid"
			}
		}

	case "responseprocessors":

		prefixes := []string{
			GL_CONFIG_NAME + ".ResponseProcessors.Processors",
		}

		for key, _ := range v {
			found := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}
			if found {
				return "invalid"
			}
		}

	case "logger":

		prefixes := []string{
			GL_CONFIG_NAME + ".Logger",
		}

		for key, _ := range v {
			found := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					found = true
					break
				}
			}
			if found {
				return "invalid"
			}
		}
	default:
		for row, errorMsg := range v {
			if strings.HasPrefix(row, key) {
				return errorMsg
			}
		}
	}
	return "valid"
}

func errorMsgToTooltipText(errorMsg string) string {
	if errorMsg == "valid" {
		return ""
	}
	tooltips := map[string]string{
		"required:":                   "Cannot be empty, possibly ${ENV} has no value",
		"gt:0":                        "At least one valid entry is required",
		"unique:Path":                 "Router paths need to be unique",
		"file:":                       "File does not exist",
		"router:":                     "no spaces, start and end with /",
		`excludesall: /()<>@;:\"[]?=`: "Remove invalid characters",
		"invalid":                     "This section has invalid entries",
		"has rejected":                "This section is valid but has rejected entries",
		"rejected":                    "This routers has invalid entries",
		"alphanumdot:":                "Only alphanumeric characters and dots",
		"min:1":                       "Value larger than 1 required",
		"min:0":                       "Value larger than 0 required",
	}
	tooltip, found := tooltips[errorMsg]
	if !found {
		return "see https://github.com/go-playground/validator"
	}
	return tooltip
}

func (g *Gl_config_v1001) updateAreasFromConfig(v map[string]string, a []area) error {
	if len(a) != 7 {
		return fmt.Errorf("invalid number of areas")
	}
	if a[GL_CONFIG_SETTINGS].Key != "settings" {
		return fmt.Errorf("settings: invalid area key")
	}
	if a[GL_CONFIG_TLS].Key != "tls" {
		return fmt.Errorf("tls: invalid area key")
	}
	if a[GL_CONFIG_SERVICEBUSCONFIG].Key != "servicebus" {
		return fmt.Errorf("servicebus: invalid area key")
	}
	if a[GL_CONFIG_ROUTERS].Key != "routers" {
		return fmt.Errorf("routers: invalid area key")
	}
	if a[GL_CONFIG_REQUESTPROCESSORS].Key != "requestprocessors" {
		return fmt.Errorf("requestprocessors: invalid area key")
	}
	if a[GL_CONFIG_RESPONSEPROCESSORS].Key != "responseprocessors" {
		return fmt.Errorf("responseprocessors: invalid area key")
	}
	if a[GL_CONFIG_LOGGER].Key != "logger" {
		return fmt.Errorf("logger: invalid area key")
	}

	if len(a[GL_CONFIG_SETTINGS].Objects) != 7 {
		return fmt.Errorf("settings: invalid number of objects")
	}
	if len(a[GL_CONFIG_TLS].Objects) != 8 {
		return fmt.Errorf("tls: invalid number of objects")
	}
	if len(a[GL_CONFIG_SERVICEBUSCONFIG].Objects) != 5 {
		return fmt.Errorf("servicebus: invalid number of objects")
	}

	// ----- Settings -----

	a[GL_CONFIG_SETTINGS].ErrorMsg = g.findError(v, "settings", false)
	a[GL_CONFIG_SETTINGS].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_SETTINGS].ErrorMsg)
	a[GL_CONFIG_SETTINGS].Objects[0].Type = INPUT
	a[GL_CONFIG_SETTINGS].Objects[0].Headline = "Gateway ID"
	a[GL_CONFIG_SETTINGS].Objects[0].Key = "gatewayid"
	a[GL_CONFIG_SETTINGS].Objects[0].Fields = &textField{
		Value:               g.GatewayID,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".GatewayID", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".GatewayID", true)),
		Placeholder:         "TST00001",
	}

	a[GL_CONFIG_SETTINGS].Objects[1].Type = INPUT
	a[GL_CONFIG_SETTINGS].Objects[1].Headline = "Port"
	a[GL_CONFIG_SETTINGS].Objects[1].Key = "port"
	a[GL_CONFIG_SETTINGS].Objects[1].Fields = &textField{
		Value:               fmt.Sprintf("%d", g.Port),
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".Port", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".Port", true)),
		Placeholder:         "5380",
	}

	a[GL_CONFIG_SETTINGS].Objects[2].Type = INPUT
	a[GL_CONFIG_SETTINGS].Objects[2].Headline = "Session ID Header"
	a[GL_CONFIG_SETTINGS].Objects[2].Key = "sessionidheader"
	a[GL_CONFIG_SETTINGS].Objects[2].Fields = &textField{
		Value:               g.SessionIDHeader,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".SessionIDHeader", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".SessionIDHeader", true)),
		Placeholder:         "Session-Id",
	}

	a[GL_CONFIG_SETTINGS].Objects[3].Type = RADIO
	a[GL_CONFIG_SETTINGS].Objects[3].Headline = "Log Level"
	a[GL_CONFIG_SETTINGS].Objects[3].Key = "loglevel"
	a[GL_CONFIG_SETTINGS].Objects[3].Fields = &radioField{
		Value:               g.LogLevel,
		Options:             []string{"DEBUG", "INFO", "WARN", "ERROR"},
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".LogLevel", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".LogLevel", true)),
	}

	maskedHeaders := make([]pair, len(g.MaskedHeaders))
	for index, _ := range g.MaskedHeaders {
		maskedHeaders[index] = pair{
			Value:               g.MaskedHeaders[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".MaskedHeaders[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".MaskedHeaders[%d]", index), false)),
		}
	}
	sort.SliceStable(maskedHeaders, func(i, j int) bool {
		return maskedHeaders[i].Value < maskedHeaders[j].Value
	})
	a[GL_CONFIG_SETTINGS].Objects[4].Type = ARRAY
	a[GL_CONFIG_SETTINGS].Objects[4].Headline = "Masked Headers"
	a[GL_CONFIG_SETTINGS].Objects[4].Key = "maskedheaders"
	a[GL_CONFIG_SETTINGS].Objects[4].Fields = &arrayField{
		Values:      maskedHeaders,
		Placeholder: "Header",
	}

	removeHeaders := make([]pair, len(g.RemoveHeaders))
	for index, _ := range g.RemoveHeaders {
		removeHeaders[index] = pair{
			Value:               g.RemoveHeaders[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RemoveHeaders[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RemoveHeaders[%d]", index), false)),
		}
	}
	sort.SliceStable(removeHeaders, func(i, j int) bool {
		return removeHeaders[i].Value < removeHeaders[j].Value
	})
	a[GL_CONFIG_SETTINGS].Objects[5].Type = ARRAY
	a[GL_CONFIG_SETTINGS].Objects[5].Headline = "Remove Headers"
	a[GL_CONFIG_SETTINGS].Objects[5].Key = "removeheaders"
	a[GL_CONFIG_SETTINGS].Objects[5].Fields = &arrayField{
		Values:      removeHeaders,
		Placeholder: "Header",
	}

	a[GL_CONFIG_SETTINGS].Objects[6].Type = RADIO
	a[GL_CONFIG_SETTINGS].Objects[6].Headline = "Log Unauthorized"
	a[GL_CONFIG_SETTINGS].Objects[6].Key = "logunauthorized"
	a[GL_CONFIG_SETTINGS].Objects[6].Fields = &radioField{
		Value:               fmt.Sprintf("%v", g.TlsUserConfig.Ingress.Enabled),
		Options:             []string{"true", "false"},
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".LogUnauthorized", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".LogUnauthorized", true)),
	}

	// ----- TLS -----

	a[GL_CONFIG_TLS].ErrorMsg = g.findError(v, "tls", false)
	a[GL_CONFIG_TLS].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_TLS].ErrorMsg)
	a[GL_CONFIG_TLS].Objects[0].Type = HEADLINE
	a[GL_CONFIG_TLS].Objects[0].Headline = "Ingress"
	a[GL_CONFIG_TLS].Objects[0].Key = "ingress"
	a[GL_CONFIG_TLS].Objects[0].Fields = &headlineField{
		Value: "Ingress",
	}

	a[GL_CONFIG_TLS].Objects[1].Type = RADIO
	a[GL_CONFIG_TLS].Objects[1].Headline = "Enabled"
	a[GL_CONFIG_TLS].Objects[1].Key = "enabled"
	a[GL_CONFIG_TLS].Objects[1].Fields = &radioField{
		Value:               fmt.Sprintf("%v", g.TlsUserConfig.Ingress.Enabled),
		Options:             []string{"true", "false"},
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Ingress.Enabled", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Ingress.Enabled", true)),
	}

	a[GL_CONFIG_TLS].Objects[2].Type = INPUT
	a[GL_CONFIG_TLS].Objects[2].Headline = "Certificate File"
	a[GL_CONFIG_TLS].Objects[2].Key = "certificatefile"
	a[GL_CONFIG_TLS].Objects[2].Fields = &textField{
		Value:               g.TlsUserConfig.Ingress.CertificateFile,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Ingress.CertificateFile", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Ingress.CertificateFile", true)),
		Placeholder:         "/app/conf/cert.pem",
	}

	a[GL_CONFIG_TLS].Objects[3].Type = INPUT
	a[GL_CONFIG_TLS].Objects[3].Headline = "Private Key File"
	a[GL_CONFIG_TLS].Objects[3].Key = "privatekeyfile"
	a[GL_CONFIG_TLS].Objects[3].Fields = &textField{
		Value:               g.TlsUserConfig.Ingress.PrivateKeyFile,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Ingress.PrivateKeyFile", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Ingress.PrivateKeyFile", true)),
		Placeholder:         "/app/conf/key.pem",
	}

	a[GL_CONFIG_TLS].Objects[4].Type = HEADLINE
	a[GL_CONFIG_TLS].Objects[4].Headline = "Outbound"
	a[GL_CONFIG_TLS].Objects[4].Key = "outbound"
	a[GL_CONFIG_TLS].Objects[4].Fields = &headlineField{
		Value: "Outbound",
	}

	a[GL_CONFIG_TLS].Objects[5].Type = RADIO
	a[GL_CONFIG_TLS].Objects[5].Headline = "Insecure Flag"
	a[GL_CONFIG_TLS].Objects[5].Key = "insecureflag"
	a[GL_CONFIG_TLS].Objects[5].Fields = &radioField{
		Value:               fmt.Sprintf("%v", g.TlsUserConfig.Outbound.InsecureFlag),
		Options:             []string{"true", "false"},
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Outbound.InsecureFlag", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Outbound.InsecureFlag", true)),
	}

	a[GL_CONFIG_TLS].Objects[6].Type = RADIO
	a[GL_CONFIG_TLS].Objects[6].Headline = "System Cert Pool Flag"
	a[GL_CONFIG_TLS].Objects[6].Key = "systemcertpoolflag"
	a[GL_CONFIG_TLS].Objects[6].Fields = &radioField{
		Value:               fmt.Sprintf("%v", g.TlsUserConfig.Outbound.SystemCertPoolFlag),
		Options:             []string{"true", "false"},
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Outbound.SystemCertPoolFlag", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".TlsUserConfig.Outbound.SystemCertPoolFlag", true)),
	}

	a[GL_CONFIG_TLS].Objects[7].Type = ARRAY
	a[GL_CONFIG_TLS].Objects[7].Headline = "CA Certificate Bundle Files"
	a[GL_CONFIG_TLS].Objects[7].Key = "certfiles"
	certFiles := make([]pair, len(g.TlsUserConfig.Outbound.CertFiles))
	for index, _ := range g.TlsUserConfig.Outbound.CertFiles {
		certFiles[index] = pair{
			Value:               g.TlsUserConfig.Outbound.CertFiles[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".TlsUserConfig.Outbound.CertFiles[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".TlsUserConfig.Outbound.CertFiles[%d]", index), false)),
		}
	}
	sort.SliceStable(certFiles, func(i, j int) bool {
		return certFiles[i].Value < certFiles[j].Value
	})
	a[GL_CONFIG_TLS].Objects[7].Fields = &arrayField{
		Values:      certFiles,
		Placeholder: "/app/conf/ca.crt",
	}

	// ----- Service Bus Config -----

	a[GL_CONFIG_SERVICEBUSCONFIG].ErrorMsg = g.findError(v, "servicebus", false)
	a[GL_CONFIG_SERVICEBUSCONFIG].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_SERVICEBUSCONFIG].ErrorMsg)
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[0].Type = INPUT
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[0].Headline = "Hostname"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[0].Key = "hostname"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[0].Fields = &textField{
		Value:               g.ServiceBusConfig.Hostname,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.Hostname", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.Hostname", true)),
		Placeholder:         "localhost:4222",
	}

	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[1].Type = INPUT
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[1].Headline = "Topic"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[1].Key = "topic"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[1].Fields = &textField{
		Value:               g.ServiceBusConfig.Topic,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.Topic", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.Topic", true)),
		Placeholder:         "coburn.gl.gecholog",
	}

	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[2].Type = INPUT
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[2].Headline = "Topic Exact Is Alive"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[2].Key = "topicexactisalive"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[2].Fields = &textField{
		Value:               g.ServiceBusConfig.TopicExactIsAlive,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.TopicExactIsAlive", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.TopicExactIsAlive", true)),
		Placeholder:         "coburn.gl.gecholog.isalive",
	}

	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[3].Type = INPUT
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[3].Headline = "Topic Exact Logger"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[3].Key = "topicexactlogger"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[3].Fields = &textField{
		Value:               g.ServiceBusConfig.TopicExactLogger,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.TopicExactLogger", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.TopicExactLogger", true)),
		Placeholder:         "coburn.gl.logger",
	}

	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[4].Type = INPUT
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[4].Headline = "Token"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[4].Key = "token"
	a[GL_CONFIG_SERVICEBUSCONFIG].Objects[4].Fields = &textField{
		Value:               g.ServiceBusConfig.Token,
		ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.Token", true),
		ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ServiceBusConfig.Token", true)),
		Placeholder:         "changeme",
	}

	// ----- Routers -----

	a[GL_CONFIG_ROUTERS].ErrorMsg = g.findError(v, "routers", false)
	a[GL_CONFIG_ROUTERS].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_ROUTERS].ErrorMsg)

	urls := getUrls(g.Routers)

	if len(g.Routers) != len(a[GL_CONFIG_ROUTERS].Objects)+len(urls) && len(g.Routers) != 0 {
		a[GL_CONFIG_ROUTERS].Objects = make([]inputObject, 0, len(g.Routers)+len(urls))
	}
	if len(urls) == 0 {
		a[GL_CONFIG_ROUTERS].Objects = make([]inputObject, 0)
	}

	duplicatePaths := func() map[string]struct{} {
		m := make(map[string]struct{})
		d := make(map[string]struct{})
		for _, router := range g.Routers {
			_, alreadyExists := m[router.Path]
			if alreadyExists {
				d[router.Path] = struct{}{}
			}
			m[router.Path] = struct{}{}
		}
		return d
	}() // create a map of duplicate paths

	for _, headline := range urls {
		headlineObject := inputObject{}
		headlineObject.Type = HEADLINE
		headlineObject.Headline = headline
		headlineObject.Key = headline
		headlineObject.Fields = &headlineField{
			Value: headline,
		}
		a[GL_CONFIG_ROUTERS].Objects = append(a[GL_CONFIG_ROUTERS].Objects, headlineObject)

		for index, _ := range g.Routers {
			if g.Routers[index].Outbound.Url == headline {
				routerObject := inputObject{}
				routerObject.Type = ROUTERS
				routerObject.Headline = g.Routers[index].Path
				routerObject.TooltipText = "The ingress path to the router."
				routerObject.Key = fmt.Sprintf("router%d", index)
				routerErrorMsg := func() string {
					if g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d]", index), false) != "valid" {
						return "rejected"
					}
					if g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers"), true) == "unique:Path" {
						_, duplicate := duplicatePaths[g.Routers[index].Path]
						if duplicate {
							return "unique:Path"
						}
					}
					return "valid"
				}()

				routerObject.Fields = &routerField{
					ErrorMsg:            routerErrorMsg,
					ErrorMsgTooltipText: errorMsgToTooltipText(routerErrorMsg),
					Fields: []inputObject{
						inputObject{
							Type:        INPUT,
							Headline:    "Path",
							TooltipText: "The ingress path to the router.",
							Key:         "path",
							Fields: &textField{
								Value:               g.Routers[index].Path,
								ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Path", index), true),
								ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Path", index), true)),
								Placeholder:         "/service/standard/",
							},
						},
						inputObject{
							Type:     HEADLINE,
							Headline: "Ingress",
							Key:      "ingress",
							Fields: &headlineField{
								Value: "Ingress",
							},
						},
						inputObject{
							Type:     HEADER,
							Headline: "Headers",
							Key:      "ingress-headers",
							Fields: &headerField{
								Headers: func() []webHeader {
									headers := []webHeader{}
									for header, values := range g.Routers[index].Ingress.Headers {
										webHeader := webHeader{
											Header:              header,
											ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Ingress.Headers[%s]", index, header), true),
											ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Ingress.Headers[%s]", index, header), true)),
										}
										for h, value := range values {
											webHeader.Values = append(webHeader.Values, webHeaderValue{
												Value:               value,
												ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Ingress.Headers[%s][%d]", index, header, h), false),
												ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Ingress.Headers[%s][%d]", index, header, h), false)),
											})
										}
										sort.SliceStable(webHeader.Values, func(i, j int) bool {
											return webHeader.Values[i].Value < webHeader.Values[j].Value
										})
										headers = append(headers, webHeader)
									}
									sort.SliceStable(headers, func(i, j int) bool {
										return headers[i].Header < headers[j].Header
									})
									return headers
								}(),
								ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Ingress.Headers", index), false),
								ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Ingress.Headers", index), false)),
							},
						},
						inputObject{
							Type:     HEADLINE,
							Headline: "Outbound",
							Key:      "outbound",
							Fields: &headlineField{
								Value: "Outbound",
							},
						},
						inputObject{
							Type:     INPUT,
							Headline: "URL",
							Key:      "url",
							Fields: &textField{
								Value:               g.Routers[index].Outbound.Url,
								ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Url", index), true),
								ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Url", index), true)),
								Placeholder:         "https://your.llm.api/",
							},
						},
						inputObject{
							Type:     INPUT,
							Headline: "Endpoint",
							Key:      "endpoint",
							Fields: &textField{
								Value:               g.Routers[index].Outbound.Endpoint,
								ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Endpoint", index), true),
								ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Endpoint", index), true)),
								Placeholder:         "/api/v1/",
							},
						},
						inputObject{
							Type:     HEADER,
							Headline: "Headers",
							Key:      "outbound-headers",
							Fields: &headerField{
								Headers: func() []webHeader {
									headers := []webHeader{}
									for header, values := range g.Routers[index].Outbound.Headers {
										headerErrorMsg := g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Headers[%s]", index, header), true)
										webHeader := webHeader{
											Header:              header,
											ErrorMsg:            headerErrorMsg,
											ErrorMsgTooltipText: errorMsgToTooltipText(headerErrorMsg),
										}
										for h, value := range values {
											webHeader.Values = append(webHeader.Values, webHeaderValue{
												Value:               value,
												ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Headers[%s][%d]", index, header, h), false),
												ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Headers[%s][%d]", index, header, h), false)),
											})
										}
										sort.SliceStable(webHeader.Values, func(i, j int) bool {
											return webHeader.Values[i].Value < webHeader.Values[j].Value
										})
										headers = append(headers, webHeader)
									}
									sort.SliceStable(headers, func(i, j int) bool {
										return headers[i].Header < headers[j].Header
									})
									return headers
								}(),
								ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Headers", index), false),
								ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Routers[%d].Router.Outbound.Headers", index), false)),
							},
						},
					},
				}
				a[GL_CONFIG_ROUTERS].Objects = append(a[GL_CONFIG_ROUTERS].Objects, routerObject)
			}
		}
	}

	// Request Processors

	a[GL_CONFIG_REQUESTPROCESSORS].ErrorMsg = g.findError(v, "requestprocessors", false)
	a[GL_CONFIG_REQUESTPROCESSORS].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_REQUESTPROCESSORS].ErrorMsg)
	a[GL_CONFIG_REQUESTPROCESSORS].Objects = make([]inputObject, len(g.RequestProcessors.Processors))
	for index, _ := range g.RequestProcessors.Processors {
		processorColumn := inputObject{}
		processorColumn.Type = PROCESSORS
		processorColumn.Headline = fmt.Sprintf("column%d", index)
		processorColumn.Key = fmt.Sprintf("processor%d", index)
		processorRows := make([]inputObject, 0, len(g.RequestProcessors.Processors[index]))
		for i, _ := range g.RequestProcessors.Processors[index] {
			processor := inputObject{}
			processor.Type = PROCESSORS
			processor.Headline = g.RequestProcessors.Processors[index][i].Name
			processor.Key = fmt.Sprintf("processor%d%d", index, i)
			rowErrorMsg := func() string {
				if g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d]", index, i), false) == "valid" {
					if g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][", index), false) != "valid" {
						return "valid"
					}
					return g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d]", index), false)
				}
				return "invalid"
			}()
			processor.Fields = &processorField{
				Async:               g.RequestProcessors.Processors[index][i].Async,
				ErrorMsg:            rowErrorMsg,
				ErrorMsgTooltipText: errorMsgToTooltipText(rowErrorMsg),
				Objects: []inputObject{
					inputObject{
						Type:     INPUT,
						Headline: "Name",
						Key:      "name",
						Fields: &textField{
							Value:               g.RequestProcessors.Processors[index][i].Name,
							ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].Name", index, i), true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].Name", index, i), true)),
							Placeholder:         "processor",
						},
					},
					inputObject{
						Type:     RADIO,
						Headline: "Modifier",
						Key:      "modifier",
						Fields: &radioField{
							Value:               fmt.Sprintf("%v", g.RequestProcessors.Processors[index][i].Modifier),
							Options:             []string{"true", "false"},
							ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".RequestProcessors.Processors[index][i].Modifier", true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".RequestProcessors.Processors[index][i].Modifier", true)),
						},
					},
					inputObject{
						Type:     RADIO,
						Headline: "Required",
						Key:      "required",
						Fields: &radioField{
							Value:               fmt.Sprintf("%v", g.RequestProcessors.Processors[index][i].Required),
							Options:             []string{"true", "false"},
							ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".RequestProcessors.Processors[index][i].Required", true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".RequestProcessors.Processors[index][i].Required", true)),
						},
					},
					inputObject{
						Type:     RADIO,
						Headline: "Async",
						Key:      "async",
						Fields: &radioField{
							Value:               fmt.Sprintf("%v", g.RequestProcessors.Processors[index][i].Async),
							Options:             []string{"true", "false"},
							ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".RequestProcessors.Processors[index][i].Async", true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".RequestProcessors.Processors[index][i].Async", true)),
						},
					},
					inputObject{
						Type:     ARRAY,
						Headline: "Input Fields Include",
						Key:      "inputfieldsinclude",
						Fields: &arrayField{
							Values: func() []pair {
								values := make([]pair, len(g.RequestProcessors.Processors[index][i].InputFieldsInclude))
								for index, value := range g.RequestProcessors.Processors[index][i].InputFieldsInclude {
									values[index] = pair{
										Value:               value,
										ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].InputFieldsInclude[%d]", index, i, index), false),
										ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].InputFieldsInclude[%d]", index, i, index), false)),
									}
								}
								sort.SliceStable(values, func(i, j int) bool {
									return values[i].Value < values[j].Value
								})
								return values
							}(),
							Placeholder: "field",
						},
					},
					inputObject{
						Type:     ARRAY,
						Headline: "Input Fields Exclude",
						Key:      "inputfieldsexclude",
						Fields: &arrayField{
							Values: func() []pair {
								values := make([]pair, len(g.RequestProcessors.Processors[index][i].InputFieldsExclude))
								for index, value := range g.RequestProcessors.Processors[index][i].InputFieldsExclude {
									values[index] = pair{
										Value:               value,
										ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].InputFieldsExclude[%d]", index, i, index), false),
										ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].InputFieldsExclude[%d]", index, i, index), false)),
									}
								}
								sort.SliceStable(values, func(i, j int) bool {
									return values[i].Value < values[j].Value
								})
								return values
							}(),
							Placeholder: "field",
						},
					},
					inputObject{
						Type:     ARRAY,
						Headline: "Output Fields Write",
						Key:      "outputfieldswrite",
						Fields: &arrayField{
							Values: func() []pair {
								values := make([]pair, len(g.RequestProcessors.Processors[index][i].OutputFieldsWrite))
								for index, value := range g.RequestProcessors.Processors[index][i].OutputFieldsWrite {
									values[index] = pair{
										Value:               value,
										ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].OutputFieldsWrite[%d]", index, i, index), false),
										ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].OutputFieldsWrite[%d]", index, i, index), false)),
									}
								}
								sort.SliceStable(values, func(i, j int) bool {
									return values[i].Value < values[j].Value
								})
								return values
							}(),
							Placeholder: "field",
						},
					},
					inputObject{
						Type:     INPUT,
						Headline: "Service Bus Topic",
						Key:      "servicebustopic",
						Fields: &textField{
							Value:               g.RequestProcessors.Processors[index][i].ServiceBusTopic,
							ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].ServiceBusTopic", index, i), true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].ServiceBusTopic", index, i), true)),
							Placeholder:         "coburn.gl.gecholog",
						},
					},
					inputObject{
						Type:     INPUT,
						Headline: "Timeout",
						Key:      "timeout",
						Fields: &textField{
							Value:               fmt.Sprintf("%d", g.RequestProcessors.Processors[index][i].Timeout),
							ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].Timeout", index, i), true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d][%d].Timeout", index, i), true)),
							Placeholder:         "50",
						},
					},
				},
			}
			processorRows = append(processorRows, processor)
		}
		processorColumn.Fields = &processorField{
			Objects:             processorRows,
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".RequestProcessors.Processors[%d]", index), false)),
		}
		a[GL_CONFIG_REQUESTPROCESSORS].Objects[index] = processorColumn
	}

	// Response Processors

	a[GL_CONFIG_RESPONSEPROCESSORS].ErrorMsg = g.findError(v, "responseprocessors", false)
	a[GL_CONFIG_RESPONSEPROCESSORS].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_RESPONSEPROCESSORS].ErrorMsg)
	a[GL_CONFIG_RESPONSEPROCESSORS].Objects = make([]inputObject, len(g.ResponseProcessors.Processors))
	for index, _ := range g.ResponseProcessors.Processors {
		processorColumn := inputObject{}
		processorColumn.Type = PROCESSORS
		processorColumn.Headline = fmt.Sprintf("column%d", index)
		processorColumn.Key = fmt.Sprintf("processor%d", index)
		processorRows := make([]inputObject, 0, len(g.ResponseProcessors.Processors[index]))
		for i, _ := range g.ResponseProcessors.Processors[index] {
			processor := inputObject{}
			processor.Type = PROCESSORS
			processor.Headline = g.ResponseProcessors.Processors[index][i].Name
			processor.Key = fmt.Sprintf("processor%d%d", index, i)
			rowErrorMsg := func() string {
				if g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d]", index, i), false) == "valid" {
					if g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][", index), false) != "valid" {
						return "valid"
					}
					return g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d]", index), false)
				}
				return "invalid"
			}()
			processor.Fields = &processorField{
				Async:               g.ResponseProcessors.Processors[index][i].Async,
				ErrorMsg:            rowErrorMsg,
				ErrorMsgTooltipText: errorMsgToTooltipText(rowErrorMsg),
				Objects: []inputObject{
					inputObject{
						Type:     INPUT,
						Headline: "Name",
						Key:      "name",
						Fields: &textField{
							Value:               g.ResponseProcessors.Processors[index][i].Name,
							ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].Name", index, i), true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].Name", index, i), true)),
							Placeholder:         "processor",
						},
					},
					inputObject{
						Type:     RADIO,
						Headline: "Modifier",
						Key:      "modifier",
						Fields: &radioField{
							Value:               fmt.Sprintf("%v", g.ResponseProcessors.Processors[index][i].Modifier),
							Options:             []string{"true", "false"},
							ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ResponseProcessors.Processors[index][i].Modifier", true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ResponseProcessors.Processors[index][i].Modifier", true)),
						},
					},
					inputObject{
						Type:     RADIO,
						Headline: "Required",
						Key:      "required",
						Fields: &radioField{
							Value:               fmt.Sprintf("%v", g.ResponseProcessors.Processors[index][i].Required),
							Options:             []string{"true", "false"},
							ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ResponseProcessors.Processors[index][i].Required", true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ResponseProcessors.Processors[index][i].Required", true)),
						},
					},
					inputObject{
						Type:     RADIO,
						Headline: "Async",
						Key:      "async",
						Fields: &radioField{
							Value:               fmt.Sprintf("%v", g.ResponseProcessors.Processors[index][i].Async),
							Options:             []string{"true", "false"},
							ErrorMsg:            g.findError(v, GL_CONFIG_NAME+".ResponseProcessors.Processors[index][i].Async", true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, GL_CONFIG_NAME+".ResponseProcessors.Processors[index][i].Async", true)),
						},
					},
					inputObject{
						Type:     ARRAY,
						Headline: "Input Fields Include",
						Key:      "inputfieldsinclude",
						Fields: &arrayField{
							Values: func() []pair {
								values := make([]pair, len(g.ResponseProcessors.Processors[index][i].InputFieldsInclude))
								for index, value := range g.ResponseProcessors.Processors[index][i].InputFieldsInclude {
									values[index] = pair{
										Value:               value,
										ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].InputFieldsInclude[%d]", index, i, index), false),
										ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].InputFieldsInclude[%d]", index, i, index), false)),
									}
								}
								sort.SliceStable(values, func(i, j int) bool {
									return values[i].Value < values[j].Value
								})
								return values
							}(),
							Placeholder: "field",
						},
					},
					inputObject{
						Type:     ARRAY,
						Headline: "Input Fields Exclude",
						Key:      "inputfieldsexclude",
						Fields: &arrayField{
							Values: func() []pair {
								values := make([]pair, len(g.ResponseProcessors.Processors[index][i].InputFieldsExclude))
								for index, value := range g.ResponseProcessors.Processors[index][i].InputFieldsExclude {
									values[index] = pair{
										Value:               value,
										ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].InputFieldsExclude[%d]", index, i, index), false),
										ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].InputFieldsExclude[%d]", index, i, index), false)),
									}
								}
								sort.SliceStable(values, func(i, j int) bool {
									return values[i].Value < values[j].Value
								})
								return values
							}(),
							Placeholder: "field",
						},
					},
					inputObject{
						Type:     ARRAY,
						Headline: "Output Fields Write",
						Key:      "outputfieldswrite",
						Fields: &arrayField{
							Values: func() []pair {
								values := make([]pair, len(g.ResponseProcessors.Processors[index][i].OutputFieldsWrite))
								for index, value := range g.ResponseProcessors.Processors[index][i].OutputFieldsWrite {
									values[index] = pair{
										Value:               value,
										ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].OutputFieldsWrite[%d]", index, i, index), false),
										ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].OutputFieldsWrite[%d]", index, i, index), false)),
									}
								}
								sort.SliceStable(values, func(i, j int) bool {
									return values[i].Value < values[j].Value
								})
								return values
							}(),
							Placeholder: "field",
						},
					},
					inputObject{
						Type:     INPUT,
						Headline: "Service Bus Topic",
						Key:      "servicebustopic",
						Fields: &textField{
							Value:               g.ResponseProcessors.Processors[index][i].ServiceBusTopic,
							ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].ServiceBusTopic", index, i), true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].ServiceBusTopic", index, i), true)),
							Placeholder:         "coburn.gl.gecholog",
						},
					},
					inputObject{
						Type:     INPUT,
						Headline: "Timeout",
						Key:      "timeout",
						Fields: &textField{
							Value:               fmt.Sprintf("%d", g.ResponseProcessors.Processors[index][i].Timeout),
							ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].Timeout", index, i), true),
							ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d][%d].Timeout", index, i), true)),
							Placeholder:         "50",
						},
					},
				},
			}
			processorRows = append(processorRows, processor)
		}
		processorColumn.Fields = &processorField{
			Objects:             processorRows,
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".ResponseProcessors.Processors[%d]", index), false)),
		}
		a[GL_CONFIG_RESPONSEPROCESSORS].Objects[index] = processorColumn
	}

	// Logger

	a[GL_CONFIG_LOGGER].ErrorMsg = g.findError(v, "logger", false)
	a[GL_CONFIG_LOGGER].ErrorMsgTooltipText = errorMsgToTooltipText(a[GL_CONFIG_LOGGER].ErrorMsg)
	a[GL_CONFIG_LOGGER].Objects[0].Type = HEADLINE
	a[GL_CONFIG_LOGGER].Objects[0].Headline = "Request"
	a[GL_CONFIG_LOGGER].Objects[0].Key = "request"
	a[GL_CONFIG_LOGGER].Objects[0].Fields = &headlineField{
		Value: "Request",
	}

	a[GL_CONFIG_LOGGER].Objects[1].Type = ARRAY
	a[GL_CONFIG_LOGGER].Objects[1].Headline = "Fields Include"
	a[GL_CONFIG_LOGGER].Objects[1].Key = "requestfieldsinclude"
	requestFieldsInclude := make([]pair, len(g.Logger.Request.FieldsInclude))
	for index, _ := range g.Logger.Request.FieldsInclude {
		requestFieldsInclude[index] = pair{
			Value:               g.Logger.Request.FieldsInclude[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Request.FieldsInclude[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Request.FieldsInclude[%d]", index), false)),
		}
	}
	sort.SliceStable(requestFieldsInclude, func(i, j int) bool {
		return requestFieldsInclude[i].Value < requestFieldsInclude[j].Value
	})
	a[GL_CONFIG_LOGGER].Objects[1].Fields = &arrayField{
		Values:      requestFieldsInclude,
		Placeholder: "field",
	}

	a[GL_CONFIG_LOGGER].Objects[2].Type = ARRAY
	a[GL_CONFIG_LOGGER].Objects[2].Headline = "Fields Exclude"
	a[GL_CONFIG_LOGGER].Objects[2].Key = "requestfieldsexclude"
	requestFieldsExclude := make([]pair, len(g.Logger.Request.FieldsExclude))
	for index, _ := range g.Logger.Request.FieldsExclude {
		requestFieldsExclude[index] = pair{
			Value:               g.Logger.Request.FieldsExclude[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Request.FieldsExclude[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Request.FieldsExclude[%d]", index), false)),
		}
	}
	sort.SliceStable(requestFieldsExclude, func(i, j int) bool {
		return requestFieldsExclude[i].Value < requestFieldsExclude[j].Value
	})
	a[GL_CONFIG_LOGGER].Objects[2].Fields = &arrayField{
		Values:      requestFieldsExclude,
		Placeholder: "field",
	}

	a[GL_CONFIG_LOGGER].Objects[3].Type = HEADLINE
	a[GL_CONFIG_LOGGER].Objects[3].Headline = "Response"
	a[GL_CONFIG_LOGGER].Objects[3].Key = "response"
	a[GL_CONFIG_LOGGER].Objects[3].Fields = &headlineField{
		Value: "Response",
	}

	a[GL_CONFIG_LOGGER].Objects[4].Type = ARRAY
	a[GL_CONFIG_LOGGER].Objects[4].Headline = "Fields Include"
	a[GL_CONFIG_LOGGER].Objects[4].Key = "responsefieldsinclude"
	responseFieldsInclude := make([]pair, len(g.Logger.Response.FieldsInclude))
	for index, _ := range g.Logger.Response.FieldsInclude {
		responseFieldsInclude[index] = pair{
			Value:               g.Logger.Response.FieldsInclude[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Response.FieldsInclude[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Response.FieldsInclude[%d]", index), false)),
		}
	}
	sort.SliceStable(responseFieldsInclude, func(i, j int) bool {
		return responseFieldsInclude[i].Value < responseFieldsInclude[j].Value
	})
	a[GL_CONFIG_LOGGER].Objects[4].Fields = &arrayField{
		Values:      responseFieldsInclude,
		Placeholder: "field",
	}

	a[GL_CONFIG_LOGGER].Objects[5].Type = ARRAY
	a[GL_CONFIG_LOGGER].Objects[5].Headline = "Fields Exclude"
	a[GL_CONFIG_LOGGER].Objects[5].Key = "responsefieldsexclude"
	responseFieldsExclude := make([]pair, len(g.Logger.Response.FieldsExclude))
	for index, _ := range g.Logger.Response.FieldsExclude {
		responseFieldsExclude[index] = pair{
			Value:               g.Logger.Response.FieldsExclude[index],
			ErrorMsg:            g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Response.FieldsExclude[%d]", index), false),
			ErrorMsgTooltipText: errorMsgToTooltipText(g.findError(v, fmt.Sprintf(GL_CONFIG_NAME+".Logger.Response.FieldsExclude[%d]", index), false)),
		}
	}
	sort.SliceStable(responseFieldsExclude, func(i, j int) bool {
		return responseFieldsExclude[i].Value < responseFieldsExclude[j].Value
	})
	a[GL_CONFIG_LOGGER].Objects[5].Fields = &arrayField{
		Values:      responseFieldsExclude,
		Placeholder: "field",
	}

	return nil
}

func (g *Gl_config_v1001) createAreas(v map[string]string) ([]area, error) {
	a := make([]area, 7)

	// Settings
	settingsObjects := make([]inputObject, 7)
	for i := range settingsObjects {
		settingsObjects[i] = inputObject{}
	}
	a[GL_CONFIG_SETTINGS] = area{
		Headline: "Settings",
		Key:      "settings",
		Redirect: "menu",
		Form:     "settings-form",
		Objects:  settingsObjects,
	}

	// TLS
	tlsObjects := make([]inputObject, 8)
	for i := range tlsObjects {
		tlsObjects[i] = inputObject{}
	}
	a[GL_CONFIG_TLS] = area{
		Headline: "TLS",
		Key:      "tls",
		Redirect: "menu",
		Form:     "tls-form",
		Objects:  tlsObjects,
	}

	// Service Bus Config
	serviceBusObjects := make([]inputObject, 5)
	for i := range serviceBusObjects {
		serviceBusObjects[i] = inputObject{}
	}
	a[GL_CONFIG_SERVICEBUSCONFIG] = area{
		Headline: "Service Bus Config",
		Key:      "servicebus",
		Redirect: "menu",
		Form:     "servicebus-form",
		Objects:  serviceBusObjects,
	}

	// Routers
	urls := getUrls(g.Routers)
	routerObjects := make([]inputObject, 0, len(g.Routers)+len(urls))
	for i := range routerObjects {
		routerObjects[i] = inputObject{}
	}
	a[GL_CONFIG_ROUTERS] = area{
		Headline: "Routers",
		Key:      "routers",
		Redirect: "routers",
		Form:     "routers",
		Objects:  routerObjects,
	}

	// Request Processors sync
	processorObjects := make([]inputObject, 0, len(g.RequestProcessors.Processors))
	for i := range processorObjects {
		processorObjects[i] = inputObject{}
	}
	a[GL_CONFIG_REQUESTPROCESSORS] = area{
		Headline: "Request Processors",
		Key:      "requestprocessors",
		Redirect: "requestprocessors",
		Form:     "requestprocessors",
		Objects:  processorObjects,
	}

	// Response Processors sync
	processorObjects = make([]inputObject, 0, len(g.ResponseProcessors.Processors))
	for i := range processorObjects {
		processorObjects[i] = inputObject{}
	}
	a[GL_CONFIG_RESPONSEPROCESSORS] = area{
		Headline: "Response Processors",
		Key:      "responseprocessors",
		Redirect: "responseprocessors",
		Form:     "responseprocessors",
		Objects:  processorObjects,
	}

	// Logger
	loggerObjects := make([]inputObject, 6)
	for i := range loggerObjects {
		loggerObjects[i] = inputObject{}
	}
	a[GL_CONFIG_LOGGER] = area{
		Headline: "Logger",
		Key:      "logger",
		Redirect: "menu",
		Form:     "logger-form",
		Objects:  loggerObjects,
	}

	err := g.updateAreasFromConfig(v, a)
	return a, err
}

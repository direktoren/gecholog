package processorconfiguration

type ProcessorConfiguration struct {
	Name               string   `json:"name" validate:"required,alphanumunderscore"`
	Modifier           bool     `json:"modifier"`
	Required           bool     `json:"required"`
	Async              bool     `json:"async"`
	InputFieldsInclude []string `json:"input_fields_include" validate:"unique,dive,alphanumunderscore"`
	InputFieldsExclude []string `json:"input_fields_exclude" validate:"unique,dive,alphanumunderscore"`
	OutputFieldsWrite  []string `json:"output_fields_write" validate:"unique,dive,alphanumunderscore"`
	ServiceBusTopic    string   `json:"service_bus_topic" validate:"required,alphanumdot"`
	Timeout            int      `json:"timeout" validate:"min=1"`
}

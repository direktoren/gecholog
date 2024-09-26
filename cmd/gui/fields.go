package main

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

const (
	INPUT      = iota // 0
	BOOL              // 1
	RADIO             // 2
	ARRAY             // 3
	HEADLINE          // 4
	ROUTERS           // 5
	HEADER            // 6
	PROCESSORS        // 7
)

type inputObject struct {
	Type        int
	Headline    string
	TooltipText string
	Key         string
	Fields      field
}

type field interface {
	ErrorMessage() string
	ErrorMessageTooltipText() string
	setValue([]string) error
	copy() field
	new() field
}

type textField struct {
	Value       string
	Placeholder string

	ErrorMsg            string
	ErrorMsgTooltipText string
}

func (t *textField) ErrorMessage() string {
	return t.ErrorMsg
}

func (t *textField) ErrorMessageTooltipText() string {
	return t.ErrorMsgTooltipText
}

func (t *textField) setValue(value []string) error {
	if len(value) == 0 {
		return nil // not an error
	}
	t.Value = value[0]
	return nil
}

func (t *textField) copy() field {
	return &textField{
		Value:               t.Value,
		Placeholder:         t.Placeholder,
		ErrorMsg:            t.ErrorMsg,
		ErrorMsgTooltipText: t.ErrorMsgTooltipText,
	}
}

func (t *textField) new() field {
	return &textField{}
}

type radioField struct {
	Value   string
	Options []string

	ErrorMsg            string
	ErrorMsgTooltipText string
}

func (r *radioField) ErrorMessage() string {
	return r.ErrorMsg
}

func (r *radioField) ErrorMessageTooltipText() string {
	return r.ErrorMsgTooltipText
}

func (r *radioField) setValue(value []string) error {
	if len(value) == 0 {
		return nil // not an error
	}

	for _, v := range r.Options {
		if v == value[0] {
			r.Value = value[0]
			return nil // Success
		}
	}
	return fmt.Errorf("value %s not in options", value[0])
}

func (r *radioField) copy() field {
	return &radioField{
		Value: r.Value,
		Options: func() []string {
			opt := make([]string, len(r.Options))
			copy(opt, r.Options)
			return opt
		}(),
		ErrorMsg:            r.ErrorMsg,
		ErrorMsgTooltipText: r.ErrorMsgTooltipText,
	}
}

func (r *radioField) new() field {
	return &radioField{
		Options: []string{},
	}
}

type pair struct {
	Value               string
	ErrorMsg            string
	ErrorMsgTooltipText string
}

type arrayField struct {
	Values      []pair
	Placeholder string
}

func (a *arrayField) ErrorMessage() string {
	str := ""
	for _, v := range a.Values {
		str += v.ErrorMsg + " "
	}
	return str
}

func (a *arrayField) ErrorMessageTooltipText() string {
	str := ""
	for _, v := range a.Values {
		str += v.ErrorMsgTooltipText + " "
	}
	return str
}

func (a *arrayField) setValue(value []string) error {

	a.Values = make([]pair, 0, len(value))
	for _, v := range value {
		if v == "" {
			continue
		}
		a.Values = append(a.Values, pair{Value: v})
	}
	sort.SliceStable(a.Values, func(i, j int) bool {
		return a.Values[i].Value < a.Values[j].Value
	})
	return nil
}

func (a *arrayField) copy() field {
	values := make([]pair, len(a.Values))
	for i, v := range a.Values {
		values[i] = pair{
			Value:               v.Value,
			ErrorMsg:            v.ErrorMsg,
			ErrorMsgTooltipText: v.ErrorMsgTooltipText,
		}
	}
	return &arrayField{
		Values:      values,
		Placeholder: a.Placeholder,
	}
}

func (a *arrayField) new() field {
	return &arrayField{
		Values: []pair{},
	}
}

type headlineField struct {
	Value string
}

func (h *headlineField) ErrorMessage() string {
	return ""
}

func (h *headlineField) ErrorMessageTooltipText() string {
	return ""
}

func (h *headlineField) setValue(value []string) error {
	/*if len(value) == 0 {
		return nil // not an error
	}
	h.Value = value[0]*/
	return nil
}

func (h *headlineField) copy() field {
	return &headlineField{
		Value: h.Value,
	}
}

func (h *headlineField) new() field {
	return &headlineField{}
}

type routerField struct {
	Fields              []inputObject
	ErrorMsg            string
	ErrorMsgTooltipText string
}

func (r *routerField) ErrorMessage() string {
	return r.ErrorMsg
}

func (r *routerField) ErrorMessageTooltipText() string {
	return r.ErrorMsgTooltipText
}

func (r *routerField) setValue(value []string) error {
	// This is setting path
	if len(r.Fields) == 0 {
		logger.Warn("routerField setValue: Fields is empty")
		return fmt.Errorf("fields is empty")
	}
	if len(value) == 0 {
		logger.Warn("routerField setValue: value is empty")
		return fmt.Errorf("value is empty")
	}
	for i, _ := range r.Fields {
		if r.Fields[i].Key == "path" {
			logger.Debug("routerField setValue", slog.String("value", value[0]))
			r.Fields[i].Fields.setValue(value)
			return nil
		}
	}
	return fmt.Errorf("path not found")
}

func (r *routerField) new() field {
	return &routerField{
		Fields: []inputObject{
			inputObject{
				Type:     INPUT,
				Headline: "Path",
				Key:      "path",
				Fields: &textField{
					Value:       "",
					ErrorMsg:    "",
					Placeholder: "/service/standard/",
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
					Headers:  []webHeader{},
					ErrorMsg: "",
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
					Value:       "",
					ErrorMsg:    "",
					Placeholder: "https://your.llm.api/",
				},
			},
			inputObject{
				Type:     INPUT,
				Headline: "Endpoint",
				Key:      "endpoint",
				Fields: &textField{
					Value:       "",
					ErrorMsg:    "",
					Placeholder: "/api/v1/",
				},
			},
			inputObject{
				Type:     HEADER,
				Headline: "Headers",
				Key:      "outbound-headers",
				Fields: &headerField{
					Headers:  []webHeader{},
					ErrorMsg: "",
				},
			},
		},
		ErrorMsg: "",
	}
}

func (r *routerField) copy() field {
	fields := make([]inputObject, len(r.Fields))
	for i, f := range r.Fields {
		fields[i] = inputObject{
			Type:     f.Type,
			Headline: f.Headline,
			Key:      f.Key,
			Fields:   f.Fields.copy(),
		}
	}
	for i, f := range fields {
		if f.Key == "path" {
			fields[i].Fields = &textField{
				Value:               f.Fields.(*textField).Value + "copy/",
				ErrorMsg:            f.Fields.ErrorMessage(),
				ErrorMsgTooltipText: f.Fields.ErrorMessageTooltipText(),
			}
		}
	}
	return &routerField{
		Fields:   fields,
		ErrorMsg: r.ErrorMsg,
	}
}

type webHeaderValue struct {
	Value               string
	ErrorMsg            string
	ErrorMsgTooltipText string
}

type webHeader struct {
	Header              string
	Values              []webHeaderValue
	ErrorMsg            string
	ErrorMsgTooltipText string
}

type headerField struct {
	Headers             []webHeader
	ErrorMsg            string
	ErrorMsgTooltipText string
}

func (h *headerField) ErrorMessage() string {
	return h.ErrorMsg
}

func (h *headerField) ErrorMessageTooltipText() string {
	return h.ErrorMsgTooltipText
}

func (h *headerField) setValue(value []string) error {
	// Data will arrive in pairs in a long string
	// header1, value1, header1, value2, header2, value3,  etc
	if len(value)%2 != 0 {
		logger.Warn("headerField setValue: len(value) % 2 != 0", slog.Any("value", value))
		return fmt.Errorf("len(value) mod 2 != 0")
	}
	tmpHeaders := map[string]webHeader{}
	for i := 0; i < len(value); i += 2 {
		if value[i] == "" {
			continue
		}
		_, exists := tmpHeaders[value[i]]
		if exists {
			webHeader := tmpHeaders[value[i]]
			webHeader.Values = append(tmpHeaders[value[i]].Values, webHeaderValue{Value: strings.TrimSpace(value[i+1])})
			tmpHeaders[value[i]] = webHeader
			continue
		}
		tmpHeaders[value[i]] = webHeader{
			Header: value[i],
			Values: []webHeaderValue{{Value: strings.TrimSpace(value[i+1])}},
		}
	}
	h.Headers = []webHeader{}
	for _, v := range tmpHeaders {
		sort.SliceStable(v.Values, func(i, j int) bool {
			return v.Values[i].Value < v.Values[j].Value
		})
		h.Headers = append(h.Headers, v)
	}
	sort.SliceStable(h.Headers, func(i, j int) bool {
		return h.Headers[i].Header < h.Headers[j].Header
	})
	return nil
}

func (h *headerField) copy() field {
	headers := make([]webHeader, len(h.Headers))
	for i, header := range h.Headers {
		values := make([]webHeaderValue, len(header.Values))
		for j, value := range header.Values {
			values[j] = webHeaderValue{
				Value:               value.Value,
				ErrorMsg:            value.ErrorMsg,
				ErrorMsgTooltipText: value.ErrorMsgTooltipText,
			}
		}
		headers[i] = webHeader{
			Header:              header.Header,
			Values:              values,
			ErrorMsg:            header.ErrorMsg,
			ErrorMsgTooltipText: header.ErrorMsgTooltipText,
		}
	}
	return &headerField{
		Headers:             headers,
		ErrorMsg:            h.ErrorMsg,
		ErrorMsgTooltipText: h.ErrorMsgTooltipText,
	}
}

func (h *headerField) new() field {
	return &headerField{
		Headers:             []webHeader{},
		ErrorMsg:            "",
		ErrorMsgTooltipText: "",
	}
}

type processorField struct {
	Async               bool
	Objects             []inputObject
	ErrorMsg            string
	ErrorMsgTooltipText string
}

func (p *processorField) ErrorMessage() string {
	return p.ErrorMsg
}

func (p *processorField) ErrorMessageTooltipText() string {
	return p.ErrorMsgTooltipText
}

func (p *processorField) setValue(value []string) error {
	// This is setting path
	if len(p.Objects) == 0 {
		logger.Warn("processorRowField setValue: Fields is empty")
		return fmt.Errorf("fields is empty")
	}
	if len(value) == 0 {
		logger.Warn("processorRowField setValue: value is empty")
		return fmt.Errorf("value is empty")
	}
	for i, _ := range p.Objects {
		if p.Objects[i].Key == "name" {
			logger.Debug("processorRowField setValue", slog.String("value", value[0]))
			p.Objects[i].Fields.setValue(value)
			return nil
		}
		if p.Objects[i].Type == PROCESSORS {
			subFields := p.Objects[i].Fields.(*processorField)
			for j, _ := range subFields.Objects {
				if subFields.Objects[j].Key == "name" {
					subFields.Objects[j].Fields.setValue(value)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("name not found")
}

func (p *processorField) new() field {
	return &processorField{
		Async:    false,
		ErrorMsg: "",
		Objects: []inputObject{
			inputObject{
				Type:     INPUT,
				Headline: "Name",
				Key:      "name",
				Fields: &textField{
					Value:       "",
					ErrorMsg:    "",
					Placeholder: "processor",
				},
			},
			inputObject{
				Type:     RADIO,
				Headline: "Modifier",
				Key:      "modifier",
				Fields: &radioField{
					Value:    "false",
					Options:  []string{"true", "false"},
					ErrorMsg: "",
				},
			},
			inputObject{
				Type:     RADIO,
				Headline: "Required",
				Key:      "required",
				Fields: &radioField{
					Value:    "false",
					Options:  []string{"true", "false"},
					ErrorMsg: "",
				},
			},
			inputObject{
				Type:     RADIO,
				Headline: "Async",
				Key:      "async",
				Fields: &radioField{
					Value:    "false",
					Options:  []string{"true", "false"},
					ErrorMsg: "",
				},
			},
			inputObject{
				Type:     ARRAY,
				Headline: "Input Fields Include",
				Key:      "inputfieldsinclude",
				Fields: &arrayField{
					Values:      []pair{},
					Placeholder: "field",
				},
			},
			inputObject{
				Type:     ARRAY,
				Headline: "Input Fields Exclude",
				Key:      "inputfieldsexclude",
				Fields: &arrayField{
					Values:      []pair{},
					Placeholder: "field",
				},
			},

			inputObject{
				Type:     ARRAY,
				Headline: "Output Fields Write",
				Key:      "outputfieldswrite",
				Fields: &arrayField{
					Values:      []pair{},
					Placeholder: "field",
				},
			},
			inputObject{
				Type:     INPUT,
				Headline: "Service Bus Topic",
				Key:      "servicebustopic",
				Fields: &textField{
					Value:       "",
					ErrorMsg:    "",
					Placeholder: "coburn.gl.processor",
				},
			},
			inputObject{
				Type:     INPUT,
				Headline: "Timeout",
				Key:      "timeout",
				Fields: &textField{
					Value:       "50",
					ErrorMsg:    "",
					Placeholder: "50",
				},
			},
		},
	}
}

func (p *processorField) copy() field {
	// noop
	return nil
}

package router

import (
	"fmt"

	"github.com/direktoren/gecholog/internal/protectedheader"
	"github.com/direktoren/gecholog/internal/validate"
)

// Not sure if this is the right place/needed. Kept here for legacy reasons
var CommonHeaders = map[string]struct{}{
	"Accept":                    {},
	"Cache-Control":             {},
	"Content-Length":            {},
	"Content-Type":              {},
	"Date":                      {},
	"Host":                      {},
	"User-Agent":                {},
	"Referer":                   {},
	"Authorization":             {},
	"Cookie":                    {},
	"Connection":                {},
	"Upgrade-Insecure-Requests": {},
	"Pragma":                    {},
	"Warning":                   {},
	"Expires":                   {},
	"Location":                  {},
	"If-Modified-Since":         {},
	"Last-Modified":             {},
	"Server":                    {},
	"Etag":                      {},
}

type IngressNode struct {
	Headers protectedheader.ProtectedHeader `json:"headers" validate:"omitempty,dive,keys,ascii,excludesall= /()<>@;:\\\"[]?=,endkeys,gt=0,dive,required,ascii"`
	//	Headers protectedheader.ProtectedHeader `json:"headers" validate:"omitempty,dive,keys,ascii,excludesall= /()<>@;:\\\"[]?=,endkeys,gt=0,dive,ascii"`
}

type OutboundNode struct {
	Url      string                          `json:"url" validate:"required,http_url"`
	Endpoint string                          `json:"endpoint" validate:"omitempty,endpoint"`
	Headers  protectedheader.ProtectedHeader `json:"headers" validate:"dive,keys,ascii,excludesall= /()<>@;:\\\"[]?=,endkeys,gt=0,dive,required,ascii"`
}

// Stringer
func (ni *IngressNode) String() string {
	return fmt.Sprintf("headers:%s", ni.Headers.String())
}

// Stringer
func (ni *OutboundNode) String() string {
	return fmt.Sprintf("url:%s endpoint:%s headers:%s", ni.Url, ni.Endpoint, (ni.Headers).String())
}

type Router struct {
	Path     string       `json:"path" validate:"router"`
	Ingress  IngressNode  `json:"ingress" validate:"omitempty"`
	Outbound OutboundNode `json:"outbound" validate:"required"`
}

// Stringer
func (r *Router) String() string {
	return fmt.Sprintf("path:%s ingress:{%s} outbound:{%s}", r.Path, r.Ingress.String(), r.Outbound.String())
}

func (c *Router) Validate() validate.ValidationErrors {
	// Add map validation as well
	v := validate.New()
	return validate.ValidateStruct(v, c)
}

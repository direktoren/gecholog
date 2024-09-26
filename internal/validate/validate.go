package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationErrors map[string]string

func (v ValidationErrors) String() string {
	str := ""
	for k, v := range v {
		str += fmt.Sprintf("[%s:%s] ", k, v)
	}
	return str
}

// Custom validation functions
func isAlphaNumDot(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9._]+$`).MatchString(fl.Field().String())
}

// Custom validation functions
func isAlphaNumUnderscore(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(fl.Field().String())
}

func isEndpoint(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[A-Za-z0-9?=./_-]+$`).MatchString(fl.Field().String()) && !strings.Contains(fl.Field().String(), "//") && !strings.Contains(fl.Field().String(), "..") && !strings.Contains(fl.Field().String(), "??") && !strings.Contains(fl.Field().String(), "./") && !strings.Contains(fl.Field().String(), "/=") && !strings.Contains(fl.Field().String(), "=/") && !strings.Contains(fl.Field().String(), "/?") && !strings.Contains(fl.Field().String(), "?/")
}

func isRouter(fl validator.FieldLevel) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9/]+$`).MatchString(fl.Field().String()) && (strings.HasSuffix(fl.Field().String(), "/") && strings.HasPrefix(fl.Field().String(), "/"))
}

func New() *validator.Validate {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation("alphanumdot", isAlphaNumDot)
	validate.RegisterValidation("alphanumunderscore", isAlphaNumUnderscore)
	validate.RegisterValidation("router", isRouter)
	validate.RegisterValidation("endpoint", isEndpoint)
	//	validate.RegisterValidation("httpheader", isHTTPHeader)

	return validate
}

func ValidateMapValuesFunc(isValidValue func(string) bool) validator.Func {
	return func(fl validator.FieldLevel) bool {
		mapField, ok := fl.Field().Interface().(map[string]string)
		if !ok {
			return false // Or true, depending on how you want to handle type mismatches
		}

		for _, value := range mapField {
			if !isValidValue(value) { // Implement isValidValue according to your requirements
				return false
			}
		}

		return true
	}
}

func ValidateStruct(v *validator.Validate, s interface{}) ValidationErrors {

	err := v.Struct(s)
	if err != nil {
		errors := ValidationErrors{}
		for _, err := range err.(validator.ValidationErrors) {
			// Can format the key differently here if needed
			errors[err.Namespace()] = err.Tag() + ":" + err.Param()
		}
		return errors
	}
	return nil
}

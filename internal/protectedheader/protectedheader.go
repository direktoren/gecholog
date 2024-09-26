package protectedheader

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProtectedHeader is a type derived from http.Header
type ProtectedHeader http.Header

// maskedHeaders is a map that tracks which headers should be masked.
// var maskedHeaders = map[string]struct{}{"Authorization": {}, "Api-Key": {}}
var maskedHeaders = map[string]struct{}{}
var once bool = false

// AddMaskedHeadersOnce adds headers to the list of headers to be masked.
func AddMaskedHeadersOnce(m []string) {
	if once {
		return
	}
	for _, header := range m {
		// Add header to the list of masked headers.
		maskedHeaders[header] = struct{}{}
	}
	once = true // Lazy protection against concurrency issues since writing to local map isn't thread safe
}

// maskSensitiveHeaders returns a copy of headers with sensitive data masked.
func maskSensitiveHeaders(headers ProtectedHeader) http.Header {
	masked := http.Header{}

	// Loop through each header in headers.
	for k, v := range headers {
		_, isSensitive := maskedHeaders[k]
		if isSensitive {
			// If it's sensitive, mask the value.
			masked[k] = []string{"*****MASKED*****"}
		} else {
			// If it's not sensitive, keep the original value.
			masked[k] = v
		}
	}

	return masked
}

// String converts the header keys and values into a string, masking sensitive headers.
func (ph ProtectedHeader) String() string {
	masked := maskSensitiveHeaders(ph)
	return fmt.Sprint(masked)
}

// Marshaller
func (ph ProtectedHeader) MarshalJSON() ([]byte, error) {
	masked := maskSensitiveHeaders(ph)
	return json.Marshal(masked)
}

// returns copy of a with f as filter/transformer 1-to-1
func transformedCopy(a ProtectedHeader, f func(header string, values []string) []string) ProtectedHeader {
	copyPH := ProtectedHeader{}
	for h, values := range a {
		newValue := f(h, values)
		if len(newValue) != 0 {
			copyPH[h] = newValue
		}
	}
	return copyPH
}

// Returns a new ProtectedHeader after Union with Preference for the First Operand
func AppendNew(a ProtectedHeader, b ProtectedHeader) ProtectedHeader {
	phAppend := transformedCopy(a, func(h string, v []string) []string { return v }) // Clean copy
	for h, v := range b {
		_, alreadyexists := phAppend[h]
		if !alreadyexists {
			phAppend[h] = v
		}
	}
	return phAppend
}

// Returns a map with the list of header names
func (ph ProtectedHeader) GetHeaderList() map[string]struct{} {
	l := map[string]struct{}{}
	for h, _ := range ph {
		l[h] = struct{}{}
	}
	return l
}

// Returns copy with removing headers in the nonAllowedHeaderNames map
func Remove(a ProtectedHeader, nonAllowedHeaderNames map[string]struct{}) ProtectedHeader {
	removeIfHeaderNameIsNotAllowed := removeIfHeaderNameIsNotAllowedFunc(nonAllowedHeaderNames)
	return transformedCopy(a, removeIfHeaderNameIsNotAllowed)
}

// equalStringSlices checks if two slices of strings are identical.
func equalStringSlices(a []string, b []string) bool {
	if len(a) != len(b) {
		// Different number of elements, they're not equal.
		return false
	}

	// Check if all elements are equal.
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// All headers and values have to be equal
func Equal(a ProtectedHeader, b ProtectedHeader) bool {
	if len(a) != len(b) {
		return false
	}
	same := transformedCopy(a, func(h string, v []string) []string {
		bValues, exists := b[h]
		if !exists {
			return []string{}
		}
		if !equalStringSlices(bValues, v) {
			return []string{}
		}
		return v
	})
	return len(same) == len(a)
}

// All headers and values have to be equal i non-empty, but exists in b if empty
func EqualIfNonEmptyExistsIfCatchall(a ProtectedHeader, b ProtectedHeader) (bool, string) {
	for header, values := range a {
		_, exists := b[header]
		if !exists {
			return false, fmt.Sprintf("header:'%s' missing", header)
		}
		if len(values) == 1 && values[0] == "regex:.+" {
			// If the header exists as a requirement, b values just need to be non-empty
			if len(b[header]) == 0 {
				return false, fmt.Sprintf("header:'%s' values missing", header)
			}
			if func() bool {
				for _, s := range b[header] {
					if s != "" {
						return false
					}
				}
				return true
			}() {
				return false, fmt.Sprintf("header:'%s' values missing", header)
			}
			continue
		}
		if !equalStringSlices(b[header], values) {
			return false, fmt.Sprintf("header:'%s' values invalid", header)
		}
	}
	return true, ""
}

func removeIfHeaderNameIsNotAllowedFunc(nonAllowedHeaderNames map[string]struct{}) func(string, []string) []string {
	return func(h string, v []string) []string {
		// Transformer that removes headers in the black list
		_, invalid := nonAllowedHeaderNames[h]
		if invalid {
			return []string{}
		}
		return v
	}
}

// Does a basic check on a ProtectedHeader
func ValidateProtectedHeader(a ProtectedHeader, nonAllowedHeaderNames map[string]struct{}) error {
	removeIfHeaderNameHasInvalidCharacter := func(h string, v []string) []string {
		// Transformer that removes header names with non ASCI strings and spaces
		for i := 0; i < len(h); i++ {
			if h[i] > 127 || h[i] == ' ' {
				return []string{}
			}
		}
		return v
	}
	m := transformedCopy(a, removeIfHeaderNameHasInvalidCharacter)
	if len(m) != len(a) {
		return fmt.Errorf("Contains >127 or ' ' ")
	}

	removeIfHeaderNameIsNotAllowed := removeIfHeaderNameIsNotAllowedFunc(nonAllowedHeaderNames)
	m = transformedCopy(a, removeIfHeaderNameIsNotAllowed)
	if len(m) != len(a) {
		return fmt.Errorf("Contains invalid header")
	}

	return nil
}

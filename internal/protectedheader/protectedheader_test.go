package protectedheader

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddMaskedHeaders uses a test case to check that a header is added to maskedHeaders.
func TestAddMaskedHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    map[string]struct{}
	}{
		{
			name:    "Add one header",
			headers: []string{"New-Header"},
			want:    map[string]struct{}{"New-Header": {}, "Authorization": {}, "Api-Key": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// reset for preventing test contamination
			once = false
			maskedHeaders = map[string]struct{}{"Authorization": {}, "Api-Key": {}}

			AddMaskedHeadersOnce(tt.headers)
			assert.Equal(t, tt.want, maskedHeaders)
		})
	}
}

// TestMaskSensitiveHeaders checks that headers are masked correctly.
func TestMaskSensitiveHeaders(t *testing.T) {
	tests := []struct {
		name           string
		inputHeaders   http.Header
		maskedHeaders  map[string]struct{}
		expectedOutput http.Header
	}{
		{
			name: "Masking Authorization",
			inputHeaders: http.Header{
				"Authorization": {"Bearer Token"},
				"User-Agent":    {"Mozilla"},
			},
			maskedHeaders: map[string]struct{}{"Authorization": {}},
			expectedOutput: http.Header{
				"Authorization": {"*****MASKED*****"},
				"User-Agent":    {"Mozilla"},
			},
		},
		{
			name: "No Masking",
			inputHeaders: http.Header{
				"User-Agent": {"Mozilla"},
			},
			maskedHeaders:  map[string]struct{}{"Authorization": {}},
			expectedOutput: http.Header{"User-Agent": {"Mozilla"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set maskedHeaders to the current test value
			maskedHeaders = tt.maskedHeaders
			result := maskSensitiveHeaders(ProtectedHeader(tt.inputHeaders))
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

// TestProtectedHeaderString checks the string masking of ProtectedHeader.
func TestProtectedHeaderString(t *testing.T) {
	tests := []struct {
		name           string
		headers        http.Header
		maskedHeaders  map[string]struct{}
		expectedOutput string
	}{
		{
			name: "Masking Authorization and Api-Key",
			headers: http.Header{
				"Authorization": {"Bearer Token"},
				"Api-Key":       {"ExampleKey"},
				"User-Agent":    {"Mozilla"},
			},
			maskedHeaders:  map[string]struct{}{"Authorization": {}, "Api-Key": {}},
			expectedOutput: "map[Api-Key:[*****MASKED*****] Authorization:[*****MASKED*****] User-Agent:[Mozilla]]",
		},
		{
			name: "No headers to mask",
			headers: http.Header{
				"User-Agent": {"Mozilla"},
			},
			maskedHeaders:  map[string]struct{}{"Authorization": {}},
			expectedOutput: "map[User-Agent:[Mozilla]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global maskedHeaders for this test
			maskedHeaders = tt.maskedHeaders
			ph := ProtectedHeader(tt.headers)

			assert.Equal(t, tt.expectedOutput, ph.String())
		})
	}
}

// TestProtectedHeaderMarshalJSON checks the JSON marshalling of ProtectedHeader.
func TestProtectedHeaderMarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		headers       http.Header
		maskedHeaders map[string]struct{}
		expectedJSON  string
	}{
		{
			name: "Masking Authorization and Api-Key",
			headers: http.Header{
				"Authorization": {"Bearer Token"},
				"Api-Key":       {"ExampleKey"},
				"User-Agent":    {"Mozilla"},
			},
			maskedHeaders: map[string]struct{}{"Authorization": {}, "Api-Key": {}},
			expectedJSON:  `{"Api-Key":["*****MASKED*****"],"Authorization":["*****MASKED*****"],"User-Agent":["Mozilla"]}`,
		},
		{
			name: "No headers to mask",
			headers: http.Header{
				"User-Agent": {"Mozilla"},
			},
			maskedHeaders: map[string]struct{}{"Authorization": {}},
			expectedJSON:  `{"User-Agent":["Mozilla"]}`,
		},
		// Additional test cases can be added here.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global maskedHeaders for this test
			maskedHeaders = tt.maskedHeaders
			ph := ProtectedHeader(tt.headers)

			// Marshal to JSON and check against expected output
			outputJSON, err := ph.MarshalJSON()
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expectedJSON, string(outputJSON))
		})
	}
}

func TestTransformedCopy(t *testing.T) {
	testCases := []struct {
		name     string
		input    ProtectedHeader
		function func(header string, values []string) []string
		expected ProtectedHeader
	}{
		{
			name:  "Uppercase Header Keys",
			input: ProtectedHeader{"Content-Type": {"application/json"}, "Accept": {"text/plain"}},
			function: func(header string, values []string) []string {
				return []string{strings.ToUpper(header)}
			},
			expected: ProtectedHeader{"Content-Type": {"CONTENT-TYPE"}, "Accept": {"ACCEPT"}},
		},
		{
			name:  "Concatenate Values",
			input: ProtectedHeader{"X-Custom": {"val1", "val2"}},
			function: func(header string, values []string) []string {
				if header == "X-Custom" {
					return []string{strings.Join(values, ",")}
				}
				return values
			},
			expected: ProtectedHeader{"X-Custom": {"val1,val2"}},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := transformedCopy(tc.input, tc.function)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAppend(t *testing.T) {
	testCases := []struct {
		name     string
		a        ProtectedHeader
		b        ProtectedHeader
		expected ProtectedHeader
	}{
		{
			name:     "Distinct Keys",
			a:        ProtectedHeader{"A": {"1"}},
			b:        ProtectedHeader{"B": {"2"}},
			expected: ProtectedHeader{"A": {"1"}, "B": {"2"}},
		},
		{
			name:     "Overlapping Keys",
			a:        ProtectedHeader{"Key": {"Value1"}},
			b:        ProtectedHeader{"Key": {"Value2"}, "Key2": {"Value2"}},
			expected: ProtectedHeader{"Key": {"Value1"}, "Key2": {"Value2"}},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendNew(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetHeaderList(t *testing.T) {
	testCases := []struct {
		name     string
		ph       ProtectedHeader
		expected map[string]struct{}
	}{
		{
			name:     "No Headers",
			ph:       ProtectedHeader{},
			expected: map[string]struct{}{},
		},
		{
			name: "Single Header",
			ph:   ProtectedHeader{"Content-Type": {"application/json"}},
			expected: map[string]struct{}{
				"Content-Type": {},
			},
		},
		{
			name: "Multiple Headers",
			ph:   ProtectedHeader{"Accept": {"application/json"}, "Content-Length": {"123"}},
			expected: map[string]struct{}{
				"Accept":         {},
				"Content-Length": {},
			},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.ph.GetHeaderList()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRemove(t *testing.T) {
	testCases := []struct {
		name                string
		a                   ProtectedHeader
		nonAllowedHeaderMap map[string]struct{}
		expected            ProtectedHeader
	}{
		{
			name: "Remove Single Header",
			a:    ProtectedHeader{"Accept": {"application/json"}, "Content-Type": {"application/xml"}},
			nonAllowedHeaderMap: map[string]struct{}{
				"Accept": {},
			},
			expected: ProtectedHeader{"Content-Type": {"application/xml"}},
		},
		{
			name: "Remove Multiple Headers",
			a:    ProtectedHeader{"Accept": {"application/json"}, "Content-Type": {"application/xml"}, "Authorization": {"Bearer token"}},
			nonAllowedHeaderMap: map[string]struct{}{
				"Accept":        {},
				"Authorization": {},
			},
			expected: ProtectedHeader{"Content-Type": {"application/xml"}},
		},
		{
			name: "Remove Non-Existing Header",
			a:    ProtectedHeader{"Accept": {"application/json"}},
			nonAllowedHeaderMap: map[string]struct{}{
				"X-Non-Existing": {},
			},
			expected: ProtectedHeader{"Accept": {"application/json"}},
		},
		{
			name:                "Remove From Empty Header",
			a:                   ProtectedHeader{},
			nonAllowedHeaderMap: map[string]struct{}{"Accept": {}},
			expected:            ProtectedHeader{},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Remove(tc.a, tc.nonAllowedHeaderMap)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEqualStringSlices(t *testing.T) {
	testCases := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "Equal Slices",
			a:        []string{"hello", "world"},
			b:        []string{"hello", "world"},
			expected: true,
		},
		{
			name:     "Different Length",
			a:        []string{"hello", "world", "test"},
			b:        []string{"hello", "world"},
			expected: false,
		},
		{
			name:     "Same Length Different Elements",
			a:        []string{"hello", "world"},
			b:        []string{"hello", "test"},
			expected: false,
		},
		{
			name:     "Different Order",
			a:        []string{"world", "hello"},
			b:        []string{"hello", "world"},
			expected: false,
		},
		{
			name:     "Both Empty",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "One Empty",
			a:        []string{"hello"},
			b:        []string{},
			expected: false,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := equalStringSlices(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEqualProtectedHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		a        ProtectedHeader
		b        ProtectedHeader
		expected bool
	}{
		{
			name:     "Identical Headers and Values",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}, "Accept": []string{"text/plain"}},
			b:        ProtectedHeader{"Content-Type": []string{"application/json"}, "Accept": []string{"text/plain"}},
			expected: true,
		},
		{
			name:     "Different Headers",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}},
			b:        ProtectedHeader{"Authorization": []string{"Bearer token"}},
			expected: false,
		},
		{
			name:     "Same Headers, Different Values",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}},
			b:        ProtectedHeader{"Content-Type": []string{"text/html"}},
			expected: false,
		},
		{
			name:     "Same Headers, Values Differ in Order",
			a:        ProtectedHeader{"Accept": []string{"text/plain", "application/json"}},
			b:        ProtectedHeader{"Accept": []string{"application/json", "text/plain"}},
			expected: false,
		},
		{
			name:     "One Empty",
			a:        ProtectedHeader{},
			b:        ProtectedHeader{"Content-Type": []string{"application/json"}},
			expected: false,
		},
		{
			name:     "Both Empty",
			a:        ProtectedHeader{},
			b:        ProtectedHeader{},
			expected: true,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Equal(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEqualIfNonEmptyExistsIfEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		a        ProtectedHeader
		b        ProtectedHeader
		expected bool
	}{
		{
			name:     "Identical Headers and Values",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}, "Accept": []string{"text/plain"}},
			b:        ProtectedHeader{"Content-Type": []string{"application/json"}, "Accept": []string{"text/plain"}},
			expected: true,
		},
		{
			name:     "Identical Headers - but no requirement on value",
			a:        ProtectedHeader{"Content-Type": []string{"regex:.+"}, "Accept": []string{"regex:.+"}},
			b:        ProtectedHeader{"Content-Type": []string{"application/json"}, "Accept": []string{"text/plain"}},
			expected: true,
		},
		{
			name:     "Identical Headers - no requirement on value, but empty value not allowed",
			a:        ProtectedHeader{"Content-Type": []string{"regex:.+"}, "Accept": []string{"regex:.+"}},
			b:        ProtectedHeader{"Content-Type": []string{""}, "Accept": []string{"text/plain"}},
			expected: false,
		},
		{
			name:     "Identical Headers - At least one value non-empty",
			a:        ProtectedHeader{"Content-Type": []string{"regex:.+"}, "Accept": []string{"regex:.+"}},
			b:        ProtectedHeader{"Content-Type": []string{"", "application/json"}, "Accept": []string{"text/plain"}},
			expected: true,
		},
		{
			name:     "Empty matching value not allowed",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}, "Accept": []string{"text/plain"}},
			b:        ProtectedHeader{"Content-Type": []string{""}, "Accept": []string{"text/plain"}},
			expected: false,
		},
		{
			name:     "Different Headers",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}},
			b:        ProtectedHeader{"Authorization": []string{"Bearer token"}},
			expected: false,
		},
		{
			name:     "Same Headers, Different Values",
			a:        ProtectedHeader{"Content-Type": []string{"application/json"}},
			b:        ProtectedHeader{"Content-Type": []string{"text/html"}},
			expected: false,
		},
		{
			name:     "Same Headers, Values Differ in Order",
			a:        ProtectedHeader{"Accept": []string{"text/plain", "application/json"}},
			b:        ProtectedHeader{"Accept": []string{"application/json", "text/plain"}},
			expected: false,
		},
		{
			name:     "One Empty",
			a:        ProtectedHeader{},
			b:        ProtectedHeader{"Content-Type": []string{"application/json"}},
			expected: true,
		},
		{
			name:     "Both Empty",
			a:        ProtectedHeader{},
			b:        ProtectedHeader{},
			expected: true,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, str := EqualIfNonEmptyExistsIfCatchall(tc.a, tc.b)
			assert.Equal(t, tc.expected, result, str)
		})
	}
}

func TestRemoveIfHeaderNameIsNotAllowed(t *testing.T) {
	nonAllowed := map[string]struct{}{
		"Forbidden-Header":   {},
		"Another-Bad-Header": {},
	}

	testCases := []struct {
		name     string
		input    ProtectedHeader
		expected ProtectedHeader
	}{
		{
			name:     "Header in Non-Allowed List",
			input:    ProtectedHeader{"Forbidden-Header": []string{"value"}},
			expected: ProtectedHeader{},
		},
		{
			name:     "Header Not in Non-Allowed List",
			input:    ProtectedHeader{"Good-Header": []string{"value"}},
			expected: ProtectedHeader{"Good-Header": []string{"value"}},
		},
		{
			name:     "Mixed Headers",
			input:    ProtectedHeader{"Forbidden-Header": []string{"bad"}, "Good-Header": []string{"good"}},
			expected: ProtectedHeader{"Good-Header": []string{"good"}},
		},
		{
			name:     "Empty Non-Allowed List",
			input:    ProtectedHeader{"Some-Header": []string{"value"}},
			expected: ProtectedHeader{"Some-Header": []string{"value"}},
		},
		{
			name:     "No Headers",
			input:    ProtectedHeader{},
			expected: ProtectedHeader{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transformer := removeIfHeaderNameIsNotAllowedFunc(nonAllowed)
			result := transformedCopy(tc.input, transformer)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateProtectedHeader(t *testing.T) {
	nonAllowedHeaders := map[string]struct{}{
		"Invalid-Header": {},
	}

	testCases := []struct {
		name    string
		headers ProtectedHeader
		isError bool
	}{
		{
			name:    "Valid Headers",
			headers: ProtectedHeader{"Valid-Header": []string{"value"}, "Another-Header": []string{"value"}},
			isError: false,
		},
		{
			name:    "Header With Non-ASCII Characters",
			headers: ProtectedHeader{"Bad☠Header": []string{"value"}},
			isError: true,
		},
		{
			name:    "Header With Space",
			headers: ProtectedHeader{"Bad Header": []string{"value"}},
			isError: true,
		},
		{
			name:    "Header In Non-Allowed List",
			headers: ProtectedHeader{"Invalid-Header": []string{"value"}},
			isError: true,
		},
		{
			name:    "Mixed Valid and Invalid Headers",
			headers: ProtectedHeader{"Valid-Header": []string{"value"}, "Invalid-Header": []string{"value"}, "Another Bad Header": []string{"value"}},
			isError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProtectedHeader(tc.headers, nonAllowedHeaders)
			if tc.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

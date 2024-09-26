package gechologobject

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	obj := New()
	assert.NotNil(t, obj)
	assert.NotNil(t, obj.fields)
	assert.Equal(t, 0, len(obj.fields))
}

func TestObject_MarshalJSON(t *testing.T) {
	obj := New()
	obj.fields["key"] = json.RawMessage(`"value"`)

	data, err := obj.MarshalJSON()

	assert.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, string(data))
}

func TestObject_UnmarshalJSON(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	obj := &GechoLogObject{}

	err := obj.UnmarshalJSON(data)

	assert.NoError(t, err)
	assert.NotNil(t, obj.fields)
	value, exists := obj.fields["key"]
	assert.True(t, exists)
	assert.Equal(t, json.RawMessage(`"value"`), value)
}

func TestObject_DebugString(t *testing.T) {
	// Create a new object
	obj := New()
	obj.fields["key1"] = json.RawMessage(`"value1"`)
	obj.fields["key2"] = json.RawMessage(`"value2"`)

	// Get the debug string
	debugStr := obj.DebugString()

	// Assert the content
	assert.Contains(t, debugStr, "key1: \"value1\"")
	assert.Contains(t, debugStr, "key2: \"value2\"")

	// Additional test: if the map is empty
	objEmpty := New()
	assert.Equal(t, "", objEmpty.DebugString())
}

func TestObject_AssignFieldRaw(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		rawMessage json.RawMessage
		wantErr    bool
		errMessage string
	}{
		{
			name:       "Valid raw message",
			key:        "validKey",
			rawMessage: json.RawMessage(`{"test": "value"}`),
			wantErr:    false,
		},
		{
			name:       "Invalid raw message",
			key:        "invalidKey",
			rawMessage: json.RawMessage(`{"test": "value"`),
			wantErr:    true,
			errMessage: "Not a valid json",
		},
		{
			name:       "Invalid raw message",
			key:        "Empty",
			rawMessage: json.RawMessage{},
			wantErr:    true,
			errMessage: "Not a valid json",
		},
		{
			name:       "Invalid raw message",
			key:        "Empty string",
			rawMessage: json.RawMessage(""),
			wantErr:    true,
			errMessage: "Not a valid json",
		},
	}

	obj := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := obj.AssignFieldRaw(tt.key, tt.rawMessage)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Equal(t, tt.errMessage, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.rawMessage, obj.fields[tt.key])
			}
		})
	}
}

func TestObject_AssignField(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        any
		wantErr      bool
		expectedJSON string
	}{
		{
			name: "Valid object",
			key:  "validKey",
			value: struct {
				Test string `json:"test"`
			}{Test: "value"},
			wantErr:      false,
			expectedJSON: `{"test":"value"}`,
		},
		{
			name:    "Invalid object",
			key:     "invalidKey",
			value:   func() {},
			wantErr: true,
		},
		{
			name:    "Invalid object",
			key:     "Nil",
			value:   json.RawMessage{},
			wantErr: true,
		},
	}

	obj := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := obj.AssignField(tt.key, tt.value)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedJSON, string(obj.fields[tt.key]))
			}
		})
	}
}

func TestObject_GetField(t *testing.T) {
	// Prepare test data
	raw := json.RawMessage(`{"test": "value"}`)
	tests := []struct {
		name       string
		setupFunc  func(*GechoLogObject)
		key        string
		wantValue  json.RawMessage
		wantErrStr string
	}{
		{
			name: "Get existing field",
			setupFunc: func(obj *GechoLogObject) {
				obj.fields = map[string]json.RawMessage{"existingKey": raw}
			},
			key:        "existingKey",
			wantValue:  raw,
			wantErrStr: "",
		},
		{
			name: "Get non-existing field",
			setupFunc: func(obj *GechoLogObject) {
				obj.fields = map[string]json.RawMessage{"existingKey": raw}
			},
			key:        "nonExistingKey",
			wantValue:  nil,
			wantErrStr: "gechologobject->GetField: key nonExistingKey does not exist",
		},
		{
			name: "Get field from object with uninitialized fields",
			setupFunc: func(obj *GechoLogObject) {
				obj.fields = nil
			},
			key:        "anyKey",
			wantValue:  nil,
			wantErrStr: "gechologobject->GetField: Uninitialized fields",
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := New()
			if tt.setupFunc != nil {
				tt.setupFunc(&obj)
			}
			value, err := obj.GetField(tt.key)
			assert.Equal(t, tt.wantValue, value)
			if tt.wantErrStr == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tt.wantErrStr, err.Error())
			}
		})
	}
}

func TestTransformedCopy(t *testing.T) {
	testCases := []struct {
		name     string
		input    GechoLogObject
		function func(header string, values json.RawMessage) json.RawMessage
		expected GechoLogObject
	}{
		{
			name:  "Insert simple json",
			input: GechoLogObject{fields: map[string]json.RawMessage{"Content-Type": ValueForEmptyJson(), "Accept": ValueForEmptyJson()}},
			function: func(header string, values json.RawMessage) json.RawMessage {
				return json.RawMessage("{ \"key\": \"value\" }")
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"Content-Type": json.RawMessage("{ \"key\": \"value\" }"),
					"Accept":       json.RawMessage("{ \"key\": \"value\" }"),
				},
			},
		},
		{
			name:  "Simple Copy",
			input: GechoLogObject{fields: map[string]json.RawMessage{"Content-Type": ValueForEmptyJson(), "Accept": ValueForEmptyJson()}},
			function: func(h string, jrm json.RawMessage) json.RawMessage {
				return jrm
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"Content-Type": ValueForEmptyJson(),
					"Accept":       ValueForEmptyJson(),
				},
			},
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

func TestFieldNames(t *testing.T) {
	testCases := []struct {
		name     string
		input    GechoLogObject
		expected []string
	}{
		{
			name: "No fields",
			input: GechoLogObject{
				fields: map[string]json.RawMessage{},
			},
			expected: []string{},
		},
		{
			name: "Single field",
			input: GechoLogObject{
				fields: map[string]json.RawMessage{"field1": json.RawMessage("{}")},
			},
			expected: []string{"field1"},
		},
		{
			name: "Multiple fields",
			input: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage("{}"),
					"field2": json.RawMessage("{}"),
					"field3": json.RawMessage("{}"),
				},
			},
			expected: []string{"field1", "field2", "field3"},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.input.FieldNames()
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}

func TestFilter(t *testing.T) {
	testCases := []struct {
		name            string
		input           GechoLogObject
		fieldsToInclude []string
		expected        GechoLogObject
	}{
		{
			name: "Exclude single field",
			input: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"key1":"value1"}`),
					"field2": json.RawMessage(`{"key2":"value2"}`),
				},
			},
			fieldsToInclude: []string{"field1"},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"key1":"value1"}`),
				},
			},
		},
		{
			name: "Include multiple fields",
			input: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"key1":"value1"}`),
					"field2": json.RawMessage(`{"key2":"value2"}`),
					"field3": json.RawMessage(`{"key3":"value3"}`),
				},
			},
			fieldsToInclude: []string{"field1", "field3"},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"key1":"value1"}`),
					"field3": json.RawMessage(`{"key3":"value3"}`),
				},
			},
		},
		{
			name: "Include non-existent field",
			input: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"key1":"value1"}`),
				},
			},
			fieldsToInclude: []string{"fieldX"},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{},
			},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Filter(tc.input, tc.fieldsToInclude)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestReplace(t *testing.T) {
	testCases := []struct {
		name     string
		original GechoLogObject
		replace  GechoLogObject
		expected GechoLogObject
	}{
		{
			name: "Replace overlapping field",
			original: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
					"field2": json.RawMessage(`{"original":"value2"}`),
				},
			},
			replace: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"replaced":"newValue1"}`),
				},
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"replaced":"newValue1"}`),
					"field2": json.RawMessage(`{"original":"value2"}`),
				},
			},
		},
		{
			name: "Replace multiple overlapping fields",
			original: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
					"field2": json.RawMessage(`{"original":"value2"}`),
					"field3": json.RawMessage(`{"original":"value3"}`),
				},
			},
			replace: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"replaced":"newValue1"}`),
					"field3": json.RawMessage(`{"replaced":"newValue3"}`),
				},
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"replaced":"newValue1"}`),
					"field2": json.RawMessage(`{"original":"value2"}`),
					"field3": json.RawMessage(`{"replaced":"newValue3"}`),
				},
			},
		},
		{
			name: "No overlapping fields",
			original: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
				},
			},
			replace: GechoLogObject{
				fields: map[string]json.RawMessage{
					"fieldX": json.RawMessage(`{"replaced":"newValueX"}`),
				},
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
				},
			},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Replace(tc.original, tc.replace)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAppendNew(t *testing.T) {
	testCases := []struct {
		name     string
		original GechoLogObject
		append   GechoLogObject
		expected GechoLogObject
	}{
		{
			name: "Append non-overlapping fields",
			original: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
				},
			},
			append: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field2": json.RawMessage(`{"appended":"value2"}`),
				},
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
					"field2": json.RawMessage(`{"appended":"value2"}`),
				},
			},
		},
		{
			name: "Do not append overlapping fields",
			original: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
					"field2": json.RawMessage(`{"original":"value2"}`),
				},
			},
			append: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field2": json.RawMessage(`{"shouldNotBeAppended":"newValue2"}`),
					"field3": json.RawMessage(`{"appended":"value3"}`),
				},
			},
			expected: GechoLogObject{
				fields: map[string]json.RawMessage{
					"field1": json.RawMessage(`{"original":"value1"}`),
					"field2": json.RawMessage(`{"original":"value2"}`), // Should remain unchanged
					"field3": json.RawMessage(`{"appended":"value3"}`),
				},
			},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendNew(tc.original, tc.append)
			assert.Equal(t, tc.expected, result)
		})
	}
}

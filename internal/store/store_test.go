package store

import (
	"encoding/json"
	"testing"

	"github.com/direktoren/gecholog/internal/gechologobject"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	tests := []struct {
		name        string
		inputValue  any
		key         string
		errKey      string
		expectError bool
	}{
		{
			name:        "Store string value",
			inputValue:  "test string",
			key:         "stringKey",
			errKey:      "stringKey",
			expectError: false,
		},
		{
			name:        "Store RawMessage value",
			inputValue:  json.RawMessage(`{"json_key":"json_value"}`),
			key:         "jsonRawKey",
			errKey:      "jsonRawKey",
			expectError: false,
		},
		{
			name:        "Empty correct RawMessage value",
			inputValue:  gechologobject.ValueForEmptyJson(),
			key:         "jsonRawEmptyKey",
			errKey:      "jsonRawEmptyKey",
			expectError: false,
		},
		{
			name: "Struct",
			inputValue: struct {
				Name     string
				LastName string
				Age      int
			}{
				Name:     "John",
				LastName: "Doe",
				Age:      37,
			},
			key:         "struct",
			errKey:      "struct",
			expectError: false,
		},
		{
			name:        "map",
			inputValue:  map[string]string{"key": "value"},
			key:         "map",
			errKey:      "map",
			expectError: false,
		},
		{
			name:        "Empty key",
			inputValue:  json.RawMessage(`{"json_key":"json_value"}`),
			key:         "",
			errKey:      "ignore",
			expectError: true,
		},
		{
			name:        "Store byte slice",
			inputValue:  []byte(`byte slice content`),
			key:         "byteKey",
			errKey:      "byteKey",
			expectError: true, // Since its not a valid json
		},
		{
			name:        "Falsely formatted json.RawMessage",
			inputValue:  json.RawMessage(`{"json_key":"json_val`),
			key:         "jrmbadKey",
			errKey:      "jrmbadKey",
			expectError: true, // Since its not a valid json
		},
		{
			name:        "Empty formatted json.RawMessage",
			inputValue:  json.RawMessage{},
			key:         "jrmbademptyKey",
			errKey:      "jrmbademptyKey",
			expectError: true, // Since its not a valid json
		},

		// Add more test cases for error scenarios and other types
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := gechologobject.New()
			errObject := gechologobject.New()
			expectedValue, _ := json.Marshal(tc.inputValue)

			Store(&o, &errObject, tc.key, tc.inputValue)

			// Verify the stored value in 'o'
			storedValue, err := o.GetField(tc.key)

			if tc.expectError {
				assert.Error(t, err)
				// Verify the error is stored in errObject
				var errValue json.RawMessage
				errValue, _ = errObject.GetField(tc.errKey)
				assert.NotEmpty(t, errValue)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, json.RawMessage(expectedValue), storedValue)

				// Verify no error is stored in errObject
				_, noErr := errObject.GetField(tc.errKey)
				assert.Error(t, noErr)
			}
		})
	}
}

func TestStoreInArray(t *testing.T) {
	tests := []struct {
		name        string
		sourceData  map[string]interface{} // Data to be stored in the source object
		arrayKey    string                 // Key under which the array will be stored
		expectError bool
	}{
		{
			name: "Multiple fields",
			sourceData: map[string]interface{}{
				"stringKey": "test string",
				"intKey":    42,
				"boolKey":   true,
			},
			arrayKey:    "dataArray",
			expectError: false,
		},
		// Additional test cases can be added here
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sourceObject := gechologobject.New()
			targetObject := gechologobject.New()
			errObject := gechologobject.New()

			// Populating the source object
			for key, value := range tc.sourceData {
				Store(&sourceObject, &errObject, key, &value)
			}

			StoreInArray(&targetObject, &errObject, tc.arrayKey, &sourceObject)

			if tc.expectError {
				// Check error conditions

				errValue, _ := errObject.GetField(tc.arrayKey)
				assert.NotEmpty(t, errValue)
				// Add assertions to validate the error scenario
			} else {
				// Check if the array is stored correctly
				storedArray, err := targetObject.GetField(tc.arrayKey)
				assert.NoError(t, err)
				assert.NotEmpty(t, storedArray)

				// Validate contents of the stored array
				var resultArray []ArrayLog
				err = json.Unmarshal(storedArray, &resultArray)
				assert.NoError(t, err)

				for _, item := range resultArray {
					expectedValue, _ := json.Marshal(tc.sourceData[item.Name])
					assert.Equal(t, json.RawMessage(expectedValue), item.Details)
				}

				// Additional checks can be made depending on the ArrayLog structure and requirements
			}
		})
	}

	// test nil pointer source
	targetObject := gechologobject.New()
	errObject := gechologobject.New()
	StoreInArray(&targetObject, &errObject, "anyKey", nil)
	errValue, _ := errObject.GetField("anyKey")
	assert.Equal(t, string(errValue), "\"Empty source\"")

}

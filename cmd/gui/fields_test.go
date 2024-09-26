package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextFieldSetValue(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name        string
		textField   textField
		input       []string
		expectValue string
	}{
		{
			name: "Set simple value",
			textField: textField{
				Value: "",
			},
			input:       []string{"test"},
			expectValue: "test",
		},
		{
			name: "Overwrite",
			textField: textField{
				Value: "before",
			},
			input:       []string{"test"},
			expectValue: "test",
		},
		{
			name: "Empty value set 1",
			textField: textField{
				Value: "before",
			},
			input:       []string{""},
			expectValue: "",
		},
		{
			name: "Empty value set 1",
			textField: textField{
				Value: "before",
			},
			input:       []string{"", "test"},
			expectValue: "",
		},
		{
			name: "Empty array",
			textField: textField{
				Value: "before",
			},
			input:       []string{},
			expectValue: "before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.textField.setValue(tt.input)

			assert.Equal(t, tt.expectValue, tt.textField.Value, "Expected value to match")
		})
	}
}

func TestTextFieldCopy(t *testing.T) {
	// Define test scenarios
	tests := []struct {
		name           string
		original       textField
		modifyOriginal func(*textField)
		expectCopy     textField
	}{
		{
			name: "Copy with modifications",
			original: textField{
				Value:               "Hello",
				Placeholder:         "Enter text",
				ErrorMsg:            "Error occurred",
				ErrorMsgTooltipText: "Tooltip error message",
			},
			modifyOriginal: func(tf *textField) {
				tf.Value = "test"
				tf.Placeholder = "test"
				tf.ErrorMsg = "test"
				tf.ErrorMsgTooltipText = "test"
			},
			expectCopy: textField{
				Value:               "Hello",
				Placeholder:         "Enter text",
				ErrorMsg:            "Error occurred",
				ErrorMsgTooltipText: "Tooltip error message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original textField
			copied := tt.original.copy().(*textField)

			// Assert that the copy has not changed
			assert.Equal(t, tt.expectCopy.Value, copied.Value, "Expected Values to match")
			assert.Equal(t, tt.expectCopy.Placeholder, copied.Placeholder, "Expected Placeholders to match")
			assert.Equal(t, tt.expectCopy.ErrorMsg, copied.ErrorMsg, "Expected ErrorMsg to match")
			assert.Equal(t, tt.expectCopy.ErrorMsgTooltipText, copied.ErrorMsgTooltipText, "Expected ErrorMsgTooltipText to match")

			// Modify the original textField as per the test case
			tt.modifyOriginal(&tt.original)

			// Assert that the copy has not changed
			assert.Equal(t, tt.expectCopy.Value, copied.Value, "Expected Values to match")
			assert.Equal(t, tt.expectCopy.Placeholder, copied.Placeholder, "Expected Placeholders to match")
			assert.Equal(t, tt.expectCopy.ErrorMsg, copied.ErrorMsg, "Expected ErrorMsg to match")
			assert.Equal(t, tt.expectCopy.ErrorMsgTooltipText, copied.ErrorMsgTooltipText, "Expected ErrorMsgTooltipText to match")

		})
	}
}

func TestRadioFieldSetValue(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name        string
		radioField  radioField
		input       []string
		expectValue string
		expectedErr bool
	}{
		{
			name: "Set simple value",
			radioField: radioField{
				Value:   "",
				Options: []string{"test"},
			},
			input:       []string{"test"},
			expectValue: "test",
			expectedErr: false,
		},
		{
			name: "Overwrite",
			radioField: radioField{
				Value:   "before",
				Options: []string{"test"},
			},
			input:       []string{"test"},
			expectValue: "test",
			expectedErr: false,
		},
		{
			name: "Empty value set 1",
			radioField: radioField{
				Value:   "before",
				Options: []string{""},
			},
			input:       []string{""},
			expectValue: "",
			expectedErr: false,
		},
		{
			name: "Empty value set 2",
			radioField: radioField{
				Value:   "before",
				Options: []string{""},
			},
			input:       []string{"", "test"},
			expectValue: "",
			expectedErr: false,
		},
		{
			name: "Empty array",
			radioField: radioField{
				Value: "before",
			},
			input:       []string{},
			expectValue: "before",
			expectedErr: false,
		},
		{
			name: "Not in options",
			radioField: radioField{
				Value:   "before",
				Options: []string{"another"},
			},
			input:       []string{"test"},
			expectValue: "before",
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.radioField.setValue(tt.input)

			if tt.expectedErr {
				assert.NotNil(t, err, "Expected error to be non-nil")
				return
			}
			assert.Equal(t, tt.expectValue, tt.radioField.Value, "Expected value to match")
		})
	}
}

func TestRadioFieldCopy(t *testing.T) {
	// Define test scenarios
	tests := []struct {
		name           string
		original       radioField
		modifyOriginal func(*radioField)
		expectCopy     radioField
	}{
		{
			name: "Copy with modifications",
			original: radioField{
				Value:               "one",
				Options:             []string{"one", "two", "three"},
				ErrorMsg:            "Error occurred",
				ErrorMsgTooltipText: "Tooltip error message",
			},
			modifyOriginal: func(tf *radioField) {
				tf.Value = "test"
				tf.Options = []string{"test"}
				tf.ErrorMsg = "test"
				tf.ErrorMsgTooltipText = "test"
			},
			expectCopy: radioField{
				Value:               "one",
				Options:             []string{"one", "two", "three"},
				ErrorMsg:            "Error occurred",
				ErrorMsgTooltipText: "Tooltip error message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original textField
			copied := tt.original.copy().(*radioField)

			// Assert that the copy has not changed
			assert.Equal(t, tt.expectCopy.Value, copied.Value, "Expected Values to match")
			assert.Equal(t, len(tt.expectCopy.Options), len(copied.Options), "Expected len Options to match")
			for i := range tt.expectCopy.Options {
				assert.Equal(t, tt.expectCopy.Options[i], copied.Options[i], "Expected Options to match")
			}
			assert.Equal(t, tt.expectCopy.ErrorMsg, copied.ErrorMsg, "Expected ErrorMsg to match")
			assert.Equal(t, tt.expectCopy.ErrorMsgTooltipText, copied.ErrorMsgTooltipText, "Expected ErrorMsgTooltipText to match")

			// Modify the original textField as per the test case
			tt.modifyOriginal(&tt.original)

			// Assert that the copy has not changed
			assert.Equal(t, tt.expectCopy.Value, copied.Value, "Expected Values to match")
			assert.Equal(t, len(tt.expectCopy.Options), len(copied.Options), "Expected len Options to match")
			for i := range tt.expectCopy.Options {
				assert.Equal(t, tt.expectCopy.Options[i], copied.Options[i], "Expected Options to match")
			}
			assert.Equal(t, tt.expectCopy.ErrorMsg, copied.ErrorMsg, "Expected ErrorMsg to match")
			assert.Equal(t, tt.expectCopy.ErrorMsgTooltipText, copied.ErrorMsgTooltipText, "Expected ErrorMsgTooltipText to match")

		})
	}
}

func TestArrayFieldSetValue(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name          string
		arrayField    arrayField
		input         []string
		expectedField arrayField
		expectedErr   bool
	}{
		{
			name: "Set simple value",
			arrayField: arrayField{
				Values: []pair{pair{Value: "before"}},
			},
			input: []string{"after"},
			expectedField: arrayField{
				Values: []pair{pair{Value: "after"}},
			},
			expectedErr: false,
		},
		{
			name: "Set simple value(s)",
			arrayField: arrayField{
				Values: []pair{pair{Value: "before"}, pair{Value: "before"}, pair{Value: "before"}},
			},
			input: []string{"one", "two", "three"},
			expectedField: arrayField{
				// Expect alphabetical order
				Values: []pair{pair{Value: "one"}, pair{Value: "three"}, pair{Value: "two"}},
			},
			expectedErr: false,
		},
		{
			name: "One gap",
			arrayField: arrayField{
				Values: []pair{pair{Value: "before"}, pair{Value: "before"}, pair{Value: "before"}},
			},
			input: []string{"one", "", "three"},
			expectedField: arrayField{
				// Expect alphabetical order
				Values: []pair{pair{Value: "one"}, pair{Value: "three"}},
			},
			expectedErr: false,
		},
		{
			name: "Three empty",
			arrayField: arrayField{
				Values: []pair{pair{Value: "before"}, pair{Value: "before"}, pair{Value: "before"}},
			},
			input: []string{"one", "", "", "three", ""},
			expectedField: arrayField{
				// Expect alphabetical order
				Values: []pair{pair{Value: "one"}, pair{Value: "three"}},
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.arrayField.setValue(tt.input)

			if tt.expectedErr {
				assert.NotNil(t, err, "Expected error to be non-nil")
				return
			}

			assert.Equal(t, len(tt.arrayField.Values), len(tt.expectedField.Values), "Expected same length of array")
			for i, _ := range tt.arrayField.Values {
				assert.Equal(t, tt.arrayField.Values[i].Value, tt.expectedField.Values[i].Value, "Expected value to match")
			}

		})
	}
}

func TestArrayFieldCopy(t *testing.T) {
	// Define test scenarios
	tests := []struct {
		name           string
		original       arrayField
		modifyOriginal func(*arrayField)
		expectCopy     arrayField
	}{
		{
			name: "Copy with modifications",
			original: arrayField{
				// Expect alphabetical order
				Values: []pair{
					pair{
						Value:               "one",
						ErrorMsg:            "errorMsgOne",
						ErrorMsgTooltipText: "errorMgsTooltipTextOne",
					},
					pair{
						Value:               "three",
						ErrorMsg:            "errorMsgThree",
						ErrorMsgTooltipText: "errorMgsTooltipTextThree",
					},
					pair{
						Value:               "two",
						ErrorMsg:            "errorMsgTwo",
						ErrorMsgTooltipText: "errorMgsTooltipTextTwo",
					},
				},
				Placeholder: "Placeholder",
			},
			modifyOriginal: func(tf *arrayField) {
				tf.Values = []pair{pair{Value: "before"}, pair{Value: "before"}, pair{Value: "before"}}
				tf.Placeholder = "errorgsfsr"
			},
			expectCopy: arrayField{
				// Expect alphabetical order
				Values: []pair{
					pair{
						Value:               "one",
						ErrorMsg:            "errorMsgOne",
						ErrorMsgTooltipText: "errorMgsTooltipTextOne",
					},
					pair{
						Value:               "three",
						ErrorMsg:            "errorMsgThree",
						ErrorMsgTooltipText: "errorMgsTooltipTextThree",
					},
					pair{
						Value:               "two",
						ErrorMsg:            "errorMsgTwo",
						ErrorMsgTooltipText: "errorMgsTooltipTextTwo",
					},
				},
				Placeholder: "Placeholder",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original textField
			copied := tt.original.copy().(*arrayField)

			// Assert that the copy has not changed
			assert.Equal(t, tt.expectCopy.Placeholder, copied.Placeholder, "Expected Values to match")
			assert.Equal(t, len(tt.expectCopy.Values), len(copied.Values), "Expected same length")
			for i, _ := range copied.Values {
				assert.Equal(t, tt.expectCopy.Values[i].Value, copied.Values[i].Value, "Expected values to match")
				assert.Equal(t, tt.expectCopy.Values[i].ErrorMsg, copied.Values[i].ErrorMsg, "Expected values to match")
				assert.Equal(t, tt.expectCopy.Values[i].ErrorMsg, copied.Values[i].ErrorMsg, "Expected values to match")

			}

			// Modify the original textField as per the test case
			tt.modifyOriginal(&tt.original)

			// Assert that the copy has not changed
			assert.Equal(t, tt.expectCopy.Placeholder, copied.Placeholder, "Expected Values to match")
			assert.Equal(t, len(tt.expectCopy.Values), len(copied.Values), "Expected same length")
			for i, _ := range copied.Values {
				assert.Equal(t, tt.expectCopy.Values[i].Value, copied.Values[i].Value, "Expected values to match")
				assert.Equal(t, tt.expectCopy.Values[i].ErrorMsg, copied.Values[i].ErrorMsg, "Expected values to match")
				assert.Equal(t, tt.expectCopy.Values[i].ErrorMsg, copied.Values[i].ErrorMsg, "Expected values to match")

			}

		})
	}
}

func TestHeaderFieldSetValue(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name          string
		headerField   headerField
		input         []string
		expectedField headerField
		expectedErr   bool
	}{
		{
			name: "Set simple value",
			headerField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "before",
							},
						},
					},
				},
			},
			input: []string{"my-header", "after"},
			expectedField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "after",
							},
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Set simple value - with need for trim",
			headerField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "before",
							},
						},
					},
				},
			},
			input: []string{"my-header", "   after  "},
			expectedField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "after",
							},
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Set simple value - expand current header",
			headerField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "before",
							},
						},
					},
				},
			},
			input: []string{"my-header", "before", "my-header", "after"},
			expectedField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "after", // Sorted alphabetically
							},
							{
								Value: "before",
							},
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Set several values",
			headerField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "before",
							},
						},
					},
				},
			},
			input: []string{"my-header", "before", "my-header", "after", "another-header", "value1", "another-header", "value2", "another-header", "value3"},
			expectedField: headerField{
				Headers: []webHeader{
					{
						Header: "another-header",
						Values: []webHeaderValue{
							{
								Value: "value1", // Sorted alphabetically
							},
							{
								Value: "value2",
							},
							{
								Value: "value3",
							},
						},
					},
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "after", // Sorted alphabetically
							},
							{
								Value: "before",
							},
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "Uneven arguments",
			headerField: headerField{
				Headers: []webHeader{
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "before",
							},
						},
					},
				},
			},
			input:         []string{"my-header"},
			expectedField: headerField{},
			expectedErr:   true,
		},
		{
			name: "Remove header",
			headerField: headerField{
				Headers: []webHeader{
					{
						Header: "another-header",
						Values: []webHeaderValue{
							{
								Value: "value1", // Sorted alphabetically
							},
							{
								Value: "value2",
							},
							{
								Value: "value3",
							},
						},
					},
					{
						Header: "my-header",
						Values: []webHeaderValue{
							{
								Value: "before",
							},
						},
					},
				},
			},
			input: []string{"", "after", "another-header", "value1", "another-header", "value2", "another-header", "value3"},
			expectedField: headerField{
				Headers: []webHeader{
					{
						Header: "another-header",
						Values: []webHeaderValue{
							{
								Value: "value1", // Sorted alphabetically
							},
							{
								Value: "value2",
							},
							{
								Value: "value3",
							},
						},
					},
				},
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.headerField.setValue(tt.input)

			if tt.expectedErr {
				assert.NotNil(t, err, "Expected error to be non-nil")
				return
			}

			for i, h := range tt.headerField.Headers {
				assert.Equal(t, tt.expectedField.Headers[i].Header, h.Header, "Expected header to match")
				for j, v := range h.Values {
					assert.Equal(t, tt.expectedField.Headers[i].Values[j].Value, v.Value, "Expected value to match")
				}
			}

		})
	}
}

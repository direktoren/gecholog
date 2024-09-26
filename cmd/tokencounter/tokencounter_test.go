package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessIngress(t *testing.T) {
	_ = os.Setenv("NATS_TOKEN", "some_value")
	args := []string{"-o", "../../config/tokencounter_config.json"}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	setupConfig(fs, args)
	logLevel.Set(slog.LevelDebug)

	responseBytes := "{\"control\":{\"Error\":\"Consumption Cap Exceeded. Try again later\",\"Path\":\"/service/capped/\"}}"
	//fmt.Print(responseBytes)

	tests := []struct {
		name           string
		glPath         string
		currentTokens  routerCountFields
		expectedResult []byte
		expectedError  error
	}{
		{
			name:   "Consumption Cap Exceeded",
			glPath: "/service/capped/",
			currentTokens: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					tokenCount{Field: "prompt_tokens", Value: 100},
					tokenCount{Field: "completion_tokens", Value: 100},
					tokenCount{Field: "total_tokens", Value: 200},
				},
				mappedFields: make(map[string]*tokenCount),
			},
			expectedResult: []byte(responseBytes), // define expected result when cap is exceeded
			expectedError:  nil,
		},
		{
			name:   "Double Consumption Cap Exceeded",
			glPath: "/service/capped/",
			currentTokens: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					tokenCount{Field: "prompt_tokens", Value: 1000},
					tokenCount{Field: "completion_tokens", Value: 100},
					tokenCount{Field: "total_tokens", Value: 200},
				},
				mappedFields: make(map[string]*tokenCount),
			},

			expectedResult: []byte(responseBytes), // define expected result when cap is exceeded
			expectedError:  nil,
		},
		{
			name:   "Successful Request",
			glPath: "/service/capped/",
			currentTokens: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					tokenCount{Field: "prompt_tokens", Value: 100},
					tokenCount{Field: "completion_tokens", Value: 100},
					tokenCount{Field: "total_tokens", Value: 50},
				},
				mappedFields: make(map[string]*tokenCount),
			},

			expectedResult: []byte{}, // define expected successful response
			expectedError:  nil,
		},
		{
			name:   "Successful Request Unlimited",
			glPath: "/service/standard/",
			currentTokens: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					tokenCount{Field: "prompt_tokens", Value: 10000},
					tokenCount{Field: "completion_tokens", Value: 10000},
					tokenCount{Field: "total_tokens", Value: 20000},
				},
				mappedFields: make(map[string]*tokenCount),
			},

			expectedResult: []byte{}, // define expected successful response
			expectedError:  nil,
		},
		// Add more test cases as needed

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset()
			add(tt.currentTokens)
			result, err := processIngress(tt.glPath)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestProcessEgress(t *testing.T) {
	_ = os.Setenv("NATS_TOKEN", "some_value")
	args := []string{"-o", "../../config/tokencounter_config.json"}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	setupConfig(fs, args)
	logLevel.Set(slog.LevelDebug)

	//key := "token_count"

	tests := []struct {
		name             string
		glPath           string
		inputData        []byte
		expectedResponse routerCountFields
		expectedError    error
	}{
		{
			name:      "Valid Input1",
			glPath:    "validPath1",
			inputData: []byte(`{"inbound_payload": { "usage": { "total_tokens": 123}}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"total_tokens": &tokenCount{
						Field: "total_tokens",
						Value: 123,
					},
				},
			}, // define expected response for valid input
			expectedError: nil,
		},
		{
			name:      "Valid Input2",
			glPath:    "validPath2",
			inputData: []byte(`{"inbound_payload": { "usage": { "total_tokens": 123,"prompt_tokens": 4}}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"total_tokens": &tokenCount{
						Field: "total_tokens",
						Value: 123,
					},
					"prompt_tokens": &tokenCount{
						Field: "prompt_tokens",
						Value: 4,
					},
				},
			}, // define expected response for valid input
			//			expectedResponse: tokenUsage{"total_tokens": 123, "prompt_tokens": 4},                                  // define expected response for valid input
			expectedError: nil,
		},
		{
			name:      "Valid Input3",
			glPath:    "validPath3",
			inputData: []byte(`{"inbound_payload": { "usage": { "total_tokens": 123,"prompt_tokens": 4, "completion_tokens": 56}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"total_tokens": &tokenCount{
						Field: "total_tokens",
						Value: 123,
					},
					"prompt_tokens": &tokenCount{
						Field: "prompt_tokens",
						Value: 4,
					},
					"completion_tokens": &tokenCount{
						Field: "completion_tokens",
						Value: 56,
					},
				},
				//			expectedResponse: tokenUsage{"total_tokens": 123, "prompt_tokens": 4, "completion_tokens": 56},                                 // define expected response for valid input

			},
			expectedError: nil,
		},
		{
			name:      "Valid Input4",
			glPath:    "validPath4",
			inputData: []byte(`{"inbound_payload": { "usage": { "prompt_tokens": 4, "completion_tokens": 56}}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"prompt_tokens": &tokenCount{
						Field: "prompt_tokens",
						Value: 4,
					},
					"completion_tokens": &tokenCount{
						Field: "completion_tokens",
						Value: 56,
					},
				},
			}, //				expectedResponse: tokenUsage{"prompt_tokens": 4, "completion_tokens": 56},                                   // define expected response for valid input
			expectedError: nil,
		},
		{
			name:      "Valid Input5",
			glPath:    "validPath5",
			inputData: []byte(`{"inbound_payload": { "usage": { "prompt_tokens": 10}}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"prompt_tokens": &tokenCount{
						Field: "prompt_tokens",
						Value: 10,
					},
				},
				//			expectedResponse: tokenUsage{"total_tokens": 123, "prompt_tokens": 4, "completion_tokens": 56},                                 // define expected response for valid input

			}, //			expectedResponse: tokenUsage{"completion_tokens": 30},                                   // define expected response for valid input
			expectedError: nil,
		},
		{
			name:      "Valid Input7",
			glPath:    "validPath7",
			inputData: []byte(`{"inbound_payload": { "usage": { "total_tokens": 50, "completion_tokens": 20}}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"total_tokens": &tokenCount{
						Field: "total_tokens",
						Value: 50,
					},
					"completion_tokens": &tokenCount{
						Field: "completion_tokens",
						Value: 20,
					},
				},
			}, //				expectedResponse: tokenUsage{"prompt_tokens": 4, "completion_tokens": 56},                                   // define expected response for valid input
			//			expectedResponse: tokenUsage{"total_tokens": 50, "completion_tokens": 20},                                   // define expected response for valid input
			expectedError: nil,
		},

		{
			name:      "Valid Empty Path",
			glPath:    "",
			inputData: []byte(`{"inbound_payload": { "usage": { "total_tokens": 123}}}`), // Example JSON input
			expectedResponse: routerCountFields{
				Router: "validPath1",
				mappedFields: map[string]*tokenCount{
					"total_tokens": &tokenCount{
						Field: "total_tokens",
						Value: 123,
					},
				},
			}, // define expected response for valid input                                  // define expected response for valid input
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a channel for output
			outputChan := make(chan routerCountFields, 1)
			defer close(outputChan)

			result, err := processEgress(outputChan, tt.glPath, tt.inputData)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				return
			}

			assert.NoError(t, err)

			responseObj := map[string]routerCountFields{"token_count": routerCountFields{}}
			err = json.Unmarshal(result, &responseObj)
			if err != nil {
				t.Fatal(err)
			}

			for _, v := range responseObj["token_count"].Fields {
				assert.Equal(t, v.Value, tt.expectedResponse.mappedFields[v.Field].Value)
			}

			// Test the output channel behavior
			fromChannel := <-outputChan
			assert.Equal(t, tt.glPath, fromChannel.Router)
			for _, v := range fromChannel.Fields {
				assert.Equal(t, v.Value, tt.expectedResponse.mappedFields[v.Field].Value)
			}

		})
	}
}

func TestAdd(t *testing.T) {
	_ = os.Setenv("NATS_TOKEN", "some_value")
	args := []string{"-o", "../../config/tokencounter_config.json"}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	setupConfig(fs, args)
	logLevel.Set(slog.LevelDebug)

	type addTest struct {
		name           string
		consumption    routerCountFields
		expectedError  string                        // Using string to compare error messages
		expectedTotals map[string]*routerCountFields // Define the expected state of globalConfig.totals after the function call
	}

	tests := []addTest{
		{
			name: "ValidRouterUpdate1",
			consumption: routerCountFields{
				Router: "router1",
				Fields: []tokenCount{
					{Field: "field1", Value: 10},
				},
			},
			expectedError: "",
			expectedTotals: map[string]*routerCountFields{
				"router1": {
					Router: "router1",
					mappedFields: map[string]*tokenCount{
						"field1": {Field: "field1", Value: 10},
					},
				},
			},
		},
		{
			name: "ValidRouterUpdate2",
			consumption: routerCountFields{
				Router: "router1",
				Fields: []tokenCount{
					{Field: "field2", Value: 103},
				},
			},
			expectedError: "",
			expectedTotals: map[string]*routerCountFields{
				"router1": {
					Router: "router1",
					mappedFields: map[string]*tokenCount{
						"field2": {Field: "field2", Value: 103},
					},
				},
			},
		},
		{
			name: "ValidRouterUpdate3",
			consumption: routerCountFields{
				Router: "router1",
				Fields: []tokenCount{
					{Field: "field1", Value: 10},
					{Field: "field2", Value: 103},
				},
			},
			expectedError: "",
			expectedTotals: map[string]*routerCountFields{
				"router1": {
					Router: "router1",
					mappedFields: map[string]*tokenCount{
						"field1": {Field: "field1", Value: 10},
						"field2": {Field: "field2", Value: 103},
					},
				},
			},
		},
		// ... add more test cases to cover different scenarios ...
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset globalConfig to a clean state before each test
			reset()

			// Call the add function
			err := add(tc.consumption)

			// Check the error
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
			}

			// Check the state of globalConfig.totals
			assert.Equal(t, tc.expectedTotals, globalConfig.consumed)
		})
	}

}

func TestIsWithinCap(t *testing.T) {
	_ = os.Setenv("NATS_TOKEN", "some_value")
	args := []string{"-o", "../../config/tokencounter_config.json"}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	setupConfig(fs, args)
	logLevel.Set(slog.LevelDebug)

	type addTest struct {
		name           string
		consumption    routerCountFields
		expectedResult bool // Using string to compare error messages
	}

	tests := []addTest{
		{
			name: "InfiniteRouterUpdate1",
			consumption: routerCountFields{
				Router: "router1",
				Fields: []tokenCount{
					{Field: "prompt_tokens", Value: 10000},
				},
			},
			expectedResult: true,
		},
		{
			name: "CappedRouterUpdate1",
			consumption: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					{Field: "prompt_tokens", Value: 1000},
				},
			},
			expectedResult: false,
		},
		{
			name: "CappedRouterUpdate2",
			consumption: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					{Field: "prompt_tokens", Value: 10},
				},
			},
			expectedResult: true,
		},
		{
			name: "CappedRouterUpdate3",
			consumption: routerCountFields{
				Router: "/service/capped/",
				Fields: []tokenCount{
					{Field: "prompt_tokens", Value: 10},
					{Field: "completion_tokens", Value: 1000},
				},
			},
			expectedResult: false,
		},
		{
			name: "StandardRouterUpdate1",
			consumption: routerCountFields{
				Router: "/service/standard/",
				Fields: []tokenCount{
					{Field: "prompt_tokens", Value: 1000},
				},
			},
			expectedResult: true,
		},
		{
			name: "StandardRouterUpdate1",
			consumption: routerCountFields{
				Router: "/service/standard/",
				Fields: []tokenCount{
					{Field: "prompt_tokens", Value: 1000},
					{Field: "completion_tokens", Value: 1000},
					{Field: "total_tokens", Value: 1000},
				},
			},
			expectedResult: true,
		},

		// ... add more test cases to cover different scenarios ...
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset globalConfig to a clean state before each test
			reset()

			// Call the add function
			add(tc.consumption)

			// Check the error
			assert.Equal(t, tc.expectedResult, isWithinCap(tc.consumption.Router))

		})
	}

}

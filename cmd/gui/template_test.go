package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateTutorials(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "tutorials.html",
			TemplateFiles: []string{"templates/tutorials.html"},
			Data:          struct{}{},
			Expected:      []string{"\n"},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			name := path.Base(tc.TemplateFiles[0])
			testTemplate := template.New(name)
			testTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			parsedTemplate, err := testTemplate.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplatePython_Render(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Template      string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "python.html",
			TemplateFiles: []string{"templates/python.html"},
			Template:      "",
			Data:          struct{}{},
			Expected:      []string{""},
		},
		{
			Test:          "rejecteddisclaimer",
			TemplateFiles: []string{"templates/python.html"},
			Template:      `{{ template "rejecteddisclaimer" . }}`,
			Data: struct {
				ErrorMsg string
			}{
				ErrorMsg: "error",
			},
			Expected: []string{"Disclaimer"},
		},
		{
			Test:          "import os - MY_API_KEY",
			TemplateFiles: []string{"templates/python.html"},
			Template:      `{{ template "importos" . }}`,
			Data: struct {
				Objects interface{}
			}{
				Objects: []inputObject{
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Test-Header",
									Values: []webHeaderValue{
										{
											Value: "Test-Value",
										},
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{"import os", "MY_API_KEY"},
		},
		{
			Test:          "import os - env var",
			TemplateFiles: []string{"templates/python.html"},
			Template:      `{{ template "importos" . }}`,
			Data: struct {
				Objects interface{}
			}{
				Objects: []inputObject{
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Api-Key",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{"import os", "ENV_VAR"},
		},
		{
			Test:          "requestheaders - env var",
			TemplateFiles: []string{"templates/python.html"},
			Template:      `{{ template "requestheaders" . }}`,
			Data: struct {
				Objects interface{}
			}{
				Objects: []inputObject{
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Api-Key",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{"headers = {", "f&#34;{ENV_VAR}&#34;"},
		},
		{
			Test:          "pythonrequest",
			TemplateFiles: []string{"templates/python.html"},
			Template:      `{{ template "pythonrequest" . }}`,
			Data: struct {
				ErrorMsg string
				Objects  interface{}
			}{
				ErrorMsg: "valid",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "My-Header",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{"import os", "headers = {", "f&#34;{MY_API_KEY}&#34;", "/test", "application/json"},
		},
		{
			Test:          "import os - MY_API_KEY - endpoint populated",
			TemplateFiles: []string{"templates/python.html"},
			Template:      `{{ template "pythonrequest" . }}`,
			Data: struct {
				Objects  interface{}
				ErrorMsg string
			}{
				ErrorMsg: "valid",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "My-Header",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "endpoint",
						Fields: &textField{
							Value: "/test",
						},
					},
				},
			},
			Expected: []string{"import os", `endpoint = &#34;&#34;`},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			name := path.Base(tc.TemplateFiles[0])
			testTemplate := template.New(name)
			testTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			parsedTemplate, err := testTemplate.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if tc.Template != "" {
				parsedTemplate, err = parsedTemplate.Parse(tc.Template)
				assert.NoError(t, err, "Failed to parse template %s", tc.Template)
			}

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplatePython_Compile(t *testing.T) {

	templateFiles := []string{"templates/python.html"}

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test     string
		Template string
		Data     interface{}
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:     "compile python with ENV_VAR",
			Template: `{{ template "pythonrequest" . }}`,
			Data: struct {
				ErrorMsg string
				Objects  interface{}
			}{
				ErrorMsg: "valid",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "My-Header",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Test:     "compile python without ENV_VAR but Api-Key hard coded",
			Template: `{{ template "pythonrequest" . }}`,
			Data: struct {
				ErrorMsg string
				Objects  interface{}
			}{
				ErrorMsg: "valid",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Api-Key",
									Values: []webHeaderValue{
										{
											Value: "mykey",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Test:     "rejected - compile python with ENV_VAR",
			Template: `{{ template "pythonrequest" . }}`,
			Data: struct {
				ErrorMsg string
				Objects  interface{}
			}{
				ErrorMsg: "rejected",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "My-Header",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Test:     "compile python with regex:.+",
			Template: `{{ template "pythonrequest" . }}`,
			Data: struct {
				ErrorMsg string
				Objects  interface{}
			}{
				ErrorMsg: "valid",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "My-Header",
									Values: []webHeaderValue{
										{
											Value: "regex:.+",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Test:     "compile python with ENV_VAR end outbound endpoint specified",
			Template: `{{ template "pythonrequest" . }}`,
			Data: struct {
				ErrorMsg string
				Objects  interface{}
			}{
				ErrorMsg: "valid",
				Objects: []inputObject{
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:     HEADER,
						Headline: "Test",
						Key:      "ingress-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "My-Header",
									Values: []webHeaderValue{
										{
											Value: "${ENV_VAR}",
										},
									},
								},
								{
									Header: "Content-Type",
									Values: []webHeaderValue{
										{
											Value: "application/json",
										},
									},
								},
							},
						},
					},
					{
						Type:     INPUT,
						Headline: "Test",
						Key:      "endpoint",
						Fields: &textField{
							Value: "test/",
						},
					},
				},
			},
		},
	}

	pythonPath, err := exec.LookPath("python3")
	assert.NoError(t, err, "Failed to find python: %s", err)

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			name := path.Base(templateFiles[0])
			testTemplate := template.New(name)
			testTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			parsedTemplate, err := testTemplate.ParseFiles(templateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", templateFiles)
			if tc.Template != "" {
				parsedTemplate, err = parsedTemplate.Parse(tc.Template)
				assert.NoError(t, err, "Failed to parse template %s", tc.Template)
			}

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", templateFiles)

			actual := buf.String()
			replaced := strings.Replace(actual, "&#39;", "'", -1)
			replaced = strings.Replace(replaced, "&#34;", "\"", -1)

			// Define the file name
			fileName := "test/tmp.py"

			// Remove the file if it exists
			if _, err := os.Stat(fileName); err == nil {
				err := os.Remove(fileName)
				if err != nil {
					fmt.Printf("Error removing file: %v\n", err)
					return
				}
			}

			// Write the buffer to the file
			err = os.WriteFile(fileName, []byte(replaced), 0644)
			if err != nil {
				fmt.Printf("Error writing to file: %v\n", err)
				return
			}

			fmt.Println("File written successfully")

			cmd := exec.Command(pythonPath, "-m", "py_compile", fileName)
			err = cmd.Run()

			// Check the exit code
			assert.NoError(t, err, "Failed to execute python code: %s", err)
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode := exitError.ExitCode()
					fmt.Printf("Compilation failed with exit code %d\n", exitCode)
				} else {
					fmt.Printf("Failed to execute python code: %v\n", err)
				}
			}

		})
	}
}

func TestTemplateHeader(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "header.html",
			TemplateFiles: []string{"templates/header.html"},
			Data:          struct{}{},
			Expected:      []string{"\n"},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateMainMenu(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "mainmenu.html",
			TemplateFiles: []string{"templates/mainmenu.html"},
			Data: struct {
				NatsConnected bool
			}{
				NatsConnected: true,
			},
		},
		{
			Test:          "mainmenu.html",
			TemplateFiles: []string{"templates/mainmenu.html"},
			Data: struct {
				NatsConnected bool
			}{
				NatsConnected: false,
			},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateLogs(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "logs.html - empty",
			TemplateFiles: []string{"templates/logs.html"},
			Data: struct {
				Logs    []logRecord
				FocusID string
				Focus   string
			}{
				Logs:    []logRecord{},
				FocusID: "",
				Focus:   "",
			},
			Expected: []string{`<button type="submit" class="standard-button validate">Reload</button>`},
		},
		{
			Test:          "logs.html - List including focus",
			TemplateFiles: []string{"templates/logs.html"},
			Data: struct {
				Logs    []logRecord
				FocusID string
				Focus   string
			}{
				Logs: []logRecord{
					{
						TransactionID: "TST00001_1723627212920003288_2_0",
						Router:        "/test/unique/",
						Time:          "	2024-08-14T09:20:12.408985312Z",
						Latency:       256,
						StatusCode:    200,
					},
					{
						TransactionID: "TST00001_1723627212408985312_1_0",
						Router:        "/echo/",
						Time:          "2024-08-14T09:20:12.920003288Z",
						Latency:       4597,
						StatusCode:    500,
					},
				},
				FocusID: "TST00001_1723627212920003288_2_0",
				Focus: `{
   "egress_post_timer": {
      "start": "2024-08-14T09:20:12.921622653Z",
      "stop": "2024-08-14T09:20:12.923226254Z",
      "duration": 1
   },
   "ingress_egress_timer": {
      "start": "2024-08-14T09:20:12.920003288Z",
      "stop": "2024-08-14T09:20:12.921622653Z",
      "duration": 1
   },
   "request": {
      "gl_path": "/echo/",
      "ingress_headers": {
         "Accept": [
            "*/*"
         ],
         "Content-Length": [
            "71"
         ],
         "Content-Type": [
            "application/json"
         ],
         "User-Agent": [
            "curl/8.7.1"
         ]
      },
      "ingress_outbound_timer": {
         "start": "2024-08-14T09:20:12.920003288Z",
         "stop": "2024-08-14T09:20:12.921389969Z",
         "duration": 1
      },
      "ingress_payload": {
         "message": {
            "content": "Hello World!"
         }
      },
      "ingress_subpath": "",
      "outbound_headers": {
         "Accept": [
            "*/*"
         ],
         "Content-Type": [
            "application/json"
         ],
         "User-Agent": [
            "curl/8.7.1"
         ]
      },
      "outbound_payload": {
         "message": {
            "content": "Hello World!"
         }
      },
      "outbound_subpath": "",
      "processors": [
         {
            "name": "token_counter",
            "details": {
               "required": false,
               "completed": true,
               "timestamp": {
                  "start": "2024-08-14T09:20:12.920140862Z",
                  "stop": "2024-08-14T09:20:12.921273341Z",
                  "duration": 1
               }
            }
         }
      ],
      "url_path": "https://localhost"
   },
   "response": {
      "egress_payload": {
         "message": {
            "content": "Hello World!"
         }
      },
      "egress_status_code": 200,
      "gl_path": "/echo/",
      "inbound_egress_timer": {
         "start": "2024-08-14T09:20:12.921590021Z",
         "stop": "2024-08-14T09:20:12.921622653Z",
         "duration": 0
      },
      "inbound_headers": {
         "Accept": [
            "*/*"
         ],
         "Content-Type": [
            "application/json"
         ]
      },
      "inbound_payload": {
         "message": {
            "content": "Hello World!"
         }
      },
      "inbound_status_code": 200,
      "outbound_inbound_timer": {
         "start": "2024-08-14T09:20:12.921389969Z",
         "stop": "2024-08-14T09:20:12.921590021Z",
         "duration": 0
      },
      "processors_async": [
         {
            "name": "token_counter",
            "details": {
               "required": false,
               "completed": true,
               "timestamp": {
                  "start": "2024-08-14T09:20:12.921898943Z",
                  "stop": "2024-08-14T09:20:12.922993136Z",
                  "duration": 1
               }
            }
         }
      ],
      "token_count": {}
   },
   "session_id": "TST00001_1723627212920003288_2_0",
   "transaction_id": "TST00001_1723627212920003288_2_0"
}`,
			},
			Expected: []string{`<button type="submit" class="standard-button validate">Reload</button>`, `href="logs?noreload=true&transactionID=TST00001_1723627212920003288_2_0"`, `<td>/echo/</td>`, `&#34;transaction_id&#34;: &#34;TST00001_1723627212920003288_2_0&#34;`, `<td>500</td>`},
		},
		{
			Test:          "logs.html - Just a list",
			TemplateFiles: []string{"templates/logs.html"},
			Data: struct {
				Logs    []logRecord
				FocusID string
				Focus   string
			}{
				Logs: []logRecord{
					{
						TransactionID: "TST00001_1723627212920003288_2_0",
						Router:        "/test/unique/",
						Time:          "	2024-08-14T09:20:12.408985312Z",
						Latency:       256,
						StatusCode:    200,
					},
					{
						TransactionID: "TST00001_1723627212408985312_1_0",
						Router:        "/echo/",
						Time:          "2024-08-14T09:20:12.920003288Z",
						Latency:       4597,
						StatusCode:    500,
					},
				},
				FocusID: "",
				Focus:   "",
			},
			Expected: []string{`<button type="submit" class="standard-button validate">Reload</button>`, `href="logs?noreload=true&transactionID=TST00001_1723627212920003288_2_0"`, `<td>/echo/</td>`},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}
func TestTemplateLogin(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "login.html",
			TemplateFiles: []string{"templates/login.html"},
			Data: struct {
				Error string
			}{
				Error: "",
			},
			Expected: []string{`class="standard-button edit"`},
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateRouter(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "No tutorial- routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "101 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "102",
					NextID:         "103",
					PreviousButton: "",
					NextButton:     "103",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "102 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "102",
					NextID:         "103",
					PreviousButton: "101",
					NextButton:     "103",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "103 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "103",
					NextID:         "104",
					PreviousButton: "102",
					NextButton:     "104",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="input-group tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "104 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "104",
					NextID:         "105",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`},
		},
		{
			Test:          "105 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "105",
					NextID:         "106",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`},
		},
		{
			Test:          "201 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "201",
					NextID:         "202",
					PreviousButton: "",
					NextButton:     "202",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "202 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "202",
					NextID:         "203",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`},
		},
		{
			Test:          "203 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "203",
					NextID:         "204",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "error",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`},
		},
		{
			Test:          "502 - routers.html",
			TemplateFiles: []string{"templates/routers.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Objects             interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "502",
					NextID:         "503",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},
					{
						Type:        ROUTERS,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test0",
						Fields: &routerField{
							ErrorMsg:            "valid",
							ErrorMsgTooltipText: "error",
							Fields: []inputObject{
								{
									Type:        INPUT,
									Headline:    "path",
									TooltipText: "Test",
									Key:         "path",
									Fields: &textField{
										Value: "/test/",
									},
								},
							},
						},
					},
				},
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateForm_Router(t *testing.T) {

	templateFiles := []string{"templates/form.html", "templates/header.html", "templates/tutorials.html", "templates/python.html"}
	// Define the structure for test cases
	type TemplateTestCase struct {
		Test     string
		Data     interface{}
		Expected []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test: "No tutorial - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "path",
						TooltipText: "Test",
						Key:         "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "106 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "106",
					NextID:         "107",
					PreviousButton: "",
					NextButton:     "107",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "path",
						TooltipText: "Test",
						Key:         "path",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="input-group tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "107 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "107",
					NextID:         "108",
					PreviousButton: "",
					NextButton:     "108",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "URL",
						TooltipText: "Test",
						Key:         "url",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="input-group tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "108 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "108",
					NextID:         "108",
					PreviousButton: "",
					NextButton:     "108",
				},
				ErrorMsg:            "error", // This will trigger highlighting
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "URL",
						TooltipText: "Test",
						Key:         "url",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "204 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "204",
					NextID:         "205",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "URL",
						TooltipText: "Test",
						Key:         "url",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="input-group tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "205 - form.html - before clicking +",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "205",
					NextID:         "206",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        HEADER,
						Headline:    "ingress header",
						TooltipText: "Test",
						Key:         "ingress-headers",
						Fields:      &headerField{},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "206 - form.html - before clicking +",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "206",
					NextID:         "207",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        HEADER,
						Headline:    "outbound header",
						TooltipText: "Test",
						Key:         "outbound-headers",
						Fields:      &headerField{},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "207 - form.html - errors",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "207",
					NextID:         "208",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        HEADER,
						Headline:    "outbound header",
						TooltipText: "Test",
						Key:         "outbound-headers",
						Fields:      &headerField{},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<span class="tooltip tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "208 - form.html - before clicking +",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "208",
					NextID:         "209",
					PreviousButton: "207",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        HEADER,
						Headline:    "ingress header",
						TooltipText: "Test",
						Key:         "ingress-headers",
						Fields:      &headerField{},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tooltip tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "209 - form.html - errors",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "209",
					NextID:         "209",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        HEADER,
						Headline:    "ingress header",
						TooltipText: "Test",
						Key:         "ingress-headers",
						Fields:      &headerField{},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<span class="tooltip tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "503 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "503",
					NextID:         "504",
					PreviousButton: "",
					NextButton:     "504",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "URL",
						TooltipText: "Test",
						Key:         "url",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:        HEADER,
						Headline:    "outbound header",
						TooltipText: "Test",
						Key:         "outbound-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Key",
									Values: []webHeaderValue{
										{
											Value: "Value",
										},
									},
									ErrorMsg:            "error",
									ErrorMsgTooltipText: "error",
								},
							},
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "504 - form.html - code",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "504",
					NextID:         "505",
					PreviousButton: "",
					NextButton:     "505",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "URL",
						TooltipText: "Test",
						Key:         "url",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:        HEADER,
						Headline:    "outbound header",
						TooltipText: "Test",
						Key:         "outbound-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Key",
									Values: []webHeaderValue{
										{
											Value: "Value",
										},
									},
									ErrorMsg:            "error",
									ErrorMsgTooltipText: "error",
								},
							},
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `import requests`, `Disclaimer`}, // This is a snippet of the expected output towards the end
		},

		{
			Test: "505 - form.html - example response",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "505",
					NextID:         "505",
					PreviousButton: "504",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "URL",
						TooltipText: "Test",
						Key:         "url",
						Fields: &textField{
							Value: "/test/",
						},
					},
					{
						Type:        HEADER,
						Headline:    "outbound header",
						TooltipText: "Test",
						Key:         "outbound-headers",
						Fields: &headerField{
							Headers: []webHeader{
								{
									Header: "Key",
									Values: []webHeaderValue{
										{
											Value: "Value",
										},
									},
									ErrorMsg:            "error",
									ErrorMsgTooltipText: "error",
								},
							},
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `{'choices': [{'c`, `{'messages': [{'role': 'system', 'content'`}, // This is a snippet of the expected output towards the end
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			name := path.Base(templateFiles[0])
			testTemplate := template.New(name)
			testTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			parsedTemplate, err := testTemplate.ParseFiles(templateFiles...)
			//			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			//parsedTemplate, err := template.New("test").ParseFiles(tc.TemplateFiles...)
			/*		parsedTemplate, err := template.New("test").Funcs(template.FuncMap{
					"findEndVarInString": findEndVarInString,
				}).ParseFiles(tc.TemplateFiles...)*/
			assert.NoError(t, err, "Failed to parse templates %s", templateFiles)

			if err != nil {
				return
			}
			// Add the custom function map to the parsed template
			parsedTemplate = parsedTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", templateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateMenu(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "No tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Test",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "301 tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "301",
					NextID:         "302",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Test",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
					{
						Headline:            "Service Bus Config",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "304 tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "304",
					NextID:         "305",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Valid",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
					{
						Headline:            "Service Bus Config",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "308 tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "308",
					NextID:         "308",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Valid",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
					{
						Headline:            "Service Bus Config",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "401 tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "401",
					NextID:         "402",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Valid",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
					{
						Headline:            "Service Bus Config",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "405 tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "405",
					NextID:         "405",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Valid",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
					{
						Headline:            "Service Bus Config",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "501 tutorial - menu.html",
			TemplateFiles: []string{"templates/menu.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Status              interface{}
				Areas               interface{}
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "501",
					NextID:         "402",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Status: struct {
					Headline             string
					ProductionFile       string
					DeployedChecksum     string
					Deployed             string
					ProductionChecksum   string
					WorkingFile          string
					WorkingChecksum      string
					StagedFormat         string
					Staged               string
					ExitCode             string
					ExitCodeFormat       string
					RejectedFields       string
					RejectedFieldsFormat string
					DeployButton         string
				}{
					Headline:             "Test",
					ProductionFile:       "Test",
					DeployedChecksum:     "Test",
					Deployed:             "Test",
					ProductionChecksum:   "Test",
					WorkingFile:          "Test",
					WorkingChecksum:      "Test",
					StagedFormat:         "valid",
					Staged:               "Test",
					ExitCode:             "Valid",
					ExitCodeFormat:       "test",
					RejectedFields:       "Test",
					RejectedFieldsFormat: "Test",
					DeployButton:         "test",
				},
				Areas: []area{
					{
						Headline:            "Test",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
					{
						Headline:            "Routers",
						Key:                 "test",
						ErrorMsg:            "error",
						ErrorMsgTooltipText: "error",
					},
				},
			},
			Expected: []string{`class="standard-button test"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateForm_Nats2FileServiceBusConfig(t *testing.T) {

	templateFiles := []string{"templates/form.html", "templates/header.html", "templates/tutorials.html", "templates/python.html"}
	// Define the structure for test cases
	type TemplateTestCase struct {
		Test     string
		Data     interface{}
		Expected []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test: "No tutorial - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "path",
						TooltipText: "Test",
						Key:         "topicexactlogger",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "302 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "302",
					NextID:         "303",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "path",
						TooltipText: "Test",
						Key:         "topicexactlogger",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`, `<div class="input-group tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test: "303 - form.html",
			Data: struct {
				Headline            string
				Tutorial            interface{}
				ErrorMsg            string
				ErrorMsgTooltipText string
				Submit              string
				Reload              string
				Quit                string
				Objects             interface{}
				Key                 string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "303",
					NextID:         "303",
					PreviousButton: "",
					NextButton:     "",
				},
				ErrorMsg:            "error",
				ErrorMsgTooltipText: "error",
				Submit:              "submit",
				Reload:              "reload",
				Quit:                "quit",
				Objects: []inputObject{
					{
						Type:        HEADLINE,
						Headline:    "Test",
						TooltipText: "Test",
						Key:         "test",
						Fields: &headlineField{
							Value: "Test",
						},
					},

					{
						Type:        INPUT,
						Headline:    "path",
						TooltipText: "Test",
						Key:         "topicexactlogger",
						Fields: &textField{
							Value: "/test/",
						},
					},
				},
				Key: "test0",
			},
			Expected: []string{`class="standard-button edit"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			name := path.Base(templateFiles[0])
			testTemplate := template.New(name)
			testTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			parsedTemplate, err := testTemplate.ParseFiles(templateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", templateFiles)

			if err != nil {
				return
			}
			// Add the custom function map to the parsed template
			parsedTemplate = parsedTemplate.Funcs(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", templateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplatePublish(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "No tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "305 tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "305",
					NextID:         "306",
					PreviousButton: "",
					NextButton:     "306",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "306 tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "306",
					NextID:         "307",
					PreviousButton: "305",
					NextButton:     "307",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "307 tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "307",
					NextID:         "308",
					PreviousButton: "306",
					NextButton:     "",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "402 tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "402",
					NextID:         "403",
					PreviousButton: "",
					NextButton:     "403",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "403 tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "403",
					NextID:         "404",
					PreviousButton: "402",
					NextButton:     "404",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`, `class="standard-button tutorial"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "404 tutorial - publish.html",
			TemplateFiles: []string{"templates/publish.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Headline string
				Tutorial interface{}
				Flag     interface{}
				FileInfo []fileMeta
				Redirect string
			}{
				Headline: "Test Title",
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "404",
					NextID:         "405",
					PreviousButton: "403",
					NextButton:     "",
				},
				Flag: struct {
					IsArchived bool
				}{
					IsArchived: true,
				},
				FileInfo: []fileMeta{
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
					{
						Filename: "test",
						Match:    false,
						Comment:  "test",
					},
				},
				Redirect: "test",
			},
			Expected: []string{`onclick="window.location.href='archive-productionfile';"`, `class="standard-button tutorial"`, `<div class="tutorial-highlighted">`}, // This is a snippet of the expected output towards the end
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

func TestTemplateArchiveQuery(t *testing.T) {

	// Define the structure for test cases
	type TemplateTestCase struct {
		Test          string
		TemplateFiles []string
		Data          interface{}
		Expected      []string
	}
	// Define your test cases

	testCases := []TemplateTestCase{
		{
			Test:          "No tutorial - archive-query.html",
			TemplateFiles: []string{"templates/archive-query.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Form        string
				Headline    string
				Source      string
				Prefix      string
				UniqueParts map[string]bool
				Tutorial    interface{}
			}{
				Form:     "test",
				Headline: "Test Title",
				Source:   "test",
				Prefix:   "test_",
				UniqueParts: map[string]bool{
					"test": true,
				},
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
			},
			Expected: []string{`class="standard-button disabled"`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "406 - archive-query.html",
			TemplateFiles: []string{"templates/archive-query.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Form        string
				Headline    string
				Source      string
				Prefix      string
				UniqueParts map[string]bool
				Tutorial    interface{}
			}{
				Form:     "test",
				Headline: "Test Title",
				Source:   "test",
				Prefix:   "test_",
				UniqueParts: map[string]bool{
					"test": true,
				},
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "406",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
			},
			Expected: []string{`class="standard-button disabled"`, `<div class="tutorial-highlighted">`, `<div id="tutorialSubmit">`}, // This is a snippet of the expected output towards the end
		},
		{
			Test:          "309 - archive-query.html",
			TemplateFiles: []string{"templates/archive-query.html", "templates/header.html", "templates/tutorials.html"},
			Data: struct {
				Form        string
				Headline    string
				Source      string
				Prefix      string
				UniqueParts map[string]bool
				Tutorial    interface{}
			}{
				Form:     "test",
				Headline: "Test Title",
				Source:   "test",
				Prefix:   "test_",
				UniqueParts: map[string]bool{
					"test": true,
				},
				Tutorial: struct {
					ID             string
					NextID         string
					PreviousButton string
					NextButton     string
				}{
					ID:             "309",
					NextID:         "",
					PreviousButton: "",
					NextButton:     "",
				},
			},
			Expected: []string{`class="standard-button disabled"`, `<div class="tutorial-highlighted">`, `<div id="tutorialSubmit">`}, // This is a snippet of the expected output towards the end
		},
	}

	// Iterate over the test cases
	for _, tc := range testCases {
		t.Run(tc.Test, func(t *testing.T) {
			// Parse the template file
			parsedTemplate, err := template.ParseFiles(tc.TemplateFiles...)
			assert.NoError(t, err, "Failed to parse templates %s", tc.TemplateFiles)

			if err != nil {
				return
			}
			// Execute the template and write to a buffer
			var buf bytes.Buffer
			err = parsedTemplate.Execute(&buf, tc.Data)
			assert.NoError(t, err, "Failed to execute template %s", tc.TemplateFiles)

			actual := buf.String()
			for _, expected := range tc.Expected {
				assert.Contains(t, actual, expected, "Expected '%s' to contain '%s'", actual, expected)
			}
		})
	}
}

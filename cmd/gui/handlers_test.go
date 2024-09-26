package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"github.com/stretchr/testify/assert"
)

type mockConfigFailedSetConfigFromAreas struct {
}

func (f *mockConfigFailedSetConfigFromAreas) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfigFailedSetConfigFromAreas) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfigFailedSetConfigFromAreas) setConfigFromAreas(areas []area) error {
	return fmt.Errorf("Failed to set config from areas")
}
func (f *mockConfigFailedSetConfigFromAreas) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfigFailedSetConfigFromAreas) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfigFailedSetConfigFromAreas) update() (map[string]string, error) {
	return nil, nil
}

type mockConfigFailedUpdate struct {
}

func (f *mockConfigFailedUpdate) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfigFailedUpdate) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfigFailedUpdate) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfigFailedUpdate) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfigFailedUpdate) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfigFailedUpdate) update() (map[string]string, error) {
	return nil, fmt.Errorf("Failed to update")
}

type mockConfigFailedWriteConfigFile struct {
}

func (f *mockConfigFailedWriteConfigFile) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfigFailedWriteConfigFile) writeConfigFile(file string) (string, error) {
	return "", fmt.Errorf("Failed to write config file")
}
func (f *mockConfigFailedWriteConfigFile) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfigFailedWriteConfigFile) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfigFailedWriteConfigFile) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfigFailedWriteConfigFile) update() (map[string]string, error) {
	return nil, nil
}

type mockConfigChangedKeys struct {
}

func (f *mockConfigChangedKeys) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfigChangedKeys) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfigChangedKeys) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfigChangedKeys) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfigChangedKeys) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfigChangedKeys) update() (map[string]string, error) {
	return map[string]string{"key0": "key1", "key1": "key0"}, nil
}

type mockConfigFailedUpdateAreasFromConfig struct {
}

func (f *mockConfigFailedUpdateAreasFromConfig) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfigFailedUpdateAreasFromConfig) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfigFailedUpdateAreasFromConfig) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfigFailedUpdateAreasFromConfig) updateAreasFromConfig(v map[string]string, a []area) error {
	return fmt.Errorf("Failed to update areas from config")
}
func (f *mockConfigFailedUpdateAreasFromConfig) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfigFailedUpdateAreasFromConfig) update() (map[string]string, error) {
	return nil, nil
}

type mockConfigFailedLoadConfigFile struct {
}

func (f *mockConfigFailedLoadConfigFile) loadConfigFile(file string) error {
	return fmt.Errorf("Failed to load config file")
}
func (f *mockConfigFailedLoadConfigFile) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfigFailedLoadConfigFile) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfigFailedLoadConfigFile) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfigFailedLoadConfigFile) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfigFailedLoadConfigFile) update() (map[string]string, error) {
	return nil, nil
}

type mockConfigFailedCreateAreas struct {
}

func (f *mockConfigFailedCreateAreas) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfigFailedCreateAreas) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfigFailedCreateAreas) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfigFailedCreateAreas) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfigFailedCreateAreas) createAreas(v map[string]string) ([]area, error) {
	return nil, fmt.Errorf("Failed to create areas")
}
func (f *mockConfigFailedCreateAreas) update() (map[string]string, error) {
	return nil, nil
}

type mockConfig struct {
}

func (f *mockConfig) loadConfigFile(file string) error {
	return nil
}
func (f *mockConfig) writeConfigFile(file string) (string, error) {
	return "", nil
}
func (f *mockConfig) setConfigFromAreas(areas []area) error {
	return nil
}
func (f *mockConfig) updateAreasFromConfig(v map[string]string, a []area) error {
	return nil
}
func (f *mockConfig) createAreas(v map[string]string) ([]area, error) {
	return nil, nil
}
func (f *mockConfig) update() (map[string]string, error) {
	return nil, nil
}
func TestWrapSubmitHandlerFunc(t *testing.T) {
	// Define test scenarios
	tests := []struct {
		name                 string
		w                    *webbAppState
		areaIndex            int
		submitHandler        updateSubmitHandler
		redirect             string
		expectedResponseBody string
		expectedResponseCode int
	}{
		{
			name:      "webbAppState w nil",
			w:         nil, // This is the error
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "objectlist is nil",
			w: &webbAppState{
				areas: nil, // This is the error
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "objectlist is nil",
			w: &webbAppState{
				areas: []area{
					{
						Objects: nil, // This is the error
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},

		{
			name: "index < 0 ",
			w: &webbAppState{
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: -2, // This is the error,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "index > len(areas) ",
			w: &webbAppState{
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 1, // This is the error,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "Empty redirect",
			w: &webbAppState{
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "", // This is the error
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "handler is nil",
			w: &webbAppState{
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex:            0,
			submitHandler:        nil, // This is the error
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "Error from handler",
			w: &webbAppState{
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				// This is the error
				return http.StatusBadRequest, fmt.Errorf("an error occurred")
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusBadRequest,
		},
		{
			name: "Failed setConfigFromAreas",
			w: &webbAppState{
				config: &mockConfigFailedSetConfigFromAreas{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "Failed update",
			w: &webbAppState{
				config: &mockConfigFailedUpdate{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "Failed writeConfigFile",
			w: &webbAppState{
				config: &mockConfigFailedWriteConfigFile{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			testFunc := wrapSubmitHandlerFunc(tt.w, tt.areaIndex, tt.redirect, tt.redirect, tt.submitHandler)

			ginRouter.GET("/test", testFunc)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
		})
	}
}

func TestWrapSubmitHandlerFunc_Redirects(t *testing.T) {
	// Define test scenarios
	tests := []struct {
		name                 string
		w                    *webbAppState
		areaIndex            int
		submitHandler        updateSubmitHandler
		redirect             string
		changedKeysRedirect  string
		expectedRedirect     string
		expectedResponseBody string
		expectedResponseCode int
	}{
		{
			name: "Changed Keys",
			w: &webbAppState{
				config: &mockConfigChangedKeys{}, // This will trigger the feature to change keys
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "/anypath?key=key0",
			changedKeysRedirect:  "/newpath",
			expectedRedirect:     "/anypath?key=key1",
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
		},
		{
			name: "No Changed Keys",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				return http.StatusFound, nil
			},
			redirect:             "/anypath",
			changedKeysRedirect:  "/newpath",
			expectedRedirect:     "/anypath",
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
		},
		{
			name: "Redirect from handler",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			submitHandler: func(c *gin.Context, objectList *[]inputObject, redirect *string) (int, error) {
				// This will trigger redirect
				*redirect = "/handlervalue"
				return http.StatusFound, nil
			},
			redirect:             "/anypath",
			changedKeysRedirect:  "/newpath",
			expectedRedirect:     "/handlervalue",
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			testFunc := wrapSubmitHandlerFunc(tt.w, tt.areaIndex, tt.redirect, tt.changedKeysRedirect, tt.submitHandler)

			ginRouter.GET("/test", testFunc)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")
		})
	}
}

func TestAddNewRouterSubmitHandler(t *testing.T) {

	handlerToTest := addNewRouterSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		c                    *gin.Context
		routerList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedLenghtAfter  int
	}{
		{
			name:                 "Empty list of inputObjects",
			handler:              handlerToTest,
			c:                    &gin.Context{},
			routerList:           &[]inputObject{},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  1,
		},
		{
			name:    "List of two inputObjects",
			handler: handlerToTest,
			c:       &gin.Context{},
			routerList: &[]inputObject{
				{},
				{},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  3,
		},
		{
			name:                 "Nil objectlist",
			handler:              handlerToTest,
			c:                    &gin.Context{},
			routerList:           nil, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			status, err := tt.handler(tt.c, tt.routerList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, len(*tt.routerList), tt.expectedLenghtAfter, "Expected routerObjectList to increase by 1")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestCopyRouterSubmitHandler_Copy(t *testing.T) {

	handlerToTest := copyRouterSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		queryParams          map[string]string
		routerList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedLenghtAfter  int
	}{
		{
			name:    "Copy in list of one",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "Copy in list of three - first",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey0",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey0",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey1",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey2",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  4,
		},
		{
			name:    "Copy in list of three - middle",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey1",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey0",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey1",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey2",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  4,
		},
		{
			name:    "Copy in list of three - last",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey2",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey0",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey1",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey2",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.routerList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tt.expectedLenghtAfter, len(*tt.routerList), "Expected routerObjectList to increase by 1")

				// Check that the new object is added to the list
				originalFound := 0
				copyFound := 0

				for _, obj := range *tt.routerList {
					if obj.Key == tt.queryParams["key"] {
						originalFound++
					}
					if obj.Key == tt.queryParams["key"]+"_copy" {
						copyFound++
					}
				}

				assert.Equal(t, 1, originalFound, "Expected (only one) original object to be in list")
				assert.Equal(t, 1, copyFound, "Expected (only one) copy object to be in list")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestCopyRouterSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := copyRouterSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		queryParams          map[string]string
		routerList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedLenghtAfter  int
	}{
		{
			name:    "nil objectlist",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList:           nil, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "empty objectlist",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList:           &[]inputObject{}, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:        "no key query parameter",
			handler:     handlerToTest,
			queryParams: map[string]string{}, // This is the error
			routerList: &[]inputObject{
				{},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "No key found",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key: "anotherKey", // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "Object not ROUTERS type",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key:  "testKey",
					Type: HEADLINE, // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "Object Field is nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey",
					Type:   ROUTERS,
					Fields: nil, // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.routerList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tt.expectedLenghtAfter, len(*tt.routerList), "Expected routerObjectList to increase by 1")

				// Check that the new object is added to the list
				originalFound := 0
				copyFound := 0

				for _, obj := range *tt.routerList {
					if obj.Key == tt.queryParams["key"] {
						originalFound++
					}
					if obj.Key == tt.queryParams["key"]+"_copy" {
						copyFound++
					}
				}

				assert.Equal(t, 1, originalFound, "Expected (only one) original object to be in list")
				assert.Equal(t, 1, copyFound, "Expected (only one) copy object to be in list")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestDeleteRouterSubmitHandler_Delete(t *testing.T) {

	handlerToTest := deleteRouterSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		queryParams          map[string]string
		routerList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedLenghtAfter  int
	}{
		{
			name:    "Delete in list of one",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  0,
		},
		{
			name:    "Delete in list of three - first",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey0",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey0",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey1",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey2",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "Delete in list of three - second",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey1",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey0",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey1",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey2",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "Delete in list of three - last",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey2",
			},
			routerList: &[]inputObject{
				{
					Key:    "testKey0",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey1",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
				{
					Key:    "testKey2",
					Type:   ROUTERS,
					Fields: &routerField{},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.routerList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tt.expectedLenghtAfter, len(*tt.routerList), "Expected routerObjectList length to match")

				// Check that the new object is removed from the list
				originalFound := 0

				for _, obj := range *tt.routerList {
					if obj.Key == tt.queryParams["key"] {
						originalFound++
					}
				}

				assert.Equal(t, 0, originalFound, "Expected not to find the object in the list")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestDeleteRouterSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := deleteRouterSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		queryParams          map[string]string
		routerList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedLenghtAfter  int
	}{
		{
			name:    "nil objectlist",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList:           nil, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "empty objectlist",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList:           &[]inputObject{}, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:        "no key query parameter",
			handler:     handlerToTest,
			queryParams: map[string]string{}, // This is the error
			routerList: &[]inputObject{
				{},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "No key found",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key: "anotherKey", // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
		{
			name:    "Object not ROUTERS type",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "testKey",
			},
			routerList: &[]inputObject{
				{
					Key:  "testKey",
					Type: HEADLINE, // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.routerList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tt.expectedLenghtAfter, len(*tt.routerList), "Expected routerObjectList to increase by 1")

				// Check that the new object is added to the list
				originalFound := 0
				copyFound := 0

				for _, obj := range *tt.routerList {
					if obj.Key == tt.queryParams["key"] {
						originalFound++
					}
					if obj.Key == tt.queryParams["key"]+"_copy" {
						copyFound++
					}
				}

				assert.Equal(t, 1, originalFound, "Expected (only one) original object to be in list")
				assert.Equal(t, 1, copyFound, "Expected (only one) copy object to be in list")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestNewSequenceProcessorSubmitHandler_Add(t *testing.T) {

	handlerToTest := newSequenceProcessorSubmitHandler
	processor := processorField{}

	// Define test scenarios
	tests := []struct {
		name                   string
		handler                updateSubmitHandler
		queryParams            map[string]string
		objectList             *[]inputObject
		redirect               string
		expectedError          bool
		expectedResponseCode   int
		expectedRedirect       string
		expectedLenghtAfter    int
		expectedCountAfter     int
		expectedFirstElemetKey string
		expectedLastELementKey string
	}{
		{
			name:                   "Empty list",
			handler:                handlerToTest,
			queryParams:            map[string]string{},
			objectList:             &[]inputObject{},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedLenghtAfter:    1,
			expectedCountAfter:     1,
			expectedFirstElemetKey: "added",
			expectedLastELementKey: "added",
		},
		{
			name:        "Longer list",
			handler:     handlerToTest,
			queryParams: map[string]string{},
			objectList: &[]inputObject{
				{
					Type:     PROCESSORS,
					Headline: "test",
					Key:      "key",
					Fields: &processorField{
						Async:    false,
						ErrorMsg: "",
						Objects: []inputObject{
							inputObject{
								Type:     PROCESSORS,
								Headline: "",
								Key:      "new0",
								Fields:   processor.new(),
							},
							inputObject{
								Type:     PROCESSORS,
								Headline: "",
								Key:      "new1",
								Fields:   processor.new(),
							},
						},
					},
				},
				{
					Type:     PROCESSORS,
					Headline: "test",
					Key:      "key",
					Fields: &processorField{
						Async:    false,
						ErrorMsg: "",
						Objects: []inputObject{
							inputObject{
								Type:     PROCESSORS,
								Headline: "",
								Key:      "new0",
								Fields:   processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedLenghtAfter:    3, // row length
			expectedCountAfter:     4, // count
			expectedFirstElemetKey: "key",
			expectedLastELementKey: "added",
		},
		{
			name:    "Empty list - query before",
			handler: handlerToTest,
			queryParams: map[string]string{
				"before": "true",
			},
			objectList:             &[]inputObject{},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedLenghtAfter:    1,
			expectedCountAfter:     1,
			expectedFirstElemetKey: "added",
			expectedLastELementKey: "added",
		},
		{
			name:    "Longer lis - query before",
			handler: handlerToTest,
			queryParams: map[string]string{
				"before": "true",
			},
			objectList: &[]inputObject{
				{
					Type:     PROCESSORS,
					Headline: "test",
					Key:      "key",
					Fields: &processorField{
						Async:    false,
						ErrorMsg: "",
						Objects: []inputObject{
							inputObject{
								Type:     PROCESSORS,
								Headline: "",
								Key:      "new0",
								Fields:   processor.new(),
							},
							inputObject{
								Type:     PROCESSORS,
								Headline: "",
								Key:      "new1",
								Fields:   processor.new(),
							},
						},
					},
				},
				{
					Type:     PROCESSORS,
					Headline: "test",
					Key:      "key",
					Fields: &processorField{
						Async:    false,
						ErrorMsg: "",
						Objects: []inputObject{
							inputObject{
								Type:     PROCESSORS,
								Headline: "",
								Key:      "new0",
								Fields:   processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedLenghtAfter:    3, // row length
			expectedCountAfter:     4, // count
			expectedFirstElemetKey: "added",
			expectedLastELementKey: "key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tt.expectedLenghtAfter, len(*tt.objectList), "Expected routerObjectList length to match")
				count := 0
				for _, obj := range *tt.objectList {
					count += len(obj.Fields.(*processorField).Objects)
				}
				assert.Equal(t, tt.expectedCountAfter, count, "Expected count to match")
				assert.Equal(t, tt.expectedFirstElemetKey, (*tt.objectList)[0].Key, "Expected first element key to match")
				assert.Equal(t, tt.expectedLastELementKey, (*tt.objectList)[len(*tt.objectList)-1].Key, "Expected last element key to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestNewSequenceProcessorSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := newSequenceProcessorSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		objectList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedLenghtAfter  int
	}{
		{
			name:                 "objectlist nil",
			handler:              handlerToTest,
			objectList:           nil, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedLenghtAfter:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tt.expectedLenghtAfter, len(*tt.objectList), "Expected routerObjectList length to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestNewParallelProcessorSubmitHandler_Add(t *testing.T) {

	handlerToTest := newParallelProcessorSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                   string
		handler                updateSubmitHandler
		row                    int
		queryParams            map[string]string
		objectList             *[]inputObject
		redirect               string
		expectedError          bool
		expectedResponseCode   int
		expectedRedirect       string
		expectedRowLengthAfter int
		expectedCountAfter     int
	}{
		{
			name:    "Add in list of one",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 2,
			expectedCountAfter:     2,
		},
		{
			name:    "Add in list of two",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 3,
			expectedCountAfter:     3,
		},
		{
			name:    "Add to second row",
			handler: handlerToTest,
			row:     1,
			queryParams: map[string]string{
				"row": "1",
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
							{},
						},
					},
				},
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
							{},
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 2,
			expectedCountAfter:     7,
		},
		{
			name:    "Add to last row",
			handler: handlerToTest,
			row:     2,
			queryParams: map[string]string{
				"row": "2",
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
							{},
						},
					},
				},
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
							{},
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 4,
			expectedCountAfter:     7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				// Check that the new object is added to the list
				count := 0
				rowcount := 0

				for r, obj := range *tt.objectList {
					count += len(obj.Fields.(*processorField).Objects)
					if tt.row == r {
						rowcount = len(obj.Fields.(*processorField).Objects)
					}
				}

				assert.Equal(t, tt.expectedRowLengthAfter, rowcount, "Expected row length to match")
				assert.Equal(t, tt.expectedCountAfter, count, "Expected count to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestNewParallelProcessorSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := newParallelProcessorSubmitHandler

	// Define test scenarios
	tests := []struct {
		name                   string
		handler                updateSubmitHandler
		row                    int
		queryParams            map[string]string
		objectList             *[]inputObject
		redirect               string
		expectedError          bool
		expectedResponseCode   int
		expectedRedirect       string
		expectedRowLengthAfter int
		expectedCountAfter     int
	}{
		{
			name:    "Objectlist nil",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList:             nil, // This is the error
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Objectlist is empty",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList:             &[]inputObject{}, // This is the error
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Non-numeric row",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "notanumber", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Empty row",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:        "No row",
			handler:     handlerToTest,
			row:         0,
			queryParams: map[string]string{}, // This is the error
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Negative row",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "-3", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Too high row",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "699", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Type not processors",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: ROUTERS, // This is the error
					Fields: &processorField{
						Objects: []inputObject{
							{
								Type:   HEADLINE,
								Fields: &processorField{},
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Fields nil",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList: &[]inputObject{
				{
					Key:    "testKey",
					Type:   PROCESSORS,
					Fields: nil, // This is the error
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Fields not processorField",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList: &[]inputObject{
				{
					Key:    "testKey",
					Type:   PROCESSORS,
					Fields: &headerField{}, // This is the error
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
		{
			name:    "Fields objects is nil",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"row": "0",
			},
			objectList: &[]inputObject{
				{
					Key:  "testKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: nil, // This is the error
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 1,
			expectedCountAfter:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				// Check that the new object is added to the list
				count := 0
				rowcount := 0

				for r, obj := range *tt.objectList {
					count += len(obj.Fields.(*processorField).Objects)
					if tt.row == r {
						rowcount = len(obj.Fields.(*processorField).Objects)
					}
				}

				assert.Equal(t, tt.expectedRowLengthAfter, rowcount, "Expected row length to match")
				assert.Equal(t, tt.expectedCountAfter, count, "Expected count to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestDeleteProcessorSubmitHandler_Delete(t *testing.T) {

	handlerToTest := deleteProcessorSubmitHandler
	processor := processorField{}

	// Define test scenarios
	tests := []struct {
		name                   string
		handler                updateSubmitHandler
		row                    int
		queryParams            map[string]string
		objectList             *[]inputObject
		redirect               string
		expectedError          bool
		expectedResponseCode   int
		expectedRedirect       string
		expectedRowLengthAfter int
		expectedCountAfter     int
	}{
		{
			name:    "Remove one from list of one",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Remove one in the middle - last in the row",
			handler: handlerToTest,
			row:     1,
			queryParams: map[string]string{
				"key": "key4",
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key2",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
							{
								Key:    "key3",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
							{
								Key:    "key4",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key5",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 2,
			expectedCountAfter:     4,
		},
		{
			name:    "Remove one in the middle - first in the row",
			handler: handlerToTest,
			row:     1,
			queryParams: map[string]string{
				"key": "key2",
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key2",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
							{
								Key:    "key3",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
							{
								Key:    "key4",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key5",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 2,
			expectedCountAfter:     4,
		},
		{
			name:    "Remove Last object",
			handler: handlerToTest,
			row:     2,
			queryParams: map[string]string{
				"key": "key5",
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key2",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
							{
								Key:    "key3",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
							{
								Key:    "key4",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key5",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          false,
			expectedResponseCode:   http.StatusFound,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				// Check that the new object is added to the list
				count := 0
				rowcount := 0

				for r, obj := range *tt.objectList {
					count += len(obj.Fields.(*processorField).Objects)
					if tt.row == r {
						rowcount = len(obj.Fields.(*processorField).Objects)
					}
					for _, o := range obj.Fields.(*processorField).Objects {
						assert.NotEqual(t, tt.queryParams["key"], o.Key, "Expected key to be removed")
					}
				}

				assert.Equal(t, tt.expectedRowLengthAfter, rowcount, "Expected row length to match")
				assert.Equal(t, tt.expectedCountAfter, count, "Expected count to match")

			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestDeleteProcessorSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := deleteProcessorSubmitHandler
	processor := processorField{}

	// Define test scenarios
	tests := []struct {
		name                   string
		handler                updateSubmitHandler
		row                    int
		queryParams            map[string]string
		objectList             *[]inputObject
		redirect               string
		expectedError          bool
		expectedResponseCode   int
		expectedRedirect       string
		expectedRowLengthAfter int
		expectedCountAfter     int
	}{
		{
			name:    "objectlist nil",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList:             nil, // This is the error
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "empty objectlist",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList:             &[]inputObject{}, // This is the error
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:        "empty key",
			handler:     handlerToTest,
			row:         0,
			queryParams: map[string]string{}, // This is the error
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Type not processors",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: ROUTERS, // This is the error
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Field is nil",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList: &[]inputObject{
				{
					Key:    "anyKey",
					Type:   PROCESSORS,
					Fields: nil, // This is the error
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Field is not processorField",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList: &[]inputObject{
				{
					Key:    "anyKey",
					Type:   PROCESSORS,
					Fields: &headlineField{}, // This is the error
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "key not found",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "wrongkey", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   PROCESSORS,
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusBadRequest,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
		{
			name:    "Found object Type not processors",
			handler: handlerToTest,
			row:     0,
			queryParams: map[string]string{
				"key": "key1",
			},
			objectList: &[]inputObject{
				{
					Key:  "anyKey",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key1",
								Type:   ROUTERS, // This is the error
								Fields: processor.new(),
							},
						},
					},
				},
			},
			redirect:               "anypath",
			expectedError:          true,
			expectedResponseCode:   http.StatusInternalServerError,
			expectedRedirect:       "anypath",
			expectedRowLengthAfter: 0,
			expectedCountAfter:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				// Check that the new object is added to the list
				count := 0
				rowcount := 0

				for r, obj := range *tt.objectList {
					count += len(obj.Fields.(*processorField).Objects)
					if tt.row == r {
						rowcount = len(obj.Fields.(*processorField).Objects)
					}
					for _, o := range obj.Fields.(*processorField).Objects {
						assert.NotEqual(t, tt.queryParams["key"], o.Key, "Expected key to be removed")
					}
				}

				assert.Equal(t, tt.expectedRowLengthAfter, rowcount, "Expected row length to match")
				assert.Equal(t, tt.expectedCountAfter, count, "Expected count to match")

			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestWrapFormHandlerFunc(t *testing.T) {
	// Define test scenarios
	tests := []struct {
		name                       string
		ctx                        context.Context
		w                          *webbAppState
		areaIndex                  int
		form                       string
		headline                   string
		defaultErrorMsg            string
		defaultErrorMsgToolTipText string
		submit                     string
		reload                     string
		quit                       string
		formHandler                populateFormHandler
		redirect                   string
		expectedResponseBody       string
		expectedResponseCode       int
	}{
		{
			name:      "w is nil",
			ctx:       context.Background(),
			w:         nil, // This is the error
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "areas is nil",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas:  nil, // This is the error
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "areaIndex < 0 ",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: -2, // This is the error
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "areaIndex to high",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 677, // This is the error
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "form is empty",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			form:      "", // This is the error
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "submit is empty",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "", // This is the error
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "reload is empty",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "", // This is the error
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "quit is empty",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "", // This is the error
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "validate is nil",
			ctx:  context.Background(),
			w: &webbAppState{
				config:   &mockConfig{},
				validate: nil, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "formHandler is nil",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				validate: func(ctx context.Context) error {
					return nil
				},
			},
			areaIndex:            0,
			form:                 "form.html",
			headline:             "Test Headline",
			submit:               "submit",
			reload:               "reload",
			quit:                 "quit",
			formHandler:          nil, // This is the error
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "validation fails",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				validate: func(ctx context.Context) error {
					// This is the error
					return fmt.Errorf("Validation failed")
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "updateAreasFromConfig fails",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfigFailedUpdateAreasFromConfig{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				validate: func(ctx context.Context) error {
					return nil
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// No filtering or selection needed
				return http.StatusOK, "", (*objectList), nil
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "handler fails",
			ctx:  context.Background(),
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				validate: func(ctx context.Context) error {
					return nil
				},
			},
			areaIndex: 0,
			form:      "form.html",
			headline:  "Test Headline",
			submit:    "submit",
			reload:    "reload",
			quit:      "quit",
			formHandler: func(c *gin.Context, objectList *[]inputObject, errorMsg *string, errorMsgToolTipText *string) (int, string, []inputObject, error) {
				// This is the error
				return http.StatusBadRequest, "", (*objectList), fmt.Errorf("Handler failed")
			},
			redirect:             "anypath",
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			testFunc := wrapFormHandlerFunc(tt.ctx, tt.w, tt.areaIndex, tt.form, tt.submit, tt.reload, tt.quit, tt.formHandler)

			ginRouter.GET("/test", testFunc)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
		})
	}
}

type mockField struct {
	count int
}

func (m mockField) ErrorMessage() string {
	return ""
}

func (m mockField) ErrorMessageTooltipText() string {
	return ""
}

// Will convert the values to an integer and add it to the count
func (m *mockField) setValue(value []string) error {
	logger.Error("values received", slog.Any("values", value))
	for _, s := range value {
		v, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		m.count += v
	}
	return nil
}

func (m *mockField) copy() field {
	return &mockField{
		count: m.count,
	}
}
func (m *mockField) new() field {
	return &mockField{}
}

func TestUpdateObjectsSubmitHandler_Update(t *testing.T) {

	handlerToTest := updateObjectsSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		row                  int
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "Set one value",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "Set one value from array",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1", "3", "5"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   9,
		},
		{
			name:    "Set multiple values",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
				"key1": {"3"},
				"key2": {"5"},
				"key3": {"7"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key1",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key2",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key3",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   16,
		},
		{
			name:    "Set multiple values - one with array",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
				"key1": {"3", "5", "7"},
				"key2": {"11"},
				"key3": {"13"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key1",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key2",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key3",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   40,
		},
		{
			name:    "Set multiple values - one is missing",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
				"key1": {"3"},
				"key3": {"7"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key1",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key2",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
				{
					Key:    "key3",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   11,
		},
		{
			name:    "Set one value - redirect",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "newpath",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "newpath",
			expectedTotalCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestUpdateObjectsSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := updateObjectsSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		row                  int
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             *string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "objectList is nil",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect:         "",
			objectList:           nil, // This is the error
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "redirect is nil",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             nil, // This is the error
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "field is nil",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: nil,   // This is the error
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "setValue error is nil",
			handler: handlerToTest,
			row:     0,
			formData: map[string][]string{
				"key0": {"notaninteger"}, // This is the error
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   INPUT, // This is not used
					Fields: &field,
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			if tt.redirect != nil {
				assert.Equal(t, tt.expectedRedirect, *tt.redirect, "Expected redirect location to match")
			}

		})
	}
}

func TestFindObject_Find(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name                 string
		objectList           *[]inputObject
		key                  string
		expectedResponseCode int
		expectedError        bool
		expectedHeadline     string
	}{
		{
			name: "list of one",
			objectList: &[]inputObject{
				{
					Key:      "key0",
					Type:     INPUT, // This is not used
					Headline: "headline0",
				},
			},
			key:                  "key0",
			expectedResponseCode: http.StatusFound,
			expectedError:        false,
			expectedHeadline:     "headline0",
		},
		{
			name: "list of two",
			objectList: &[]inputObject{
				{
					Key:      "key0",
					Type:     INPUT, // This is not used
					Headline: "headline0",
				},
				{
					Key:      "key1",
					Type:     INPUT, // This is not used
					Headline: "headline1",
				},
			},
			key:                  "key1",
			expectedResponseCode: http.StatusFound,
			expectedError:        false,
			expectedHeadline:     "headline1",
		},
		{
			name: "Longer list",
			objectList: &[]inputObject{
				{
					Key:      "key0",
					Type:     INPUT, // This is not used
					Headline: "headline0",
				},
				{
					Key:      "key1",
					Type:     INPUT, // This is not used
					Headline: "headline1",
				},
				{
					Key:      "key2",
					Type:     INPUT, // This is not used
					Headline: "headline2",
				},
				{
					Key:      "key3",
					Type:     INPUT, // This is not used
					Headline: "headline3",
				},
				{
					Key:      "key4",
					Type:     INPUT, // This is not used
					Headline: "headline4",
				},
			},
			key:                  "key3",
			expectedResponseCode: http.StatusFound,
			expectedError:        false,
			expectedHeadline:     "headline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			status, object, err := findObject(tt.objectList, tt.key)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.NotNil(t, object, "Expected object to be found")
				if object != nil {
					assert.Equal(t, tt.expectedHeadline, object.Headline, "Expected headline to match")
					assert.Equal(t, tt.key, object.Key, "Expected key to match")
				}

			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

		})
	}
}

func TestFindObject_BadVars(t *testing.T) {

	// Define test scenarios
	tests := []struct {
		name                 string
		objectList           *[]inputObject
		key                  string
		expectedResponseCode int
		expectedError        bool
		expectedHeadline     string
	}{
		{
			name:                 "Objectlist nil",
			objectList:           nil, // This is the error
			key:                  "key0",
			expectedResponseCode: http.StatusInternalServerError,
			expectedError:        true,
			expectedHeadline:     "headline0",
		},
		{
			name: "Cant find key",
			objectList: &[]inputObject{
				{
					Key:      "key0",
					Type:     INPUT, // This is not used
					Headline: "headline0",
				},
				{
					Key:      "key1",
					Type:     INPUT, // This is not used
					Headline: "headline1",
				},
			},
			key:                  "wrong key", // This is the error
			expectedResponseCode: http.StatusBadRequest,
			expectedError:        true,
			expectedHeadline:     "headline1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			status, object, err := findObject(tt.objectList, tt.key)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")
				assert.NotNil(t, object, "Expected object to be found")
				if object != nil {
					assert.Equal(t, tt.expectedHeadline, object.Headline, "Expected headline to match")
					assert.Equal(t, tt.key, object.Key, "Expected key to match")
				}

			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

		})
	}
}

func TestUpdateRouterObjectsSubmitHandler_Update(t *testing.T) {

	handlerToTest := updateRouterObjectSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		key                  string
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "Set one value in one router list",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":   {"findme"},
				"field": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "findme",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "field",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "Set mulitple values in longer router list",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
				"field2": {"5", "7", "9"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "notme",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "field1",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "findme",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "field0",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
							{
								Key:    "field1",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
							{
								Key:    "field2",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "norme",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "field2",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestUpdateRouterObjectsSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := updateRouterObjectSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		key                  string
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "objectlist is ni",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":   {"findme"},
				"field": {"1"},
			},
			formRedirect:         "",
			objectList:           nil, // This is the error
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "routerObject field is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
				"field2": {"5", "7", "9"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "findme",
					Type:   ROUTERS,
					Fields: nil, // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   25,
		},
		{
			name:    "field is not routerField",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
				"field2": {"5", "7", "9"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "findme",
					Type:   ROUTERS,
					Fields: &field, // This is the error
				},
			},
			redirect:             "anypath",
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestUpdateAllRouterPathsSubmitHandler_Update(t *testing.T) {

	handlerToTest := updateAllRouterPathsSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "One router list",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "Longer router list",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
				"key1": {"3"},
				"key2": {"5", "7", "9"},
				"key3": {"11"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "key1",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "key2",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "key3",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   36,
		},
		{
			name:    "Redirect",
			handler: handlerToTest,
			formData: map[string][]string{
				"redirect": {"newpath"},
				"key0":     {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "newpath",
			expectedTotalCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestUpdateAllRouterPathsSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := updateAllRouterPathsSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             *string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "objectlist is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect:         "",
			objectList:           nil, // This is the error
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "redirect is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             nil, // This is the error
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "field is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   ROUTERS,
					Fields: nil, // This is the error
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "field is not routerfield",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   ROUTERS,
					Fields: &field, // This is the error
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "field field is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: nil, // This is the error
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "Path not found",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"1"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "notpath", // This is the error
								Type:   INPUT,     // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
		{
			name:    "setValue fails found",
			handler: handlerToTest,
			formData: map[string][]string{
				"key0": {"notaninteger"}, // This is the error
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			if tt.redirect != nil {
				assert.Equal(t, tt.expectedRedirect, *tt.redirect, "Expected redirect location to match")
			}
		})
	}
}

func TestUpdateProcessorObjectSubmitHandler_Update(t *testing.T) {

	handlerToTest := updateProcessorObjectSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		key                  string
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "Set two value in one processor list",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "findme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Set two value in larger processor list",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "notme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
				{
					Key:  "row1",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "norme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
							{
								Key:  "findme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			redirect:             "anypath",
			expectedError:        false,
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, &tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			assert.Equal(t, tt.expectedRedirect, tt.redirect, "Expected redirect location to match")

		})
	}
}

func TestUpdateProcessorObjectSubmitHandler_BadVars(t *testing.T) {

	handlerToTest := updateProcessorObjectSubmitHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                 string
		handler              updateSubmitHandler
		key                  string
		formData             map[string][]string
		formRedirect         string
		objectList           *[]inputObject
		redirect             *string
		expectedError        bool
		expectedResponseCode int
		expectedRedirect     string
		expectedTotalCount   int
	}{
		{
			name:    "objectlist is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect:         "",
			objectList:           nil, // This is the error
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Redirect is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "notme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			redirect:             nil, // This is the error
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Key not found",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"wrongkey"}, // This is the error
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "findme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Type not processor",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: ROUTERS, // This is the error
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "findme",
								Type: PROCESSORS,
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Field is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "row0",
					Type:   PROCESSORS,
					Fields: nil, // This is the error
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Field is not processorfield",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:    "row0",
					Type:   PROCESSORS,
					Fields: &field, // This is the error
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Found object has wrong type",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "findme",
								Type: ROUTERS, // This is the error
								Fields: &processorField{
									Objects: []inputObject{
										{
											Key:    "field0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "field1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Foundobject field is nil",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "findme",
								Type:   PROCESSORS,
								Fields: nil, // This is the error
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
		{
			name:    "Foundobject field is not processorfield",
			handler: handlerToTest,
			formData: map[string][]string{
				"key":    {"findme"},
				"field0": {"1"},
				"field1": {"3"},
			},
			formRedirect: "",
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "findme",
								Type:   PROCESSORS,
								Fields: &field, // This is the error
							},
						},
					},
				},
			},
			redirect:             func() *string { s := "anypath"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "anypath",
			expectedTotalCount:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with for post form
			form := make(url.Values)
			for key, value := range tt.formData {
				for _, v := range value {
					form.Add(key, v)
				}
			}
			form.Add("redirect", tt.formRedirect)
			req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Assign the request to the context
			c.Request = req

			status, err := tt.handler(c, tt.objectList, tt.redirect)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedTotalCount, field.count, "Expected totals to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")
			if tt.redirect != nil {
				assert.Equal(t, tt.expectedRedirect, *tt.redirect, "Expected redirect location to match")
			}

		})
	}
}

func TestPopulateRouterObjectsFormHHandler_Populate(t *testing.T) {

	handlerToTest := populateRouterObjectsFormHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                        string
		handler                     populateFormHandler
		queryParams                 map[string]string
		objectList                  *[]inputObject
		errorMsg                    string
		errorMsgToolTipText         string
		expectedError               bool
		expectedResponseCode        int
		expectedKeys                map[string]struct{}
		expectedErrorMsg            string
		expectedErrorMsgToolTipText string
		expectedKey                 string
	}{
		{
			name:    "One router list",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             "noerror",
			errorMsgToolTipText:  "noerror",
			expectedError:        false,
			expectedResponseCode: http.StatusOK,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Bigger router list",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key2",
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "key1",
					Type: ROUTERS,
					Fields: &routerField{
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
				{
					Key:  "key2",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path0",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
							{
								Key:    "path1",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
							{
								Key:    "path2",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             "noerror",
			errorMsgToolTipText:  "noerror",
			expectedError:        false,
			expectedResponseCode: http.StatusOK,
			expectedKeys: map[string]struct{}{
				"path0": struct{}{},
				"path1": struct{}{},
				"path2": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, key, objects, err := tt.handler(c, tt.objectList, &tt.errorMsg, &tt.errorMsgToolTipText)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedKey, key, "Expected keys to match")
				assert.NotNil(t, objects, "Expected objects to be populated")
				for _, object := range objects {
					_, ok := tt.expectedKeys[object.Key]
					assert.True(t, ok, "Expected key to be found")
				}
				assert.Equal(t, tt.expectedErrorMsg, tt.errorMsg, "Expected error message to match")
				assert.Equal(t, tt.expectedErrorMsgToolTipText, tt.errorMsgToolTipText, "Expected error message tooltip text to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

		})
	}
}

func TestPopulateRouterObjectsFormHHandler_BadVars(t *testing.T) {

	handlerToTest := populateRouterObjectsFormHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                        string
		handler                     populateFormHandler
		queryParams                 map[string]string
		objectList                  *[]inputObject
		errorMsg                    *string
		errorMsgToolTipText         *string
		expectedError               bool
		expectedResponseCode        int
		expectedKeys                map[string]struct{}
		expectedErrorMsg            string
		expectedErrorMsgToolTipText string
		expectedKey                 string
	}{
		{
			name:    "Objectlist nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList:           nil, // This is the error
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "ErrorMsg nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             nil, // This is the error
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "ErrorMsgTooltipText nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  nil, // This is the error
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:        "No key",
			handler:     handlerToTest,
			queryParams: map[string]string{}, // This is the error
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Key not found",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "badkey", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Type is not router",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: HEADLINE, // This is the error
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Found object fields are empty",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   ROUTERS,
					Fields: nil, // This is the error
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Field is not routerfield",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:    "key0",
					Type:   ROUTERS,
					Fields: &field, // This is the error
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "One router list",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "key0",
					Type: ROUTERS,
					Fields: &routerField{
						ErrorMsg:            "newerror",
						ErrorMsgTooltipText: "newerror",
						Fields: []inputObject{
							{
								Key:    "path",
								Type:   INPUT, // This is not used
								Fields: &field,
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        false,
			expectedResponseCode: http.StatusOK,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, key, objects, err := tt.handler(c, tt.objectList, tt.errorMsg, tt.errorMsgToolTipText)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedKey, key, "Expected keys to match")
				assert.NotNil(t, objects, "Expected objects to be populated")
				for _, object := range objects {
					_, ok := tt.expectedKeys[object.Key]
					assert.True(t, ok, "Expected key to be found")
				}

				if tt.errorMsg != nil {
					assert.Equal(t, tt.expectedErrorMsg, *tt.errorMsg, "Expected error message to match")
				}
				if tt.errorMsgToolTipText != nil {
					assert.Equal(t, tt.expectedErrorMsgToolTipText, *tt.errorMsgToolTipText, "Expected error message tooltip text to match")
				}
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

		})
	}
}

func TestPopulateProcessorObjectsFormHHandler_Populate(t *testing.T) {

	handlerToTest := populateProcessorObjectsFormHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                        string
		handler                     populateFormHandler
		queryParams                 map[string]string
		objectList                  *[]inputObject
		errorMsg                    string
		errorMsgToolTipText         string
		expectedError               bool
		expectedResponseCode        int
		expectedKeys                map[string]struct{}
		expectedErrorMsg            string
		expectedErrorMsgToolTipText string
		expectedKey                 string
	}{
		{
			name:    "One processor list",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             "noerror",
			errorMsgToolTipText:  "noerror",
			expectedError:        false,
			expectedResponseCode: http.StatusOK,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Large processor list",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key2",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "error",
									ErrorMsgTooltipText: "error",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
							{
								Key:  "key1",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "error",
									ErrorMsgTooltipText: "error",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
				{
					Key:  "row1",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key2",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path0",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "path1",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
										{
											Key:    "path2",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
							{
								Key:  "key3",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "error",
									ErrorMsgTooltipText: "error",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             "noerror",
			errorMsgToolTipText:  "noerror",
			expectedError:        false,
			expectedResponseCode: http.StatusOK,
			expectedKeys: map[string]struct{}{
				"path0": struct{}{},
				"path1": struct{}{},
				"path2": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, key, objects, err := tt.handler(c, tt.objectList, &tt.errorMsg, &tt.errorMsgToolTipText)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedKey, key, "Expected keys to match")
				assert.NotNil(t, objects, "Expected objects to be populated")
				for _, object := range objects {
					_, ok := tt.expectedKeys[object.Key]
					assert.True(t, ok, "Expected key to be found")
				}
				assert.Equal(t, tt.expectedErrorMsg, tt.errorMsg, "Expected error message to match")
				assert.Equal(t, tt.expectedErrorMsgToolTipText, tt.errorMsgToolTipText, "Expected error message tooltip text to match")
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

		})
	}
}

func TestPopulateProcessorObjectsFormHHandler_BadVars(t *testing.T) {

	handlerToTest := populateProcessorObjectsFormHandler
	field := mockField{
		count: 0,
	}

	// Define test scenarios
	tests := []struct {
		name                        string
		handler                     populateFormHandler
		queryParams                 map[string]string
		objectList                  *[]inputObject
		errorMsg                    *string
		errorMsgToolTipText         *string
		expectedError               bool
		expectedResponseCode        int
		expectedKeys                map[string]struct{}
		expectedErrorMsg            string
		expectedErrorMsgToolTipText string
		expectedKey                 string
	}{
		{
			name:    "Objectlist nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList:           nil, // This is the error
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "ErrorMsg nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             nil, // This is the error
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "ErrorMsgTooltip nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  nil, // This is the error
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:        "Empty key",
			handler:     handlerToTest,
			queryParams: map[string]string{}, // This is the error
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Key not found",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "wrongkey", // This is the error
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusBadRequest,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Field is nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:    "row0",
					Type:   PROCESSORS,
					Fields: nil, // This is the error
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Field is not proessorfield",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:    "row0",
					Type:   PROCESSORS,
					Fields: &field, // This is the error
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Object is nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: nil, // This is the error
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Found object type not processor",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: ROUTERS, // This the error
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "foundObject field is nil",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key0",
								Type:   PROCESSORS,
								Fields: nil, // This is the error
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "Found object field is not processorField",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:    "key0",
								Type:   PROCESSORS,
								Fields: &field, // This is the error
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        true,
			expectedResponseCode: http.StatusInternalServerError,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
		{
			name:    "One processor list",
			handler: handlerToTest,
			queryParams: map[string]string{
				"key": "key0",
			},
			objectList: &[]inputObject{
				{
					Key:  "row0",
					Type: PROCESSORS,
					Fields: &processorField{
						Objects: []inputObject{
							{
								Key:  "key0",
								Type: PROCESSORS,
								Fields: &processorField{
									ErrorMsg:            "newerror",
									ErrorMsgTooltipText: "newerror",
									Objects: []inputObject{
										{
											Key:    "path",
											Type:   INPUT, // This is not used
											Fields: &field,
										},
									},
								},
							},
						},
					},
				},
			},
			errorMsg:             func() *string { s := "noerror"; return &s }(),
			errorMsgToolTipText:  func() *string { s := "noerror"; return &s }(),
			expectedError:        false,
			expectedResponseCode: http.StatusOK,
			expectedKeys: map[string]struct{}{
				"path": struct{}{},
			},
			expectedErrorMsg:            "newerror",
			expectedErrorMsgToolTipText: "newerror",
			expectedKey:                 "key0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			field.count = 0

			w := httptest.NewRecorder()

			// Create a new gin context
			c, _ := gin.CreateTestContext(w)

			// Create a new HTTP request with the query parameters
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Assign the request to the context
			c.Request = req

			status, key, objects, err := tt.handler(c, tt.objectList, tt.errorMsg, tt.errorMsgToolTipText)

			switch tt.expectedError {
			case true:
				assert.NotNil(t, err, "Expected an error")
			case false:
				assert.Nil(t, err, "Expected no error")

				assert.Equal(t, tt.expectedKey, key, "Expected keys to match")
				assert.NotNil(t, objects, "Expected objects to be populated")
				for _, object := range objects {
					_, ok := tt.expectedKeys[object.Key]
					assert.True(t, ok, "Expected key to be found")
				}
				if tt.errorMsg != nil {
					assert.Equal(t, tt.expectedErrorMsg, *tt.errorMsg, "Expected error message to match")
				}
				if tt.errorMsgToolTipText != nil {
					assert.Equal(t, tt.expectedErrorMsgToolTipText, *tt.errorMsgToolTipText, "Expected error message tooltip text to match")
				}
			}

			assert.Equal(t, tt.expectedResponseCode, status, "Expected response code to match")

		})
	}
}

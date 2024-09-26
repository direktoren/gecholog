package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"text/template"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"github.com/stretchr/testify/assert"
)

func TestWrapFileListHandlerFunc(t *testing.T) {

	handlerToTest := wrapFileListHandlerFunc

	// Define test scenarios
	tests := []struct {
		name        string
		w           *webbAppState
		form        string
		source      string
		redirect    string
		fileHandler fileMetaHandler

		expectedResponseBody string
		expectedResponseCode int
	}{
		{
			name:     "w is nil",
			w:        nil, // This is the error
			form:     "form.html",
			source:   "source",
			redirect: "anypath",
			fileHandler: func(w *webbAppState) ([]fileMeta, fileFlags, error) {
				return []fileMeta{}, fileFlags{}, nil
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "form is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			form:     "", // This is the error
			source:   "source",
			redirect: "anypath",
			fileHandler: func(w *webbAppState) ([]fileMeta, fileFlags, error) {
				return []fileMeta{}, fileFlags{}, nil
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "source is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			form:     "form.html",
			source:   "", // This is the error
			redirect: "anypath",
			fileHandler: func(w *webbAppState) ([]fileMeta, fileFlags, error) {
				return []fileMeta{}, fileFlags{}, nil
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "redirect is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			form:     "form.html",
			source:   "source",
			redirect: "", // This is the error
			fileHandler: func(w *webbAppState) ([]fileMeta, fileFlags, error) {
				return []fileMeta{}, fileFlags{}, nil
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "handler is nil",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			form:                 "form.html",
			source:               "source",
			redirect:             "anypath",
			fileHandler:          nil, // This is the error
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "handler fails",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			form:     "form.html",
			source:   "source",
			redirect: "anypath",
			fileHandler: func(w *webbAppState) ([]fileMeta, fileFlags, error) {
				return []fileMeta{}, fileFlags{}, fmt.Errorf("error") // This is the error
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "basic success",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			form:     "form.html",
			source:   "source",
			redirect: "anypath",
			fileHandler: func(w *webbAppState) ([]fileMeta, fileFlags, error) {
				return []fileMeta{}, fileFlags{}, nil
			},
			expectedResponseBody: "",
			expectedResponseCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			ginRouter.SetFuncMap(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			ginRouter.LoadHTMLGlob("templates/*")
			ginRouter.Static("/static", "./static")

			testFunc := handlerToTest(tt.w, tt.form, tt.source, tt.redirect, tt.fileHandler, 0)

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

func TestFileWriteHandlerFunc(t *testing.T) {

	handlerToTest := fileWriteHandlerFunc

	// Define test scenarios
	tests := []struct {
		name        string
		w           *webbAppState
		redirect    string
		fileHandler filenameFunction

		postForm map[string]string

		expectedResponseBody string
		expectedResponseCode int
		expectedRedirect     string
	}{
		{
			name:     "w is nil",
			w:        nil, // This is the error
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"filename": "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "w.config is nil",
			w: &webbAppState{
				config: nil,
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "postform redirect",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "newpath",
			},
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "/newpath",
		},
		{
			name: "redirect is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect: "", // This is the error
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "Filename handler is nil",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect:    "anypath",
			fileHandler: nil, // This is the error
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "filename handler fails",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return "", fmt.Errorf("error") // This is the error
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "write config file fails",
			w: &webbAppState{
				config: &mockConfigFailedWriteConfigFile{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "basic success",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "/anypath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			testFunc := handlerToTest(tt.w, tt.redirect, tt.fileHandler)

			ginRouter.POST("/test", testFunc)

			w := httptest.NewRecorder()
			form := make(url.Values)
			for key, value := range tt.postForm {
				form.Add(key, value)
			}
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")

		})
	}
}

func TestReadFileHandlerFunc(t *testing.T) {

	handlerToTest := readFileHandlerFunc

	// Define test scenarios
	tests := []struct {
		name           string
		w              *webbAppState
		cancelFunction context.CancelFunc
		redirect       string
		fileHandler    filenameFunction

		postForm map[string]string

		expectedResponseBody string
		expectedResponseCode int
		expectedRedirect     string
	}{
		{
			name: "w is nil",
			w:    nil, // This is the error
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "basic success",
			w: &webbAppState{
				config: nil, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "redirect is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "", // This is the error
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "filename handler is nil",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect:    "anypath",
			fileHandler: nil, // This is the error
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "filename handler fails",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return "", fmt.Errorf("error") // This is the error
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "load config file fails",
			w: &webbAppState{
				config: &mockConfigFailedLoadConfigFile{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "create areas fails",
			w: &webbAppState{
				config: &mockConfigFailedCreateAreas{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "failed update",
			w: &webbAppState{
				config: &mockConfigFailedUpdate{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "failed write config file",
			w: &webbAppState{
				config: &mockConfigFailedWriteConfigFile{}, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "basic success",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
			},
			cancelFunction: func() context.CancelFunc {
				_, cancel := context.WithCancel(context.Background())
				return cancel
			}(),
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			postForm: map[string]string{
				"file": "filename",
			},
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "/anypath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			testFunc := handlerToTest(tt.w, tt.cancelFunction, tt.redirect, tt.fileHandler)

			ginRouter.POST("/test", testFunc)

			w := httptest.NewRecorder()
			form := make(url.Values)
			for key, value := range tt.postForm {
				form.Add(key, value)
			}
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")

		})
	}
}

func TestCopyFileHandlerFunc(t *testing.T) {

	handlerToTest := copyFileHandlerFunc

	// Define test scenarios
	tests := []struct {
		name        string
		w           *webbAppState
		redirect    string
		fileHandler filenameFunction
		copyHandler copyFunction

		postForm map[string]string

		expectedResponseBody string
		expectedResponseCode int
		expectedRedirect     string
	}{
		{
			name:     "w is nil",
			w:        nil, // This is the error
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "w.config is nil",
			w: &webbAppState{
				config: nil, // This is the error
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "production filename is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "", // This is the error
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "redirect is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect: "", // This is the error
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "filehandler is nil",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect:    "anypath",
			fileHandler: nil, // This is the error
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "filehandler fails",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return "", fmt.Errorf("error") // This is the error
			},
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "copyhandler is nil",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: nil,
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "copyhandler fails",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: func(source, destination string) error {
				return fmt.Errorf("error") // This is the error
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
			expectedRedirect:     "",
		},
		{
			name: "basic success",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					productionFile: "filename",
				},
			},
			redirect: "anypath",
			fileHandler: func(filename string) (string, error) {
				return filename, nil
			},
			copyHandler: func(source, destination string) error {
				return nil
			},
			postForm: map[string]string{
				"file":     "filename",
				"redirect": "",
			},
			expectedResponseBody: "",
			expectedResponseCode: http.StatusFound,
			expectedRedirect:     "/anypath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			testFunc := handlerToTest(tt.w, tt.redirect, tt.fileHandler, tt.copyHandler)

			ginRouter.POST("/test", testFunc)

			w := httptest.NewRecorder()
			form := make(url.Values)
			for key, value := range tt.postForm {
				form.Add(key, value)
			}
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")

		})
	}
}

func TestCountChecksumMatches(t *testing.T) {
	tests := []struct {
		name             string
		files            []string
		expectedChecksum string
		generateChecksum checksumGenerator
		expectedMatches  []fileMeta
		expectedError    error
	}{
		{
			name:             "All matches",
			files:            []string{"file1", "file2", "file3"},
			expectedChecksum: "expectedChecksum",
			generateChecksum: func(file string) (string, error) {
				return "expectedChecksum", nil
			},
			expectedMatches: []fileMeta{
				{Filename: "file1", Match: true, Comment: "current"},
				{Filename: "file2", Match: true, Comment: "current"},
				{Filename: "file3", Match: true, Comment: "current"},
			},
			expectedError: nil,
		},
		{
			name:             "No matches",
			files:            []string{"file1", "file2", "file3"},
			expectedChecksum: "expectedChecksum",
			generateChecksum: func(file string) (string, error) {
				return "differentChecksum", nil
			},
			expectedMatches: []fileMeta{
				{Filename: "file1"},
				{Filename: "file2"},
				{Filename: "file3"},
			},
			expectedError: nil,
		},
		{
			name:             "Mixed matches",
			files:            []string{"file1", "file2", "file3"},
			expectedChecksum: "expectedChecksum",
			generateChecksum: func(file string) (string, error) {
				if file == "file2" {
					return "expectedChecksum", nil
				}
				return "differentChecksum", nil
			},
			expectedMatches: []fileMeta{
				{Filename: "file1"},
				{Filename: "file2", Match: true, Comment: "current"},
				{Filename: "file3"},
			},
			expectedError: nil,
		},
		{
			name:             "Checksum generation error",
			files:            []string{"file1", "file2", "file3"},
			expectedChecksum: "expectedChecksum",
			generateChecksum: func(file string) (string, error) {
				if file == "file2" {
					return "", fmt.Errorf("checksum generation error")
				}
				return "differentChecksum", nil
			},
			expectedMatches: []fileMeta{},
			expectedError:   fmt.Errorf("checksum generation error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := countChecksumMatches(tt.files, tt.expectedChecksum, tt.generateChecksum)

			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.Nil(t, err)
			}

			assert.NotNil(t, matches)
			assert.Equal(t, len(tt.expectedMatches), len(matches))
			assert.Equal(t, tt.expectedMatches, matches)
		})
	}
}

func TestGetUniquePostfixFunc_BadSetup(t *testing.T) {
	tests := []struct {
		name                  string
		filepattern           string
		prefix                string
		expectError           bool
		expectPostFixFunction bool
	}{

		{
			name:                  "Filepattern is empty",
			filepattern:           "", // This is the error
			prefix:                "gl_config_",
			expectError:           true,
			expectPostFixFunction: false,
		},
		{
			name:                  "prefix is empty",
			filepattern:           "gl_config_*.json",
			prefix:                "",
			expectError:           true,
			expectPostFixFunction: false,
		},
		{
			name:                  "prefix should not compile",
			filepattern:           "gl_config_*.json",
			prefix:                "[[[[",
			expectError:           true,
			expectPostFixFunction: false,
		},
		{
			name:                  "Happy path",
			filepattern:           "gl_config_*.json",
			prefix:                "gl_config_",
			expectError:           false,
			expectPostFixFunction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postfixFunc, err := getUniquePostfixFunc(tt.filepattern, tt.prefix)

			if tt.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			if tt.expectPostFixFunction {
				assert.NotNil(t, postfixFunc)
			} else {
				assert.Nil(t, postfixFunc)
			}
		})
	}
}

func TestGetUniquePostfixFunc_FindPostfix(t *testing.T) {

	filepattern := "./test/gl_config_*.json"
	prefix := "gl_config_"

	postfixFunc, err := getUniquePostfixFunc(filepattern, prefix)
	assert.Nil(t, err)
	assert.NotNil(t, postfixFunc)

	postfix, err := postfixFunc()
	assert.Nil(t, err)

	shouldExist := []string{"tmp1", "tmp2", "tmp4"}
	shouldNotExist := []string{"tmp3", "tmp5"}

	for _, file := range shouldExist {
		assert.Contains(t, postfix, file)
	}

	for _, file := range shouldNotExist {
		assert.NotContains(t, postfix, file)
	}

}

func TestArchiveFormHandler(t *testing.T) {

	handlerToTest := archiveFormHandler

	// Define test scenarios
	tests := []struct {
		name        string
		w           *webbAppState
		form        string
		headline    string
		redirect    string
		postFix     postfixFunction
		queryParams map[string]string

		expectedResponseBody string
		expectedResponseCode int
	}{
		{
			name:     "w is nil",
			w:        nil, // This is the error
			form:     "submit",
			headline: "headline",
			redirect: "anypath",
			postFix: func() (map[string]bool, error) {
				return map[string]bool{"tmp1": true, "tmp2": true}, nil
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "form is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "gl_config_",
				},
			},
			form:     "", // This is the error
			headline: "headline",
			redirect: "anypath",
			postFix: func() (map[string]bool, error) {
				return map[string]bool{"tmp1": true, "tmp2": true}, nil
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "headline is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "gl_config_",
				},
			},
			form:     "submit",
			headline: "", // This is the error
			redirect: "anypath",
			postFix: func() (map[string]bool, error) {
				return map[string]bool{"tmp1": true, "tmp2": true}, nil
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "redirect is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "gl_config_",
				},
			},
			form:     "submit",
			headline: "headline",
			redirect: "", // This is the error
			postFix: func() (map[string]bool, error) {
				return map[string]bool{"tmp1": true, "tmp2": true}, nil
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "prefix is empty",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "", // This is the error
				},
			},
			form:     "submit",
			headline: "headline",
			redirect: "anypath",
			postFix: func() (map[string]bool, error) {
				return map[string]bool{"tmp1": true, "tmp2": true}, nil
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "getPostfix is nil",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "gl_config_",
				},
			},
			form:     "submit",
			headline: "headline",
			redirect: "anypath",
			postFix:  nil, // This is the error
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "getPostFix fails",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "gl_config_",
				},
			},
			form:     "submit",
			headline: "headline",
			redirect: "anypath",
			postFix: func() (map[string]bool, error) {
				return nil, fmt.Errorf("error") // This is the error
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "Internal error",
			expectedResponseCode: http.StatusInternalServerError,
		},
		{
			name: "basic success",
			w: &webbAppState{
				config: &mockConfig{},
				areas: []area{
					{
						Objects: []inputObject{},
					},
				},
				webMisc: misc{
					prefix: "gl_config_",
				},
			},
			form:     "submit",
			headline: "headline",
			redirect: "anypath",
			postFix: func() (map[string]bool, error) {
				return map[string]bool{"tmp1": true, "tmp2": true}, nil
			},
			queryParams: map[string]string{
				"redirect": "",
			},
			expectedResponseBody: "",
			expectedResponseCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions

			gin.SetMode(gin.ReleaseMode)
			ginRouter := gin.New()
			ginRouter.Use(sloggin.New(logger))
			ginRouter.Use(gin.Recovery())

			ginRouter.SetFuncMap(template.FuncMap{
				"findEndVarInString": findEndVarInString,
			})

			ginRouter.LoadHTMLGlob("templates/*")
			ginRouter.Static("/static", "./static")

			testFunc := handlerToTest(tt.w, tt.form, tt.headline, tt.redirect, tt.postFix, 0)

			ginRouter.GET("/test", testFunc)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedResponseCode, w.Code, "Expected response code to match")
			if tt.expectedResponseBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedResponseBody, "Expected response body to contain")
			}
		})
	}
}

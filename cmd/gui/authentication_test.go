package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	sloggin "github.com/samber/slog-gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocking the logger and session store
type MockStore struct {
	mock.Mock
	sessions.Store
}

func (s *MockStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	args := s.Called(r, name)
	return args.Get(0).(*sessions.Session), args.Error(1)
}
func (s *MockStore) New(r *http.Request, name string) (*sessions.Session, error) {
	args := s.Called(r, name)
	return args.Get(0).(*sessions.Session), args.Error(1)
}
func (s *MockStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	args := s.Called(r, w, session)
	return args.Error(0)
}

func TestAuthRequiredFunc_TestSessions(t *testing.T) {
	// Initialize Gin engine and middleware
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(sloggin.New(logger))
	ginRouter.Use(gin.Recovery())

	store := &MockStore{}
	session := sessions.NewSession(store, "session-name")
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   10 * 60, // 10 minutes
		HttpOnly: true,
	}

	store.On("Get", mock.Anything, "session-name").Return(session, nil)
	store.On("New", mock.Anything, "session-name").Return(session, nil)
	store.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Creating an instance of the application state
	appState := &appState{
		lastActivity:    time.Now().Add(-30 * time.Minute), // simulate last activity 30 minutes ago
		activeSessionID: "validSessionID",
	}

	ginRouter.Use(func(c *gin.Context) {
		c.Set("session", session)
	})

	ginRouter.GET("/test", authRequiredFunc(appState))

	// Define test scenarios
	tests := []struct {
		name                    string
		setup                   func()
		expectCode              int
		expectBody              string
		expectedActiveSessionID string
		expectedRedirect        string
	}{
		{
			name: "Session Timeout 1 - everything empty",
			setup: func() {
				appState.lastActivity = time.Now().Add(-30 * time.Minute) // Set last activity to simulate timeout
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
		{
			name: "Session Timeout 2 - valid user session",
			setup: func() {
				appState.lastActivity = time.Now().Add(-30 * time.Minute) // Set last activity to simulate timeout
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "validSessionID"
				appState.activeSessionID = "validSessionID"
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
		{
			name: "Session Timeout 3 - invalid user session",
			setup: func() {
				appState.lastActivity = time.Now().Add(-30 * time.Minute) // Set last activity to simulate timeout
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.activeSessionID = "validSessionID"
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
		{
			name: "Session Timeout 4 - empty user",
			setup: func() {
				appState.lastActivity = time.Now().Add(-30 * time.Minute) // Set last activity to simulate timeout
				session.Values["user"] = nil
				appState.activeSessionID = "validSessionID"
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
		{
			name: "Session Timeout 5 - No active session",
			setup: func() {
				appState.lastActivity = time.Now().Add(-30 * time.Minute) // Set last activity to simulate timeout
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "inactiveSessionID"
				appState.activeSessionID = ""
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
		{
			name: "Invalid Session 1 - empty user empty session",
			setup: func() {
				session.Values["user"] = nil // Simulate an invalid user session
				appState.activeSessionID = "validSessionID"
				appState.lastActivity = time.Now()
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "/login",
		},
		{
			name: "Invalid Session 2 - user set invalid session ID",
			setup: func() {
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.activeSessionID = "validSessionID"
				appState.lastActivity = time.Now()
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "/login",
		},
		{
			name: "Invalid Session 3  empty user invalid session ID",
			setup: func() {
				session.Values["user"] = nil
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.activeSessionID = "validSessionID"
				appState.lastActivity = time.Now()
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "/login",
		},
		{
			name: "Invalid Session 4 - empty session ID",
			setup: func() {
				session.Values["user"] = "user"
				appState.activeSessionID = "validSessionID"
				appState.lastActivity = time.Now()
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "/login",
		},
		{
			name: "Valid Session 1 ",
			setup: func() {
				session.Values["user"] = "user1"
				session.Values["customSessionID"] = "validSessionID"
				appState.activeSessionID = "validSessionID"
				appState.lastActivity = time.Now()
			},
			expectCode:              http.StatusOK,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "",
		},
		{
			name: "Valid Session 2 ",
			setup: func() {
				session.Values["user"] = "user2"
				session.Values["customSessionID"] = "validSessionID2"
				appState.activeSessionID = "validSessionID2"
				appState.lastActivity = time.Now()
			},
			expectCode:              http.StatusOK,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID2",
			expectedRedirect:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			session.Values = make(map[interface{}]interface{})
			tt.setup()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code, "Expected response code to match")
			if tt.expectBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedActiveSessionID, appState.activeSessionID, "Expected active session ID to match")
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")
		})
	}
}

func TestAuthRequiredFunc_TestFailedSessionSave(t *testing.T) {

	// Initialize Gin engine and middleware
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(sloggin.New(logger))
	ginRouter.Use(gin.Recovery())

	store := &MockStore{}
	session := sessions.NewSession(store, "session-name")
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   10 * 60, // 10 minutes
		HttpOnly: true,
	}

	store.On("Get", mock.Anything, "session-name").Return(session, nil)
	store.On("New", mock.Anything, "session-name").Return(session, nil)
	store.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("Error saving session"))

	// Creating an instance of the application state
	appState := &appState{
		lastActivity:    time.Now().Add(-30 * time.Minute), // simulate last activity 30 minutes ago
		activeSessionID: "validSessionID",
	}

	ginRouter.Use(func(c *gin.Context) {
		c.Set("session", session)
	})

	ginRouter.GET("/test", authRequiredFunc(appState))

	// Define test scenarios
	tests := []struct {
		name                    string
		setup                   func()
		expectCode              int
		expectBody              string
		expectedActiveSessionID string
		expectedRedirect        string
	}{
		{
			name: "Error Saving Session",
			setup: func() {
				appState.lastActivity = time.Now()
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "validSessionID"
				appState.activeSessionID = "validSessionID"
			},
			expectCode:              http.StatusInternalServerError,
			expectBody:              "Failed to save session",
			expectedActiveSessionID: "",
			expectedRedirect:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			session.Values = make(map[interface{}]interface{})
			tt.setup()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code, "Expected response code to match")
			if tt.expectBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedActiveSessionID, appState.activeSessionID, "Expected active session ID to match")
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")
		})
	}
}

func TestLoginPOSTHandlerFunc_Login(t *testing.T) {
	// Initialize Gin engine and middleware
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(sloggin.New(logger))
	ginRouter.Use(gin.Recovery())

	store := &MockStore{}
	session := sessions.NewSession(store, "session-name")
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   10 * 60, // 10 minutes
		HttpOnly: true,
	}

	store.On("Get", mock.Anything, "session-name").Return(session, nil)
	store.On("New", mock.Anything, "session-name").Return(session, nil)
	store.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Creating an instance of the application state
	appState := &appState{
		lastActivity:                 time.Now().Add(-30 * time.Minute), // simulate last activity 30 minutes ago
		activeSessionID:              "validSessionID",
		failedAuthenticationRequests: 0,
	}

	testSecret := "4n0th3r53cr3t"
	globalConfig.Secret = testSecret

	ginRouter.Use(func(c *gin.Context) {
		c.Set("session", session)
	})

	ginRouter.SetFuncMap(template.FuncMap{
		"findEndVarInString": findEndVarInString,
	})

	ginRouter.LoadHTMLGlob("templates/*")
	ginRouter.Static("/static", "./static")

	ctx, cancel := context.WithCancel(context.Background())
	ginRouter.POST("/login", loginPOSTHandlerFunc(ctx, cancel, appState))

	// Define test scenarios
	tests := []struct {
		name                    string
		secret                  string
		setup                   func()
		expectCode              int
		expectBody              string
		expectedActiveSessionID string
		expectedRedirect        string
		expectedFailedAttempts  int
	}{
		{
			name:   "Successful Login",
			secret: testSecret,
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "new",
			expectedRedirect:        "/mainmenu",
			expectedFailedAttempts:  0,
		},
		{
			name:   "Successful Login - clear failed attempts",
			secret: testSecret,
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 2
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "new",
			expectedRedirect:        "/mainmenu",
			expectedFailedAttempts:  0,
		},
		{
			name:   "Successful Login - good password, but already logged in",
			secret: testSecret,
			setup: func() {
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "validSessionID"
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "new",
			expectedRedirect:        "/mainmenu",
			expectedFailedAttempts:  0,
		},
		{
			name:   "Successful Login - After idle timeout",
			secret: testSecret,
			setup: func() {
				session.Values["user"] = "user"
				session.Values["customSessionID"] = ""
				appState.lastActivity = time.Now().Add(-30 * time.Minute) // Set last activity to simulate timeout
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "new",
			expectedRedirect:        "/mainmenu",
			expectedFailedAttempts:  0,
		},
		{
			name:   "Successful Login - but busy",
			secret: testSecret,
			setup: func() {
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusLocked,
			expectBody:              "Busy. Multiple logins not allowed",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "",
			expectedFailedAttempts:  0,
		},
		{
			name:   "Failed Login - bad credentials",
			secret: "badsecret",
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusBadRequest,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "",
			expectedFailedAttempts:  1,
		},
		{
			name:   "Failed Login - bad credentials, increase attempts",
			secret: "badsecret",
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 2
			},
			expectCode:              http.StatusBadRequest,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "",
			expectedFailedAttempts:  3,
		},
		{
			name:   "Failed Login - bad password, but already logged in",
			secret: "badsecret",
			setup: func() {
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "validSessionID"
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusBadRequest,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "",
			expectedFailedAttempts:  1,
		},
		{
			name:   "Failed Login - bad password, but another session active",
			secret: "badsecret",
			setup: func() {
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusBadRequest,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "",
			expectedFailedAttempts:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			session.Values = make(map[interface{}]interface{})
			tt.setup()

			w := httptest.NewRecorder()

			// Prepare form data
			formData := url.Values{}
			formData.Set("password", tt.secret)

			// Create a new POST request with form-encoded data
			req, _ := http.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code, "Expected response code to match")
			if tt.expectBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectBody, "Expected response body to contain")
			}
			switch tt.expectedActiveSessionID {
			case "new":
				assert.NotEqual(t, "", appState.activeSessionID, "Expected active session ID to be set")
			case "":
				assert.Equal(t, "", appState.activeSessionID, "Expected active session ID to be empty")
			default:
				assert.Equal(t, tt.expectedActiveSessionID, appState.activeSessionID, "Expected active session ID to match")
			}

			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")
			assert.Equal(t, tt.expectedFailedAttempts, appState.failedAuthenticationRequests, "Expected failed attempts to match")
		})
	}
}

func TestLoginPOSTHandlerFunc_Cancel(t *testing.T) {
	// Initialize Gin engine and middleware
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(sloggin.New(logger))
	ginRouter.Use(gin.Recovery())

	store := &MockStore{}
	session := sessions.NewSession(store, "session-name")
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   10 * 60, // 10 minutes
		HttpOnly: true,
	}

	store.On("Get", mock.Anything, "session-name").Return(session, nil)
	store.On("New", mock.Anything, "session-name").Return(session, nil)
	store.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("Error saving session"))

	// Creating an instance of the application state
	appState := &appState{
		lastActivity:                 time.Now().Add(-30 * time.Minute), // simulate last activity 30 minutes ago
		activeSessionID:              "validSessionID",
		failedAuthenticationRequests: 0,
	}

	testSecret := "4n0th3r53cr3t"
	globalConfig.FailedAuthenticationLimit = 3

	ginRouter.Use(func(c *gin.Context) {
		c.Set("session", session)
	})

	ginRouter.SetFuncMap(template.FuncMap{
		"findEndVarInString": findEndVarInString,
	})

	ginRouter.LoadHTMLGlob("templates/*")
	ginRouter.Static("/static", "./static")

	ctx := context.Background()
	cancelCalled := false
	cancel := func() {
		cancelCalled = true
	}
	ginRouter.POST("/login", loginPOSTHandlerFunc(ctx, cancel, appState))

	// Define test scenarios
	tests := []struct {
		name                    string
		globalConfigSecret      string
		secret                  string
		setup                   func()
		expectCode              int
		expectBody              string
		expectedActiveSessionID string
		expectedRedirect        string
		expectedFailedAttempts  int
		contextCancelled        bool
	}{
		{
			name:               "Too many failed attempts",
			globalConfigSecret: testSecret,
			secret:             "badsecret",
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 800
			},
			expectCode:       http.StatusLocked,
			expectBody:       "Too many failed authentications",
			contextCancelled: true,
		},
		{
			name:               "Empty secret",
			globalConfigSecret: "",
			secret:             testSecret,
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 0
			},
			expectCode:       http.StatusInternalServerError,
			expectBody:       "Internal error",
			contextCancelled: true,
		},
		{
			name:               "Cannot save session",
			globalConfigSecret: testSecret,
			secret:             testSecret,
			setup: func() {
				appState.lastActivity = time.Now() // Set last activity to simulate timeout
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 0
			},
			expectCode:       http.StatusInternalServerError,
			expectBody:       "Internal error",
			contextCancelled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			session.Values = make(map[interface{}]interface{})

			cancelCalled = false
			globalConfig.Secret = tt.globalConfigSecret

			tt.setup()

			w := httptest.NewRecorder()

			// Prepare form data
			formData := url.Values{}
			formData.Set("password", tt.secret)

			// Create a new POST request with form-encoded data
			req, _ := http.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code, "Expected response code to match")
			if tt.expectBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectBody, "Expected response body to contain")
			}
			if tt.contextCancelled {
				assert.True(t, cancelCalled, "Expected cancel function to be called")
			}
		})
	}
}

func TestLogoutHandlerFunc(t *testing.T) {
	// Initialize Gin engine and middleware
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(sloggin.New(logger))
	ginRouter.Use(gin.Recovery())

	store := &MockStore{}
	session := sessions.NewSession(store, "session-name")
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   10 * 60, // 10 minutes
		HttpOnly: true,
	}

	store.On("Get", mock.Anything, "session-name").Return(session, nil)
	store.On("New", mock.Anything, "session-name").Return(session, nil)
	store.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Creating an instance of the application state
	appState := &appState{
		lastActivity:    time.Now().Add(-30 * time.Minute), // simulate last activity 30 minutes ago
		activeSessionID: "validSessionID",
	}

	ginRouter.Use(func(c *gin.Context) {
		c.Set("session", session)
	})

	ginRouter.GET("/logout", logoutHandlerFunc(appState))

	// Define test scenarios
	tests := []struct {
		name                    string
		setup                   func()
		expectCode              int
		expectBody              string
		expectedActiveSessionID string
		expectedRedirect        string
	}{
		{
			name: "User logged in",
			setup: func() {
				appState.lastActivity = time.Now()
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "validSessionID"
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
		{
			name: "Different User logged in 1",
			setup: func() {
				appState.lastActivity = time.Now()
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "/login",
		},
		{
			name: "Different User logged in 2",
			setup: func() {
				appState.lastActivity = time.Now()
				appState.activeSessionID = "validSessionID"
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "validSessionID",
			expectedRedirect:        "/login",
		},
		{
			name: "no User logged in",
			setup: func() {
				appState.lastActivity = time.Now()
				session.Values["user"] = "user"
				session.Values["customSessionID"] = "INvalidSessionID"
				appState.activeSessionID = ""
				appState.failedAuthenticationRequests = 0
			},
			expectCode:              http.StatusFound,
			expectBody:              "",
			expectedActiveSessionID: "",
			expectedRedirect:        "/login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset session values and setup test conditions
			session.Values = make(map[interface{}]interface{})
			tt.setup()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/logout", nil)
			ginRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code, "Expected response code to match")
			if tt.expectBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectBody, "Expected response body to contain")
			}
			assert.Equal(t, tt.expectedActiveSessionID, appState.activeSessionID, "Expected active session ID to match")
			assert.Equal(t, tt.expectedRedirect, w.Header().Get("Location"), "Expected redirect location to match")
		})
	}
}

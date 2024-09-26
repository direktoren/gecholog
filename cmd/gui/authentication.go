package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

// ------------------------------ Authentication handlers and middleware ------------------------------

func authRequiredFunc(s *appState) gin.HandlerFunc {
	return func(c *gin.Context) {

		/*
			Either
				Timeout due to inactivity -> delete session -> redirect to login
				Invalid session -> delete session -> redirect to login
			Else
				Extend session
		*/

		session := c.MustGet("session").(*sessions.Session)

		if time.Since(s.lastActivity) > 10*time.Minute {
			logger.Debug("Timeout", slog.String("lastRequest", s.lastActivity.String()))

			delete(session.Values, "user")
			delete(session.Values, "customSessionID")
			session.Save(c.Request, c.Writer)
			s.m.Lock()
			s.activeSessionID = ""
			s.m.Unlock()

			c.Redirect(http.StatusFound, s.basePath+"/login")
			c.Abort()

			return
		}

		user := session.Values["user"]
		sessionID := session.Values["customSessionID"]

		// User is not set
		if user == nil || sessionID == nil || sessionID != s.activeSessionID {
			logger.Debug("invalid session", slog.Any("user", user), slog.Any("sessionID", sessionID), slog.Any("activeSessionID", s.activeSessionID))

			delete(session.Values, "user")
			delete(session.Values, "customSessionID")
			session.Save(c.Request, c.Writer)

			c.Redirect(http.StatusFound, s.basePath+"/login")
			c.Abort()
			return
		}

		session.Options.MaxAge = 10 * 60 // 10 minutes
		s.m.Lock()
		s.lastActivity = time.Now()
		s.m.Unlock()

		if err := session.Save(c.Request, c.Writer); err != nil {
			logger.Error("Failed to save session", slog.Any("error", err))

			s.m.Lock()
			s.activeSessionID = ""
			s.m.Unlock()

			c.String(http.StatusInternalServerError, "Failed to save session")
			return
		}
		c.Next()
	}
}

func loginPOSTHandlerFunc(ctx context.Context, cancelTheContext context.CancelFunc, s *appState) gin.HandlerFunc {
	return func(c *gin.Context) {

		/*
			Either
				Redirect already logged in users to menu
				Block if someone else is logged in
				Block failed authentications
				Die if too many failed authentications
			Else
				logged in -> redirect to menu
		*/

		if globalConfig.Secret == "" {
			logger.Error("Secret is empty")
			c.String(http.StatusInternalServerError, "Internal error")
			cancelTheContext()
			return
		}

		session := c.MustGet("session").(*sessions.Session)
		logger.Debug("before login processing",
			slog.String("currentActiveSession", s.activeSessionID),
			slog.String("lastActivity", s.lastActivity.String()),
			slog.Any("sessionID", session.Values["customSessionID"]),
		)

		// Retrieve password from the form submission
		password := c.PostForm("password")

		if password != globalConfig.Secret {
			s.m.Lock()
			s.failedAuthenticationRequests++
			s.m.Unlock()

			logger.Debug("Invalid credentials", slog.Int("failedAuthentications", s.failedAuthenticationRequests))

			// zero means infinite
			if globalConfig.FailedAuthenticationLimit > 0 && s.failedAuthenticationRequests > globalConfig.FailedAuthenticationLimit {
				logger.Error("Too many failed authentications", slog.Int("failedAuthentications", s.failedAuthenticationRequests))
				c.String(http.StatusLocked, "Too many failed authentications")
				cancelTheContext()
				return
			}
			c.HTML(http.StatusBadRequest, "login.html", gin.H{"Error": "Invalid credentials"})
			return
		}

		if s.activeSessionID != "" && s.activeSessionID != session.Values["customSessionID"] {
			if time.Since(s.lastActivity) < 10*time.Minute {
				logger.Debug("Busy. Multiple logins not allowed", slog.Any("time.Since(w.lastActivity)", time.Since(s.lastActivity)))
				c.String(http.StatusLocked, "Busy. Multiple logins not allowed")
				return
			}
			// timeout - Clear the session
			s.m.Lock()
			s.activeSessionID = ""
			s.m.Unlock()
		}

		// Successful login

		customSessionID := uuid.New().String()
		session.Values["user"] = "user"
		session.Values["customSessionID"] = customSessionID

		s.m.Lock()
		s.failedAuthenticationRequests = 0
		s.activeSessionID = customSessionID // Save the custom session ID globally
		s.lastActivity = time.Now()
		s.m.Unlock()

		logger.Debug("User authenticated", slog.String("customSessionID", customSessionID), slog.Any("c.Request.URL", c.Request.URL), slog.String("redirect", "mainmenu"))
		logger.Debug("details", slog.String("c.Request.Host", c.Request.Host), slog.String("c.Request.RequestURI", c.Request.RequestURI), slog.String("c.Request.URL.Path", c.Request.URL.Path), slog.String("c.Request.URL.RawQuery", c.Request.URL.RawQuery), slog.String("c.Request.Referrer", c.Request.Referer()))
		// Set base path
		referredURL, err := url.Parse(c.Request.Referer())
		if err != nil {
			logger.Error("Failed to parse referrer", slog.Any("error", err))
		}
		splits := strings.Split(referredURL.Path, "/")
		s.basePath = strings.Join(splits[:len(splits)-1], "/")
		logger.Debug("basePath", slog.String("basePath", s.basePath))

		if err := session.Save(c.Request, c.Writer); err != nil {
			logger.Error("Failed to save session", slog.Any("error", err))
			c.String(http.StatusInternalServerError, "Internal error")
			return
		}
		c.Redirect(http.StatusFound, s.basePath+"/mainmenu")
	}
}

func logoutHandlerFunc(s *appState) gin.HandlerFunc {
	return func(c *gin.Context) {

		/*
			Either
				Logged in user is logged out, and redirected to login
			Else
				Non-logged in user is redirected to login
		*/

		session := c.MustGet("session").(*sessions.Session)
		logger.Debug("before logout processing",
			slog.String("currentActiveSession", s.activeSessionID),
			slog.String("lastActivity", s.lastActivity.String()),
			slog.Any("sessionID", session.Values["customSessionID"]),
		)

		if s.activeSessionID == "" {
			c.Redirect(http.StatusFound, s.basePath+"/login")
			return
		}

		if s.activeSessionID != session.Values["customSessionID"] {
			c.Redirect(http.StatusFound, s.basePath+"/login")
			return
		}

		logger.Debug("User logged out")

		delete(session.Values, "user")
		delete(session.Values, "customSessionID")
		session.Save(c.Request, c.Writer)

		s.m.Lock()
		s.activeSessionID = ""
		s.m.Unlock()

		c.Redirect(http.StatusFound, s.basePath+"/login")

	}
}

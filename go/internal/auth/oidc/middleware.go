package oidc

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/session"
	"go.uber.org/zap"
)

// AuthMiddleware creates a middleware that checks for valid authentication
func AuthMiddleware(sessionStore session.Store, logger *zap.Logger, excludePaths []string) gin.HandlerFunc {
	// Create a map for faster lookup of excluded paths
	excludeMap := make(map[string]bool)
	for _, path := range excludePaths {
		excludeMap[path] = true
	}

	return func(c *gin.Context) {
		// Check if path is excluded
		if excludeMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get session ID from cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			logger.Debug("No session cookie found")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		// Retrieve user session
		var userSession UserSession
		err = sessionStore.Get(c.Request.Context(), sessionID, &userSession)
		if err != nil {
			logger.Debug("Failed to retrieve session",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired session",
			})
			c.Abort()
			return
		}

		// Check if token is expired
		if time.Now().After(userSession.ExpiresAt) {
			logger.Debug("Session expired",
				zap.String("user_id", userSession.ID),
				zap.Time("expired_at", userSession.ExpiresAt),
			)
			
			// Delete expired session
			if err := sessionStore.Delete(c.Request.Context(), sessionID); err != nil {
				logger.Warn("Failed to delete expired session", zap.Error(err), zap.String("session_id", sessionID))
			}
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Session expired",
			})
			c.Abort()
			return
		}

		// Add user information to context
		c.Set("user_id", userSession.ID)
		c.Set("user_email", userSession.Email)
		c.Set("user_name", userSession.Name)
		c.Set("user_session", &userSession)

		// Add user headers for proxy
		c.Request.Header.Set("X-User-ID", userSession.ID)
		c.Request.Header.Set("X-User-Email", userSession.Email)
		c.Request.Header.Set("X-User-Name", userSession.Name)

		logger.Debug("User authenticated",
			zap.String("user_id", userSession.ID),
			zap.String("email", userSession.Email),
		)

		c.Next()
	}
}

// OptionalAuthMiddleware is like AuthMiddleware but doesn't block unauthenticated requests
func OptionalAuthMiddleware(sessionStore session.Store, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID from cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil || sessionID == "" {
			// No session, but that's okay
			c.Next()
			return
		}

		// Try to retrieve user session
		var userSession UserSession
		err = sessionStore.Get(c.Request.Context(), sessionID, &userSession)
		if err != nil {
			// Session invalid, but continue anyway
			logger.Debug("Failed to retrieve optional session",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
			c.Next()
			return
		}

		// Check if token is expired
		if time.Now().After(userSession.ExpiresAt) {
			// Session expired, delete it but continue
			if err := sessionStore.Delete(c.Request.Context(), sessionID); err != nil {
				logger.Warn("Failed to delete expired session", zap.Error(err), zap.String("session_id", sessionID))
			}
			c.Next()
			return
		}

		// Add user information to context
		c.Set("user_id", userSession.ID)
		c.Set("user_email", userSession.Email)
		c.Set("user_name", userSession.Name)
		c.Set("user_session", &userSession)
		c.Set("authenticated", true)

		// Add user headers for proxy
		c.Request.Header.Set("X-User-ID", userSession.ID)
		c.Request.Header.Set("X-User-Email", userSession.Email)
		c.Request.Header.Set("X-User-Name", userSession.Name)

		c.Next()
	}
}
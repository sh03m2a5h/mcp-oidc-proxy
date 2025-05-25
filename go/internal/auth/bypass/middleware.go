package bypass

import (
	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"go.uber.org/zap"
)

// Default mock user values for bypass mode
const (
	DefaultUserID    = "bypass-user"
	DefaultUserEmail = "bypass@example.com"
	DefaultUserName  = "Bypass User"
)

// AuthMiddleware creates a middleware that bypasses authentication
func AuthMiddleware(logger *zap.Logger, headerConfig *config.HeadersConfig) gin.HandlerFunc {
	// Use default header names if not configured
	userIDHeader := headerConfig.UserID
	if userIDHeader == "" {
		userIDHeader = "X-User-ID"
	}
	userEmailHeader := headerConfig.UserEmail
	if userEmailHeader == "" {
		userEmailHeader = "X-User-Email"
	}
	userNameHeader := headerConfig.UserName
	if userNameHeader == "" {
		userNameHeader = "X-User-Name"
	}
	
	return func(c *gin.Context) {
		// In bypass mode, set mock user headers using configured header names
		c.Request.Header.Set(userIDHeader, DefaultUserID)
		c.Request.Header.Set(userEmailHeader, DefaultUserEmail)
		c.Request.Header.Set(userNameHeader, DefaultUserName)
		
		// Set context values for handlers
		c.Set("user_id", DefaultUserID)
		c.Set("user_email", DefaultUserEmail)
		c.Set("user_name", DefaultUserName)
		
		logger.Debug("Bypass auth mode - setting mock user headers",
			zap.String("user_id", DefaultUserID),
			zap.String("user_email", DefaultUserEmail),
			zap.String("user_name", DefaultUserName),
		)
		
		c.Next()
	}
}

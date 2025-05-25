package bypass

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware creates a middleware that bypasses authentication
func AuthMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// In bypass mode, set mock user headers
		c.Request.Header.Set("X-User-ID", "bypass-user")
		c.Request.Header.Set("X-User-Email", "bypass@example.com")
		c.Request.Header.Set("X-User-Name", "Bypass User")
		
		// Set context values for handlers
		c.Set("user_id", "bypass-user")
		c.Set("user_email", "bypass@example.com")
		c.Set("user_name", "Bypass User")
		
		logger.Debug("Bypass auth mode - setting mock user headers")
		
		c.Next()
	}
}

package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/oidc"
	"go.uber.org/zap"
)

// StructuredLoggingMiddleware creates a middleware that logs requests with structured data
func StructuredLoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Get additional context from the request context
		userID := param.Keys["user_id"]
		userEmail := param.Keys["user_email"]
		requestID := param.Keys["request_id"]

		// Build structured log fields
		fields := []zap.Field{
			zap.Time("timestamp", param.TimeStamp),
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.String("query", param.Request.URL.RawQuery),
			zap.String("ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.Int("body_size", param.BodySize),
		}

		// Add request ID if available
		if requestID != nil {
			if id, ok := requestID.(string); ok {
				fields = append(fields, zap.String("request_id", id))
			}
		}

		// Add user context if available
		if userID != nil {
			if id, ok := userID.(string); ok {
				fields = append(fields, zap.String("user_id", id))
			}
		}
		if userEmail != nil {
			if email, ok := userEmail.(string); ok {
				fields = append(fields, zap.String("user_email", email))
			}
		}

		// Add error information if present
		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		// Log based on status code level
		if param.StatusCode >= 500 {
			logger.Error("HTTP Request", fields...)
		} else if param.StatusCode >= 400 {
			logger.Warn("HTTP Request", fields...)
		} else {
			logger.Info("HTTP Request", fields...)
		}

		// Return empty string as we handle logging ourselves
		return ""
	})
}

// RequestContextMiddleware adds request context information for logging
// and transfers session data from Gin context to standard HTTP context
func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Before processing request, transfer session from Gin context to HTTP context
		if userSession, exists := c.Get("user_session"); exists {
			if sess, ok := userSession.(*oidc.UserSession); ok {
				// Add session to the HTTP request context using typed key
				ctx := context.WithValue(c.Request.Context(), oidc.SessionContextKey{}, sess)
				c.Request = c.Request.WithContext(ctx)
			}
		}

		// Process request
		c.Next()

		// Extract user information from context (set by auth middleware)
		if userID, exists := c.Get("user_id"); exists {
			c.Set("user_id", userID)
		}
		if userEmail, exists := c.Get("user_email"); exists {
			c.Set("user_email", userEmail)
		}
		if requestID, exists := c.Get("request_id"); exists {
			c.Set("request_id", requestID)
		}
	}
}
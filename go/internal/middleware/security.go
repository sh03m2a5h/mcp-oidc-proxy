package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Enable XSS protection (legacy browsers)
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions policy (replace Feature-Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// Process request
		c.Next()
	}
}
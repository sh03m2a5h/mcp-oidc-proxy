package middleware

import (
	"github.com/gin-gonic/gin"
)

// Security header constants
const (
	HeaderXFrameOptions       = "X-Frame-Options"
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	HeaderXXSSProtection      = "X-XSS-Protection"
	HeaderReferrerPolicy      = "Referrer-Policy"
	HeaderPermissionsPolicy   = "Permissions-Policy"
	HeaderContentSecurityPolicy = "Content-Security-Policy"
)

// Default security header values
var DefaultSecurityHeaders = map[string]string{
	HeaderXFrameOptions:       "DENY",
	HeaderXContentTypeOptions: "nosniff",
	HeaderXXSSProtection:      "1; mode=block",
	HeaderReferrerPolicy:      "strict-origin-when-cross-origin",
	HeaderPermissionsPolicy:   "geolocation=(), microphone=(), camera=()",
	// Basic CSP that allows self-hosted resources only
	HeaderContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'",
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Apply all security headers
		for header, value := range DefaultSecurityHeaders {
			c.Header(header, value)
		}
		
		// Process request
		c.Next()
	}
}
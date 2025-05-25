package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router with security middleware
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Create test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify security headers using constants from implementation
	for header, expectedValue := range DefaultSecurityHeaders {
		t.Run(header, func(t *testing.T) {
			actualValue := w.Header().Get(header)
			assert.Equal(t, expectedValue, actualValue, "Header %s should have value %s", header, expectedValue)
		})
	}
}
package bypass

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAuthMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	tests := []struct {
		name           string
		headerConfig   *config.HeadersConfig
		expectedUserID string
		expectedEmail  string
		expectedName   string
	}{
		{
			name: "Default headers",
			headerConfig: &config.HeadersConfig{
				UserID:    "",
				UserEmail: "",
				UserName:  "",
			},
			expectedUserID: "bypass-user",
			expectedEmail:  "bypass@example.com",
			expectedName:   "Bypass User",
		},
		{
			name: "Custom headers",
			headerConfig: &config.HeadersConfig{
				UserID:    "X-Custom-User-ID",
				UserEmail: "X-Custom-Email",
				UserName:  "X-Custom-Name",
			},
			expectedUserID: "bypass-user",
			expectedEmail:  "bypass@example.com",
			expectedName:   "Bypass User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()
			router.Use(AuthMiddleware(logger, tt.headerConfig))
			
			// Add test endpoint
			router.GET("/test", func(c *gin.Context) {
				// Get user info from context
				userID, _ := c.Get("user_id")
				userEmail, _ := c.Get("user_email")
				userName, _ := c.Get("user_name")
				
				assert.Equal(t, tt.expectedUserID, userID)
				assert.Equal(t, tt.expectedEmail, userEmail)
				assert.Equal(t, tt.expectedName, userName)
				
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			
			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			recorder := httptest.NewRecorder()
			
			// Serve request
			router.ServeHTTP(recorder, req)
			
			// Verify response
			assert.Equal(t, http.StatusOK, recorder.Code)
			
			// Verify headers were set
			expectedHeaders := map[string]string{
				"X-User-ID":    tt.expectedUserID,
				"X-User-Email": tt.expectedEmail,
				"X-User-Name":  tt.expectedName,
			}
			
			// Use custom headers if provided
			if tt.headerConfig.UserID != "" {
				expectedHeaders[tt.headerConfig.UserID] = tt.expectedUserID
				delete(expectedHeaders, "X-User-ID")
			}
			if tt.headerConfig.UserEmail != "" {
				expectedHeaders[tt.headerConfig.UserEmail] = tt.expectedEmail
				delete(expectedHeaders, "X-User-Email")
			}
			if tt.headerConfig.UserName != "" {
				expectedHeaders[tt.headerConfig.UserName] = tt.expectedName
				delete(expectedHeaders, "X-User-Name")
			}
			
			// Check that headers were set on the request
			for header, value := range expectedHeaders {
				assert.Equal(t, value, req.Header.Get(header), "Header %s should be set", header)
			}
		})
	}
}

func TestAuthMiddlewareHeaderForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	
	headerConfig := &config.HeadersConfig{
		UserID:    "X-Custom-ID",
		UserEmail: "X-Custom-Mail",
		UserName:  "X-Custom-Name",
	}
	
	// Create test router
	router := gin.New()
	router.Use(AuthMiddleware(logger, headerConfig))
	
	// Add endpoint that echoes headers
	router.GET("/echo", func(c *gin.Context) {
		headers := make(map[string]string)
		headers["X-Custom-ID"] = c.Request.Header.Get("X-Custom-ID")
		headers["X-Custom-Mail"] = c.Request.Header.Get("X-Custom-Mail")
		headers["X-Custom-Name"] = c.Request.Header.Get("X-Custom-Name")
		
		c.JSON(http.StatusOK, headers)
	})
	
	// Create test request
	req := httptest.NewRequest("GET", "/echo", nil)
	recorder := httptest.NewRecorder()
	
	// Serve request
	router.ServeHTTP(recorder, req)
	
	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)
	
	// Verify custom headers were used
	assert.Equal(t, "bypass-user", req.Header.Get("X-Custom-ID"))
	assert.Equal(t, "bypass@example.com", req.Header.Get("X-Custom-Mail"))
	assert.Equal(t, "Bypass User", req.Header.Get("X-Custom-Name"))
}
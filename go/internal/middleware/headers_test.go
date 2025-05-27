package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/oidc"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestHeaderInjector_InjectStaticHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		Custom: map[string]string{
			"X-Service-Name":    "mcp-oidc-proxy",
			"X-Service-Version": "1.0.0",
			"X-Environment":     "test",
		},
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	injector.injectStaticHeaders(req)
	
	assert.Equal(t, "mcp-oidc-proxy", req.Header.Get("X-Service-Name"))
	assert.Equal(t, "1.0.0", req.Header.Get("X-Service-Version"))
	assert.Equal(t, "test", req.Header.Get("X-Environment"))
}

func TestHeaderInjector_InjectDynamicHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		Dynamic: config.DynamicHeaders{
			Timestamp: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Request-Timestamp",
				Format:     "rfc3339",
			},
			RequestID: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Request-ID",
			},
			ClientIP: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Client-IP",
			},
			UserAgent: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-User-Agent",
			},
			CorrelationID: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Correlation-ID",
			},
		},
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "test-agent/1.0")
	req.RemoteAddr = "192.168.1.100:12345"
	
	injector.injectDynamicHeaders(req, nil)
	
	// Check timestamp header
	timestamp := req.Header.Get("X-Request-Timestamp")
	assert.NotEmpty(t, timestamp)
	_, err := time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err, "Timestamp should be valid RFC3339")
	
	// Check request ID header
	requestID := req.Header.Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
	assert.True(t, strings.HasPrefix(requestID, "req_"))
	
	// Check client IP header
	clientIP := req.Header.Get("X-Client-IP")
	assert.Equal(t, "192.168.1.100", clientIP)
	
	// Check user agent header
	userAgent := req.Header.Get("X-User-Agent")
	assert.Equal(t, "test-agent/1.0", userAgent)
	
	// Check correlation ID header
	correlationID := req.Header.Get("X-Correlation-ID")
	assert.NotEmpty(t, correlationID)
	assert.True(t, strings.HasPrefix(correlationID, "corr_"))
}

func TestHeaderInjector_InjectUserHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		UserID:     "X-User-ID",
		UserEmail:  "X-User-Email",
		UserName:   "X-User-Name",
		UserGroups: "X-User-Groups",
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	sess := &oidc.UserSession{
		ID:    "user123",
		Email: "test@example.com",
		Name:  "Test User",
		Claims: map[string]interface{}{
			"groups": []string{"admin", "users"},
		},
	}
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	injector.injectUserHeaders(req, sess)
	
	assert.Equal(t, "user123", req.Header.Get("X-User-ID"))
	assert.Equal(t, "test@example.com", req.Header.Get("X-User-Email"))
	assert.Equal(t, "Test User", req.Header.Get("X-User-Name"))
	assert.Equal(t, "admin,users", req.Header.Get("X-User-Groups"))
}

func TestHeaderInjector_InjectSessionIDHeader(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		Dynamic: config.DynamicHeaders{
			SessionID: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Session-ID",
			},
		},
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	sess := &oidc.UserSession{
		ID: "user123",
	}
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	injector.injectDynamicHeaders(req, sess)
	
	assert.Equal(t, "user123", req.Header.Get("X-Session-ID"))
}

func TestHeaderInjector_PreserveExistingCorrelationID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		Dynamic: config.DynamicHeaders{
			CorrelationID: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Correlation-ID",
			},
		},
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	existingCorrelationID := "existing-correlation-id"
	req.Header.Set("X-Correlation-ID", existingCorrelationID)
	
	injector.injectDynamicHeaders(req, nil)
	
	// Should preserve existing correlation ID
	assert.Equal(t, existingCorrelationID, req.Header.Get("X-Correlation-ID"))
}

func TestHeaderInjector_FormatTimestamp(t *testing.T) {
	logger := zaptest.NewLogger(t)
	injector := NewHeaderInjector(&config.HeadersConfig{}, logger)
	
	tests := []struct {
		format   string
		validate func(string) bool
	}{
		{
			format: "unix",
			validate: func(s string) bool {
				if len(s) != 10 {
					return false
				}
				for _, r := range s {
					if r < '0' || r > '9' {
						return false
					}
				}
				return true
			},
		},
		{
			format: "rfc3339",
			validate: func(s string) bool {
				_, err := time.Parse(time.RFC3339, s)
				return err == nil
			},
		},
		{
			format: "2006-01-02",
			validate: func(s string) bool {
				_, err := time.Parse("2006-01-02", s)
				return err == nil
			},
		},
		{
			format: "",
			validate: func(s string) bool {
				_, err := time.Parse(time.RFC3339, s)
				return err == nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := injector.formatTimestamp(tt.format)
			assert.True(t, tt.validate(result), "Format %q produced invalid result: %s", tt.format, result)
		})
	}
}

func TestHeaderInjector_GetClientIP(t *testing.T) {
	logger := zaptest.NewLogger(t)
	injector := NewHeaderInjector(&config.HeadersConfig{}, logger)
	
	tests := []struct {
		name     string
		headers  map[string]string
		remoteAddr string
		expected string
	}{
		{
			name: "X-Forwarded-For single IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
			},
			expected: "192.168.1.100",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100, 10.0.0.1, 172.16.0.1",
			},
			expected: "192.168.1.100",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.195",
			},
			expected: "203.0.113.195",
		},
		{
			name: "CF-Connecting-IP",
			headers: map[string]string{
				"CF-Connecting-IP": "198.51.100.178",
			},
			expected: "198.51.100.178",
		},
		{
			name:       "RemoteAddr",
			remoteAddr: "192.168.1.100:12345",
			expected:   "192.168.1.100",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.100",
			expected:   "192.168.1.100",
		},
		{
			name:       "No IP information",
			remoteAddr: "",
			expected:   "unknown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr
			
			result := injector.getClientIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHeaderInjector_Middleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		UserID: "X-User-ID",
		Custom: map[string]string{
			"X-Service": "test",
		},
		Dynamic: config.DynamicHeaders{
			RequestID: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "X-Request-ID",
			},
		},
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	// Create a test handler that captures the request
	var capturedRequest *http.Request
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	})
	
	// Create middleware
	middleware := injector.Middleware(testHandler)
	
	// Create request with session in context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	sess := &oidc.UserSession{
		ID: "user123",
	}
	ctx := context.WithValue(req.Context(), oidc.SessionContextKey{}, sess)
	req = req.WithContext(ctx)
	
	// Execute middleware
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	
	// Verify headers were injected
	require.NotNil(t, capturedRequest)
	assert.Equal(t, "user123", capturedRequest.Header.Get("X-User-ID"))
	assert.Equal(t, "test", capturedRequest.Header.Get("X-Service"))
	assert.NotEmpty(t, capturedRequest.Header.Get("X-Request-ID"))
}

func TestHeaderInjector_DisabledHeaders(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	headerConfig := &config.HeadersConfig{
		Dynamic: config.DynamicHeaders{
			RequestID: config.HeaderTemplate{
				Enabled:    false, // Disabled
				HeaderName: "X-Request-ID",
			},
			Timestamp: config.HeaderTemplate{
				Enabled:    true,
				HeaderName: "", // Empty header name
			},
		},
	}
	
	injector := NewHeaderInjector(headerConfig, logger)
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	injector.injectDynamicHeaders(req, nil)
	
	// Should not inject disabled or empty header name
	assert.Empty(t, req.Header.Get("X-Request-ID"))
	assert.Empty(t, req.Header.Get("X-Timestamp"))
}

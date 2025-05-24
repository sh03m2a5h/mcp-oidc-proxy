package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNew(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "Empty target host",
			config: &Config{
				TargetHost: "",
				TargetPort: 3000,
			},
			expectError: true,
			errorMsg:    "target host is required",
		},
		{
			name: "Invalid target port",
			config: &Config{
				TargetHost: "localhost",
				TargetPort: -1,
			},
			expectError: true,
			errorMsg:    "target port must be positive",
		},
		{
			name: "Valid config",
			config: &Config{
				TargetHost:   "localhost",
				TargetPort:   3000,
				TargetScheme: "http",
				Retry: RetryConfig{
					MaxAttempts: 3,
					Backoff:     100 * time.Millisecond,
				},
				CircuitBreaker: CircuitBreakerConfig{
					Threshold: 5,
					Timeout:   60 * time.Second,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := New(tt.config, logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, proxy)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, proxy)
				assert.Equal(t, "http://localhost:3000", proxy.Target().String())
			}
		})
	}
}

func TestProxy_ServeHTTP(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Header", "test")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	config := &Config{
		TargetHost:   "127.0.0.1",
		TargetPort:   8080, // Will be overridden
		TargetScheme: "http",
		Retry: RetryConfig{
			MaxAttempts: 1,
			Backoff:     10 * time.Millisecond,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Threshold: 3,
			Timeout:   1 * time.Second,
		},
	}

	proxy, err := New(config, logger)
	require.NoError(t, err)

	// Override target to point to test server
	backendURL, err := url.Parse(backend.URL)
	require.NoError(t, err)
	proxy.target = backendURL
	
	// Create new reverse proxy with correct target
	proxy.httputil = httputil.NewSingleHostReverseProxy(backendURL)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Test-Header", "test-value")

	recorder := httptest.NewRecorder()

	// Execute proxy request
	proxy.ServeHTTP(recorder, req)

	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "backend response", recorder.Body.String())
	assert.Equal(t, "test", recorder.Header().Get("X-Backend-Header"))
}

func TestProxy_Health(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name           string
		backendHandler http.HandlerFunc
		expectError    bool
	}{
		{
			name: "Healthy backend",
			backendHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}),
			expectError: false,
		},
		{
			name: "Unhealthy backend",
			backendHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := httptest.NewServer(tt.backendHandler)
			defer backend.Close()

			config := &Config{
				TargetHost:   "127.0.0.1",
				TargetPort:   8080,
				TargetScheme: "http",
			}

			proxy, err := New(config, logger)
			require.NoError(t, err)

			// Override target
			backendURL, err := url.Parse(backend.URL)
			require.NoError(t, err)
			proxy.target = backendURL

			// Test health check
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = proxy.Health(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
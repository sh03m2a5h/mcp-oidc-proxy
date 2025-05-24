package proxy

import (
	"context"
	"fmt"
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
	proxy.reverseProxy = httputil.NewSingleHostReverseProxy(backendURL)

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

func TestProxy_RetryBehavior(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name           string
		attempts       int
		responses      []int // Status codes for each attempt
		expectedCalls  int
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Success on first attempt",
			attempts:       3,
			responses:      []int{200},
			expectedCalls:  1,
			expectedStatus: 200,
			expectError:    false,
		},
		{
			name:           "Success after retry",
			attempts:       3,
			responses:      []int{500, 200},
			expectedCalls:  2,
			expectedStatus: 200,
			expectError:    false,
		},
		{
			name:           "All attempts fail",
			attempts:       3,
			responses:      []int{500, 500, 500},
			expectedCalls:  3,
			expectedStatus: 500,
			expectError:    true,
		},
		{
			name:           "Success on last attempt",
			attempts:       3,
			responses:      []int{500, 503, 200},
			expectedCalls:  3,
			expectedStatus: 200,
			expectError:    false,
		},
		{
			name:           "4xx errors not retried",
			attempts:       3,
			responses:      []int{404},
			expectedCalls:  1,
			expectedStatus: 404,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			
			// Create test backend that returns different status codes
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if callCount < len(tt.responses) {
					w.WriteHeader(tt.responses[callCount])
					w.Write([]byte(fmt.Sprintf("response %d", callCount)))
					callCount++
				} else {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("unexpected call"))
				}
			}))
			defer backend.Close()

			config := &Config{
				TargetHost:   "127.0.0.1",
				TargetPort:   8080,
				TargetScheme: "http",
				Retry: RetryConfig{
					MaxAttempts: tt.attempts,
					Backoff:     10 * time.Millisecond,
				},
				CircuitBreaker: CircuitBreakerConfig{
					Threshold: 10,
					Timeout:   1 * time.Second,
				},
			}

			proxy, err := New(config, logger)
			require.NoError(t, err)

			// Override target
			backendURL, err := url.Parse(backend.URL)
			require.NoError(t, err)
			proxy.target = backendURL
			proxy.reverseProxy = httputil.NewSingleHostReverseProxy(backendURL)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			// Execute proxy request
			proxy.ServeHTTP(recorder, req)

			// Verify results
			assert.Equal(t, tt.expectedCalls, callCount, "unexpected number of backend calls")
			assert.Equal(t, tt.expectedStatus, recorder.Code, "unexpected status code")
			
			// Verify error recording for circuit breaker
			if tt.expectError {
				assert.Equal(t, 1, proxy.circuitBreaker.Failures())
			} else {
				assert.Equal(t, 0, proxy.circuitBreaker.Failures())
			}
		})
	}
}
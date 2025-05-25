package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
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

	tests := []struct {
		name             string
		setupBackend     func() *httptest.Server
		setupRequest     func() *http.Request
		expectedStatus   int
		expectedBody     string
		verifyHeaders    func(t *testing.T, w *httptest.ResponseRecorder, backendReq *http.Request)
	}{
		{
			name: "Successful proxy with custom headers",
			setupBackend: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify custom headers were added
					assert.NotEmpty(t, r.Header.Get("X-Forwarded-Proto"))
					assert.NotEmpty(t, r.Header.Get("X-Forwarded-Host"))
					
					// Verify hop-by-hop headers were removed
					assert.Empty(t, r.Header.Get("Connection"))
					assert.Empty(t, r.Header.Get("Keep-Alive"))
					
					w.Header().Set("X-Backend-Header", "test")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("backend response"))
				}))
			},
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Test-Header", "test-value")
				req.Header.Set("Connection", "keep-alive") // Should be removed
				req.Header.Set("Keep-Alive", "timeout=5")  // Should be removed
				return req
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "backend response",
			verifyHeaders: func(t *testing.T, w *httptest.ResponseRecorder, backendReq *http.Request) {
				assert.Equal(t, "test", w.Header().Get("X-Backend-Header"))
			},
		},
		{
			name: "Backend error triggers error handler",
			setupBackend: func() *httptest.Server {
				// Create a backend that immediately closes connections
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Close connection to trigger error
					hj, ok := w.(http.Hijacker)
					if ok {
						conn, _, _ := hj.Hijack()
						conn.Close()
					}
				}))
			},
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			expectedStatus: http.StatusBadGateway,
			expectedBody:   "Bad Gateway",
			verifyHeaders: func(t *testing.T, w *httptest.ResponseRecorder, backendReq *http.Request) {
				// No additional verification needed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := tt.setupBackend()
			defer backend.Close()

			// Parse backend URL
			backendURL, err := url.Parse(backend.URL)
			require.NoError(t, err)
			
			// Update config to use test backend
			config := &Config{
				TargetHost:   backendURL.Hostname(),
				TargetPort:   func() int { 
					port, _ := strconv.Atoi(backendURL.Port())
					return port
				}(),
				TargetScheme: backendURL.Scheme,
				Retry: RetryConfig{
					MaxAttempts: 1,
					Backoff:     10 * time.Millisecond,
				},
				CircuitBreaker: CircuitBreakerConfig{
					Threshold: 3,
					Timeout:   1 * time.Second,
				},
			}

			// Create proxy with proper configuration
			proxy, err := New(config, logger)
			require.NoError(t, err)

			// Create test request
			req := tt.setupRequest()
			recorder := httptest.NewRecorder()

			// Execute proxy request
			proxy.ServeHTTP(recorder, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, recorder.Code)
			assert.Equal(t, tt.expectedBody, recorder.Body.String())
			
			// Run additional verifications
			tt.verifyHeaders(t, recorder, req)
		})
	}
}

func TestProxy_RequestBodyReplay(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name           string
		method         string
		body           string
		hasGetBody     bool
		backendFails   int // Number of times backend should fail before success
		expectedCalls  int
		expectedStatus int
	}{
		{
			name:           "POST with replayable body",
			method:         http.MethodPost,
			body:           `{"test": "data"}`,
			hasGetBody:     true,
			backendFails:   1,
			expectedCalls:  2, // Should retry once
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST without replayable body",
			method:         http.MethodPost,
			body:           `{"test": "data"}`,
			hasGetBody:     false,
			backendFails:   1,
			expectedCalls:  1, // Should NOT retry
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "GET request always retries",
			method:         http.MethodGet,
			body:           "",
			hasGetBody:     false,
			backendFails:   1,
			expectedCalls:  2, // Should retry
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			
			// Create test backend
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				
				// Fail for the first N attempts
				if callCount <= tt.backendFails {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("temporary error"))
					return
				}
				
				// Success after failures
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			}))
			defer backend.Close()

			// Parse backend URL
			backendURL, err := url.Parse(backend.URL)
			require.NoError(t, err)
			
			config := &Config{
				TargetHost:   backendURL.Hostname(),
				TargetPort:   func() int { 
					port, _ := strconv.Atoi(backendURL.Port())
					return port
				}(),
				TargetScheme: backendURL.Scheme,
				Retry: RetryConfig{
					MaxAttempts: 3,
					Backoff:     10 * time.Millisecond,
				},
				CircuitBreaker: CircuitBreakerConfig{
					Threshold: 10,
					Timeout:   1 * time.Second,
				},
			}

			proxy, err := New(config, logger)
			require.NoError(t, err)

			// Create test request
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
				
				// Simulate replayable body if needed
				if tt.hasGetBody {
					req.GetBody = func() (io.ReadCloser, error) {
						return io.NopCloser(strings.NewReader(tt.body)), nil
					}
				}
			} else {
				req = httptest.NewRequest(tt.method, "/test", nil)
			}
			
			recorder := httptest.NewRecorder()

			// Execute proxy request
			proxy.ServeHTTP(recorder, req)

			// Verify results
			assert.Equal(t, tt.expectedCalls, callCount, "unexpected number of backend calls")
			assert.Equal(t, tt.expectedStatus, recorder.Code, "unexpected status code")
		})
	}
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
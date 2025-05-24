package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		logger *zap.Logger
	}{
		{
			name:   "with default config",
			config: nil,
			logger: zap.NewNop(),
		},
		{
			name:   "with custom config",
			config: &Config{
				Host:         "127.0.0.1",
				Port:         9090,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			logger: zap.NewNop(),
		},
		{
			name:   "with nil logger",
			config: DefaultConfig(),
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.config, tt.logger)
			assert.NotNil(t, s)
			assert.NotNil(t, s.router)
			assert.NotNil(t, s.config)
			assert.NotNil(t, s.logger)
		})
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := New(DefaultConfig(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status)
	assert.NotEmpty(t, response.Version)
	assert.GreaterOrEqual(t, response.Uptime, int64(0))
	assert.Equal(t, "unknown", response.BackendStatus)
}

func TestServer_VersionEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := New(DefaultConfig(), zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response VersionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Version)
	assert.NotEmpty(t, response.GoVersion)
	assert.NotEmpty(t, response.Platform)
}

func TestServer_Shutdown(t *testing.T) {
	t.Run("shutdown without starting", func(t *testing.T) {
		s := New(DefaultConfig(), zap.NewNop())
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := s.Shutdown(ctx)
		assert.NoError(t, err)
	})
	
	t.Run("shutdown after starting", func(t *testing.T) {
		// Use a random port to avoid conflicts
		config := &Config{
			Host:         "127.0.0.1",
			Port:         0, // Let the system assign a port
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		
		s := New(config, zap.NewNop())
		
		// Start server in goroutine
		serverStarted := make(chan bool)
		serverError := make(chan error, 1)
		
		go func() {
			// Create a test listener to get the actual port
			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))
			if err != nil {
				serverError <- err
				return
			}
			defer listener.Close()
			
			// Update server with actual address
			s.httpServer = &http.Server{
				Handler:      s.router,
				ReadTimeout:  config.ReadTimeout,
				WriteTimeout: config.WriteTimeout,
				IdleTimeout:  config.IdleTimeout,
			}
			
			serverStarted <- true
			if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
				serverError <- err
			}
		}()
		
		// Wait for server to start or error
		select {
		case <-serverStarted:
			// Server started successfully
		case err := <-serverError:
			t.Fatalf("Failed to start server: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("Server start timeout")
		}
		
		// Test shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := s.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		assert.True(t, exists)
		assert.NotEmpty(t, requestID)
		c.String(http.StatusOK, "ok")
	})

	// Test without X-Request-ID header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// Test with X-Request-ID header
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-request-id", w.Header().Get("X-Request-ID"))
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		expectAllowed  bool
	}{
		{
			name:           "wildcard allows all",
			allowedOrigins: []string{"*"},
			requestOrigin:  "https://example.com",
			expectAllowed:  true,
		},
		{
			name:           "specific origin allowed",
			allowedOrigins: []string{"https://example.com", "https://test.com"},
			requestOrigin:  "https://example.com",
			expectAllowed:  true,
		},
		{
			name:           "origin not allowed",
			allowedOrigins: []string{"https://example.com"},
			requestOrigin:  "https://evil.com",
			expectAllowed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORSMiddleware(tt.allowedOrigins))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectAllowed {
				assert.Equal(t, tt.requestOrigin, w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}

	// Test OPTIONS request
	router := gin.New()
	router.Use(CORSMiddleware([]string{"*"}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
}

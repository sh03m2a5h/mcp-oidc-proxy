package proxy

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIsStreamingRequest(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "SSE request",
			headers: map[string]string{
				"Accept": "text/event-stream",
			},
			expected: true,
		},
		{
			name: "WebSocket request",
			headers: map[string]string{
				"Connection": "Upgrade",
				"Upgrade":    "websocket",
			},
			expected: true,
		},
		{
			name: "Regular HTTP request",
			headers: map[string]string{
				"Accept": "application/json",
			},
			expected: false,
		},
		{
			name: "Mixed accept with SSE",
			headers: map[string]string{
				"Accept": "text/html, text/event-stream, */*",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			
			result := isStreamingRequest(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSSEStreaming(t *testing.T) {
	// Create a test SSE server
	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "ResponseWriter must support flushing")
		
		// Send a few events
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: Event %d\n\n", i)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer sseServer.Close()
	
	// Parse test server URL
	serverURL, err := url.Parse(sseServer.URL)
	require.NoError(t, err)
	
	// Create proxy
	config := &Config{
		TargetHost:   serverURL.Hostname(),
		TargetPort:   func() int { 
			port, _ := strconv.Atoi(serverURL.Port())
			return port
		}(),
		TargetScheme: serverURL.Scheme,
		Retry: RetryConfig{
			MaxAttempts: 1,
			Backoff:     10 * time.Millisecond,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Threshold: 3,
			Timeout:   1 * time.Second,
		},
	}
	
	logger := zap.NewNop()
	proxy, err := New(config, logger)
	require.NoError(t, err)
	
	// Create test request
	req := httptest.NewRequest("GET", "/events", nil)
	req.Header.Set("Accept", "text/event-stream")
	
	// Record response
	recorder := httptest.NewRecorder()
	
	// Handle request
	proxy.ServeHTTP(recorder, req)
	
	// Verify response
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	
	// Parse SSE events
	scanner := bufio.NewScanner(strings.NewReader(recorder.Body.String()))
	events := []string{}
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			events = append(events, strings.TrimPrefix(line, "data: "))
		}
	}
	
	// Verify we received all events
	assert.Equal(t, 3, len(events))
	assert.Equal(t, "Event 0", events[0])
	assert.Equal(t, "Event 1", events[1])
	assert.Equal(t, "Event 2", events[2])
}

func TestStreamingWithAuthHeaders(t *testing.T) {
	// Create test server that verifies auth headers
	var receivedHeaders http.Header
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: test\n\n")
	}))
	defer testServer.Close()
	
	// Parse test server URL
	serverURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)
	
	// Create proxy
	config := &Config{
		TargetHost:   serverURL.Hostname(),
		TargetPort:   func() int { 
			port, _ := strconv.Atoi(serverURL.Port())
			return port
		}(),
		TargetScheme: serverURL.Scheme,
		Retry: RetryConfig{
			MaxAttempts: 1,
			Backoff:     10 * time.Millisecond,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Threshold: 3,
			Timeout:   1 * time.Second,
		},
	}
	
	logger := zap.NewNop()
	proxy, err := New(config, logger)
	require.NoError(t, err)
	
	// Create request with auth headers
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-User-ID", "test-user")
	req.Header.Set("X-User-Email", "test@example.com")
	
	// Handle request
	recorder := httptest.NewRecorder()
	proxy.ServeHTTP(recorder, req)
	
	// Verify auth headers were forwarded
	assert.Equal(t, "test-user", receivedHeaders.Get("X-User-ID"))
	assert.Equal(t, "test@example.com", receivedHeaders.Get("X-User-Email"))
}

func TestStreamingMetrics(t *testing.T) {
	// TODO: Add metrics verification once metrics are properly mocked
	t.Skip("Metrics testing requires proper mocking")
}
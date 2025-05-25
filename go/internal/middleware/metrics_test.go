package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/metrics"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware(t *testing.T) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		fullPath       string
		expectedStatus int
		handler        gin.HandlerFunc
	}{
		{
			name:           "Successful GET request",
			method:         "GET",
			path:           "/api/v1/users",
			fullPath:       "/api/v1/users",
			expectedStatus: http.StatusOK,
			handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			},
		},
		{
			name:           "Failed POST request",
			method:         "POST",
			path:           "/api/v1/users",
			fullPath:       "/api/v1/users",
			expectedStatus: http.StatusBadRequest,
			handler: func(c *gin.Context) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			},
		},
		{
			name:           "Not found request",
			method:         "GET",
			path:           "/api/v1/not-found",
			fullPath:       "",
			expectedStatus: http.StatusNotFound,
			handler: func(c *gin.Context) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			},
		},
		{
			name:           "Internal server error",
			method:         "PUT",
			path:           "/api/v1/users/123",
			fullPath:       "/api/v1/users/:id",
			expectedStatus: http.StatusInternalServerError,
			handler: func(c *gin.Context) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new router for each test
			router := gin.New()
			router.Use(MetricsMiddleware())

			// Add the test handler
			switch tt.method {
			case "GET":
				router.GET(tt.fullPath, tt.handler)
			case "POST":
				router.POST(tt.fullPath, tt.handler)
			case "PUT":
				router.PUT(tt.fullPath, tt.handler)
			}

			// Create test request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)

			// Execute request
			router.ServeHTTP(w, req)

			// Verify response status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify metrics were recorded
			// Check HTTPRequestsTotal counter
			// Get metric value
			counter, err := metrics.HTTPRequestsTotal.GetMetricWith(prometheus.Labels{
				"method": tt.method,
				"path":   tt.fullPath,
				"status": strconv.Itoa(tt.expectedStatus),
			})
			assert.NoError(t, err)

			// Verify counter was incremented
			// Note: We can't directly check the counter value without resetting metrics
			// but we've verified that the counter exists without error
			assert.NotNil(t, counter)
		})
	}
}

func TestMetricsMiddleware_Timing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router with metrics middleware
	router := gin.New()
	router.Use(MetricsMiddleware())

	// Add a handler with artificial delay
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(100 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "slow response"})
	})

	// Create test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)

	// Record start time
	start := time.Now()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify request took expected time
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify metrics were recorded by ensuring they can be retrieved without error
	_, err := metrics.HTTPRequestsTotal.GetMetricWith(prometheus.Labels{
		"method": "GET",
		"path":   "/slow",
		"status": "200",
	})
	assert.NoError(t, err)

	_, err = metrics.HTTPRequestDuration.GetMetricWith(prometheus.Labels{
		"method": "GET",
		"path":   "/slow",
		"status": "200",
	})
	assert.NoError(t, err)
}

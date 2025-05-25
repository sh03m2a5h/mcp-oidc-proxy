package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/metrics"
)

// MetricsMiddleware records HTTP metrics for each request
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get status code
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			status,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			status,
		).Observe(duration)
	}
}
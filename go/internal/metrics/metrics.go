package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_oidc_proxy_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_oidc_proxy_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// Proxy metrics
	ProxyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_oidc_proxy_proxy_requests_total",
			Help: "Total number of proxy requests",
		},
		[]string{"method", "status", "backend"},
	)

	ProxyRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_oidc_proxy_proxy_request_duration_seconds",
			Help:    "Proxy request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status", "backend"},
	)

	ProxyRetryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_oidc_proxy_proxy_retry_total",
			Help: "Total number of proxy request retries",
		},
		[]string{"method", "backend"},
	)

	// Circuit Breaker metrics
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mcp_oidc_proxy_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"backend"},
	)

	CircuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_oidc_proxy_circuit_breaker_failures_total",
			Help: "Total number of circuit breaker failures",
		},
		[]string{"backend"},
	)

	// Authentication metrics
	AuthRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_oidc_proxy_auth_requests_total",
			Help: "Total number of authentication requests",
		},
		[]string{"provider", "status"},
	)

	AuthCallbackDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "mcp_oidc_proxy_auth_callback_duration_seconds",
			Help:    "Authentication callback processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Session metrics
	SessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "mcp_oidc_proxy_sessions_active",
			Help: "Number of active sessions",
		},
	)

	SessionOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_oidc_proxy_session_operations_total",
			Help: "Total number of session operations",
		},
		[]string{"operation", "store_type", "status"},
	)

	SessionOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_oidc_proxy_session_operation_duration_seconds",
			Help:    "Session operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "store_type"},
	)

	// Application info
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mcp_oidc_proxy_build_info",
			Help: "Build information",
		},
		[]string{"version", "commit", "build_date"},
	)
)

// SetBuildInfo sets the build information metric
func SetBuildInfo(version, commit, buildDate string) {
	BuildInfo.WithLabelValues(version, commit, buildDate).Set(1)
}

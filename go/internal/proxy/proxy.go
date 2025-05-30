package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/auth/oidc"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/metrics"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Proxy handles reverse proxy operations
type Proxy struct {
	target         *url.URL
	reverseProxy   *httputil.ReverseProxy
	circuitBreaker *CircuitBreaker
	retryConfig    RetryConfig
	logger         *zap.Logger
	tracer         trace.Tracer
	headerInjector *middleware.HeaderInjector
}

// Config holds proxy configuration
type Config struct {
	TargetHost     string
	TargetPort     int
	TargetScheme   string
	Retry          RetryConfig
	CircuitBreaker CircuitBreakerConfig
	Headers        *config.HeadersConfig
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int
	Backoff     time.Duration
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Threshold int
	Timeout   time.Duration
}

// New creates a new reverse proxy
func New(config *Config, logger *zap.Logger) (*Proxy, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.TargetHost == "" {
		return nil, errors.New("target host is required")
	}

	if config.TargetPort <= 0 {
		return nil, errors.New("target port must be positive")
	}

	// Build target URL
	targetURL := &url.URL{
		Scheme: config.TargetScheme,
		Host:   fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort),
	}

	// Create reverse proxy
	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize director to handle path rewriting and headers
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		// Add standard proxy headers
		req.Header.Set("X-Forwarded-Proto", getScheme(req))
		req.Header.Set("X-Forwarded-Host", req.Host)
		
		// Inject trace context into outgoing request headers
		propagator := otel.GetTextMapPropagator()
		propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
		
		// Remove hop-by-hop headers
		removeHopHeaders(req.Header)
	}

	// Custom error handler
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("Proxy error", 
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("remote_addr", r.RemoteAddr),
		)
		
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	}

	// Create circuit breaker
	circuitBreaker := NewCircuitBreaker(config.CircuitBreaker.Threshold, config.CircuitBreaker.Timeout, logger)

	// Create tracer
	tracer := otel.Tracer("mcp-oidc-proxy/proxy")

	// Create header injector if headers config is provided
	var headerInjector *middleware.HeaderInjector
	if config.Headers != nil {
		headerInjector = middleware.NewHeaderInjector(config.Headers, logger)
	}

	return &Proxy{
		target:         targetURL,
		reverseProxy:   reverseProxy,
		circuitBreaker: circuitBreaker,
		retryConfig:    config.Retry,
		logger:         logger,
		tracer:         tracer,
		headerInjector: headerInjector,
	}, nil
}

// ServeHTTP implements http.Handler interface
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create proxy span
	ctx, span := p.tracer.Start(ctx, "proxy.request",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPMethod(r.Method),
			semconv.HTTPURL(r.URL.String()),
			semconv.HTTPTarget(r.URL.Path),
			semconv.NetHostName(p.target.Host),
			attribute.String("proxy.target", p.target.String()),
		),
	)
	defer span.End()

	// Update request context
	r = r.WithContext(ctx)
	
	// Inject custom headers if configured
	if p.headerInjector != nil {
		// Get session from context if available
		sess := oidc.GetSessionFromContext(r.Context())
		p.headerInjector.InjectHeaders(r, sess)
	}
	
	// Check if this is a streaming request
	if isStreamingRequest(r) {
		span.SetAttributes(attribute.Bool("proxy.streaming", true))
		p.handleStreaming(w, r)
		return
	}

	// Check circuit breaker state
	allow := p.circuitBreaker.Allow()
	state := float64(0) // closed
	if !allow {
		state = float64(1) // open
	}
	metrics.CircuitBreakerState.WithLabelValues(p.target.String()).Set(state)

	// Handle circuit breaker being open
	if !allow {
		p.logger.Warn("Circuit breaker open, rejecting request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("user_agent", r.UserAgent()),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("target", p.target.String()),
		)
		span.SetStatus(codes.Error, "Circuit breaker open")
		span.SetAttributes(
			semconv.HTTPStatusCode(http.StatusServiceUnavailable),
			attribute.String("error.type", "circuit_breaker_open"),
		)
		metrics.ProxyRequestsTotal.WithLabelValues(r.Method, "503", p.target.String()).Inc()
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service Unavailable"))
		return
	}

	// Execute with retry
	statusCode, err := p.executeWithRetry(ctx, w, r)
	
	// Calculate duration
	duration := time.Since(start).Seconds()
	
	// Record metrics
	status := strconv.Itoa(statusCode)
	metrics.ProxyRequestsTotal.WithLabelValues(r.Method, status, p.target.String()).Inc()
	metrics.ProxyRequestDuration.WithLabelValues(r.Method, status, p.target.String()).Observe(duration)
	
	// Update span with final status
	span.SetAttributes(
		semconv.HTTPStatusCode(statusCode),
		attribute.Float64("proxy.duration_seconds", duration),
	)
	
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.message", err.Error()))
	} else if statusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	}
	
	// Record result in circuit breaker
	if err != nil {
		p.circuitBreaker.RecordFailure()
		// Record circuit breaker failure metric
		metrics.CircuitBreakerFailures.WithLabelValues(p.target.String()).Inc()
	} else {
		p.circuitBreaker.RecordSuccess()
	}
}

// executeWithRetry executes the proxy request with retry logic
func (p *Proxy) executeWithRetry(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var lastErr error

	// For requests with body, ensure we can replay it
	if r.Body != nil && r.GetBody == nil {
		// For methods that typically have bodies, we need to be careful
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			p.logger.Warn("Request body cannot be replayed for retries",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("content_type", r.Header.Get("Content-Type")),
				zap.Int64("content_length", r.ContentLength),
				zap.String("target", p.target.String()),
			)
			// Set MaxAttempts to 1 to disable retry for non-replayable bodies
			p.retryConfig.MaxAttempts = 1
		}
	}

	for attempt := 1; attempt <= p.retryConfig.MaxAttempts; attempt++ {
		if attempt > 1 {
			// Wait before retry
			select {
			case <-time.After(p.retryConfig.Backoff):
			case <-ctx.Done():
				return http.StatusRequestTimeout, ctx.Err()
			}

			// Reset request body if possible
			if r.GetBody != nil {
				newBody, err := r.GetBody()
				if err != nil {
					return http.StatusBadRequest, fmt.Errorf("failed to reset request body: %w", err)
				}
				r.Body = newBody
			}

			p.logger.Debug("Retrying proxy request",
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", p.retryConfig.MaxAttempts),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("backoff", p.retryConfig.Backoff),
				zap.String("target", p.target.String()),
			)
			metrics.ProxyRetryTotal.WithLabelValues(r.Method, p.target.String()).Inc()
		}

		// Always use response recorder to capture status
		recorder := NewResponseRecorder()

		// Execute request
		p.reverseProxy.ServeHTTP(recorder, r)


		// Check if retry is needed
		if recorder.StatusCode >= 500 && recorder.StatusCode < 600 {
			lastErr = fmt.Errorf("server error: %d", recorder.StatusCode)
			
			// If this is not the last attempt, continue to retry
			if attempt < p.retryConfig.MaxAttempts {
				continue
			}
			
			// Last attempt with 5xx error - still write the response
			// but return error for circuit breaker
			recorder.WriteTo(w)
			return recorder.StatusCode, lastErr
		}

		// Success - write to actual response
		recorder.WriteTo(w)
		return recorder.StatusCode, nil
	}

	// If we get here, all retries failed
	// Return 502 Bad Gateway as we couldn't reach the backend
	return http.StatusBadGateway, lastErr
}

// Health checks if the target server is healthy
func (p *Proxy) Health(ctx context.Context) error {
	// Create health check span
	ctx, span := p.tracer.Start(ctx, "proxy.health_check",
		trace.WithAttributes(
			attribute.String("proxy.target", p.target.String()),
			semconv.HTTPMethod(http.MethodGet),
		),
	)
	defer span.End()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	healthURL := *p.target
	healthURL.Path = "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL.String(), nil)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to create request")
		span.SetAttributes(attribute.String("error.message", err.Error()))
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Inject trace context into health check request
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "Health check failed")
		span.SetAttributes(attribute.String("error.message", err.Error()))
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))

	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	span.SetStatus(codes.Ok, "Health check passed")
	return nil
}

// Target returns the target URL
func (p *Proxy) Target() *url.URL {
	return p.target
}

// getScheme returns the scheme of the request
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// removeHopHeaders removes hop-by-hop headers that shouldn't be forwarded
func removeHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, h := range hopHeaders {
		header.Del(h)
	}
}
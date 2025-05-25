package proxy

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/metrics"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// isStreamingRequest detects if the request is for SSE or WebSocket
func isStreamingRequest(r *http.Request) bool {
	// Check for SSE
	if accept := r.Header.Get("Accept"); strings.Contains(accept, "text/event-stream") {
		return true
	}
	
	// Check for WebSocket
	connection := r.Header.Get("Connection")
	upgrade := r.Header.Get("Upgrade")
	if strings.Contains(strings.ToLower(connection), "upgrade") && strings.ToLower(upgrade) == "websocket" {
		return true
	}
	
	return false
}

// handleStreaming handles SSE and WebSocket requests without buffering
func (p *Proxy) handleStreaming(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	startTime := time.Now()
	
	// Check circuit breaker
	if !p.circuitBreaker.Allow() {
		span.SetStatus(codes.Error, "circuit breaker open")
		metrics.ProxyStreamingErrorsTotal.WithLabelValues("circuit_breaker_open", p.target.Host).Inc()
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	
	// Log streaming request
	p.logger.Debug("Handling streaming request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("target", p.target.String()),
		zap.Bool("sse", strings.Contains(r.Header.Get("Accept"), "text/event-stream")),
		zap.Bool("websocket", r.Header.Get("Upgrade") == "websocket"),
	)
	
	// Metrics
	streamType := "sse"
	if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
		streamType = "websocket"
	}
	metrics.ProxyStreamingRequestsTotal.WithLabelValues(streamType, p.target.String()).Inc()
	
	// For WebSocket, use the standard reverse proxy which handles upgrades automatically
	if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
		// The httputil.ReverseProxy handles WebSocket upgrades correctly
		p.reverseProxy.ServeHTTP(w, r)
		
		// Record success (we can't easily get the actual status for WebSocket)
		p.circuitBreaker.RecordSuccess()
		duration := time.Since(startTime)
		metrics.ProxyRequestDuration.WithLabelValues(r.Method, "101", p.target.String()).Observe(duration.Seconds())
		return
	}
	
	// For SSE, use our custom streaming proxy
	status := p.streamingProxy(w, r)
	
	// Record duration
	duration := time.Since(startTime)
	metrics.ProxyRequestDuration.WithLabelValues(r.Method, strconv.Itoa(status), p.target.String()).Observe(duration.Seconds())
}

// streamingProxy performs direct streaming proxy without buffering
func (p *Proxy) streamingProxy(w http.ResponseWriter, r *http.Request) int {
	// Set target URL
	targetURL := *r.URL
	targetURL.Scheme = p.target.Scheme
	targetURL.Host = p.target.Host
	
	// Create client request
	client := &http.Client{
		Transport: p.reverseProxy.Transport,
		// No timeout for streaming connections
		Timeout: 0,
	}
	
	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		p.logger.Error("Failed to create proxy request",
			zap.Error(err),
			zap.String("target", targetURL.String()),
		)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return http.StatusBadGateway
	}
	
	// Copy headers
	copyHeaders(proxyReq.Header, r.Header)
	
	// Perform request
	resp, err := client.Do(proxyReq)
	if err != nil {
		p.logger.Error("Proxy request failed",
			zap.Error(err),
			zap.String("target", targetURL.String()),
		)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	
	// Copy response headers
	copyHeaders(w.Header(), resp.Header)
	
	// Set status code
	w.WriteHeader(resp.StatusCode)
	
	// Handle SSE streaming
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		p.handleSSEStream(w, resp.Body)
		return resp.StatusCode
	}
	
	// Standard streaming copy for SSE
	io.Copy(w, resp.Body)
	return resp.StatusCode
}

// handleSSEStream handles Server-Sent Events streaming
func (p *Proxy) handleSSEStream(w http.ResponseWriter, body io.Reader) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		p.logger.Error("ResponseWriter does not support flushing")
		return
	}
	
	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				p.logger.Error("Error reading SSE stream", zap.Error(err))
			}
			break
		}
		
		// Write line to response
		if _, err := w.Write(line); err != nil {
			p.logger.Error("Error writing SSE response", zap.Error(err))
			break
		}
		
		// Flush to send immediately
		flusher.Flush()
	}
}

// handleWebSocketUpgrade handles WebSocket protocol upgrade
func (p *Proxy) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request, resp *http.Response) {
	// WebSocket requires special handling that is more complex than our current implementation
	// For now, we'll use the standard reverse proxy which handles WebSocket correctly
	p.logger.Warn("WebSocket upgrade detected but using standard proxy",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("upgrade", r.Header.Get("Upgrade")),
	)
	
	// The reverse proxy in httputil handles WebSocket upgrades automatically
	// when it detects the Upgrade header, so we don't need custom handling here.
	// The streamingProxy function should be modified to use the reverse proxy directly.
}

// copyHeaders copies headers from source to destination
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
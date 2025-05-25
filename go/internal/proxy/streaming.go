package proxy

import (
	"bufio"
	"io"
	"net"
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
	if r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") == "websocket" {
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
	
	// Set target URL
	r.URL.Scheme = p.target.Scheme
	r.URL.Host = p.target.Host
	r.Host = p.target.Host
	
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
	if r.Header.Get("Upgrade") == "websocket" {
		streamType = "websocket"
	}
	metrics.ProxyStreamingRequestsTotal.WithLabelValues(streamType, p.target.String()).Inc()
	
	// Direct proxy without ResponseRecorder
	status := p.streamingProxy(w, r)
	
	// Record duration
	duration := time.Since(startTime)
	metrics.ProxyRequestDuration.WithLabelValues(r.Method, strconv.Itoa(status), p.target.String()).Observe(duration.Seconds())
}

// streamingProxy performs direct streaming proxy without buffering
func (p *Proxy) streamingProxy(w http.ResponseWriter, r *http.Request) int {
	// Create client request
	client := &http.Client{
		Transport: p.reverseProxy.Transport,
		// No timeout for streaming connections
		Timeout: 0,
	}
	
	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		p.logger.Error("Failed to create proxy request",
			zap.Error(err),
			zap.String("target", r.URL.String()),
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
			zap.String("target", r.URL.String()),
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
	
	// Handle WebSocket
	if resp.Header.Get("Upgrade") == "websocket" {
		p.handleWebSocketUpgrade(w, r, resp)
		return resp.StatusCode
	}
	
	// Standard streaming copy
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
	// Get hijacker
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		p.logger.Error("ResponseWriter does not support hijacking")
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		return
	}
	
	// Hijack the connection
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		p.logger.Error("Failed to hijack connection", zap.Error(err))
		http.Error(w, "WebSocket hijack failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()
	
	// Get backend connection
	backendConn, ok := resp.Body.(io.ReadWriteCloser)
	if !ok {
		p.logger.Error("Backend response does not support ReadWriteCloser")
		return
	}
	defer backendConn.Close()
	
	// Write upgrade response
	if err := resp.Write(clientConn); err != nil {
		p.logger.Error("Failed to write upgrade response", zap.Error(err))
		return
	}
	
	// Start bidirectional copy
	errChan := make(chan error, 2)
	
	go func() {
		_, err := io.Copy(backendConn, clientConn)
		errChan <- err
	}()
	
	go func() {
		_, err := io.Copy(clientConn, backendConn)
		errChan <- err
	}()
	
	// Wait for either direction to close
	err = <-errChan
	if err != nil && err != io.EOF {
		p.logger.Error("WebSocket proxy error", zap.Error(err))
	}
}

// copyHeaders copies headers from source to destination
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// StreamingResponseWriter wraps http.ResponseWriter to support streaming
type StreamingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (w *StreamingResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

// Write implements io.Writer
func (w *StreamingResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher
func (w *StreamingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker
func (w *StreamingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates a gin middleware for distributed tracing
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	
	return func(c *gin.Context) {
		// Extract trace context from incoming request headers
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		
		// Create span name from HTTP method and route
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if c.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}
		
		// Start new span
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Request.Method),
				semconv.HTTPURL(c.Request.URL.String()),
				semconv.HTTPScheme(c.Request.URL.Scheme),
				semconv.NetHostName(c.Request.Host),
				semconv.HTTPTarget(c.Request.URL.Path),
				semconv.HTTPUserAgent(c.Request.UserAgent()),
				attribute.Int64("http.request.content_length", c.Request.ContentLength),
			),
		)
		defer span.End()
		
		// Set request context with span
		c.Request = c.Request.WithContext(ctx)
		
		// Add span context to Gin context for downstream usage
		c.Set("trace_span", span)
		c.Set("trace_context", ctx)
		
		// Process request
		c.Next()
		
		// Set response attributes
		span.SetAttributes(
			semconv.HTTPStatusCode(c.Writer.Status()),
			attribute.Int64("http.response.content_length", int64(c.Writer.Size())),
		)
		
		// Set span status based on HTTP status code
		if c.Writer.Status() >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
		}
		
		// Add error information if available
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			span.SetAttributes(attribute.String("error.message", c.Errors.String()))
		}
		
		// Add user context if available (from auth middleware)
		if userID := c.GetString("user_id"); userID != "" {
			span.SetAttributes(
				semconv.EnduserID(userID),
			)
		}
		
		if userEmail := c.GetString("user_email"); userEmail != "" {
			span.SetAttributes(
				attribute.String("user.email", userEmail),
			)
		}
	}
}
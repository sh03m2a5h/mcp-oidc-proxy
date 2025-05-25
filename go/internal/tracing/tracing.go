package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/internal/config"
	"github.com/sh03m2a5h/mcp-oidc-proxy-go/pkg/version"
)

// Initialize sets up OpenTelemetry tracing based on configuration
func Initialize(ctx context.Context, cfg *config.TracingConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// Return a no-op shutdown function
		return func(context.Context) error { return nil }, nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(version.Version),
			semconv.DeploymentEnvironmentKey.String("production"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter trace.SpanExporter

	switch cfg.Provider {
	case "otlp", "jaeger":
		// Create OTLP HTTP exporter
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}

		// Add insecure option if not using HTTPS
		if cfg.Endpoint != "" && cfg.Endpoint[:5] != "https" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}

		exporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported tracing provider: %s", cfg.Provider)
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(cfg.SampleRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Return shutdown function
	return tp.Shutdown, nil
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) oteltrace.Tracer {
	return otel.Tracer(name)
}
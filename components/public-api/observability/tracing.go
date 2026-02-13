package observability

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// TracingEnabled returns true if OpenTelemetry tracing is enabled
func TracingEnabled() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_ENABLED") == "true"
}

// InitTracer initializes the OpenTelemetry tracer provider
// Returns a shutdown function that should be called on application exit
func InitTracer() func() {
	if !TracingEnabled() {
		Logger.Info().Msg("OpenTelemetry tracing disabled (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
		return func() {}
	}

	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		Logger.Error().Err(err).Msg("Failed to create OTLP exporter, tracing disabled")
		return func() {}
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("public-api"),
			semconv.ServiceVersion(getVersion()),
			semconv.DeploymentEnvironment(getEnvironment()),
		),
	)
	if err != nil {
		Logger.Error().Err(err).Msg("Failed to create resource, using default")
		res = resource.Default()
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(getSampler()),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set global propagator (W3C Trace Context + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	Logger.Info().
		Str("endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")).
		Msg("OpenTelemetry tracing initialized")

	// Return shutdown function
	return func() {
		ctx := context.Background()
		if err := tp.Shutdown(ctx); err != nil {
			Logger.Error().Err(err).Msg("Error shutting down tracer provider")
		}
	}
}

// getSampler returns the appropriate sampler based on environment
func getSampler() sdktrace.Sampler {
	// Check for custom sampling ratio
	if ratio := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); ratio != "" {
		// OpenTelemetry SDK handles this automatically via env vars
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))
	}

	// Default: sample 10% of traces in production, 100% in development
	if os.Getenv("GIN_MODE") == "debug" {
		return sdktrace.AlwaysSample()
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))
}

// getVersion returns the service version
func getVersion() string {
	if v := os.Getenv("SERVICE_VERSION"); v != "" {
		return v
	}
	return "unknown"
}

// getEnvironment returns the deployment environment
func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	if os.Getenv("GIN_MODE") == "debug" {
		return "development"
	}
	return "production"
}

package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// TracingConfig defines OpenTelemetry configuration options
// Supports multiple exporters for different observability backends
// Configurable sampling for performance optimisation
type TracingConfig struct {
    ServiceName     string  `yaml:"serviceName" json:"serviceName"`
    ServiceVersion  string  `yaml:"serviceVersion" json:"serviceVersion"`
    Environment     string  `yaml:"environment" json:"environment"`
    JaegerEndpoint  string  `yaml:"jaegerEndpoint" json:"jaegerEndpoint"`
    OTLPEndpoint    string  `yaml:"otlpEndpoint" json:"otlpEndpoint"`
    SamplingRatio   float64 `yaml:"samplingRatio" json:"samplingRatio"`
    Enabled         bool    `yaml:"enabled" json:"enabled"`
}

// InitTracing initializes OpenTelemetry tracing with configured exporters
// Sets up trace provider, propagators, and sampling for distributed tracing
// Supports both Jaeger and OTLP exporters for flexibility
// Time Complexity: O(1) - initialisation setup
// Space Complexity: O(1) - fixed tracer provider overhead
func InitTracing(config TracingConfig) (func(), error) {
    if !config.Enabled {
        return func() {}, nil
    }

    // Create resource with service information
    res, err := resource.Merge(
        resource.Default(),
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(config.ServiceName),
            semconv.ServiceVersionKey.String(config.ServiceVersion),
            semconv.DeploymentEnvironmentKey.String(config.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    var exporters []trace.SpanExporter

    // Configure Jaeger exporter if endpoint provided
    if config.JaegerEndpoint != "" {
        jaegerExporter, err := jaeger.New(
            jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)),
        )
        if err != nil {
            return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
        }
        exporters = append(exporters, jaegerExporter)
    }

    // Configure OTLP exporter if endpoint provided
    if config.OTLPEndpoint != "" {
        otlpExporter, err := otlptracehttp.New(
            context.Background(),
            otlptracehttp.WithEndpoint(config.OTLPEndpoint),
            otlptracehttp.WithInsecure(),
        )
        if err != nil {
            return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
        }
        exporters = append(exporters, otlpExporter)
    }

    if len(exporters) == 0 {
        return nil, fmt.Errorf("no trace exporters configured")
    }

    // Create batch span processors for performance
    var processors []trace.SpanProcessor
    for _, exporter := range exporters {
        processor := trace.NewBatchSpanProcessor(
            exporter,
            trace.WithBatchTimeout(time.Second*5),
            trace.WithMaxExportBatchSize(512),
        )
        processors = append(processors, processor)
    }

    // Configure sampling based on ratio
    var sampler trace.Sampler
    if config.SamplingRatio <= 0 {
        sampler = trace.NeverSample()
    } else if config.SamplingRatio >= 1 {
        sampler = trace.AlwaysSample()
    } else {
        sampler = trace.ParentBased(trace.TraceIDRatioBased(config.SamplingRatio))
    }

    // Create trace provider with all processors
    tp := trace.NewTracerProvider(
        trace.WithResource(res),
        trace.WithSampler(sampler),
    )

    for _, processor := range processors {
        tp.RegisterSpanProcessor(processor)
    }

    // Set global tracer provider
    otel.SetTracerProvider(tp)

    // Set global propagator for trace context
    otel.SetTextMapPropagator(
        propagation.NewCompositeTextMapPropagator(
            propagation.TraceContext{},
            propagation.Baggage{},
        ),
    )

    // Return cleanup function
    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }, nil
}
package otel

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/config"
)

// InitTracer initializes OpenTelemetry tracer with configuration
func InitTracer(ctx context.Context, cfg *config.Config) (*sdktrace.TracerProvider, error) {
	endpoint := cfg.GetEndpoint()
	fmt.Printf("Connecting to OTLP endpoint: %s (protocol: %s)\n", endpoint, cfg.Protocol)

	exporter, err := CreateExporter(ctx, cfg.Protocol, endpoint)
	if err != nil {
		return nil, err
	}

	serviceName := fmt.Sprintf("%s/%s",
		os.Getenv("CI_PROJECT_NAMESPACE"),
		os.Getenv("CI_PROJECT_NAME"))

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(os.Getenv("CI_COMMIT_SHA")),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

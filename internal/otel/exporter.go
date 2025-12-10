package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// CreateExporter creates an OTLP exporter based on protocol
func CreateExporter(ctx context.Context, protocol, endpoint string) (sdktrace.SpanExporter, error) {
	switch protocol {
	case "http":
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
	case "grpc":
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
	case "stdout", "console":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s (supported: http, grpc, stdout)", protocol)
	}
}

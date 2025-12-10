package otel

import (
	"context"
	"os"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestExtractParentContext(t *testing.T) {
	// Test non-triggered pipeline
	_ = os.Setenv("CI_PIPELINE_SOURCE", "push")
	defer func() { _ = os.Unsetenv("CI_PIPELINE_SOURCE") }()

	ctx := context.Background()
	result := ExtractParentContext(ctx, nil, nil)
	if result != ctx {
		t.Error("non-triggered pipeline should return original context")
	}

	// Test triggered pipeline with TRACEPARENT
	_ = os.Setenv("CI_PIPELINE_SOURCE", "pipeline")
	_ = os.Setenv("TRACEPARENT", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	defer func() { _ = os.Unsetenv("TRACEPARENT") }()

	// Setup propagator for test
	otel.SetTextMapPropagator(propagation.TraceContext{})

	result = ExtractParentContext(ctx, nil, nil)
	// Check if span context was extracted by looking for trace ID
	spanCtx := trace.SpanContextFromContext(result)
	if !spanCtx.IsValid() {
		t.Error("triggered pipeline with TRACEPARENT should have valid span context")
	}
}

package otel

import (
	"context"
	"testing"
)

func TestCreateExporter(t *testing.T) {
	ctx := context.Background()

	// Test HTTP exporter
	exporter, err := CreateExporter(ctx, "http", "localhost:4318")
	if err != nil {
		t.Errorf("HTTP exporter creation failed: %v", err)
	}
	if exporter == nil {
		t.Error("HTTP exporter should not be nil")
	}

	// Test gRPC exporter
	exporter, err = CreateExporter(ctx, "grpc", "localhost:4317")
	if err != nil {
		t.Errorf("gRPC exporter creation failed: %v", err)
	}
	if exporter == nil {
		t.Error("gRPC exporter should not be nil")
	}

	// Test stdout exporter
	exporter, err = CreateExporter(ctx, "stdout", "stdout")
	if err != nil {
		t.Errorf("stdout exporter creation failed: %v", err)
	}
	if exporter == nil {
		t.Error("stdout exporter should not be nil")
	}

	// Test console alias
	exporter, err = CreateExporter(ctx, "console", "stdout")
	if err != nil {
		t.Errorf("console exporter creation failed: %v", err)
	}
	if exporter == nil {
		t.Error("console exporter should not be nil")
	}

	// Test unsupported protocol
	exporter, err = CreateExporter(ctx, "invalid", "localhost:4318")
	if err == nil {
		t.Error("invalid protocol should return error")
	}
	if exporter != nil {
		t.Error("invalid protocol exporter should be nil")
	}
}

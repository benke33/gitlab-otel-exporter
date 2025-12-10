package main

import (
	"context"
	"os"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		key      string
		fallback string
		envValue string
		want     string
	}{
		{"TEST_KEY", "default", "", "default"},
		{"TEST_KEY", "default", "custom", "custom"},
	}

	for _, tt := range tests {
		if tt.envValue != "" {
			_ = os.Setenv(tt.key, tt.envValue)
			defer func() { _ = os.Unsetenv(tt.key) }()
		}
		if got := getEnv(tt.key, tt.fallback); got != tt.want {
			t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.fallback, got, tt.want)
		}
	}
}

func TestRefType(t *testing.T) {
	tests := []struct {
		tag  string
		want string
	}{
		{"", "branch"},
		{"v1.0.0", "tag"},
	}

	for _, tt := range tests {
		_ = os.Setenv("CI_COMMIT_TAG", tt.tag)
		defer func() { _ = os.Unsetenv("CI_COMMIT_TAG") }()
		if got := refType(); got != tt.want {
			t.Errorf("refType() = %q, want %q", got, tt.want)
		}
	}
}

func TestTriggerType(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"push", "scm.push"},
		{"merge_request_event", "scm.pull_request"},
		{"schedule", "schedule"},
		{"trigger", "other_pipeline"},
		{"pipeline", "other_pipeline"},
		{"web", "manual"},
	}

	for _, tt := range tests {
		_ = os.Setenv("CI_PIPELINE_SOURCE", tt.source)
		defer func() { _ = os.Unsetenv("CI_PIPELINE_SOURCE") }()
		if got := triggerType(); got != tt.want {
			t.Errorf("triggerType() with source %q = %q, want %q", tt.source, got, tt.want)
		}
	}
}

func TestPipelineAttributes(t *testing.T) {
	os.Setenv("CI_PIPELINE_NAME", "test-pipeline")
	os.Setenv("CI_PIPELINE_ID", "123")
	defer os.Unsetenv("CI_PIPELINE_NAME")
	defer os.Unsetenv("CI_PIPELINE_ID")

	attrs := pipelineAttributes()
	if len(attrs) != 7 {
		t.Errorf("pipelineAttributes() returned %d attributes, want 7", len(attrs))
	}
}

func TestJobAttributes(t *testing.T) {
	job := &JobData{
		Job: &gitlab.Job{
			ID:     456,
			Name:   "test-job",
			Stage:  "test",
			WebURL: "https://gitlab.com/test/job/456",
		},
		Raw: map[string]interface{}{
			"id":      float64(456),
			"name":    "test-job",
			"stage":   "test",
			"web_url": "https://gitlab.com/test/job/456",
			"status":  "success",
		},
	}

	attrs := jobAttributes(job)
	if len(attrs) < 5 {
		t.Errorf("jobAttributes() returned %d attributes, want at least 5", len(attrs))
	}
}

func TestExportJobSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := otel.Tracer("test")
	ctx := context.Background()

	now := time.Now()
	started := now.Add(-5 * time.Minute)
	job := &JobData{
		Job: &gitlab.Job{
			ID:         123,
			Name:       "build",
			Stage:      "build",
			Status:     "success",
			WebURL:     "https://gitlab.com/test/job/123",
			StartedAt:  &started,
			FinishedAt: &now,
		},
		Raw: map[string]interface{}{
			"id":    float64(123),
			"name":  "build",
			"stage": "build",
		},
	}

	_ = exportJobSpan(ctx, tracer, job)

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "Stage: build - job_id: 123" {
		t.Errorf("unexpected span name: %s", spans[0].Name)
	}
}

func TestStructToMap(t *testing.T) {
	job := &gitlab.Job{
		ID:     123,
		Name:   "test",
		Stage:  "build",
		Status: "success",
	}

	m, err := structToMap(job)
	if err != nil {
		t.Fatalf("structToMap() failed: %v", err)
	}

	if m["id"] != float64(123) {
		t.Errorf("expected id=123, got %v", m["id"])
	}
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}

func TestFlattenMap(t *testing.T) {
	m := map[string]interface{}{
		"simple": "value",
		"number": float64(42),
		"nested": map[string]interface{}{
			"key": "nested_value",
		},
	}

	attrs := flattenMap("", m)
	if len(attrs) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(attrs))
	}

	found := false
	for _, attr := range attrs {
		if attr.Key == "nested.key" && attr.Value.AsString() == "nested_value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("nested attribute not properly flattened")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no ansi", "plain text", "plain text"},
		{"with ansi color", "\x1b[31mred text\x1b[0m", "red text"},
		{"multiple ansi", "\x1b[1mbold\x1b[0m \x1b[32mgreen\x1b[0m", "bold green"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanRaw(t *testing.T) {
	m := map[string]interface{}{
		"plain":  "text",
		"ansi":   "\x1b[31mred\x1b[0m",
		"number":  float64(42),
		"nested": map[string]interface{}{
			"ansi_nested": "\x1b[1mbold\x1b[0m",
		},
		"array": []interface{}{"\x1b[32mgreen\x1b[0m", "plain"},
	}

	cleanRaw(m)

	if m["plain"] != "text" {
		t.Errorf("plain text changed: %v", m["plain"])
	}
	if m["ansi"] != "red" {
		t.Errorf("ansi not stripped: %v", m["ansi"])
	}
	if nested, ok := m["nested"].(map[string]interface{}); ok {
		if nested["ansi_nested"] != "bold" {
			t.Errorf("nested ansi not stripped: %v", nested["ansi_nested"])
		}
	}
	if arr, ok := m["array"].([]interface{}); ok {
		if arr[0] != "green" {
			t.Errorf("array ansi not stripped: %v", arr[0])
		}
	}
}

func TestExportJobSpanWithNilTimestamps(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := otel.Tracer("test")
	ctx := context.Background()

	job := &JobData{
		Job: &gitlab.Job{
			ID:         123,
			Name:       "build",
			Stage:      "build",
			Status:     "pending",
			StartedAt:  nil,
			FinishedAt: nil,
		},
		Raw: map[string]interface{}{},
	}

	err := exportJobSpan(ctx, tracer, job)
	if err != nil {
		t.Errorf("exportJobSpan with nil timestamps should not error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 0 {
		t.Errorf("expected 0 spans for job with nil timestamps, got %d", len(spans))
	}
}

func TestFlattenMapWithArrays(t *testing.T) {
	m := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2"},
		"empty": []interface{}{},
	}

	attrs := flattenMap("", m)

	found := false
	for _, attr := range attrs {
		if attr.Key == "tags" && attr.Value.AsString() == "tag1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("array first element not extracted")
	}
}

func TestFlattenMapWithNilValues(t *testing.T) {
	m := map[string]interface{}{
		"null_value": nil,
		"bool_true":  true,
		"bool_false": false,
	}

	attrs := flattenMap("", m)

	for _, attr := range attrs {
		if attr.Key == "null_value" && attr.Value.AsString() != "None" {
			t.Errorf("nil value not converted to 'None': %v", attr.Value.AsString())
		}
		if attr.Key == "bool_true" && attr.Value.AsString() != "true" {
			t.Errorf("bool true not converted: %v", attr.Value.AsString())
		}
	}
}

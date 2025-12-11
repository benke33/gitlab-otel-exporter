package spans

import (
	"context"
	"os"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/config"
	gitlabpkg "gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/gitlab"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/pkg/semconv"
)

func TestCreateJobSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := &config.Config{Debug: false}
	spanExporter := &Exporter{
		config: cfg,
		tracer: otel.Tracer("test"),
	}

	ctx := context.Background()
	now := time.Now()
	started := now.Add(-5 * time.Minute)
	job := &gitlabpkg.JobData{
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

	err := spanExporter.createJobSpan(ctx, job)
	if err != nil {
		t.Errorf("createJobSpan should not error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name != "Stage: build - job_id: 123" {
		t.Errorf("unexpected span name: %s", spans[0].Name)
	}
}

func TestCreateJobSpanWithNilTimestamps(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	cfg := &config.Config{Debug: false}
	spanExporter := &Exporter{
		config: cfg,
		tracer: otel.Tracer("test"),
	}

	ctx := context.Background()
	job := &gitlabpkg.JobData{
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

	err := spanExporter.createJobSpan(ctx, job)
	if err != nil {
		t.Errorf("createJobSpan with nil timestamps should not error: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 0 {
		t.Errorf("expected 0 spans for job with nil timestamps, got %d", len(spans))
	}
}

func TestDownstreamPipelineIntegration(t *testing.T) {
	// Simulate downstream pipeline environment
	_ = os.Setenv("CI_PIPELINE_SOURCE", "pipeline")
	_ = os.Setenv("CI_PARENT_PIPELINE_ID", "100")
	_ = os.Setenv("CI_PARENT_PROJECT_ID", "200")
	_ = os.Setenv("TRACEPARENT", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	_ = os.Setenv("CI_PROJECT_NAMESPACE", "test")
	_ = os.Setenv("CI_PROJECT_NAME", "downstream")
	defer func() {
		_ = os.Unsetenv("CI_PIPELINE_SOURCE")
		_ = os.Unsetenv("CI_PARENT_PIPELINE_ID")
		_ = os.Unsetenv("CI_PARENT_PROJECT_ID")
		_ = os.Unsetenv("TRACEPARENT")
		_ = os.Unsetenv("CI_PROJECT_NAMESPACE")
		_ = os.Unsetenv("CI_PROJECT_NAME")
	}()

	// Test pipeline attributes include parent info
	attrs := semconv.PipelineAttributes()
	found := false
	for _, attr := range attrs {
		if attr.Key == "cicd.pipeline.trigger.type" && attr.Value.AsString() == "other_pipeline" {
			found = true
			break
		}
	}
	if !found {
		t.Error("downstream pipeline should have trigger.type = other_pipeline")
	}

	// Test parent attributes are generated
	pipeline := &gitlabpkg.PipelineData{
		Pipeline: &gitlab.Pipeline{
			User: &gitlab.BasicUser{ID: 300},
		},
	}
	parentAttrs := semconv.ParentPipelineAttributes(nil, pipeline)
	if len(parentAttrs) == 0 {
		t.Error("downstream pipeline should have parent attributes")
	}
}

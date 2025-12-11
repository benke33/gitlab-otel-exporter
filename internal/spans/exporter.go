package spans

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/config"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/gitlab"
	otelutil "gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/otel"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/utils"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/pkg/semconv"
)

// Exporter handles span creation and export
type Exporter struct {
	config    *config.Config
	gitClient *gitlab.Client
	tracer    trace.Tracer
}

// NewExporter creates a new span exporter
func NewExporter(cfg *config.Config, gitClient *gitlab.Client) *Exporter {
	return &Exporter{
		config:    cfg,
		gitClient: gitClient,
		tracer:    otel.Tracer("gitlab-ci-collector"),
	}
}

// ExportPipeline exports traces for the entire pipeline
func (e *Exporter) ExportPipeline(ctx context.Context) error {
	fmt.Println("Fetching pipeline data...")
	pipeline, err := e.gitClient.FetchPipeline()
	if err != nil {
		return err
	}

	// Check for parent pipeline context
	ctx = otelutil.ExtractParentContext(ctx, e.gitClient, pipeline)

	jobs, err := e.gitClient.FetchJobs()
	if err != nil {
		return err
	}
	fmt.Printf("Found %d jobs in pipeline\n", len(jobs))

	// Create pipeline span
	ctx, pipelineSpan := e.createPipelineSpan(ctx, pipeline)
	defer e.endPipelineSpan(pipelineSpan, pipeline)

	// Export trace context for downstream pipelines
	otelutil.ExportTraceContext(ctx, e.config.Debug)

	// Create job spans
	fmt.Println("Creating job spans...")
	for _, job := range jobs {
		if job.Status == "skipped" {
			continue
		}
		if err := e.createJobSpan(ctx, job); err != nil {
			log.Printf("failed to export job span for job %d: %v", job.ID, err)
		}
	}

	return nil
}

func (e *Exporter) createPipelineSpan(ctx context.Context, pipeline *gitlab.PipelineData) (context.Context, trace.Span) {
	pipelineName := fmt.Sprintf("%s/%s #%d",
		os.Getenv("CI_PROJECT_NAMESPACE"),
		os.Getenv("CI_PROJECT_NAME"),
		pipeline.ID)

	pipelineAttrs := semconv.PipelineAttributes()
	pipelineAttrs = append(pipelineAttrs, utils.FlattenMap("", pipeline.Raw)...)

	// Add parent pipeline correlation attributes
	if parentAttrs := semconv.ParentPipelineAttributes(e.gitClient, pipeline); len(parentAttrs) > 0 {
		pipelineAttrs = append(pipelineAttrs, parentAttrs...)
	}

	var startOpts []trace.SpanStartOption
	if pipeline.CreatedAt != nil {
		startOpts = append(startOpts, trace.WithTimestamp(*pipeline.CreatedAt))
	}
	startOpts = append(startOpts,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(pipelineAttrs...),
	)

	ctx, pipelineSpan := e.tracer.Start(ctx, pipelineName, startOpts...)
	fmt.Printf("Creating pipeline span: %s\n", pipelineName)

	return ctx, pipelineSpan
}

func (e *Exporter) endPipelineSpan(pipelineSpan trace.Span, pipeline *gitlab.PipelineData) {
	if pipeline.Status == "failed" {
		pipelineSpan.SetStatus(codes.Error, "pipeline failed")
	} else {
		pipelineSpan.SetStatus(codes.Ok, "")
	}

	if pipeline.UpdatedAt != nil {
		pipelineSpan.End(trace.WithTimestamp(*pipeline.UpdatedAt))
	} else {
		pipelineSpan.End()
	}
}

func (e *Exporter) createJobSpan(ctx context.Context, job *gitlab.JobData) error {
	if job.StartedAt == nil || job.FinishedAt == nil {
		return nil
	}

	spanName := fmt.Sprintf("Stage: %s - job_id: %d", job.Name, job.ID)
	attrs := semconv.JobAttributes(job)

	_, jobSpan := e.tracer.Start(ctx, spanName,
		trace.WithTimestamp(*job.StartedAt),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
	fmt.Printf("  Job: %s (%s)\n", job.Name, job.Status)
	defer jobSpan.End(trace.WithTimestamp(*job.FinishedAt))

	if job.Status == "failed" {
		jobSpan.SetStatus(codes.Error, "job failed")
	} else {
		jobSpan.SetStatus(codes.Ok, "")
	}

	return nil
}

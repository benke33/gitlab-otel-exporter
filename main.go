package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

type PipelineData struct {
	*gitlab.Pipeline
	Raw map[string]interface{}
}

type JobData struct {
	*gitlab.Job
	Raw map[string]interface{}
}

var debug = os.Getenv("DEBUG") == "true"

func main() {
	ctx := context.Background()

	fmt.Println("🚀 Starting GitLab OpenTelemetry Exporter")

	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("error shutting down tracer: %v", err)
		}
	}()

	if err := exportPipelineTrace(ctx); err != nil {
		log.Fatalf("failed to export trace: %v", err)
	}

	fmt.Println("✅ Traces exported successfully")
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	protocol := getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "http")
	endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", getDefaultEndpoint(protocol))
	fmt.Printf("📡 Connecting to OTLP endpoint: %s (protocol: %s)\n", endpoint, protocol)

	exporter, err := createExporter(ctx, protocol, endpoint)
	if err != nil {
		return nil, err
	}

	serviceName := fmt.Sprintf("%s/%s",
		os.Getenv("CI_PROJECT_NAMESPACE"),
		os.Getenv("CI_PROJECT_NAME"))
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(getEnv("CI_COMMIT_SHA", "unknown")),
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

func exportPipelineTrace(ctx context.Context) error {
	tracer := otel.Tracer("gitlab-ci-collector")

	git, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"),
		gitlab.WithBaseURL(os.Getenv("CI_SERVER_URL")),
	)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	fmt.Println("📥 Fetching pipeline data from GitLab API...")
	pipeline, err := fetchPipeline(git)
	if err != nil {
		return fmt.Errorf("failed to fetch pipeline: %w", err)
	}

	// Check for parent pipeline context
	ctx = extractParentContext(ctx, git, pipeline)

	jobs, err := fetchJobs(git)
	if err != nil {
		return fmt.Errorf("failed to fetch jobs: %w", err)
	}
	fmt.Printf("📋 Found %d jobs in pipeline\n", len(jobs))

	pipelineName := fmt.Sprintf("%s/%s #%d",
		os.Getenv("CI_PROJECT_NAMESPACE"),
		os.Getenv("CI_PROJECT_NAME"),
		pipeline.ID)

	pipelineAttrs := pipelineAttributes()
	pipelineAttrs = append(pipelineAttrs, flattenMap("", pipeline.Raw)...)

	// Add parent pipeline correlation attributes
	if parentAttrs := getParentPipelineAttributes(git, pipeline); len(parentAttrs) > 0 {
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

	ctx, pipelineSpan := tracer.Start(ctx, pipelineName, startOpts...)
	fmt.Printf("📤 Creating pipeline span: %s\n", pipelineName)
	if debug {
		fmt.Printf("   Attributes: %v\n", pipelineAttrs)
	}

	// Export trace context for downstream pipelines
	exportTraceContext(ctx)

	defer func() {
		if pipeline.UpdatedAt != nil {
			pipelineSpan.End(trace.WithTimestamp(*pipeline.UpdatedAt))
		} else {
			pipelineSpan.End()
		}
	}()

	fmt.Println("📤 Creating job spans...")
	for _, job := range jobs {
		if job.Status == "skipped" {
			continue
		}
		if err := exportJobSpan(ctx, tracer, job); err != nil {
			log.Printf("failed to export job span for job %d: %v", job.ID, err)
		}
	}

	if pipeline.Status == "failed" {
		pipelineSpan.SetStatus(codes.Error, "pipeline failed")
	} else {
		pipelineSpan.SetStatus(codes.Ok, "")
	}

	return nil
}

func exportJobSpan(ctx context.Context, tracer trace.Tracer, job *JobData) error {
	if job.StartedAt == nil || job.FinishedAt == nil {
		return nil
	}

	spanName := fmt.Sprintf("Stage: %s - job_id: %d", job.Name, job.ID)
	attrs := jobAttributes(job)
	_, jobSpan := tracer.Start(ctx, spanName,
		trace.WithTimestamp(*job.StartedAt),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
	fmt.Printf("   ├─ Job: %s (status: %s)\n", job.Name, job.Status)
	if debug {
		fmt.Printf("      Attributes: %v\n", attrs)
	}
	defer jobSpan.End(trace.WithTimestamp(*job.FinishedAt))

	if job.Status == "failed" {
		jobSpan.SetStatus(codes.Error, "job failed")
	} else {
		jobSpan.SetStatus(codes.Ok, "")
	}

	return nil
}

func fetchPipeline(git *gitlab.Client) (*PipelineData, error) {
	projectID := os.Getenv("CI_PROJECT_ID")
	pipelineID, _ := strconv.Atoi(os.Getenv("CI_PIPELINE_ID"))

	pipeline, _, err := git.Pipelines.GetPipeline(projectID, pipelineID, nil)
	if err != nil {
		return nil, err
	}

	raw, err := structToMap(pipeline)
	if err != nil {
		return nil, err
	}
	cleanRaw(raw)

	return &PipelineData{Pipeline: pipeline, Raw: raw}, nil
}

func fetchJobs(git *gitlab.Client) ([]*JobData, error) {
	projectID := os.Getenv("CI_PROJECT_ID")
	pipelineID, _ := strconv.Atoi(os.Getenv("CI_PIPELINE_ID"))

	jobs, _, err := git.Jobs.ListPipelineJobs(projectID, pipelineID, &gitlab.ListJobsOptions{}, nil)
	if err != nil {
		return nil, err
	}

	var jobData []*JobData
	for _, job := range jobs {
		raw, err := structToMap(job)
		if err != nil {
			log.Printf("failed to convert job %d to map: %v", job.ID, err)
			continue
		}
		cleanRaw(raw)
		jobData = append(jobData, &JobData{Job: job, Raw: raw})
	}

	return jobData, nil
}

func structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func cleanRaw(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = stripANSI(val)
		case map[string]interface{}:
			cleanRaw(val)
		case []interface{}:
			for i, item := range val {
				if str, ok := item.(string); ok {
					val[i] = stripANSI(str)
				}
			}
		}
	}
}

func stripANSI(s string) string {
	if !strings.Contains(s, "\x1b") {
		return s
	}
	return ansiRegex.ReplaceAllString(s, "")
}

func pipelineAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("cicd.pipeline.name", os.Getenv("CI_PIPELINE_NAME")),
		attribute.String("cicd.pipeline.run.id", os.Getenv("CI_PIPELINE_ID")),
		attribute.String("vcs.repository.url.full", os.Getenv("CI_PROJECT_URL")),
		attribute.String("vcs.repository.ref.name", os.Getenv("CI_COMMIT_REF_NAME")),
		attribute.String("vcs.repository.ref.revision", os.Getenv("CI_COMMIT_SHA")),
		attribute.String("vcs.repository.ref.type", refType()),
		attribute.String("cicd.pipeline.trigger.type", triggerType()),
	}
}

func jobAttributes(job *JobData) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("cicd.pipeline.task.name", job.Name),
		attribute.String("cicd.pipeline.task.run.id", fmt.Sprintf("%d", job.ID)),
		attribute.String("cicd.pipeline.task.run.url.full", job.WebURL),
		attribute.String("cicd.pipeline.task.type", "build"),
		attribute.String("stage", job.Stage),
	}

	attrs = append(attrs, flattenMap("", job.Raw)...)
	return attrs
}

func flattenMap(prefix string, m map[string]interface{}) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			attrs = append(attrs, flattenMap(key, val)...)
		case []interface{}:
			if len(val) > 0 {
				if str, ok := val[0].(string); ok {
					attrs = append(attrs, attribute.String(key, str))
				}
			}
		case string:
			attrs = append(attrs, attribute.String(key, val))
		case float64:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%.0f", val)))
		case bool:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%v", val)))
		case nil:
			attrs = append(attrs, attribute.String(key, "None"))
		}
	}
	return attrs
}

func refType() string {
	if os.Getenv("CI_COMMIT_TAG") != "" {
		return "tag"
	}
	return "branch"
}

func triggerType() string {
	switch os.Getenv("CI_PIPELINE_SOURCE") {
	case "push":
		return "scm.push"
	case "merge_request_event":
		return "scm.pull_request"
	case "schedule":
		return "schedule"
	case "trigger", "pipeline":
		return "other_pipeline"
	default:
		return "manual"
	}
}



func extractParentContext(ctx context.Context, git *gitlab.Client, pipeline *PipelineData) context.Context {
	// Check if this pipeline was triggered by another pipeline
	if os.Getenv("CI_PIPELINE_SOURCE") != "pipeline" && os.Getenv("CI_PIPELINE_SOURCE") != "trigger" {
		return ctx
	}

	// Look for parent pipeline trace context in variables
	if traceParent := os.Getenv("TRACEPARENT"); traceParent != "" {
		carrier := propagation.MapCarrier{"traceparent": traceParent}
		return otel.GetTextMapPropagator().Extract(ctx, carrier)
	}

	// Try to extract from pipeline variables if available
	projectID := os.Getenv("CI_PROJECT_ID")
	pipelineID, _ := strconv.Atoi(os.Getenv("CI_PIPELINE_ID"))

	if variables, _, err := git.Pipelines.GetPipelineVariables(projectID, pipelineID, nil); err == nil {
		for _, v := range variables {
			if v.Key == "TRACEPARENT" {
				carrier := propagation.MapCarrier{"traceparent": v.Value}
				return otel.GetTextMapPropagator().Extract(ctx, carrier)
			}
		}
	}

	return ctx
}

func getParentPipelineAttributes(git *gitlab.Client, pipeline *PipelineData) []attribute.KeyValue {
	var attrs []attribute.KeyValue

	// Add parent pipeline info for downstream pipelines
	if os.Getenv("CI_PIPELINE_SOURCE") == "pipeline" || os.Getenv("CI_PIPELINE_SOURCE") == "trigger" {
		if parentPipelineID := os.Getenv("CI_PARENT_PIPELINE_ID"); parentPipelineID != "" {
			attrs = append(attrs, attribute.String("cicd.pipeline.parent.id", parentPipelineID))
		}
		if parentProjectID := os.Getenv("CI_PARENT_PROJECT_ID"); parentProjectID != "" {
			attrs = append(attrs, attribute.String("cicd.pipeline.parent.project.id", parentProjectID))
		}

		// Try to get more parent info from API if available
		if pipeline.User != nil && pipeline.User.ID != 0 {
			// This is a best-effort attempt to correlate with parent
			attrs = append(attrs, attribute.String("cicd.pipeline.trigger.user.id", fmt.Sprintf("%d", pipeline.User.ID)))
		}
	}

	return attrs
}

func exportTraceContext(ctx context.Context) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if traceParent := carrier["traceparent"]; traceParent != "" {
		fmt.Printf("🔗 TRACE_PARENT=%s\n", traceParent)
		if debug {
			fmt.Printf("   Use this in downstream pipeline variables\n")
		}
	}
}

func createExporter(ctx context.Context, protocol, endpoint string) (sdktrace.SpanExporter, error) {
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

func getDefaultEndpoint(protocol string) string {
	switch protocol {
	case "http":
		return "localhost:4318"
	case "grpc":
		return "localhost:4317"
	case "stdout", "console":
		return "stdout"
	default:
		return "localhost:4318"
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

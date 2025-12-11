package semconv

import (
	"fmt"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/gitlab"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/utils"
)

// PipelineAttributes returns CI/CD semantic convention attributes for pipeline
func PipelineAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("cicd.pipeline.name", os.Getenv("CI_PIPELINE_NAME")),
		attribute.String("cicd.pipeline.run.id", os.Getenv("CI_PIPELINE_ID")),
		attribute.String("vcs.repository.url.full", os.Getenv("CI_PROJECT_URL")),
		attribute.String("vcs.repository.ref.name", os.Getenv("CI_COMMIT_REF_NAME")),
		attribute.String("vcs.repository.ref.revision", os.Getenv("CI_COMMIT_SHA")),
		attribute.String("vcs.repository.ref.type", RefType()),
		attribute.String("cicd.pipeline.trigger.type", TriggerType()),
	}
}

// JobAttributes returns CI/CD semantic convention attributes for job
func JobAttributes(job *gitlab.JobData) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("cicd.pipeline.task.name", job.Name),
		attribute.String("cicd.pipeline.task.run.id", fmt.Sprintf("%d", job.ID)),
		attribute.String("cicd.pipeline.task.run.url.full", job.WebURL),
		attribute.String("cicd.pipeline.task.type", "build"),
		attribute.String("stage", job.Stage),
	}

	attrs = append(attrs, utils.FlattenMap("", job.Raw)...)
	return attrs
}

// ParentPipelineAttributes returns attributes for parent pipeline correlation
func ParentPipelineAttributes(gitClient *gitlab.Client, pipeline *gitlab.PipelineData) []attribute.KeyValue {
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

// RefType determines if the reference is a branch or tag
func RefType() string {
	if os.Getenv("CI_COMMIT_TAG") != "" {
		return "tag"
	}
	return "branch"
}

// TriggerType determines the pipeline trigger type
func TriggerType() string {
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

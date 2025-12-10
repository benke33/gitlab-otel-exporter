package semconv

import (
	"os"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabpkg "github.com/benke33/gitlab-otel-exporter/internal/gitlab"
)

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
		if got := RefType(); got != tt.want {
			t.Errorf("RefType() = %q, want %q", got, tt.want)
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
		if got := TriggerType(); got != tt.want {
			t.Errorf("TriggerType() with source %q = %q, want %q", tt.source, got, tt.want)
		}
	}
}

func TestPipelineAttributes(t *testing.T) {
	_ = os.Setenv("CI_PIPELINE_NAME", "test-pipeline")
	_ = os.Setenv("CI_PIPELINE_ID", "123")
	defer func() { _ = os.Unsetenv("CI_PIPELINE_NAME") }()
	defer func() { _ = os.Unsetenv("CI_PIPELINE_ID") }()

	attrs := PipelineAttributes()
	if len(attrs) != 7 {
		t.Errorf("PipelineAttributes() returned %d attributes, want 7", len(attrs))
	}
}

func TestJobAttributes(t *testing.T) {
	job := &gitlabpkg.JobData{
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

	attrs := JobAttributes(job)
	if len(attrs) < 5 {
		t.Errorf("JobAttributes() returned %d attributes, want at least 5", len(attrs))
	}
}

func TestParentPipelineAttributes(t *testing.T) {
	// Test non-triggered pipeline
	_ = os.Setenv("CI_PIPELINE_SOURCE", "push")
	defer func() { _ = os.Unsetenv("CI_PIPELINE_SOURCE") }()

	attrs := ParentPipelineAttributes(nil, nil)
	if len(attrs) != 0 {
		t.Errorf("non-triggered pipeline should have no parent attributes, got %d", len(attrs))
	}

	// Test triggered pipeline with parent info
	_ = os.Setenv("CI_PIPELINE_SOURCE", "trigger")
	_ = os.Setenv("CI_PARENT_PIPELINE_ID", "123")
	_ = os.Setenv("CI_PARENT_PROJECT_ID", "456")
	defer func() {
		_ = os.Unsetenv("CI_PARENT_PIPELINE_ID")
		_ = os.Unsetenv("CI_PARENT_PROJECT_ID")
	}()

	pipeline := &gitlabpkg.PipelineData{
		Pipeline: &gitlab.Pipeline{
			User: &gitlab.BasicUser{ID: 789},
		},
	}

	attrs = ParentPipelineAttributes(nil, pipeline)
	if len(attrs) != 3 {
		t.Errorf("triggered pipeline should have 3 parent attributes, got %d", len(attrs))
	}

	// Verify specific attributes
	found := map[string]bool{}
	for _, attr := range attrs {
		switch attr.Key {
		case "cicd.pipeline.parent.id":
			if attr.Value.AsString() != "123" {
				t.Errorf("parent.id should be 123, got %s", attr.Value.AsString())
			}
			found["parent.id"] = true
		case "cicd.pipeline.parent.project.id":
			if attr.Value.AsString() != "456" {
				t.Errorf("parent.project.id should be 456, got %s", attr.Value.AsString())
			}
			found["parent.project.id"] = true
		case "cicd.pipeline.trigger.user.id":
			if attr.Value.AsString() != "789" {
				t.Errorf("trigger.user.id should be 789, got %s", attr.Value.AsString())
			}
			found["trigger.user.id"] = true
		}
	}

	if len(found) != 3 {
		t.Errorf("not all expected attributes found: %v", found)
	}
}

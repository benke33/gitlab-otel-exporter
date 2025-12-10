package gitlab

import gitlab "gitlab.com/gitlab-org/api/client-go"

// PipelineData wraps GitLab pipeline with raw data
type PipelineData struct {
	*gitlab.Pipeline
	Raw map[string]interface{}
}

// JobData wraps GitLab job with raw data
type JobData struct {
	*gitlab.Job
	Raw map[string]interface{}
}

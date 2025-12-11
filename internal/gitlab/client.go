package gitlab

import (
	"log"
	"strconv"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/config"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/utils"
)

// Client wraps GitLab API client with configuration
type Client struct {
	client *gitlab.Client
	config *config.Config
}

// NewClient creates a new GitLab client
func NewClient(cfg *config.Config) (*Client, error) {
	client, err := gitlab.NewJobClient(cfg.Token, gitlab.WithBaseURL(cfg.ServerURL))
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// FetchPipeline retrieves pipeline data from GitLab API
func (c *Client) FetchPipeline() (*PipelineData, error) {
	pipelineID, _ := strconv.Atoi(c.config.PipelineID)

	pipeline, _, err := c.client.Pipelines.GetPipeline(c.config.ProjectID, pipelineID, nil)
	if err != nil {
		return nil, err
	}

	raw, err := utils.StructToMap(pipeline)
	if err != nil {
		return nil, err
	}
	utils.CleanRaw(raw)

	return &PipelineData{Pipeline: pipeline, Raw: raw}, nil
}

// FetchJobs retrieves all jobs for the pipeline
func (c *Client) FetchJobs() ([]*JobData, error) {
	pipelineID, _ := strconv.Atoi(c.config.PipelineID)

	jobs, _, err := c.client.Jobs.ListPipelineJobs(c.config.ProjectID, pipelineID, &gitlab.ListJobsOptions{}, nil)
	if err != nil {
		return nil, err
	}

	var jobData []*JobData
	for _, job := range jobs {
		raw, err := utils.StructToMap(job)
		if err != nil {
			log.Printf("failed to convert job %d to map: %v", job.ID, err)
			continue
		}
		utils.CleanRaw(raw)
		jobData = append(jobData, &JobData{Job: job, Raw: raw})
	}

	return jobData, nil
}

// GetClient returns the underlying GitLab client
func (c *Client) GetClient() *gitlab.Client {
	return c.client
}

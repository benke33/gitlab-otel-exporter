package main

import (
	"context"
	"fmt"
	"log"

	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/config"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/gitlab"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/otel"
	"gitlab.internal.ericsson.com/ewikhen/gitlab-otel-exporter/internal/spans"
)

func main() {
	ctx := context.Background()

	fmt.Println("Starting GitLab OpenTelemetry Exporter")

	// Load configuration
	cfg := config.Load()

	// Initialize tracer
	tp, err := otel.InitTracer(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("error shutting down tracer: %v", err)
		}
	}()

	// Create GitLab client
	gitClient, err := gitlab.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to create GitLab client: %v", err)
	}

	// Create and run exporter
	exporter := spans.NewExporter(cfg, gitClient)
	if err := exporter.ExportPipeline(ctx); err != nil {
		log.Fatalf("failed to export trace: %v", err)
	}

	fmt.Println("Traces exported successfully")
}

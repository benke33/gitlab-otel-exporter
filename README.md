# GitLab OpenTelemetry Exporter

A minimal OpenTelemetry exporter for GitLab CI/CD pipelines that exports traces following the [CI/CD semantic conventions](https://opentelemetry.io/docs/specs/semconv/cicd/cicd-spans/).

## Features

- Fetches all pipeline jobs via GitLab API using official GitLab Go SDK
- Creates one span per job/stage in the pipeline
- Exports traces to OTLP HTTP endpoint with parent-child relationships
- Follows OpenTelemetry CI/CD semantic conventions
- Runs in `.post` stage (executes regardless of pipeline status)
- Real-time console output with progress indicators
- Debug mode to print all span attributes
- Comprehensive metadata export (all GitLab API data flattened as span attributes)
- ANSI escape code stripping for clean attribute values
- Written in Go 1.25 with best practices

## Installation

```bash
go mod download
go build -o gitlab-otel-exporter main.go
```

## Usage

### GitLab CI/CD Integration

The exporter runs automatically in your pipeline via the `otel-export` job in the `.post` stage:

```yaml
variables:
  OTEL_EXPORTER_OTLP_ENDPOINT: "your-collector:4318"

otel-export:
  stage: .post
  image: golang:1.25
  script:
    - export GITLAB_TOKEN=${CI_JOB_TOKEN}
    - go run main.go
  when: always
  allow_failure: true
```

The `.post` stage ensures the exporter runs after all other stages complete, regardless of pipeline success or failure. The exporter uses `CI_JOB_TOKEN` to authenticate with the GitLab API and fetch all pipeline jobs.

### Debug Mode

Enable debug mode to print all span attributes:

```yaml
otel-export:
  stage: .post
  script:
    - export GITLAB_TOKEN=${CI_JOB_TOKEN}
    - export DEBUG=true
    - go run main.go
```

### Console Output

The exporter provides real-time feedback:

```
🚀 Starting GitLab OpenTelemetry Exporter
📡 Connecting to OTLP endpoint: collector:4318
📥 Fetching pipeline data from GitLab API...
📋 Found 5 jobs in pipeline
📤 Creating pipeline span: namespace/project #12345
📤 Creating job spans...
   ├─ Job: build (status: success)
   ├─ Job: test (status: success)
   ├─ Job: deploy (status: failed)
✅ Traces exported successfully
```

### Trace Structure

**Service Name:** `namespace/project` (e.g., `ewikhen/otel-go-collector`)

**Root Span Name:** `namespace/project #pipelineID` (e.g., `ewikhen/otel-go-collector #12345`)

**Job Span Name:** `Stage: job_name - job_id: 123`

### Exported Attributes

**Pipeline Span:**
- `cicd.pipeline.name`
- `cicd.pipeline.run.id`
- `vcs.repository.url.full`
- `vcs.repository.ref.name`
- `vcs.repository.ref.revision`
- `vcs.repository.ref.type`
- `cicd.pipeline.trigger.type`
- All GitLab API pipeline metadata (flattened)

**Job Span:**
- `cicd.pipeline.task.name`
- `cicd.pipeline.task.run.id`
- `cicd.pipeline.task.run.url.full`
- `cicd.pipeline.task.type`
- `stage`
- All GitLab API job metadata (flattened)

## Docker

### Using Dockerfile

```bash
docker build -t gitlab-otel-exporter .
docker run -e OTEL_EXPORTER_OTLP_ENDPOINT=collector:4318 gitlab-otel-exporter
```

### Using Cloud Native Buildpacks

```bash
pack build gitlab-otel-exporter --builder paketobuildpacks/builder-jammy-base --trust-builder
docker run -e OTEL_EXPORTER_OTLP_ENDPOINT=collector:4318 gitlab-otel-exporter
```

The project includes `project.toml` configuration for Paketo buildpacks with Go 1.25 support.

## License

Apache 2.0

# GitLab Pipeline Examples

This directory contains example GitLab CI/CD pipeline configurations demonstrating downstream pipeline correlation with OpenTelemetry tracing.

## Files

### `parent-pipeline.yml`
**Parent pipeline** that:
- Executes normal CI/CD stages (build, test, deploy)
- Runs OTEL exporter in `.post` stage to generate traces
- Exports `TRACE_PARENT` for downstream correlation
- Triggers child pipeline with trace context

### `child-pipeline.yml`
**Child pipeline** that:
- Receives trace context from parent via `TRACEPARENT` variable
- Shows GitLab's automatic parent pipeline variables
- Executes production deployment
- Creates correlated spans linked to parent trace

## Setup Instructions

### 1. Create Parent Project
```bash
# Create parent project repository
git clone <parent-repo>
cp examples/parent-pipeline.yml .gitlab-ci.yml
git add .gitlab-ci.yml
git commit -m "Add parent pipeline with OTEL tracing"
git push
```

### 2. Create Child Project
```bash
# Create child project repository
git clone <child-repo>
cp examples/child-pipeline.yml .gitlab-ci.yml
git add .gitlab-ci.yml
git commit -m "Add child pipeline with OTEL correlation"
git push
```

### 3. Update Parent Pipeline
Edit `parent-pipeline.yml` and update the trigger target:
```yaml
trigger-downstream:
  trigger:
    project: your-namespace/your-child-project  # Update this
```

### 4. Deploy OTEL Exporter Binary
Copy the `gitlab-otel-exporter` binary to both projects or use a shared container registry.

## Expected Trace Structure

```
Parent Trace: namespace/parent-project #123
├─ build job span
├─ test job span
├─ deploy job span
└─ Child Trace: namespace/child-project #456 (correlated)
   ├─ validate job span
   └─ deploy-prod job span
```

## Environment Variables

Both pipelines use:
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP collector endpoint
- `OTEL_EXPORTER_OTLP_PROTOCOL`: Protocol (http/grpc/stdout)
- `GITLAB_TOKEN`: Automatically provided as `CI_JOB_TOKEN`

Child pipeline receives:
- `TRACEPARENT`: W3C trace context from parent
- `CI_PIPELINE_SOURCE=pipeline`: Indicates triggered pipeline
- `CI_PARENT_PIPELINE_ID`: Parent pipeline ID
- `CI_PARENT_PROJECT_ID`: Parent project ID

## Observability

View the correlated traces in your observability platform:
- **Jaeger**: http://jaeger:16686
- **Zipkin**: http://zipkin:9411
- **SigNoz**: http://signoz:3301
- **Grafana Tempo**: Via Grafana dashboards

The traces will show the complete flow from parent pipeline through child pipeline deployment with proper parent-child relationships.

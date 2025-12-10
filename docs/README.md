# GitLab OpenTelemetry Exporter - Architecture Diagrams

This directory contains PlantUML diagrams illustrating the end-to-end flows of the GitLab OpenTelemetry Exporter.

## Diagrams

### 1. Single Pipeline Flow
**File**: `single-pipeline-flow.puml`

Shows the complete flow for a single GitLab pipeline:
- Pipeline stages execute normally
- OTEL exporter runs in `.post` stage
- Fetches all pipeline/job data via GitLab API
- Creates spans and exports to OTLP collector
- Generates trace context for potential downstream use

### 2. Downstream Pipeline Correlation
**File**: `downstream-pipeline-flow.puml`

Shows distributed tracing across multiple pipelines:
- Parent pipeline generates trace context
- Uses GitLab's `trigger` keyword to start downstream pipeline
- Child pipeline extracts parent trace context
- Creates correlated spans maintaining parent-child relationship
- Results in unified distributed trace

## Key Features Illustrated

- **CI/CD Semantic Conventions**: Proper OpenTelemetry span naming and attributes
- **Trace Context Propagation**: W3C TraceContext standard for correlation
- **GitLab Integration**: Native trigger mechanism and environment variables
- **Complete Observability**: Full pipeline and job metadata as span attributes

## Viewing Diagrams

Use any PlantUML viewer or online renderer:
- [PlantUML Online Server](http://www.plantuml.com/plantuml/uml/)
- VS Code PlantUML extension
- IntelliJ PlantUML plugin

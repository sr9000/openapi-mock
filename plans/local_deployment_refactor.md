# Local deployment refactor plan and dashboard review

## What this refactor changes

- Consolidates local Docker Compose assets under `deploy/local/`.
- Splits responsibilities into two isolated entrypoints:
  - `deploy/local/dev/docker-compose.yaml`
  - `deploy/local/observability/docker-compose.yaml`
- Removes fixed `container_name` usage from the observability stack so Compose project naming can isolate resources cleanly.
- Keeps repo-root `Makefile` targets as the supported entrypoints.

## Dashboard review summary

### Findings

1. The HTTP metrics are now operation-aware:
   - `http_requests_total{method,endpoint,operation,status}`
   - `http_request_duration_seconds{method,endpoint,operation,status}`
2. Error and panic metrics no longer expose a `message` label; they expose `status` and `kind`:
   - `http_errors_total{method,endpoint,operation,status,kind}`
   - `http_panics_total{method,endpoint,operation,status,kind}`
3. Existing dashboards needed an update because some panels still queried the removed `message` label and did not expose the `operation` dimension.

### Changes made in this refactor

- `endpoints-overview.json`
  - switched legends and aggregations to include `operation`
  - replaced `message`-based error/panic queries with `status` + `kind`
- `endpoint-details.json`
  - added an `operation` variable
  - updated all request, error, panic, and latency queries to filter by `operation`
  - replaced `message`-based error/panic queries with `status` + `kind`
- `resources-overview.json`
  - retained as-is functionally; it already matches the exported Go/process resource metrics

## Why no new trace/log dashboard was added

The current Grafana setup already provisions:

- Prometheus datasource for metrics dashboards
- Tempo datasource for trace exploration
- Loki datasource for log exploration

For `openapi-mock`, the primary product requirement is visibility into request rate, latency, errors/panics, and resource usage, with logs and traces available for drill-down. That requirement is satisfied by the current combination of:

- prebuilt metrics dashboards
- provisioned Tempo/Loki datasources
- Grafana Explore for trace/log navigation

## Proof added alongside the refactor

The smoke-validation script now also checks that Grafana provisions:

- `HTTP Mock Metrics`
- `HTTP Mock Traces`
- `HTTP Mock Logs`
- the three shipped dashboards by UID

## Optional future follow-up

If operators later want a single-page stack-health view, add a dedicated dashboard for:

- OTel Collector accepted spans
- Promtail/Loki ingestion health
- Tempo readiness or ingest metrics
- datasource health badges / links into Grafana Explore

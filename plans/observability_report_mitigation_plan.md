# Mitigation plan for issues reported in `reports/observability_verification_report.md`

Date: 2026-05-24
Status: analysis complete, implementation not started

## Scope

This document captures the verified root causes behind the issues reported in `reports/observability_verification_report.md`, and a commit-by-commit mitigation plan.

> Release rule for this sequence: **do not start commit _N+1_ until the Definition of Done (DoD) for commit _N_ has passed in full.**

## What was verified

### Commands run during analysis

```bash
cd /home/sr9000/Documents/dev/openapi-mock && go test ./...
cd /home/sr9000/Documents/dev/openapi-mock && HOST=127.0.0.1 PORT=18080 METRICS_PORT=19100 MGMT_PORT=19000 HTTP_LOGGING=false TRACE_ENABLED=false go run ./cmd/openapi-mock run
curl -sS -D - -X POST http://127.0.0.1:18080/echo -H 'Content-Type: application/json' -d '{"message":"hello"}'
curl -sS http://127.0.0.1:19100/metrics | grep 'http_requests_total{endpoint="/echo"'
curl -sS -D - -X POST http://127.0.0.1:18080/echo -H 'Content-Type: application/json' -d '{invalid-json'
curl -sS http://127.0.0.1:19100/metrics | grep 'http_errors_total{endpoint="/echo"'
cd /home/sr9000/Documents/dev/openapi-mock && docker compose -f docker-compose-grafana.yaml config
```

### Observed results

- `go test ./...` passes on current HEAD.
- Successful `POST /echo` requests are still emitted as:
  - `http_requests_total{endpoint="/echo",method="POST",operation="unknown",status="200"}`
- Request-parse failures are still emitted as:
  - `http_errors_total{endpoint="/echo",kind="request_parse",method="POST",operation="unknown",status="400"}`
- `docker compose -f docker-compose-grafana.yaml config` succeeds, so the current issue is not YAML syntax; it is runtime drift / workflow hardening.

## Root cause analysis

### 1. `operation="unknown"` on successful requests

### Evidence

- `pkg/middleware/operation.go` defines `OperationContext()`, but it has no runtime call sites.
- `cmd/upd-stubs/openapi_wire.go` generates:
  - `NewStrictHandlerWithOptions(strict, nil, ...)`
- The generated runtime files `internal/app/openapi_wire.go` and `internal/app/wire_gen.go` also pass `nil` for strict middlewares.

### Root cause

The strict middleware intended to inject the OpenAPI operation ID is never wired into the generated server construction, so no operation metadata is attached during handler execution.

### Additional root cause that will still exist even after wiring

`pkg/middleware/recording.go` and `pkg/middleware/error_handlers.go` read operation from `observability.Operation(r.Context())`. However, strict middlewares operate on the `ctx context.Context` argument passed through `StrictHTTPHandlerFunc`; they do **not** mutate the outer `*http.Request` instance held by the HTTP middleware.

That means:

- even if `OperationContext()` is finally wired in,
- handler code can see the operation via the strict `ctx`,
- but outer HTTP middleware will still often read an empty operation from `r.Context()` and fall back to `unknown`.

So the real fix is **not only wiring** the middleware; it also requires a shared request-metadata carrier or an explicit operation resolver that outer middleware can read.

---

### 2. Request-parse errors cannot currently resolve the operation name

### Evidence

In generated strict handlers such as `internal/generated/echo/server.gen.go`, request parsing happens **before** the strict middleware chain is executed.

Example flow for `POST /echo`:

1. decode JSON body
2. call `RequestErrorHandlerFunc(...)` on decode error
3. only after that, build/apply `sh.middlewares`

### Root cause

Even after fixing successful-path propagation, request-parse failures will still miss operation metadata because the generated request decoder invokes `RequestErrorHandlerFunc` before strict middleware has a chance to attach the operation.

This means request-parse coverage needs a second mechanism: a `method + route -> operation` resolver available to HTTP middleware and request error handlers.

---

### 3. The observability Compose stack is vulnerable to upstream image/config drift

### Evidence

`docker-compose-grafana.yaml` still uses floating tags for several key services:

- `prom/prometheus:latest`
- `otel/opentelemetry-collector-contrib:latest`
- `grafana/loki:latest`
- `grafana/promtail:latest`

At the same time, the repo currently has no automated Compose smoke test; `Makefile` exposes `compose-up`, `compose-logs`, and `compose-down`, but nothing that validates startup health in CI or before merge.

### Root cause

The stack depends on upstream defaults that can change independently of this repository. Because images are not pinned and startup is only manually verified, collector/exporter/schema regressions can reappear without any code change in this repo.

---

### 4. The reported `docker compose` panic is most likely external CLI instability, but the repo currently drives the risky code path

### Evidence

The report captured a Docker Compose CLI panic: `concurrent map read and map write` during `docker compose ... up --build`.

The repo's default path for the full stack is currently:

```bash
docker compose -f docker-compose-grafana.yaml up --build -d
```

### Root cause

This panic is not caused by Go application code in the repo. It is consistent with a Docker Compose / progress-renderer race in the local CLI/plugin while building and starting multiple services.

However, the repository currently **amplifies** that risk by making `up --build` the default documented and automated path instead of separating build from startup or forcing plain progress output.

---

### 5. The sample Echo API returns empty payloads because the sample stubs intentionally ignore inputs

### Evidence

`internal/stubs/echo/echo.go` currently does the following:

- ignores `request`
- returns zero-value responses such as `gen.Echo200JSONResponse{}`

### Root cause

This is not an observability bug. It is sample stub behavior. The stub implementation never copies request/path/header values into the response object, so verification traffic gets empty-but-valid-looking JSON such as `{"echo":""}`.

This makes manual observability verification harder because the sample responses do not prove semantic correctness.

---

## Commit plan

## Commit 1 — propagate operation metadata on successful handler paths

### Goal

Make successful requests produce concrete OpenAPI operation names in metrics, traces, and access logs.

### Changes

- Introduce a shared per-request metadata carrier in `pkg/observability`.
  - Minimum fields: `request_id`, `trace_id`, `operation`.
  - Prefer a mutable holder stored in the original request context so inner strict middleware and outer HTTP middleware can observe the same state.
- Initialize that metadata carrier in `pkg/middleware/recording.go`.
- Update `pkg/middleware/operation.go` so strict middleware writes the resolved operation into the shared metadata carrier.
- Update `pkg/middleware/recording.go` to read operation from the shared carrier instead of relying only on `r.Context()`.
- Update `cmd/upd-stubs/openapi_wire.go` so generated providers pass `middleware.OperationContext()` into `NewStrictHandlerWithOptions(...)`.
- Regenerate `internal/app/openapi_wire.go` and `internal/app/wire_gen.go`.
- Add tests that cover at least:
  - successful `POST /echo` -> operation `Echo`
  - successful `GET /echo/{message}` -> operation `EchoPath`
  - successful `GET /status` -> operation `GetStatus`

### Definition of Done (DoD)

- `go test ./...` passes.
- A local smoke request to `POST /echo` records `operation="Echo"` instead of `unknown` in `/metrics`.
- Access logs and trace attributes for a successful request use the same concrete operation name.
- No generated file needs a manual fix after running the generator and Wire.

### Verification commands

```bash
go test ./...
make stub wire build
HOST=127.0.0.1 PORT=18080 METRICS_PORT=19100 MGMT_PORT=19000 TRACE_ENABLED=false HTTP_LOGGING=true ./bin/openapi-mock run
curl -sS -X POST http://127.0.0.1:18080/echo -H 'Content-Type: application/json' -d '{"message":"hello"}'
curl -sS http://127.0.0.1:19100/metrics | grep 'http_requests_total{endpoint="/echo"'
```

### Gate to proceed

Do **not** start Commit 2 until all DoD checks above pass.

---

## Commit 2 — resolve operation names for request-parse failures and other pre-handler errors

### Goal

Remove `operation="unknown"` from request-parse error metrics for valid OpenAPI routes.

### Changes

- Generate a deterministic `method + route -> operation` registry from OpenAPI specs inside `cmd/upd-stubs/openapi_wire.go`.
- Expose that registry to runtime code through `internal/app/openapi_wire.go`.
- Inject an operation resolver into `pkg/middleware/error_handlers.go` and `pkg/middleware/recording.go`.
- Resolve operation from:
  1. shared request metadata, if already known;
  2. otherwise the generated registry using HTTP method + matched chi route pattern.
- Add tests for at least:
  - invalid JSON on `POST /echo` -> `request_parse` metric labeled `Echo`
  - invalid path/query parsing on a route with parameters -> concrete operation label
  - unmatched paths still remain `unknown` by design

### Definition of Done (DoD)

- `go test ./...` passes.
- Invalid JSON sent to `POST /echo` increments `http_errors_total` with `operation="Echo"`.
- Request-parse failures on parameterized routes resolve to the correct OpenAPI operation.
- Truly unmatched requests still use `unknown`, so the fallback remains explicit and intentional.

### Verification commands

```bash
go test ./...
HOST=127.0.0.1 PORT=18080 METRICS_PORT=19100 MGMT_PORT=19000 TRACE_ENABLED=false HTTP_LOGGING=false ./bin/openapi-mock run
curl -sS -X POST http://127.0.0.1:18080/echo -H 'Content-Type: application/json' -d '{invalid-json'
curl -sS http://127.0.0.1:19100/metrics | grep 'http_errors_total{endpoint="/echo"'
```

### Gate to proceed

Do **not** start Commit 3 until all DoD checks above pass.

---

## Commit 3 — pin observability stack versions and align configs with those versions

### Goal

Stop regressions caused by upstream image changes.

### Changes

- Replace floating `:latest` tags in `docker-compose-grafana.yaml` with explicitly chosen versions for:
  - Prometheus
  - OTel Collector Contrib
  - Loki
  - Promtail
  - optionally Grafana as well, if not already controlled via the local Dockerfile base image
- Align `otel/collector.yaml`, `tempo/tempo.yaml`, `promtail/promtail.yaml`, and Grafana datasource config with the pinned versions.
- Remove any deprecated collector exporter/config aliases still present for the selected collector version.
- Add service health checks where they materially improve startup diagnostics.

### Definition of Done (DoD)

- `docker compose -f docker-compose-grafana.yaml config` passes.
- A compose smoke run brings up all observability services with the pinned versions.
- Collector startup logs contain no known config-schema failures or deprecation warnings for the chosen config.
- The stack still accepts app metrics, traces, and logs end-to-end.

### Verification commands

```bash
docker compose -f docker-compose-grafana.yaml config
docker compose -f docker-compose-grafana.yaml up -d
docker compose -f docker-compose-grafana.yaml ps
docker compose -f docker-compose-grafana.yaml logs --no-color otel-collector | tail -n 50
docker compose -f docker-compose-grafana.yaml down
```

### Gate to proceed

Do **not** start Commit 4 until all DoD checks above pass.

---

## Commit 4 — add automated observability stack smoke validation

### Goal

Make stack regressions fail fast instead of depending on manual verification.

### Changes

- Add `scripts/validate-observability-stack.sh` (or equivalent) that:
  1. builds the app image;
  2. starts the Compose stack;
  3. waits for Prometheus / Grafana / Loki / Tempo / collector / app health;
  4. issues one traced request and one plain request;
  5. verifies:
     - `/metrics` scrape works,
     - Prometheus reports `up{job="openapi-mock"} == 1`,
     - collector receives/export traces,
     - Loki receives application logs,
     - response contains request ID header;
  6. tears the stack down.
- Add a `make compose-smoke` target.
- Document the smoke target in `README.md`.

### Definition of Done (DoD)

- `make compose-smoke` passes from a clean checkout.
- The smoke script exits non-zero on missing metrics, traces, logs, or request ID propagation.
- The smoke script runs unattended and is suitable for CI.

### Verification commands

```bash
make compose-smoke
```

### Gate to proceed

Do **not** start Commit 5 until all DoD checks above pass.

---

## Commit 5 — harden local developer workflow against Docker Compose CLI race conditions

### Goal

Reduce exposure to the `docker compose up --build` progress-renderer panic seen in the report.

### Changes

- Change the repo workflow so `make compose-up` does **not** rely on a single `docker compose up --build -d` call.
- Prefer a safer sequence such as:
  1. `docker compose build --progress plain`
  2. `docker compose up -d`
- Optionally export plain-progress defaults for documented workflows.
- Document the known issue as an external CLI problem, plus the tested Compose plugin/version range.

### Definition of Done (DoD)

- `make compose-up` no longer invokes `docker compose ... up --build -d` directly.
- The documented path uses plain progress or an equivalent non-TUI build flow.
- Local contributor docs clearly separate repository issues from Docker CLI/plugin issues.

### Verification commands

```bash
grep -n 'compose-up' Makefile
make compose-up
```

### Gate to proceed

Do **not** start Commit 6 until all DoD checks above pass.

---

## Commit 6 — make the sample Echo stub semantically useful for verification

### Goal

Remove confusion from empty sample responses during manual verification.

### Changes

- Update `internal/stubs/echo/echo.go` so sample handlers actually echo:
  - JSON/text body for `POST /echo`
  - path value for `GET /echo/{message}`
  - selected headers for `GET /echo/headers`
- Add tests for the sample behavior.
- Update `README.md` examples if response bodies change.

### Definition of Done (DoD)

- `go test ./...` passes.
- `POST /echo` returns the submitted message in the `echo` field.
- `GET /echo/{message}` returns the path message in the `echo` field.
- The sample API is now useful as an end-to-end verification target for observability demos.

### Verification commands

```bash
go test ./...
HOST=127.0.0.1 PORT=18080 METRICS_PORT=19100 MGMT_PORT=19000 TRACE_ENABLED=false HTTP_LOGGING=false ./bin/openapi-mock run
curl -sS -X POST http://127.0.0.1:18080/echo -H 'Content-Type: application/json' -d '{"message":"hello"}'
curl -sS http://127.0.0.1:18080/echo/hello
```

### Gate to proceed

This is the last planned commit in the sequence.

---

## Priority order

If only the blocking observability issues should be addressed now, execute commits in this order:

1. Commit 1
2. Commit 2
3. Commit 3
4. Commit 4
5. Commit 5

Commit 6 is recommended but non-blocking for the observability platform itself.

## Already-fixed item from the original verification run

The earlier `oapi-codegen/runtime` mismatch described in the report appears to have already been handled in the current workspace and is **not** part of the remaining mitigation sequence.

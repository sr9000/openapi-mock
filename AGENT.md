# AGENT notes for `openapi-mock`

## Snapshot

`openapi-mock` is currently the more mature operator-facing repo of the pair.

It already provides:

- recursive OpenAPI discovery,
- generated code isolated in `internal/generated/`,
- hand-written logic isolated in `internal/stubs/`,
- modular stub updater with tests,
- rich management API,
- structured logging + metrics + traces,
- full observability compose stack,
- smoke validation for that stack.

## Strengths worth preserving

1. **Best control-plane story**
    - logs, request-id lookup, context values, reset, per-API/per-version docs
2. **Best observability story**
    - Prometheus + Grafana + Loki + Tempo + OTel Collector + Promtail
3. **Best updater maturity**
    - modular `cmd/upd-stubs/`
    - explicit CLI flags
    - meaningful test coverage
4. **Clean separation of concerns**
    - middleware owns request recording, request-id propagation, tracing, and structured access logging

## Gaps versus `grpc-mock`

1. **Dev loop is weaker**
    - `scripts/run-dev.sh` rebuilds and runs, but does not mirror the watcher-based hot reload flow used in `grpc-mock`
2. **Operator surface is richer, but not yet aligned**
    - flag semantics differ from `grpc-mock` (`--mgmt-enabled` vs `--no-mgmt`, etc.)
3. **Higher complexity cost**
    - more moving pieces in observability and updater pipeline
    - stronger platform, but also more maintenance overhead

## Recommendations for unification work

### Priority 1

- Use this repo as the baseline for the shared management API contract.
- Document a minimum common control-plane surface that `grpc-mock` should grow toward.
- Preserve the repo-owned smoke automation pattern.

### Priority 2

- Improve dev ergonomics to match `grpc-mock`:
    - watcher-based local/dev flow,
    - automatic regen/restart when `api/**` or stub code changes.

### Priority 3

- Normalize CLI and config semantics with `grpc-mock` where transport-independent:
    - flag naming conventions,
    - env-var naming patterns,
    - README section ordering.

## Extra findings

- The repo includes practical operational knowledge around Compose instability by separating build/up in the full stack
  flow; that pattern is worth copying where needed.
- `make compose-smoke` is a strong precedent for repo-level end-to-end validation.
- Per-API/per-version docs handling is a useful model for any future gRPC equivalent documentation/indexing concept,
  even if the exact endpoints differ.

## Suggested role in the unified direction

Let `openapi-mock` be the reference implementation for:

- management API maturity,
- observability depth,
- updater UX/testing discipline,
- smoke automation.

Keep transport-specific HTTP/OpenAPI machinery local to this repo, but use its operator experience as the benchmark for
convergence with `grpc-mock`.

# AGENT notes for `openapi-mock`

## Contracts

This repo implements the contracts defined in `../CONTRACTS.md` (workspace root). See that file for the unified CLI, Management API, Recorder JSON, Metrics, Make target, Stub Updater, and Docs layout contracts.

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

The genuinely-open items; the full cross-repo list lives in `../CONTRACTS.md` §8.

1. **Fewer CLI knobs** — logging/tracing are configurable via **env only**. `grpc-mock` also exposes
   `--log-format/-output/-file/-level` and `--trace-*` flags. Either add the flags here or document env-only intentionally.
2. **Dev loop parity** — `make docker-dev` is described as watch mode, but the documented primary flow still rebuilds
   via `make all`. Confirm whether true file-watch reload is wired, and align the docs/help text accordingly.
3. **Flag-name drift** — request logging is `--http-logging` (with `--logging` alias) vs `grpc-mock`'s `--logging`.
   Standardise docs on the common `--logging`.
4. **Higher complexity cost** — more moving pieces in observability and the updater pipeline; stronger platform but
   more maintenance overhead.
5. **Docs table nit** — the management-API "Extended" table lists `/metrics`, but metrics are served on the dedicated
   `METRICS_PORT` (9100), not by the management server. Make the separate port explicit.

## Recommendations for unification work

### Priority 1

- Use this repo as the baseline for the shared management API contract.
- Document a minimum common control-plane surface that `grpc-mock` should grow toward.
- Preserve the repo-owned smoke automation pattern.

### Priority 2

- Improve dev ergonomics to match `grpc-mock`:
    - watcher-based local/dev flow,
    - automatic regen/restart when `api-specs/**` or stub code changes.

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

# Repository Improvement Review — Usage & Workflow

Date: 2026-06-02
Scope: Deep review of `openapi-mock` against three goals plus the overriding "spec → working endpoint" speed goal.

## Goals under evaluation

1. **Fast code-driven mock building** (spec → generated, compilable mock).
2. **Automation-friendly management** (drive mocks from integration tests without redeploys).
3. **Observability** (traces, logs, metrics visible in Grafana).
4. **Main goal:** minimize wall-clock + manual steps from editing a spec to a working mock endpoint.

Overall the project is well-structured and already strong on observability. The biggest opportunities are in
**goal 1 / main goal**: the generation pipeline produces *empty* response bodies and requires several manual steps,
non-reproducible tooling, and does not re-trigger on spec changes.

---

## 🔴 High-impact (directly hurt the main goal)

### CANCELLED ~~H1. Generated stubs return EMPTY response bodies~~

> REASON: Avoid too much magic in the generator. Stub update already magical enough.

**Where:** `cmd/upd-stubs/openapi_stubs.go` → `generateOpenAPIMethodBody`.

The generated handler body is:

```go
return gen.ListPets200JSONResponse{}, nil // empty/zero value
```

A freshly generated endpoint returns an **empty JSON body** (`{}`, `[]`, or zero-valued struct). To get a useful
mock, a developer must hand-edit *every* handler (as was done manually in `internal/stubs/petstore/pets.go`). This is
the single largest source of "spec → working endpoint" friction.

**Recommendation (biggest win):** Populate the generated body from the spec automatically, in priority order:

1. Response `example` / `examples` (OpenAPI explicitly provides these).
2. Schema `example` values on properties.
3. Schema-faithful synthetic data (respect `type`, `format`, `enum`, `required`, arrays → 1–2 items).

Even option 1 alone would make most documented endpoints return realistic payloads with **zero** manual edits.
`kin-openapi` (already a dependency) exposes `Example`, `Examples`, and full schema info, so this is achievable inside
the existing generator. Marshal the example to JSON and emit it as the response body literal (or a `json.RawMessage`).

Impact: removes the dominant manual step. This is the highest-leverage change in the whole repo.

### H2. Tooling is non-reproducible and a manual prerequisite (`oapi-codegen@latest`, `wire@latest`)

**Where:** `scripts/gen-openapi.sh` (calls `oapi-codegen` from `PATH`), `Makefile` `wire` target
(`go run github.com/google/wire/cmd/wire@latest`), `Dockerfile` (`go install ...@latest`).

Problems:

- The README lists `oapi-codegen` install as a **manual prerequisite** — extra friction for new users.
- `@latest` is **non-reproducible**: generated files were produced with `v2.5.1`, but a fresh `make all` may pull a
  newer version and silently change output (breaking `git diff --exit-code` stability checks from the plan).

**Recommendation:** Pin both tools as Go module tool dependencies (Go 1.25 supports the `tool` directive in `go.mod`):

```go
// go.mod
tool (
github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
github.com/google/wire/cmd/wire
)
```

Then invoke via `go tool oapi-codegen ...` / `go tool wire ...`. This removes the manual install, guarantees a pinned
version, and works identically locally and in Docker.

### RETHINK H3. Dev watcher does NOT react to spec changes

> TODO: is it really required or is it enough to run `make all` manually after spec edits? The plan lists "ensure
> openapi.yaml changes trigger regen" as a recurring DoD item, but the current setup only watches Go files.
> And maybe even drop the watcher entirely in favor of "run `make all` when you change a spec" — the current watcher is
> already a bit magical and not well-documented.

**Where:** `.air.go.toml` → `include_ext = ["go"]`, and `scripts/run-dev.sh` only runs `make build`.

In dev mode (`make docker-dev`), editing `api/**/openapi.yaml` does **not** regenerate code. The developer must
manually run the full `make all` pipeline. This breaks the "edit spec → see mock update" loop.

**Recommendation:** Add spec watching. Either:

- Extend air to watch `yaml`/`yml` under `api/` and run `make all` (not just `make build`) when specs change; or
- Add a dedicated `make watch` target (e.g. a second air config `.air.spec.toml` whose `cmd = "make all"` and
  `include_ext = ["yaml","yml","json"]`, scoped to `api/`).

Impact: enables a true hot-reload loop from spec to endpoint.

### H4. `test-mgmt.sh` is broken (references removed `/clear`)

**Where:** `scripts/test-mgmt.sh` lines 112 & 148 call `POST ${MGMT_URL}/clear`; the header comment also lists
`/clear`.

Commit 1 of the management-API plan **removed** `POST /clear` (it now returns 404). This script will now fail at
`test_logs_empty_initially` / `test_clear_logs`. This is a real regression in a checked-in test script.

**Recommendation:** Replace `POST /clear` with `DELETE /logs` and update the comment. Consider adding this script to
CI so such drift is caught automatically.

---

## 🟡 Medium-impact

### CANCELLED ~~M1. No way to drive responses from integration tests without recompiling (goal 2)~~

> REASON: By design this repo uses code-driven approach rather than "programing-inside-json".
> IFF protocol and mock behavior is a simple enough - wiremock will be used instead.

The `mm` context-value store + management API is a great primitive, but using it still requires **hand-written Go**
(`mm.FromCtx(ctx, "low")`) in each handler. For integration tests, the high-value capability is overriding a response
(status + body + headers) per request-id **without touching code**.

**Recommendation:** Add a generic response-override layer driven by the management API, e.g.:

- `PUT /responses/{operationId}` or `PUT /responses/{request_id}` with `{ "status": 404, "body": {...},
  "headers": {...} }`.
- A middleware (or a thin wrapper in the strict-handler chain) consults the override store keyed by request-id and/or
  operation before delegating to the generated stub.

This combines goals 1 & 2: tests can shape any endpoint's behavior at runtime, and the default generated body (H1)
covers the happy path. The existing `mm.Store` copy-semantics and `/reset` plumbing are a ready foundation.

### RETHINK M2. Management OpenAPI document is hand-maintained

> TODO: By desing management API is static. Maybe smoke test that tightly coupled with spec is enough to prevent drift.

**Where:** `pkg/mgmt/openapi.json` is edited by hand and must be kept in sync with `server.go` routes.

This drifts easily (the plan even lists "ensure openapi.json matches" as a recurring DoD item).

**Recommendation:** Either generate the management spec from a typed route table, or add a test that asserts every
registered chi route is present in `openapi.json` (and vice-versa) to prevent drift mechanically.

### M3. Recorder stores request/response as opaque strings; doc shows structured

**Where:** `pkg/middleware/recording.go` records `Request: string(bodyBytes)` and `Response: rw.body.String()`, but
`README.md` documents a structured shape `{ "query": ..., "body": ... }`.

For integration-test assertions, structured fields (method, path, query, status code, parsed JSON body) are far more
useful than a single string, and the doc currently misrepresents the payload.

**Recommendation:** Record structured fields (status code is currently **not** stored in `CallRecord` at all — only
embedded in `Method`). Add `StatusCode int`, `Path string`, `Query string`, and store bodies as `json.RawMessage`
when JSON. Then update README to match. This also helps Grafana/log correlation.

### CANCELLED ~~M4. No scaffold for a new API~~

> REASON: do not automate what is already simple enough.

Adding an API is "create the folder and YAML by hand." A one-liner reduces friction and enforces the expected layout.

**Recommendation:** Add `make new-api NAME=foo [VER=v1]` that scaffolds `api/foo/openapi.yaml` (or
`api/foo/v1/openapi.yaml`) from a minimal template, then optionally runs `make all`.

### M5. Dead/placeholder generator config

**Where:** `scripts/oapi-codegen.yaml` contains `output: #!FIXME please` and `# NOTE another server must be added!`
— it is unused (the bash loop in `gen-openapi.sh` passes flags directly).

**Recommendation:** Either adopt config-file-driven generation (cleaner, supports per-API options, response type
maps) and delete the bash flag duplication, or remove the dead file to avoid confusion.

---

## 🟢 Lower-impact / polish

### L1. README documentation drift

- `pkg/ctxkeys/` is referenced (README lines 25, 120) but **does not exist** — the helper is now in
  `pkg/observability`. Update the structure diagram and the Request-ID helper note.
- The recorded-call JSON example (README ~line 278) does not match the actual recorder output (see M3).

### L2. Trace span naming uses raw URL path

**Where:** `recording.go` line 78: `opts.Tracer.Start(ctx, r.Method+" "+r.URL.Path, ...)`. Using the concrete path
(`/pets/123`) produces high-cardinality span names. The route template is already computed later as `pathLabel`.

**Recommendation:** Rename the span to the route template once known (chi exposes it post-routing), or set
`http.route` consistently and keep span name low-cardinality (e.g. `GET /pets/{petId}`).

### L3. No in-flight / active-requests metric

The metrics set is solid (RPS, duration, errors, panics, resources) but lacks an in-flight gauge, which is useful on
Grafana to spot stuck/slow handlers during load tests.

### L4. Reset error-handling edge case (carried over from prior review)

`cmd/openapi-mock/runtime.go` `stopLocked` sets `r.server = nil` even when `Shutdown` returns an error, which can
leave the runtime believing it can rebind a still-occupied port on the next `Reset`. Guard the nil-out behind success,
or verify the listener is fully closed.

### L5. `GET /docs/{api_name}` ambiguous case returns JSON only

Browser-facing UI route returns a JSON version index rather than an HTML page (plan allowed HTML/JSON). Minor UX.

---

## Suggested prioritization (by impact on the main goal)

| Priority | Item                                                          | Goal served | Effort  |
|:---------|:--------------------------------------------------------------|:------------|:--------|
| 1        | **H1** Generate response bodies from spec examples/schema     | 1, main     | Medium  |
| 2        | **H2** Pin `oapi-codegen`/`wire` via `go.mod tool` directives | 1, main     | Low     |
| 3        | **H3** Watch `api/**` specs in dev (hot reload)               | 1, main     | Low     |
| 4        | **H4** Fix broken `test-mgmt.sh` (`/clear` → `DELETE /logs`)  | 2           | Trivial |
| 5        | **M1** Runtime response overrides via management API          | 1, 2        | Medium  |
| 6        | **M3** Structured recorder records (+ status code)            | 2, 3        | Medium  |
| 7        | **M2** Prevent management OpenAPI drift (test or codegen)     | 2           | Low     |
| 8        | **M4 / M5 / L1–L5** Scaffold, cleanup, doc & metric polish    | all         | Low     |

## The "ideal" fast path after these changes

```bash
# 1. Scaffold + edit one YAML
make new-api NAME=orders            # H4/M4

# 2. Everything else is automatic in dev mode
make docker-dev                     # H3 watches api/** → regenerates → rebuilds
#    -> endpoints already return realistic example data (H1), reproducibly (H2)

# 3. Integration test shapes behavior at runtime, no recompile
curl -X PUT localhost:9000/responses/GetOrderById -d '{"status":404,"body":{"error":"nope"}}'  # M1
curl localhost:8080/orders/5 -H 'X-Request-ID: case-404'
curl localhost:9000/logs/case-404 | jq          # structured records (M3)
```

This collapses the current multi-command, hand-edit-every-handler flow into "edit spec → it works", which is exactly
the main goal.

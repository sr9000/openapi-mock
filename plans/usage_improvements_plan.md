# Repository Improvement Review — Usage & Workflow

Date: 2026-06-02 (re-evaluated after maintainer review)
Scope: Deep review of `openapi-mock` against three goals plus the overriding "spec → working endpoint" speed goal.

## Goals under evaluation

1. **Fast code-driven mock building** (spec → generated, compilable mock).
2. **Automation-friendly management** (drive mocks from integration tests without redeploys).
3. **Observability** (traces, logs, metrics visible in Grafana).
4. **Main goal:** minimize wall-clock + manual steps from editing a spec to a working mock endpoint.

Overall the project is well-structured and already strong on observability.

## Re-evaluation summary (maintainer decisions)

The plan was reviewed and several items were reshaped to match the project's **code-driven, low-magic** philosophy
("don't automate what is already simple"; behavior lives in Go, not in JSON; reach for WireMock when a protocol is
trivial). Net result:

| Item                           | Decision                                  | Outcome                                                            |
|:-------------------------------|:------------------------------------------|:-------------------------------------------------------------------|
| H2 — pin `oapi-codegen`/`wire` | ✅ Keep                                    | Reproducible, no manual prereq                                     |
| H3 — dev watcher               | 🔄 Re-scoped → **drop the watcher**       | Remove air-based hot-reload; `make all` is the one manual step     |
| H4 — fix broken `test-mgmt.sh` | ✅ Keep                                    | Real regression (`/clear` → 404)                                   |
| M2 — mgmt OpenAPI drift        | 🔄 Re-scoped → **static live smoke test** | Hit every documented endpoint on a running server                  |
| M3 — recorder shape            | ✅ Keep (full)                             | Add structured fields + status code, update README                 |
| M5 — dead generator config     | ✅ Keep                                    | Remove/replace placeholder `scripts/oapi-codegen.yaml`             |
| L1–L5 — polish                 | ✅ Keep                                    | Doc drift, span naming, in-flight metric, reset edge case, docs UX |

The guiding principle after re-evaluation: **reduce *pipeline* friction (one reproducible `make all`), not endpoint
behavior** — behavior is intentionally hand-written.

---

## 🔴 High-impact (directly hurt the main goal)

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

### DECIDED H3. Drop the dev watcher entirely

> DECISION: Remove the air-based hot-reload. It is "magical", under-documented, and only watches Go files anyway.
> Dev mode should simply build + run; re-running `make all` after any spec/code change is the single, explicit step.
> This matches the code-driven, low-magic philosophy and removes a moving part.

**Where:** `.air.go.toml`, `scripts/run-dev.sh` (`exec air ...`), `Dockerfile` dev stage (`go install air@latest`),
`docker-compose.dev.yaml` (`target: dev` → `CMD ["./scripts/run-dev.sh"]`).

**Why the original "add spec watching" idea was rejected:** it added more automation (a second air config / extra
include globs) to chase a loop the team is fine running manually. Editing a spec is infrequent and `make all` is fast.

**Recommended changes (to implement later):**

- Delete `.air.go.toml` and the air install from the `tools` stage of the `Dockerfile`.
- Replace `scripts/run-dev.sh` with a plain `make all && ./bin/openapi-mock run 0.0.0.0 8080` (or drop the script and
  set the dev stage `CMD` directly).
- Keep `docker-compose.dev.yaml` for the mounted-volume + ports convenience, but without hot-reload semantics.
- Document in README: "after editing `api/**` or stubs, run `make all`."

Impact: simpler, more predictable dev environment; one obvious regeneration command instead of an implicit watcher.

### H4. `test-mgmt.sh` is broken (references removed `/clear`)

**Where:** `scripts/test-mgmt.sh` lines 112 & 148 call `POST ${MGMT_URL}/clear`; the header comment also lists
`/clear`.

Commit 1 of the management-API plan **removed** `POST /clear` (it now returns 404). This script will now fail at
`test_logs_empty_initially` / `test_clear_logs`. This is a real regression in a checked-in test script.

**Recommendation:** Replace `POST /clear` with `DELETE /logs` and update the comment. Consider adding this script to
CI so such drift is caught automatically.

---

## 🟡 Medium-impact

### DECIDED M2. Guard management OpenAPI with a static live smoke test

> DECISION: The management API is static by design, so keep `openapi.json` hand-written but pin it down with a smoke
> test that starts a real server and hits **every documented endpoint**, asserting it does not 404/405. This catches
> "documented-but-unrouted" and obvious drift without introducing route-table codegen.

**Where:** `pkg/mgmt/openapi.json` (hand-maintained) vs. `pkg/mgmt/server.go` routes.

**Recommended approach (to implement later):**

- Extend the (now-fixed, see H4) `scripts/test-mgmt.sh` — or add a Go-level `httptest` smoke test — that enumerates
  the paths/methods declared in `openapi.json` and issues a representative request to each against a live server.
- Assert each documented operation returns a non-`404`/non-`405` status (i.e. the route exists and the method is
  allowed). Path params can use a placeholder value (e.g. `/logs/smoke-id`, `/docs/petstore`).
- Optionally assert the reverse direction lightly (no obviously-undocumented top-level route), but the primary goal is
  "everything documented is actually served."

This synergizes with **H4**: the same script becomes the single source of truth for management-endpoint coverage.

**Rejected alternative:** generating `openapi.json` from a typed route table — more codegen machinery than a static,
rarely-changing API warrants.

### DECIDED M3. Recorder stores opaque strings; move to structured records

> DECISION: Adopt the full structured shape (status code + path + query + JSON body) and update the README to match.
> This is the most valuable change for goal 2 (integration-test assertions) and goal 3 (Grafana/log correlation).

**Where:** `pkg/middleware/recording.go` records `Request: string(bodyBytes)` and `Response: rw.body.String()`, but
`README.md` documents a structured shape `{ "query": ..., "body": ... }`. Also note the HTTP **status code is not
stored as its own field** today — it is only embedded inside the `Method` string.

**Recommended changes (to implement later):**

- Extend `recorder.CallRecord` with `StatusCode int`, `Path string`, `Query string`; store request/response bodies as
  `json.RawMessage` when the content is JSON (fall back to string otherwise).
- Populate these in `recording.go` (status is already available via the `responseWriter.statusCode`; path/query from
  the route template + `r.URL`).
- Update the README recorded-call example to match the real payload.
- Keep backward-compatible JSON field names where possible so existing `/logs` consumers don't break.

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

## Suggested prioritization (accepted items only, by impact on the main goal)

| Priority | Item                                                          | Goal served | Effort  |
|:---------|:--------------------------------------------------------------|:------------|:--------|
| 1        | **H2** Pin `oapi-codegen`/`wire` via `go.mod tool` directives | 1, main     | Low     |
| 2        | **H4** Fix broken `test-mgmt.sh` (`/clear` → `DELETE /logs`)  | 2           | Trivial |
| 3        | **M2** Static live smoke test over all documented mgmt routes | 2           | Low     |
| 4        | **M3** Structured recorder records (+ status code) + README   | 2, 3        | Medium  |
| 5        | **H3** Drop the dev watcher; document `make all`              | 1, main     | Low     |
| 6        | **M5** Remove/replace dead `scripts/oapi-codegen.yaml`        | 1           | Trivial |
| 7        | **L1** Fix README drift (`pkg/ctxkeys`, record example)       | docs        | Trivial |
| 8        | **L2–L5** Span naming, in-flight metric, reset edge, docs UX  | 3, polish   | Low     |

Implementation status: **report-only for now** — no code changes applied yet, per maintainer decision.

## The "ideal" fast path after the accepted changes

Aligned with the code-driven philosophy: shrink the **pipeline**, keep behavior hand-written.

```bash
# 1. Add / edit one spec by hand
$EDITOR api/orders/openapi.yaml

# 2. One reproducible command regenerates types/stubs/wire and builds
make all                     # H2: pinned oapi-codegen + wire, no manual installs

# 3. Implement handler behavior in Go (intentional, not generated)
$EDITOR internal/stubs/orders/orders.go

# 4. Run — observable out of the box (traces, logs, metrics)
./bin/openapi-mock run

# 5. Drive integration tests via the existing management API (code-driven scenarios)
curl -X PUT localhost:9000/context-values/case-404 -d '{"...":"..."}'
curl localhost:8080/orders/5 -H 'X-Request-ID: case-404'
curl localhost:9000/logs/case-404 | jq    # structured records incl. status code (M3)
```

The optimization target is the **number of distinct manual commands and prerequisites** between a spec edit and a
running, observable endpoint — reduced here to "edit YAML → `make all` → edit handler → run", with reproducible
tooling and no hidden watcher.

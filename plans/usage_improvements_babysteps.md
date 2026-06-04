# Usage & Workflow Improvements — Iterative Baby-Step Plan

Derived from `usage_improvements_plan.md`. This document converts the accepted proposal items into an
ordered sequence of **small, independently shippable steps**, each driven by the same workflow loop.

## Workflow Loop (applies to every step)

1. **Implement** one small, logical unit of work (a single step below).
2. **Run DoD Verification — in order:**
    1. **Lint:** `gofmt -l .` (must print nothing) **and** `go vet ./...`
    2. **Test:** `go test ./...`
    3. **Extra Check:** the step-specific command listed under each step.
3. **Decide:**
    - ✅ **All pass →** `git commit` immediately with the step's commit message, then move to the next step
      without pausing.
    - ❌ **Any fail →** fix locally and re-run the full verification from the top. **Do not commit** until Lint,
      Test, and Extra Check all pass.

### DoD Verification reference commands

```bash
# Lint
gofmt -l .            # expect: no output
go vet ./...

# Test
go test ./...

# Extra Check (varies per step — see each step)
```

> Note: no `golangci-lint`/CI exists in the repo today. `gofmt -l .` + `go vet ./...` is the baseline lint gate.
> If a step introduces CI or a linter, later steps may upgrade this gate.

### Priority order

The steps follow the accepted prioritization from the proposal: **H2 → H4 → M2 → M3 → H3 → M5 → L1 → L2–L5.**
Each numbered phase is a separate commit (some phases are split where a smaller commit is safer).

---

## Phase 1 — H2: Pin `oapi-codegen` and `wire` (reproducible tooling)

Goal: remove `@latest` / manual-install non-reproducibility. Serves goal 1 + main goal.

### Step 1.1 — Add `tool` directives + pin versions in `go.mod`

- Add a `tool (...)` block to `go.mod` for
  `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` and `github.com/google/wire/cmd/wire`.
- Pin `oapi-codegen` to the version that produced the current generated files (**v2.5.1**) to keep `*.gen.go`
  byte-stable; pin `wire` to its current `v0.7.0`.
- Run `go mod tidy`.
- **Extra Check:** `go tool oapi-codegen --version` prints `v2.5.1` and `go tool wire help` runs without error.
- **Commit:** `build: pin oapi-codegen and wire via go.mod tool directives`

### Step 1.2 — Switch generation scripts/Makefile/Dockerfile to `go tool`

- `scripts/gen-openapi.sh`: replace bare `oapi-codegen` calls with `go tool oapi-codegen`.
- `Makefile` `wire` target: replace `go run github.com/google/wire/cmd/wire@latest` with `go tool wire`.
- `Dockerfile`: replace `go install ...@latest` of these tools with reliance on `go tool` (and drop manual
  install steps).
- Update `README.md` to remove the manual `oapi-codegen` install prerequisite.
- **Extra Check:** `make all && git diff --exit-code internal/generated internal/app/wire_gen.go`
  (regeneration is reproducible and produces no diff).
- **Commit:** `build: invoke pinned oapi-codegen/wire via go tool`

---

## Phase 2 — H4: Fix broken `test-mgmt.sh` (`/clear` → `DELETE /logs`)

Goal: repair real regression (removed `POST /clear` now 404s). Serves goal 2.

### Step 2.1 — Replace `/clear` usage with `DELETE /logs`

- `scripts/test-mgmt.sh`: change the two `POST ${MGMT_URL}/clear` calls (lines ~112 & ~148) to
  `DELETE ${MGMT_URL}/logs`; update the header comment that lists `/clear`.
- **Extra Check:** start the server, then run `./scripts/test-mgmt.sh` against it — all cases pass (no 404 on
  log clearing).
- **Commit:** `test: fix test-mgmt.sh to use DELETE /logs instead of removed /clear`

---

## Phase 3 — M2: Static live smoke test over documented mgmt routes

Goal: catch "documented-but-unrouted" drift. Serves goal 2. Build on the now-fixed script from Phase 2.

### Step 3.1 — Enumerate documented endpoints and assert they are served

- Add a smoke test (preferred: a Go `httptest`-based test next to `pkg/mgmt/server_test.go`, e.g.
  `pkg/mgmt/openapi_smoke_test.go`) that reads paths/methods from `pkg/mgmt/openapi.json`, issues a
  representative request to each against a live in-process server, and asserts the response is **not 404/405**.
- Use placeholder values for path params (`/logs/smoke-id`, `/docs/petstore`).
- **Extra Check:** `go test ./pkg/mgmt/...` (the new smoke test passes).
- **Commit:** `test: add live smoke test asserting all documented mgmt routes are served`

---

## Phase 4 — M3: Structured recorder records (+ status code) + README

Goal: most valuable for goals 2 & 3. Split into model change, then population, then docs to keep commits small.

### Step 4.1 — Extend `recorder.CallRecord` with structured fields

- Add `StatusCode int`, `Path string`, `Query string` to `recorder.CallRecord`; change body fields to
  `json.RawMessage` for JSON content (fall back to string otherwise). Keep existing JSON field names where
  possible for backward compatibility.
- Update/extend `pkg/recorder/recorder_test.go` for the new shape.
- **Extra Check:** `go test ./pkg/recorder/...`
- **Commit:** `feat(recorder): add structured StatusCode/Path/Query and JSON body fields`

### Step 4.2 — Populate structured fields in recording middleware

- `pkg/middleware/recording.go`: populate `StatusCode` (from `responseWriter.statusCode`), `Path`/`Query`
  (route template + `r.URL`), and store JSON bodies as `json.RawMessage`.
- Update `pkg/middleware/recording_test.go` accordingly.
- **Extra Check:** `go test ./pkg/middleware/...`
- **Commit:** `feat(middleware): record structured request/response with status code`

### Step 4.3 — Align README recorded-call example with real payload

- Update the README recorded-call JSON example (~line 278) to match the actual emitted payload.
- **Extra Check:** `grep` the README example fields against the `CallRecord` struct tags to confirm they match
  (manual diff of field names).
- **Commit:** `docs: update recorded-call example to match structured recorder output`

---

## Phase 5 — H3: Drop the dev watcher; document `make all`

Goal: simpler, predictable dev loop. Serves goal 1 + main goal. Split removal from doc update.

### Step 5.1 — Remove air-based hot-reload

- Delete `.air.go.toml`.
- Remove the `air` install from the `tools`/dev stage of the `Dockerfile`.
- Replace `scripts/run-dev.sh` body with `make all && ./bin/openapi-mock run 0.0.0.0 8080` (or set the dev
  stage `CMD` directly and drop the script).
- Keep `docker-compose.dev.yaml` (mounted volumes + ports) but without hot-reload semantics.
- **Extra Check:** `bash -n scripts/run-dev.sh` (script parses) **and**
  `grep -ri "air" Dockerfile scripts/ .air.go.toml 2>/dev/null`
  returns no remaining hot-reload references.
- **Commit:** `chore: drop air-based dev watcher in favor of explicit make all`

### Step 5.2 — Document the explicit regeneration step

- README: add "after editing `api/**` or stubs, run `make all`" to the dev workflow section; remove watcher
  references.
- **Extra Check:** `grep -n "make all" README.md` shows the documented step; no remaining "watch"/"air"
  references in README.
- **Commit:** `docs: document make all as the single regeneration step`

---

## Phase 6 — M5: Remove/replace dead `scripts/oapi-codegen.yaml`

Goal: remove confusing placeholder. Serves goal 1.

### Step 6.1 — Delete the unused placeholder config

- Remove `scripts/oapi-codegen.yaml` (contains `output: #!FIXME please`), since `gen-openapi.sh` passes flags
  directly and the file is unused. (Adopting config-file-driven generation is out of scope for this step.)
- **Extra Check:** `grep -rn "oapi-codegen.yaml" .` returns no references; `make openapi` still regenerates with
  no diff (`git diff --exit-code internal/generated`).
- **Commit:** `chore: remove dead scripts/oapi-codegen.yaml placeholder`

---

## Phase 7 — L1: Fix README documentation drift

Goal: docs accuracy. Trivial.

### Step 7.1 — Correct `pkg/ctxkeys` → `pkg/observability` references

- README (structure diagram ~line 25 and Request-ID note ~line 120): replace the non-existent `pkg/ctxkeys/`
  with `pkg/observability`.
- **Extra Check:** `grep -n "ctxkeys" README.md` returns nothing; referenced `pkg/observability` path exists.
- **Commit:** `docs: fix README references from pkg/ctxkeys to pkg/observability`

---

## Phase 8 — L2–L5: Observability & polish

Each is an independent small commit.

### Step 8.1 — L2: Low-cardinality trace span naming

- `pkg/middleware/recording.go` (~line 78): name the span using the route template (e.g. `GET /pets/{petId}`)
  instead of the concrete `r.URL.Path`; ensure `http.route` is set consistently.
- **Extra Check:** `go test ./pkg/middleware/...` (add/adjust an assertion that the span name uses the template).
- **Commit:** `fix(tracing): use route template for span name to lower cardinality`

### Step 8.2 — L3: In-flight / active-requests metric

- `pkg/metrics/metrics.go`: add an in-flight gauge; increment/decrement around request handling in the
  appropriate middleware.
- **Extra Check:** `go test ./pkg/metrics/... ./pkg/middleware/...`; confirm the gauge appears in metrics output.
- **Commit:** `feat(metrics): add in-flight requests gauge`

### Step 8.3 — L4: Guard runtime reset nil-out behind successful shutdown

- `cmd/openapi-mock/runtime.go` `stopLocked`: only set `r.server = nil` when `Shutdown` succeeds (or after the
  listener is confirmed closed), so a failed shutdown can't cause a rebind on the next `Reset`.
- **Extra Check:** `go test ./cmd/openapi-mock/...` (extend `runtime_test.go` with a failed-shutdown case).
- **Commit:** `fix(runtime): keep server reference when Shutdown fails to avoid rebind`

### Step 8.4 — L5: Serve HTML for ambiguous `GET /docs/{api_name}`

- Return an HTML version index (not JSON-only) for the browser-facing ambiguous-version case.
- **Extra Check:** `go test ./pkg/mgmt/...`; manual `curl -H 'Accept: text/html'` returns HTML.
- **Commit:** `feat(docs): serve HTML version index for ambiguous /docs/{api_name}`

---

## Completion criteria

- Every phase committed with all three DoD gates green.
- `make all && git diff --exit-code` clean (reproducible generation, Phase 1 invariant holds throughout).
- `./scripts/test-mgmt.sh` and `go test ./...` both pass on a fresh checkout.

## Quick step index

| #   | Step                                 | Item | Commit gate (Extra Check)                 |
|:----|:-------------------------------------|:-----|:------------------------------------------|
| 1.1 | Pin tools in go.mod                  | H2   | `go tool oapi-codegen --version` = v2.5.1 |
| 1.2 | Use `go tool` in scripts/Make/Docker | H2   | `make all && git diff --exit-code`        |
| 2.1 | `test-mgmt.sh` → `DELETE /logs`      | H4   | `./scripts/test-mgmt.sh` passes           |
| 3.1 | Live mgmt route smoke test           | M2   | `go test ./pkg/mgmt/...`                  |
| 4.1 | Structured `CallRecord`              | M3   | `go test ./pkg/recorder/...`              |
| 4.2 | Populate in recording middleware     | M3   | `go test ./pkg/middleware/...`            |
| 4.3 | README recorded-call example         | M3   | field-name diff vs struct                 |
| 5.1 | Remove air watcher                   | H3   | no `air` refs remain                      |
| 5.2 | Document `make all`                  | H3   | README shows step                         |
| 6.1 | Delete dead codegen yaml             | M5   | no refs; clean gen diff                   |
| 7.1 | README `ctxkeys` → observability     | L1   | no `ctxkeys` in README                    |
| 8.1 | Span template naming                 | L2   | `go test ./pkg/middleware/...`            |
| 8.2 | In-flight gauge                      | L3   | `go test ./pkg/metrics/...`               |
| 8.3 | Reset nil-out guard                  | L4   | `go test ./cmd/openapi-mock/...`          |
| 8.4 | HTML docs index                      | L5   | `go test ./pkg/mgmt/...`                  |

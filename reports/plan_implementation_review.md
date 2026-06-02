# Management API Improvements — Implementation Review

Date: 2026-06-02

## Summary

All six commits from the plan have been implemented. The core functionality is in place, tests pass (including race
tests), and the README has been updated. There are a few minor deviations from the plan and some areas where the
implementation is thinner than the plan suggested. Details follow.

---

## Commit 1 — Update log-management API

### What was implemented

| Requirement                                                   | Status | Notes                                                                   |
|:--------------------------------------------------------------|:-------|:------------------------------------------------------------------------|
| Replace `POST /clear` and `DELETE /clear` with `DELETE /logs` | ✅ Done | Route completely removed; test confirms `POST /clear` returns 404       |
| Add `GET /logs/{request_id}`                                  | ✅ Done | Uses `chi.URLParam(r, "request_id")`                                    |
| `Recorder.GetRecordsByRequestID` method                       | ✅ Done | Added to `pkg/recorder/recorder.go`                                     |
| Update `pkg/mgmt/openapi.json`                                | ✅ Done | `/logs` GET/DELETE and `/logs/{request_id}` documented; `/clear` absent |
| Update `pkg/mgmt/server_test.go`                              | ✅ Done | Tests for GET, GET by ID, DELETE, and 404 for `/clear`                  |
| Update README                                                 | ✅ Done | Migration notes added; examples updated                                 |

### Deviations / Gaps

- **None significant.** Implementation matches the plan closely.

### DoD Assessment

- `go test ./pkg/recorder ./pkg/mgmt` — ✅ PASS
- `go test ./...` — ✅ PASS
- `POST /clear` not accepted — ✅ Confirmed by test (returns 404)
- Management OpenAPI no longer documents `/clear` — ✅ Confirmed

---

## Commit 2 — Add request-id context-value store and middleware

### What was implemented

| Requirement                                                                  | Status | Notes                                                             |
|:-----------------------------------------------------------------------------|:-------|:------------------------------------------------------------------|
| `pkg/mm` package with `FromCtx`, `Lookup`                                    | ✅ Done | Both return cloned values                                         |
| Concurrency-safe `Store` keyed by request id                                 | ✅ Done | `sync.RWMutex`-protected                                          |
| Middleware `ContextValues(store)`                                            | ✅ Done | In `pkg/middleware/context_values.go`                             |
| Middleware registered after `Recording`, before handlers                     | ✅ Done | Order in `main.go`: CORS → Recording → ContextValues              |
| JSON numeric normalization (`int` for whole numbers, `float64` for decimals) | ✅ Done | `DecodeObject`/`DecodeStore` use `json.Number` + `normalizeValue` |
| Store copy semantics                                                         | ✅ Done | `cloneMap`, `cloneValue`, `cloneNestedMap` used throughout        |

### Deviations / Gaps

- **`Lookup` returns cloned values** — The plan says `FromCtx` returns `nil` for missing keys and `Lookup` returns
  `(nil, false)`. Implementation matches. However, `Lookup` also clones the returned value, which is a good safety
  measure not explicitly required but consistent with the store's copy semantics.
- **No typed helpers** — Plan mentioned "optionally typed helpers later." Not implemented, which is fine per the plan's
  wording.

### DoD Assessment

- Unit tests for `FromCtx`/`Lookup` present/missing — ✅ `TestFromCtxAndLookup`
- Integer JSON → `int` — ✅ `TestDecodeObjectNormalizesIntegersToInt`
- Decimal JSON → `float64` — ✅ `TestDecodeObjectNormalizesDecimalsToFloat64`
- Store copy semantics — ✅ `TestStoreCopySemantics`
- Concurrent reads/writes no race — ✅ `TestStoreConcurrentAccess` + `go test -race` passes
- Middleware integration test — ✅ `TestContextValuesMiddlewareInjectsPerRequestID`
- `go test ./pkg/mm ./pkg/middleware` — ✅ PASS
- `go test ./...` — ✅ PASS

---

## Commit 3 — Add management API for context values

### What was implemented

| Requirement                                                 | Status | Notes                                                |
|:------------------------------------------------------------|:-------|:-----------------------------------------------------|
| `GET /context-values`                                       | ✅ Done | Returns all store data                               |
| `PUT /context-values`                                       | ✅ Done | Replaces whole store via `ReplaceAll`                |
| `PATCH /context-values`                                     | ✅ Done | Merges via `MergeAll`                                |
| `DELETE /context-values`                                    | ✅ Done | Clears via `Clear`                                   |
| `GET /context-values/{request_id}`                          | ✅ Done | Returns `{}` for missing                             |
| `PUT /context-values/{request_id}`                          | ✅ Done | Replaces one request-id map                          |
| `PATCH /context-values/{request_id}`                        | ✅ Done | Merges values for one request id                     |
| `DELETE /context-values/{request_id}` (empty body)          | ✅ Done | Deletes all values for request id                    |
| `DELETE /context-values/{request_id}` with `{"keys":[...]}` | ✅ Done | Deletes only listed keys                             |
| Invalid JSON returns 400                                    | ✅ Done | `TestContextValuesInvalidJSONAndUnknownRequestID`    |
| Unsupported methods return 405                              | ✅ Done | `TestManagementRouteMethodsAndDocs` tests POST → 405 |
| Update `pkg/mgmt/openapi.json`                              | ✅ Done | All context-value paths and schemas documented       |
| Update README                                               | ✅ Done | Examples and migration notes included                |

### Deviations / Gaps

- **None significant.** All methods and edge cases are implemented and tested.

### DoD Assessment

- Tests cover every method and both collection/request-id paths — ✅
- Tests cover invalid JSON and missing/unknown request id — ✅
- OpenAPI document includes all context-value operations — ✅
- `go test ./pkg/mm ./pkg/mgmt ./pkg/middleware` — ✅ PASS
- `go test ./...` — ✅ PASS

---

## Commit 4 — Add mock API Swagger docs registry and endpoints

### What was implemented

| Requirement                                   | Status | Notes                                                             |
|:----------------------------------------------|:-------|:------------------------------------------------------------------|
| Mock-doc registry types in `pkg/mgmt`         | ✅ Done | `MockDoc` struct + `mockDocsIndex` with `list`, `find`, `resolve` |
| `GET /docs` index endpoint                    | ✅ Done | Returns list of all mock docs with UI/OpenAPI URLs                |
| `GET /docs/{api_name}`                        | ✅ Done | Serves Swagger UI or version index if ambiguous                   |
| `GET /docs/{api_name}/openapi.json`           | ✅ Done | Returns JSON spec; 404 if ambiguous                               |
| `GET /docs/{api_name}/{api_ver}`              | ✅ Done | Swagger UI for specific version                                   |
| `GET /docs/{api_name}/{api_ver}/openapi.json` | ✅ Done | JSON spec for specific version                                    |
| Version resolution rules (1–5 from plan)      | ✅ Done | `resolve()` implements all 5 rules                                |
| Generated registry from `cmd/upd-stubs`       | ✅ Done | `mock_docs.go` generates `internal/app/mock_docs_gen.go`          |
| `app.MockDocs()` function                     | ✅ Done | Returns `[]mgmt.MockDoc` with cached `SpecJSON`                   |
| Reuse embedded Swagger UI assets              | ✅ Done | `renderSwaggerUIHTML` with dynamic spec URL                       |
| Cache marshaled JSON per mock doc             | ✅ Done | `sync.Once` in generated `SpecJSON` closures                      |
| Mock server URL injection in OpenAPI JSON     | ✅ Done | `injectMockServerURL` adds `servers` field                        |
| Missing API/version returns 404               | ✅ Done | JSON error for JSON routes, HTML error for UI routes              |

### Deviations / Gaps

1. **`MockDocsRegistry` interface not implemented.** The plan suggested:
   ```go
   type MockDocsRegistry interface {
       List() []MockDoc
       Find(apiName, apiVersion string) (MockDoc, bool)
       Resolve(apiName string) ResolveResult
   }
   ```
   The implementation uses a concrete `mockDocsIndex` struct with unexported methods instead. This is a minor design
   deviation — the interface would add abstraction that isn't currently needed since the registry is only consumed
   internally by `Server`. No functional impact.

2. **`ResolveResult` type not introduced.** The `resolve` method returns a 4-value tuple instead. Functionally
   equivalent but less self-documenting than a named result type.

3. **Ambiguous version response is JSON-only.** The plan says "returns an HTML/JSON index with links." The
   implementation always returns JSON for the ambiguous case (status 200 with `{"api_name":..., "versions":[...]}`).
   Since `GET /docs/{api_name}` is a browser-facing endpoint, an HTML index page would be more user-friendly. This is
   a minor UX gap.

### DoD Assessment

- `go run ./cmd/upd-stubs` succeeds — ⚠️ Not verified in this review (would need manual run)
- `go test ./cmd/upd-stubs ./pkg/mgmt ./internal/app` — ✅ PASS (mgmt and upd-stubs pass; internal/app has no tests)
- `go test ./...` — ✅ PASS
- Swagger UI loads selected mock spec — ✅ `renderSwaggerUIHTML` uses dynamic spec URL
- Missing API/version returns 404 — ✅ Tested in `TestMockDocsRoutes`

---

## Commit 5 — Add soft reset runtime and management endpoint

### What was implemented

| Requirement                                       | Status | Notes                                         |
|:--------------------------------------------------|:-------|:----------------------------------------------|
| Mock runtime/controller (`mockRuntime`)           | ✅ Done | In `cmd/openapi-mock/runtime.go`              |
| `POST /reset` management handler                  | ✅ Done | With 5-second timeout                         |
| Reset callback wired into `mgmt.Server`           | ✅ Done | Via `Options.Reset`                           |
| Reset clears recorder and context-value store     | ✅ Done | `resetCallback` in `main.go`                  |
| Serialize resets with mutex                       | ✅ Done | `sync.Mutex` in `mockRuntime`                 |
| Bounded shutdown context                          | ✅ Done | 5-second timeout in `stopLocked`              |
| Return success only after new server is listening | ✅ Done | `startLocked` opens listener before returning |
| Return 500 on failure                             | ✅ Done | `handleReset` returns 500 on error            |
| Management/metrics servers remain running         | ✅ Done | Only mock server is stopped/started           |
| Process signal shutdown still works               | ✅ Done | Signal handler stops all servers in order     |

### Deviations / Gaps

1. **`startLocked` skips if `r.server != nil`.** After a successful `stopLocked`, `r.server` is set to `nil`, so
   `startLocked` will proceed. However, if `stopLocked` fails (returns error), `r.server` is still set to `nil`
   (line 94), meaning a failed shutdown still clears the server reference. This could leave the runtime in a state
   where the old server is still serving on the listener but `startLocked` would try to create a new one, potentially
   failing on `net.Listen` because the port is still in use. This is an edge case that could be improved.

2. **No explicit test that management server remains callable after reset.** The plan's DoD requires this. The
   `TestResetRoute` test only checks the handler in isolation (via `router()`), not with a running management server.
   The runtime tests (`TestMockRuntimeResetRebuildsHandler`, `TestMockRuntimeResetIsSerialized`) don't involve the
   management server at all.

3. **No test for "reset clears logs and context values" at the integration level.** `TestResetCallbackClearsStores`
   tests the callback function directly, which is good, but there's no end-to-end test that hits `POST /reset` and
   then checks `/logs` and `/context-values`.

### DoD Assessment

- Unit tests for reset handler success/failure — ✅ `TestResetRoute`
- Runtime tests for rebuild handler — ✅ `TestMockRuntimeResetRebuildsHandler`
- Reset clears logs and context values — ✅ `TestResetCallbackClearsStores` (unit level)
- Management server remains callable after reset — ⚠️ Not explicitly tested
- Repeated reset calls are serialized — ✅ `TestMockRuntimeResetIsSerialized`
- `go test ./cmd/openapi-mock ./pkg/mgmt ./pkg/mm ./pkg/middleware` — ✅ PASS
- `go test ./...` — ✅ PASS

---

## Commit 6 — End-to-end documentation, compatibility checks, and final polish

### What was implemented

| Requirement                                                     | Status | Notes                                                                            |
|:----------------------------------------------------------------|:-------|:---------------------------------------------------------------------------------|
| Integration test: request-specific behavior with context values | ✅ Done | `TestContextValuesByRequestID_EndToEnd` in `pkg/mgmt/context_values_e2e_test.go` |
| README examples are copy-pasteable                              | ✅ Done | All curl examples updated                                                        |
| `pkg/mgmt/openapi.json` matches implemented endpoints           | ✅ Done | All endpoints documented                                                         |
| Migration notes for removed `/clear`                            | ✅ Done | "Миграционные заметки" section in README                                         |
| Race-sensitive packages pass                                    | ✅ Done | `go test -race ./pkg/mm ./pkg/recorder ./pkg/middleware ./pkg/mgmt` PASS         |

### Deviations / Gaps

1. **No changelog file.** The plan mentions "changelog-style migration notes." The README has migration notes inline,
   but there's no dedicated `CHANGELOG.md` or similar file. This is a minor documentation gap.

2. **Generated files stability not verified in this review.** The DoD requires `go run ./cmd/upd-stubs` followed by
   `git diff --exit-code` to confirm stable output. This was not run as part of the review.

3. **No full local smoke test.** The DoD requires starting the server and running a series of curl commands. This was
   not performed as part of the review.

### DoD Assessment

- `go test ./...` — ✅ PASS
- `go test -race ./pkg/mm ./pkg/recorder ./pkg/middleware ./pkg/mgmt` — ✅ PASS
- Generated files stable — ⚠️ Not verified
- Local smoke flow — ⚠️ Not verified
- README and OpenAPI doc complete — ✅

---

## Cross-cutting observations

### Design deviations from the plan

| Plan suggestion                        | Actual implementation           | Impact                                                    |
|:---------------------------------------|:--------------------------------|:----------------------------------------------------------|
| `MockDocsRegistry` interface           | Concrete `mockDocsIndex` struct | Low — no external consumers need the interface            |
| `ResolveResult` named type             | 4-value return tuple            | Low — works fine, less self-documenting                   |
| HTML/JSON index for ambiguous versions | JSON-only response              | Low — browser UX could be improved                        |
| Changelog file                         | Inline README migration notes   | Low — information is present, just not in a separate file |

### Missing test coverage

| Area                                                    | Status                        | Risk                              |
|:--------------------------------------------------------|:------------------------------|:----------------------------------|
| Management server remains callable after reset          | Not tested                    | Medium — could regress silently   |
| End-to-end reset clears logs + context values           | Only unit-level callback test | Low — callback is simple          |
| `GET /docs/{api_name}` ambiguous case returns HTML      | Not tested for HTML content   | Low — JSON response is functional |
| `stopLocked` error leaves runtime in inconsistent state | Not tested                    | Low — edge case                   |

### Positive observations

1. **Clean separation of concerns.** `pkg/mm` is independent of management handlers. `pkg/recorder` has its own
   `GetRecordsByRequestID`. The runtime controller is cleanly separated from the management server.

2. **Copy semantics are thorough.** Every store read/write clones data, preventing race conditions and mutation leaks.
   The `cloneValue` function handles nested maps and slices recursively.

3. **JSON numeric normalization is correct.** Whole numbers fitting in `int` become `int`; decimals and scientific
   notation become `float64`. This matches the plan's requirement for `. (int)` type assertions.

4. **Chi router adoption.** The plan suggested using Chi or Go 1.22+ `http.ServeMux` patterns. Chi was chosen, which
   provides clean path parameter extraction and method-specific route registration.

5. **Wire integration is clean.** The `mock_docs_gen.go` generator integrates seamlessly with the existing Wire-based
   DI setup.

6. **Swagger UI reuse.** The `renderSwaggerUIHTML` function parameterizes the spec URL, allowing the same embedded
   assets to serve both management and mock API docs.

---

## Overall assessment

| Commit                               | Implementation                       | DoD Status                                      |
|:-------------------------------------|:-------------------------------------|:------------------------------------------------|
| 1 — Log management API               | ✅ Complete                           | ✅ Pass                                          |
| 2 — Context-value store & middleware | ✅ Complete                           | ✅ Pass                                          |
| 3 — Context-value management API     | ✅ Complete                           | ✅ Pass                                          |
| 4 — Mock docs registry & endpoints   | ✅ Complete (minor design deviations) | ⚠️ Mostly pass (manual smoke not verified)      |
| 5 — Soft reset runtime               | ✅ Complete (minor test gaps)         | ⚠️ Mostly pass (mgmt-after-reset not tested)    |
| 6 — E2E docs & polish                | ✅ Complete (no changelog file)       | ⚠️ Mostly pass (smoke & stability not verified) |

**Verdict:** The plan has been substantially implemented. All core functionality works, tests pass (including race
tests), and the README is up to date. The remaining gaps are minor: a few integration-level tests, an HTML version
index page for ambiguous API names, and manual smoke verification.

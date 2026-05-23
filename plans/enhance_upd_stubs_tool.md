# upd-stubs polishing plan

Living document. Each top-level section is independently shippable.

## 1. Bugs / weaknesses in the current code

### main.go
- `log.Fatalf` on first spec aborts every other spec. Use `errors.Join`, exit 1 only after attempting all specs.
- No CLI flags. Paths are hard-coded constants.
- No summary of created / updated / skipped files.

### openapi_discovery.go
- `os.IsNotExist` check at the bottom only catches the initial Walk stat — actual missing-dir errors come from inside the callback. Stat `specsDir` explicitly first.
- `relPath` is empty when the spec is directly at `specs/openapi.yaml`.
- `pkgName` not validated: leading digits, dots, Go keywords, dashes all reach codegen and break the build.
- Tag normalisation (`ToLower` + space→`_`) silently merges distinct tags and leaves unsafe chars (`/`, `.`, accents) untouched.
- **Operations with multiple tags** are emitted once per tag → duplicate method declarations in stub files and in the composite provider. Need a primary-tag rule (first tag wins, else `default`).
- `getMethodOrderFromInterface` error is silently dropped.
- `pathItem.Operations()` map order is non-deterministic when `server.gen.go` is absent.

### openapi_provider.go
- Same multi-tag duplication as above.
- `extractArgNames` is dead code.
- No `var _ gen.StrictServerInterface = (*XxxHandlers)(nil)` per-tag.
- Generates `gen.<MethodName>RequestObject` from a hand-rolled `toPascalCase(op.OperationID)`. oapi-codegen does its own normalisation; the two can diverge. Should derive names from the parsed AST of `server.gen.go`.

### openapi_stubs.go
- `parseExistingMethods` is line-based and brittle:
    - misses multi-line signatures,
    - hard-codes receiver name `h` (any rename → method re-added → compile error),
    - the two prefix vars (`prefix`, `altPrefix`) are identical (dead code).
- Update path does NOT run goimports → appended methods may reference symbols not imported anymore.
- No way to flag/remove operations that disappeared from the spec.
- "Smallest code" uses lexicographic `sort.Strings` → `"1XX" < "200" < "2XX" < "default"`. Needs explicit 2xx→3xx→1xx→4xx→5xx→default ordering.
- `responseBodyExpr` and `sortedStringKeys` are dead.
- `format.Source` failure dumps the full buffer to logs unconditionally — gate behind `--verbose`.

### openapi_wire.go
- **Index-mismatch bug**: `imports` built in spec order, then sorted by `StubPath`. Later loops do `for i, imp := range imports { spec := specs[i] }`. After the sort `specs[i]` no longer matches `imports[i]` — with ≥2 specs the wire wires wrong stubs to wrong gen package.
- No `goimports` pass on the final file.
- Two specs with the same base name in different sub-dirs collide on `stubAlias`/`genAlias`.
- `enableLogging` parameter of `InitializeHTTPApp` is never consumed by any provider in the generated set — wire will complain.
- `HTTPApp` field names risk collision when pkg+tag overlap across specs.

### utils.go
- `toSnakeCase("HTTPHandler")` → `h_t_t_p_handler`.
- `toPascalCase` ignores digit boundaries and doesn't match oapi-codegen exactly — every downstream name guess depends on it.
- `writeImports` is hand-rolled; emit raw and rely on `goimports.Process` instead.

---

## 2. Refactor steps (recommended order)

1. **Extract `Config` struct + CLI flags** in `main.go` (`--specs-dir`, `--gen-dir`, `--stubs-dir`, `--wire-out`, `--dry-run`, `--prune`, `--verbose`). Aggregate errors with `errors.Join`. Print summary. Unlocks testability.
2. **Read names from `server.gen.go` AST** — return `map[opKey]{MethodName, ReqType, RespType, ResponseCodes}` and use it everywhere instead of recomputing from `OperationID`.
3. **Fix wire index-mismatch** by iterating a single sorted `[]{spec, imp}` slice. Add a path-derived suffix to aliases. Decide & wire `enableLogging`.
4. **Replace `parseExistingMethods` with go/ast scanner** + run `goimports.Process` on every output file.
5. **Harden discovery**: validate pkgName, primary-tag rule, deterministic fallback ordering, surfaced warnings.
6. **`pickPrimaryResponse` helper** with explicit 2xx→3xx→1xx→4xx→5xx→default ordering.
7. **Per-tag interface assertion** appended to each stub file.
8. **Opt-in `--prune` mode** that moves orphaned methods into an annotated tail section (never deletes user code).
9. **Remove dead code** (`extractArgNames`, `responseBodyExpr`, `sortedStringKeys`, duplicate `altPrefix`).
10. **Update `UPD_STUBS.md`** with new flags, primary-tag rule, prune semantics, `enableLogging` injection model.

---

## 3. Test strategy (edge cases)

Tests live next to the code under `cmd/upd-stubs/*_test.go`. Most are table tests; a few are golden-file integration tests against `t.TempDir()`.

### utils_test.go
- `toSnakeCase`: `HTTPHandler`, `ID`, `listPetsV2`, `""`, `already_snake`, `with-dash`, unicode `Café`.
- `toPascalCase`: `list_pets`, `list-pets`, `listPets`, `123abc`, `""`, `___`, `v1_pets`.
- `sanitizeFieldName`: `default`, `Default`, `go` (keyword), `123foo`.
- `pickPrimaryResponse`: `{200,400}`, `{201,200}`, `{2XX,200}`, `{default,500}`, empty map.

### openapi_discovery_test.go
- Missing `specs/` → no error, empty result.
- Spec with zero operations → spec recorded, no tags map.
- Operation with no tags → bucketed under `default`.
- Operation with multiple tags → present in exactly one stub (primary tag).
- Two ops sharing OperationID across paths → deterministic disambiguation or clear error.
- Spec dir named `123-api` or `func` → rejected with clear error.
- Tag collisions (`Pets v1` vs `pets-v1`) → deterministic merge + warning.
- Determinism: discover twice → byte-equal results.
- Order with/without `server.gen.go` is stable.

### openapi_stubs_test.go
- Fresh generation against an in-memory spec → output parses with `go/parser`; methods present.
- Update path:
    - Receiver renamed in existing file (`handler` not `h`) → method NOT re-added.
    - Multi-line existing signature → NOT re-added.
    - Existing file missing one method → only that one appended; rest byte-equal to golden.
    - Imports previously trimmed → restored by goimports after append.
- Prune mode: dropped op with user body → moved into orphan block with TODO; without `--prune`, untouched.
- Multi-tag op: appears in primary-tag stub only; composite delegates once; package builds (verify with `go/types`).
- Response selection: only-`default`, `200+201`, `4xx`-only, `$ref` JSON body, no content.
- Name edge cases (digits, dashes, empty OperationID) match what the AST of `server.gen.go` actually declares.

### openapi_wire_test.go
- Two specs (`petstore`, `echo`) → after sort, `provideEchoHandlers` references `echostub.NewCompositeHandlers` (regression for the index bug).
- Single spec → output gofmt-stable across two runs.
- Same base name in different sub-dirs (`specs/v1/petstore`, `specs/v2/petstore`) → no alias collision; both `HTTPApp` fields distinct.
- `enableLogging` parameter either consumed by a provider or absent — assert via AST of the generated file.

### main_test.go (golden / integration)
- Run `main()` against `testdata/specs/petstore` + canned `server.gen.go` into `t.TempDir()`; diff against `testdata/golden/`.
- Re-run; assert idempotency (no diffs).
- Add a new operation to the spec; re-run; only one new method appended, every other file unchanged.
- Corrupt one of two specs; assert non-zero exit, the other spec still generated, error message references the offending path.

---

## 4. Implementation order

1. CLI / Config extraction + utils_test.go.
2. AST-driven name map from `server.gen.go`.
3. Wire index-mismatch fix.
4. AST-based existing-method scan + goimports on every output.
5. Discovery hardening + primary-tag rule + response picker + per-tag iface assertion + dead-code removal.
6. `--prune` mode.
7. Docs.

Tests are written together with each step; integration goldens land with step 4.

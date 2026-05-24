package main

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed test_srcs/*/*.go
var testSources embed.FS

func mustReadTestSource(t *testing.T, relPath string) []byte {
	t.Helper()
	b, err := testSources.ReadFile(relPath)
	if err != nil {
		t.Fatalf("failed to read embedded test source %q: %v", relPath, err)
	}
	return b
}

func TestSanitizeHelpers(t *testing.T) {
	if got := toSnakeCase("HTTPHandler"); got != "http_handler" {
		t.Fatalf("toSnakeCase(HTTPHandler) = %q, want %q", got, "http_handler")
	}
	if got := sanitizeGoIdentifier("123 go-value"); got != "v_123_go_value" {
		t.Fatalf("sanitizeGoIdentifier() = %q, want %q", got, "v_123_go_value")
	}
	if got := sanitizeGoPackageName("Func"); got != "func_" {
		t.Fatalf("sanitizeGoPackageName() = %q, want %q", got, "func_")
	}
}

func TestParseExistingMethods_ASTHandlesReceiverRenameAndMultiline(t *testing.T) {
	src := mustReadTestSource(t, "test_srcs/parse_existing_methods/pets_handlers.go")

	methods := parseExistingMethods(src, "PetsHandlers")

	if !methods["ListPets"] {
		t.Fatalf("expected ListPets to be detected")
	}
	if !methods["GetPet"] {
		t.Fatalf("expected GetPet to be detected")
	}
}

func TestPickPrimaryResponse(t *testing.T) {
	mk := func(code string) *openapi3.ResponseRef {
		return &openapi3.ResponseRef{Value: &openapi3.Response{Description: &code}}
	}

	tests := []struct {
		name string
		in   map[string]*openapi3.ResponseRef
		want string
	}{
		{name: "prefer-2xx-over-4xx", in: map[string]*openapi3.ResponseRef{"404": mk("404"), "200": mk("200")}, want: "200"},
		{name: "prefer-lowest-2xx", in: map[string]*openapi3.ResponseRef{"201": mk("201"), "200": mk("200")}, want: "200"},
		{name: "default-after-5xx", in: map[string]*openapi3.ResponseRef{"default": mk("default"), "500": mk("500")}, want: "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := pickPrimaryResponse(tt.in)
			if code != tt.want {
				t.Fatalf("pickPrimaryResponse() = %q, want %q", code, tt.want)
			}
		})
	}
}

func TestGenerateOpenAPIWireFile_UsesMatchingSpecAfterSorting(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "internal", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	specs := []*openapiSpec{
		{
			PkgName:     "petstore",
			RelPath:     "petstore",
			GenPkgPath:  "openapi-mock/internal/generated/petstore",
			StubPkgPath: "openapi-mock/internal/stubs/petstore",
			Tags:        map[string][]opInfo{"pets": nil},
		},
		{
			PkgName:     "echo",
			RelPath:     "echo",
			GenPkgPath:  "openapi-mock/internal/generated/echo",
			StubPkgPath: "openapi-mock/internal/stubs/echo",
			Tags:        map[string][]opInfo{"default": nil},
		},
	}

	if err := generateOpenAPIWireFile(specs); err != nil {
		t.Fatalf("generateOpenAPIWireFile() error = %v", err)
	}

	out, err := os.ReadFile(filepath.Join(tmp, "internal", "app", "openapi_wire.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(out)

	if !strings.Contains(content, "func provideEchoHandlers(") {
		t.Fatalf("expected provideEchoHandlers in generated wire file")
	}
	if !strings.Contains(content, "strict := echostub1.NewCompositeHandlers(") {
		t.Fatalf("expected echo provider to delegate to echostub1 composite handlers")
	}
	if !strings.Contains(content, "strict := petstorestub0.NewCompositeHandlers(") {
		t.Fatalf("expected petstore provider to delegate to petstorestub0 composite handlers")
	}
}

func TestDiscoverOpenAPISpecs_NoSpecsDir(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	oldSpecsDir := specsDir
	specsDir = "specs"
	defer func() { specsDir = oldSpecsDir }()

	specs, err := discoverOpenAPISpecs()
	if err != nil {
		t.Fatalf("discoverOpenAPISpecs() unexpected error: %v", err)
	}
	if len(specs) != 0 {
		t.Fatalf("discoverOpenAPISpecs() len = %d, want 0", len(specs))
	}
}

func TestLoadOpenAPISpec_PrimaryTagAndSanitizedPackage(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	if err := os.MkdirAll(filepath.Join("specs", "123-api"), 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join("specs", "123-api", "openapi.yaml")

	content := `openapi: 3.0.3
info:
  title: test
  version: 1.0.0
paths:
  /pets:
    get:
      operationId: listPets
      tags: [pets, admin]
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := loadOpenAPISpec(specPath)
	if err != nil {
		t.Fatalf("loadOpenAPISpec() error: %v", err)
	}
	if spec.PkgName != "v_123_api" {
		t.Fatalf("spec.PkgName = %q, want %q", spec.PkgName, "v_123_api")
	}
	if _, ok := spec.Tags["pets"]; !ok {
		t.Fatalf("expected primary tag pets to exist")
	}
	if _, ok := spec.Tags["admin"]; ok {
		t.Fatalf("secondary tag should not receive duplicated operation")
	}
}

func TestResolveOperationMethodName_UsesStrictNameMatch(t *testing.T) {
	spec := &openapiSpec{StrictNames: map[string]bool{"ListPetsV2": true}}
	op := opInfo{OperationID: "list_pets_v2"}
	if got := resolveOperationMethodName(spec, op); got != "ListPetsV2" {
		t.Fatalf("resolveOperationMethodName() = %q, want %q", got, "ListPetsV2")
	}
}

func TestAnnotateOrphanedMethods(t *testing.T) {
	src := mustReadTestSource(t, "test_srcs/orphaned_methods/pets_handlers.go")
	existing := parseExistingMethods(src, "PetsHandlers")
	spec := &openapiSpec{}
	ops := []opInfo{{OperationID: "ListPets"}}

	updated, changed := annotateOrphanedMethods(spec, src, existing, ops)
	if !changed {
		t.Fatalf("expected orphan annotation change")
	}
	text := string(updated)
	if !strings.Contains(text, orphanBlockStart) || !strings.Contains(text, "DeprecatedPets") {
		t.Fatalf("expected orphan block with DeprecatedPets, got:\n%s", text)
	}

	updated2, changed2 := annotateOrphanedMethods(spec, updated, parseExistingMethods(updated, "PetsHandlers"), ops)
	if changed2 {
		t.Fatalf("expected idempotent orphan annotation update")
	}
	if string(updated2) != string(updated) {
		t.Fatalf("expected stable output across repeated annotation")
	}
}

func TestUpdateOpenAPIStubFile_DoesNotDuplicateMultilineRenamedReceiverMethod(t *testing.T) {
	tmp := t.TempDir()
	stubPath := filepath.Join(tmp, "pets.go")

	existing := mustReadTestSource(t, "test_srcs/update_existing_methods/pets_handlers.go")
	if err := os.WriteFile(stubPath, existing, 0o644); err != nil {
		t.Fatal(err)
	}

	spec := &openapiSpec{
		PkgName:     "petstore",
		GenPkgPath:  "openapi-mock/internal/generated/petstore",
		StrictNames: map[string]bool{"ListPets": true, "CreatePet": true},
	}
	responses := &openapi3.Responses{}
	ops := []opInfo{
		{OperationID: "ListPets", Operation: &openapi3.Operation{Responses: responses}},
		{OperationID: "CreatePet", Operation: &openapi3.Operation{Responses: responses}},
	}

	if err := updateOpenAPIStubFile(stubPath, spec, "pets", ops); err != nil {
		t.Fatalf("updateOpenAPIStubFile() error = %v", err)
	}

	updated, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(updated)

	if strings.Count(content, "ListPets(") != 1 {
		t.Fatalf("expected exactly one ListPets method after update, got:\n%s", content)
	}
	if !strings.Contains(content, "func (h *PetsHandlers) CreatePet(") {
		t.Fatalf("expected missing CreatePet method to be appended, got:\n%s", content)
	}
}

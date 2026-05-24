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

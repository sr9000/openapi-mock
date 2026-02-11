package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// generateOpenAPIWireFile generates internal/app/openapi_wire.go
func generateOpenAPIWireFile(specs []*openapiSpec) error {
	if len(specs) == 0 {
		return nil
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "//go:build wireinject\n")
	fmt.Fprintf(&buf, "// +build wireinject\n\n")
	fmt.Fprintf(&buf, "package app\n\n")

	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"net/http\"\n\n")
	fmt.Fprintf(&buf, "\t\"github.com/go-chi/chi/v5\"\n")
	fmt.Fprintf(&buf, "\t\"github.com/google/wire\"\n\n")

	// Import all generated packages first, then stubs
	type specImport struct {
		StubAlias string
		StubPath  string
		GenAlias  string
		GenPath   string
	}
	var imports []specImport

	for _, spec := range specs {
		stubAlias := strings.ReplaceAll(spec.RelPath, "/", "") + "stub"
		genAlias := strings.ReplaceAll(spec.RelPath, "/", "") + "gen"
		imports = append(imports, specImport{
			StubAlias: stubAlias,
			StubPath:  spec.StubPkgPath,
			GenAlias:  genAlias,
			GenPath:   spec.GenPkgPath,
		})
	}

	sort.Slice(imports, func(i, j int) bool {
		return imports[i].StubPath < imports[j].StubPath
	})

	for _, imp := range imports {
		fmt.Fprintf(&buf, "\t%s %q\n", imp.GenAlias, imp.GenPath)
	}
	buf.WriteString("\n")
	for _, imp := range imports {
		fmt.Fprintf(&buf, "\t%s %q\n", imp.StubAlias, imp.StubPath)
	}

	fmt.Fprintf(&buf, ")\n\n")

	// HTTPApp struct
	fmt.Fprintf(&buf, "type HTTPApp struct {\n")
	fmt.Fprintf(&buf, "\tRouter *chi.Mux\n")
	for _, imp := range imports {
		var spec *openapiSpec
		for _, s := range specs {
			if s.StubPkgPath == imp.StubPath {
				spec = s
				break
			}
		}
		if spec == nil {
			continue
		}

		for _, tag := range spec.getSortedTags() {
			fieldName := toPascalCase(spec.PkgName) + toPascalCase(tag)
			structName := getHandlerStructName(tag)
			fmt.Fprintf(&buf, "\t%s *%s.%s\n", fieldName, imp.StubAlias, structName)
		}
	}
	fmt.Fprintf(&buf, "}\n\n")

	// Provider functions: per-spec strict handlers
	for i, imp := range imports {
		spec := specs[i]

		fmt.Fprintf(&buf, "func provide%sHandlers(", toPascalCase(spec.PkgName))
		var params []string
		for _, tag := range spec.getSortedTags() {
			structName := getHandlerStructName(tag)
			params = append(params, fmt.Sprintf("%s *%s.%s", sanitizeFieldName(tag), imp.StubAlias, structName))
		}
		fmt.Fprintf(&buf, "%s) %s.ServerInterface {\n", strings.Join(params, ", "), imp.GenAlias)
		fmt.Fprintf(&buf, "\tstrict := %s.NewCompositeHandlers(%s)\n", imp.StubAlias, extractFieldNames(params))
		fmt.Fprintf(&buf, "\treturn %s.NewStrictHandler(strict, nil)\n", imp.GenAlias)
		fmt.Fprintf(&buf, "}\n\n")
	}

	// Router provider
	fmt.Fprintf(&buf, "func provideHTTPRouter(middlewares []func(http.Handler) http.Handler, ")
	var routerParams []string
	for i, imp := range imports {
		spec := specs[i]
		routerParams = append(routerParams, fmt.Sprintf("%s %s.ServerInterface", spec.PkgName+"Handler", imp.GenAlias))
	}
	fmt.Fprintf(&buf, "%s) *chi.Mux {\n", strings.Join(routerParams, ", "))
	fmt.Fprintf(&buf, "\tr := chi.NewRouter()\n")
	fmt.Fprintf(&buf, "\tfor _, mw := range middlewares {\n")
	fmt.Fprintf(&buf, "\t\tr.Use(mw)\n")
	fmt.Fprintf(&buf, "\t}\n")
	for i, imp := range imports {
		spec := specs[i]
		fmt.Fprintf(&buf, "\t%s.HandlerFromMux(%s, r)\n", imp.GenAlias, spec.PkgName+"Handler")
	}
	fmt.Fprintf(&buf, "\treturn r\n")
	fmt.Fprintf(&buf, "}\n\n")

	// ProviderSet
	fmt.Fprintf(&buf, "var HTTPProviderSet = wire.NewSet(\n")
	for _, imp := range imports {
		var spec *openapiSpec
		for _, s := range specs {
			if s.StubPkgPath == imp.StubPath {
				spec = s
				break
			}
		}
		if spec == nil {
			continue
		}
		for _, tag := range spec.getSortedTags() {
			newFunc := getNewHandlerFuncName(tag)
			fmt.Fprintf(&buf, "\t%s.%s,\n", imp.StubAlias, newFunc)
		}
		fmt.Fprintf(&buf, "\tprovide%sHandlers,\n", toPascalCase(spec.PkgName))
	}
	fmt.Fprintf(&buf, "\tprovideHTTPRouter,\n")
	fmt.Fprintf(&buf, "\twire.Struct(new(HTTPApp), \"*\"),\n")
	fmt.Fprintf(&buf, ")\n\n")

	// InitializeHTTPApp
	fmt.Fprintf(&buf, "func InitializeHTTPApp(middlewares []func(http.Handler) http.Handler, enableLogging bool) (*HTTPApp, error) {\n")
	fmt.Fprintf(&buf, "\twire.Build(HTTPProviderSet)\n")
	fmt.Fprintf(&buf, "\treturn nil, nil\n")
	fmt.Fprintf(&buf, "}\n")

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Printf("failed to format openapi_wire.go: %v\nSource:\n%s", err, buf.String())
		return err
	}

	outPath := filepath.Join("internal", "app", "openapi_wire.go")
	return os.WriteFile(outPath, src, 0o644)
}

func extractFieldNames(params []string) string {
	var names []string
	for _, p := range params {
		fields := strings.Fields(strings.TrimSpace(p))
		if len(fields) > 0 {
			names = append(names, fields[0])
		}
	}
	return strings.Join(names, ", ")
}

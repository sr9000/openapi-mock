package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"sort"
	"strconv"
	"strings"

	goimports "golang.org/x/tools/imports"
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

	fmt.Fprintf(&buf, "\t\"openapi-mock/pkg/metrics\"\n")
	fmt.Fprintf(&buf, "\t\"openapi-mock/pkg/middleware\"\n")

	// Import all generated packages first, then stubs
	type specImport struct {
		StubAlias string
		StubPath  string
		GenAlias  string
		GenPath   string
	}
	type wireSpec struct {
		Spec *openapiSpec
		Imp  specImport
	}
	var wireSpecs []wireSpec

	for i, spec := range specs {
		aliasBase := sanitizeGoIdentifier(strings.ReplaceAll(spec.RelPath, "/", "_"))
		stubAlias := fmt.Sprintf("%sstub%d", aliasBase, i)
		genAlias := fmt.Sprintf("%sgen%d", aliasBase, i)
		wireSpecs = append(wireSpecs, wireSpec{
			Spec: spec,
			Imp: specImport{
				StubAlias: stubAlias,
				StubPath:  spec.StubPkgPath,
				GenAlias:  genAlias,
				GenPath:   spec.GenPkgPath,
			},
		})
	}

	sort.Slice(wireSpecs, func(i, j int) bool {
		return wireSpecs[i].Imp.StubPath < wireSpecs[j].Imp.StubPath
	})

	for _, ws := range wireSpecs {
		imp := ws.Imp
		fmt.Fprintf(&buf, "\t%s %q\n", imp.GenAlias, imp.GenPath)
	}
	buf.WriteString("\n")
	for _, ws := range wireSpecs {
		imp := ws.Imp
		fmt.Fprintf(&buf, "\t%s %q\n", imp.StubAlias, imp.StubPath)
	}

	fmt.Fprintf(&buf, ")\n\n")

	// HTTPApp struct
	fmt.Fprintf(&buf, "type HTTPApp struct {\n")
	fmt.Fprintf(&buf, "\tRouter *chi.Mux\n")
	for _, ws := range wireSpecs {
		imp := ws.Imp
		spec := ws.Spec
		for _, tag := range spec.getSortedTags() {
			fieldName := toPascalCase(strings.ReplaceAll(spec.RelPath, "/", "_")) + toPascalCase(tag)
			structName := getHandlerStructName(tag)
			fmt.Fprintf(&buf, "\t%s *%s.%s\n", fieldName, imp.StubAlias, structName)
		}
	}
	fmt.Fprintf(&buf, "}\n\n")

	// Provider functions: per-spec strict handlers
	for _, ws := range wireSpecs {
		imp := ws.Imp
		spec := ws.Spec

		fmt.Fprintf(&buf, "func provide%sHandlers(", toPascalCase(spec.PkgName))
		var params []string
		for _, tag := range spec.getSortedTags() {
			structName := getHandlerStructName(tag)
			params = append(params, fmt.Sprintf("%s *%s.%s", sanitizeFieldName(tag), imp.StubAlias, structName))
		}

		handlerParams := make([]string, len(params))
		copy(handlerParams, params)

		params = append(params, "errHandlers *middleware.ErrorHandlers")
		fmt.Fprintf(&buf, "%s) %s.ServerInterface {\n", strings.Join(params, ", "), imp.GenAlias)
		fmt.Fprintf(&buf, "\tstrict := %s.NewCompositeHandlers(%s)\n", imp.StubAlias, extractFieldNames(handlerParams))
		fmt.Fprintf(&buf, "\tstrictMiddlewares := []%s.StrictMiddlewareFunc{middleware.OperationContext()}\n", imp.GenAlias)
		fmt.Fprintf(&buf, "\treturn %s.NewStrictHandlerWithOptions(strict, strictMiddlewares, %s.StrictHTTPServerOptions{\n", imp.GenAlias, imp.GenAlias)
		fmt.Fprintf(&buf, "\t\tRequestErrorHandlerFunc:  errHandlers.RequestErrorHandler,\n")
		fmt.Fprintf(&buf, "\t\tResponseErrorHandlerFunc: errHandlers.ResponseErrorHandler,\n")
		fmt.Fprintf(&buf, "\t})\n")
		fmt.Fprintf(&buf, "}\n\n")
	}

	// Error handlers provider
	fmt.Fprintf(&buf, "func provideOperationResolver() middleware.OperationResolver {\n")
	fmt.Fprintf(&buf, "\tresolver := middleware.NewOperationResolver(map[string]string{\n")
	for _, entry := range collectResolverEntries(specs) {
		fmt.Fprintf(&buf, "\t\t%s: %s,\n", strconv.Quote(entry.Key), strconv.Quote(entry.Operation))
	}
	fmt.Fprintf(&buf, "\t})\n")
	fmt.Fprintf(&buf, "\tmiddleware.SetOperationResolver(resolver)\n")
	fmt.Fprintf(&buf, "\treturn resolver\n")
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func provideErrorHandlers(m *metrics.Metrics, operationResolver middleware.OperationResolver) *middleware.ErrorHandlers {\n")
	fmt.Fprintf(&buf, "\treturn middleware.NewErrorHandlers(m, operationResolver)\n")
	fmt.Fprintf(&buf, "}\n\n")

	// Router provider
	fmt.Fprintf(&buf, "func provideHTTPRouter(middlewares []func(http.Handler) http.Handler, errHandlers *middleware.ErrorHandlers, ")
	var routerParams []string
	for _, ws := range wireSpecs {
		imp := ws.Imp
		spec := ws.Spec
		routerParams = append(routerParams, fmt.Sprintf("%s %s.ServerInterface", spec.PkgName+"Handler", imp.GenAlias))
	}
	fmt.Fprintf(&buf, "%s) *chi.Mux {\n", strings.Join(routerParams, ", "))
	fmt.Fprintf(&buf, "\tr := chi.NewRouter()\n")
	fmt.Fprintf(&buf, "\tfor _, mw := range middlewares {\n")
	fmt.Fprintf(&buf, "\t\tr.Use(mw)\n")
	fmt.Fprintf(&buf, "\t}\n")
	for _, ws := range wireSpecs {
		imp := ws.Imp
		spec := ws.Spec
		fmt.Fprintf(&buf, "\t%s.HandlerWithOptions(%s, %s.ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: errHandlers.RequestErrorHandler})\n", imp.GenAlias, spec.PkgName+"Handler", imp.GenAlias)
	}
	fmt.Fprintf(&buf, "\treturn r\n")
	fmt.Fprintf(&buf, "}\n\n")

	// ProviderSet
	fmt.Fprintf(&buf, "var HTTPProviderSet = wire.NewSet(\n")
	for _, ws := range wireSpecs {
		imp := ws.Imp
		spec := ws.Spec
		for _, tag := range spec.getSortedTags() {
			newFunc := getNewHandlerFuncName(tag)
			fmt.Fprintf(&buf, "\t%s.%s,\n", imp.StubAlias, newFunc)
		}
		fmt.Fprintf(&buf, "\tprovide%sHandlers,\n", toPascalCase(spec.PkgName))
	}
	fmt.Fprintf(&buf, "\tprovideErrorHandlers,\n")
	fmt.Fprintf(&buf, "\tprovideOperationResolver,\n")
	fmt.Fprintf(&buf, "\tprovideHTTPRouter,\n")
	fmt.Fprintf(&buf, "\twire.Struct(new(HTTPApp), \"*\"),\n")
	fmt.Fprintf(&buf, ")\n\n")

	// InitializeHTTPApp
	fmt.Fprintf(&buf, "func InitializeHTTPApp(middlewares []func(http.Handler) http.Handler, m *metrics.Metrics, enableLogging bool) (*HTTPApp, error) {\n")
	fmt.Fprintf(&buf, "\twire.Build(HTTPProviderSet)\n")
	fmt.Fprintf(&buf, "\treturn nil, nil\n")
	fmt.Fprintf(&buf, "}\n")

	src, err := format.Source(buf.Bytes())
	if err != nil {
		if verboseLogs {
			log.Printf("failed to format openapi_wire.go: %v\nSource:\n%s", err, buf.String())
		}
		return err
	}

	src, err = goimports.Process("openapi_wire.go", src, nil)
	if err != nil {
		return err
	}

	return writeGeneratedFile(openapiWireOut, src)
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

type resolverEntry struct {
	Key       string
	Operation string
}

func collectResolverEntries(specs []*openapiSpec) []resolverEntry {
	entries := map[string]string{}
	for _, spec := range specs {
		for _, tag := range spec.getSortedTags() {
			for _, op := range spec.Tags[tag] {
				key := strings.ToUpper(op.Method) + " " + op.Path
				entries[key] = resolveOperationMethodName(spec, op)
			}
		}
	}
	out := make([]resolverEntry, 0, len(entries))
	for key, operation := range entries {
		out = append(out, resolverEntry{Key: key, Operation: operation})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

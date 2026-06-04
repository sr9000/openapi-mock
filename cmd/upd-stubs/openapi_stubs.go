package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	goimports "golang.org/x/tools/imports"
)

// generateOpenAPIStubs generates stub files for an OpenAPI spec
func generateOpenAPIStubs(spec *openapiSpec) error {
	outDir := filepath.Join(openapiStubsDir, spec.RelPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Generate a stub file for each tag
	for _, tag := range spec.getSortedTags() {
		ops := spec.Tags[tag]
		if err := generateOpenAPIStubFile(outDir, spec, tag, ops); err != nil {
			return fmt.Errorf("failed to generate stub for tag %s: %w", tag, err)
		}
	}

	// Generate provider.go
	if err := generateOpenAPIProvider(outDir, spec); err != nil {
		return fmt.Errorf("failed to generate provider: %w", err)
	}

	return nil
}

func generateOpenAPIStubFile(outDir string, spec *openapiSpec, tag string, ops []opInfo) error {
	structName := getHandlerStructName(tag)
	fileName := toSnakeCase(tag) + ".go"
	outPath := filepath.Join(outDir, fileName)

	// If file exists, update it; otherwise create new
	if _, err := os.Stat(outPath); err == nil {
		return updateOpenAPIStubFile(outPath, spec, tag, ops)
	}

	var buf bytes.Buffer

	imports := map[string]string{
		"context":                        "",
		"github.com/rs/zerolog":          "",
		spec.GenPkgPath:                  "gen",
		"openapi-mock/pkg/observability": "",
	}

	fmt.Fprintf(&buf, "package %s\n\n", spec.PkgName)
	writeImports(&buf, imports)
	buf.WriteString("\n")

	// Struct
	fmt.Fprintf(&buf, "type %s struct {\n", structName)
	fmt.Fprintf(&buf, "\tEnableLogging bool\n")
	fmt.Fprintf(&buf, "}\n\n")

	// Constructor
	fmt.Fprintf(&buf, "func %s(enableLogging bool) *%s {\n", getNewHandlerFuncName(tag), structName)
	fmt.Fprintf(&buf, "\treturn &%s{EnableLogging: enableLogging}\n", structName)
	fmt.Fprintf(&buf, "}\n\n")

	buf.WriteString(generateOpenAPILogHelper(structName))

	// Methods for each operation
	for _, op := range ops {
		method := generateOpenAPIMethod(structName, spec, op)
		fmt.Fprintf(&buf, "\n%s\n", method)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		if verboseLogs {
			log.Printf("failed to format %s: %v\nSource:\n%s", fileName, err, buf.String())
		}
		return err
	}

	// Use goimports to clean up unused imports
	src, err = goimports.Process(outPath, src, nil)
	if err != nil {
		log.Printf("failed to process imports for %s: %v", fileName, err)
		return err
	}

	return writeGeneratedFile(outPath, src)
}

func generateOpenAPIMethod(structName string, spec *openapiSpec, op opInfo) string {
	var buf bytes.Buffer

	methodName := resolveOperationMethodName(spec, op)

	// Strict signature types
	reqType := fmt.Sprintf("gen.%sRequestObject", methodName)
	respType := fmt.Sprintf("gen.%sResponseObject", methodName)

	fmt.Fprintf(&buf, "func (h *%s) %s(ctx context.Context, request %s) (%s, error) {\n", structName, methodName, reqType, respType)
	fmt.Fprintf(&buf, "\tif h.EnableLogging {\n")
	fmt.Fprintf(&buf, "\t\tlogger := h.logger(ctx)\n")
	fmt.Fprintf(&buf, "\t\tlogger.Info().Str(\"handler\", %q).Msg(%q)\n", structName, methodName)
	fmt.Fprintf(&buf, "\t}\n")

	// Avoid unused warning for request in basic stubs
	fmt.Fprintf(&buf, "\t_ = request\n\n")

	buf.WriteString(generateOpenAPIMethodBody(spec, op, methodName))
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

func generateOpenAPILogHelper(structName string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "func (h *%s) logger(ctx context.Context) zerolog.Logger {\n", structName)
	fmt.Fprintf(&buf, "\treturn observability.Logger(ctx, zerolog.Nop())\n")
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

func generateOpenAPIMethodBody(spec *openapiSpec, op opInfo, methodName string) string {
	var buf bytes.Buffer

	successCode, successResp := pickPrimaryResponse(op.Operation.Responses.Map())

	if successCode == "" || successResp == nil {
		// No declared response: return empty 200
		buf.WriteString(fmt.Sprintf("\treturn gen.%s200Response{}, nil\n", methodName))
		return buf.String()
	}

	// Normalize code (e.g. "default" -> "Default")
	successCode = toPascalCase(successCode)

	// Decide whether this is a JSON response type (oapi-codegen generates both
	// <Op><Code>JSONResponse and <Op><Code>Response depending on the spec).
	jsonBody := responseHasJSONBody(successResp)

	switch {
	case jsonBody:
		// For JSON responses, always return the generated wrapper type. For inline/anonymous
		// schemas, this is the safest option and doesn't require us to guess the body type.
		buf.WriteString(fmt.Sprintf("\treturn gen.%s%sJSONResponse{}, nil\n", methodName, successCode))
	default:
		// Empty/non-JSON response wrappers
		buf.WriteString(fmt.Sprintf("\treturn gen.%s%sResponse{}, nil\n", methodName, successCode))
	}

	return buf.String()
}

func responseHasJSONBody(resp *openapi3.Response) bool {
	if resp == nil || resp.Content == nil {
		return false
	}
	jsonContent := resp.Content.Get("application/json")
	return jsonContent != nil && jsonContent.Schema != nil
}

func updateOpenAPIStubFile(path string, spec *openapiSpec, tag string, ops []opInfo) error {
	// Read existing file
	existingSrc, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	structName := getHandlerStructName(tag)

	// Parse to find existing methods
	existingMethods := parseExistingMethods(existingSrc, structName)
	helperName := fmt.Sprintf("func (h *%s) logger(ctx context.Context) zerolog.Logger", structName)
	helperMissing := !strings.Contains(string(existingSrc), helperName)

	// Find missing methods
	var newMethods []string
	if helperMissing {
		newMethods = append(newMethods, generateOpenAPILogHelper(structName))
	}
	for _, op := range ops {
		methodName := resolveOperationMethodName(spec, op)
		if _, exists := existingMethods[methodName]; !exists {
			method := generateOpenAPIMethod(structName, spec, op)
			newMethods = append(newMethods, method)
		}
	}

	if len(newMethods) == 0 {
		if pruneStale {
			prunedSrc, changed := annotateOrphanedMethods(spec, existingSrc, existingMethods, ops)
			if changed {
				formatted, err := format.Source(prunedSrc)
				if err != nil {
					return err
				}
				formatted, err = goimports.Process(path, formatted, nil)
				if err != nil {
					return err
				}
				return writeGeneratedFile(path, formatted)
			}
		}
		return nil // No changes needed
	}

	// Append new methods
	var buf bytes.Buffer
	buf.Write(existingSrc)
	for _, method := range newMethods {
		buf.WriteString("\n\n")
		buf.WriteString(method)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Printf("failed to format updated %s: %v", path, err)
		return err
	}

	src, err = goimports.Process(path, src, nil)
	if err != nil {
		return err
	}

	return writeGeneratedFile(path, src)
}

const (
	orphanBlockStart = "// upd-stubs: orphaned-methods-start"
	orphanBlockEnd   = "// upd-stubs: orphaned-methods-end"
)

func annotateOrphanedMethods(spec *openapiSpec, existingSrc []byte, existingMethods map[string]bool, ops []opInfo) ([]byte, bool) {
	expected := map[string]bool{}
	for _, op := range ops {
		expected[resolveOperationMethodName(spec, op)] = true
	}

	orphaned := make([]string, 0)
	for method := range existingMethods {
		if !expected[method] {
			orphaned = append(orphaned, method)
		}
	}
	sort.Strings(orphaned)

	original := string(existingSrc)
	updated := removeOrphanBlock(original)
	if len(orphaned) == 0 {
		return []byte(updated), updated != original
	}

	var block strings.Builder
	block.WriteString("\n")
	block.WriteString(orphanBlockStart)
	block.WriteString("\n")
	block.WriteString("// Methods listed below exist in this file but are no longer declared in the current OpenAPI spec.\n")
	for _, method := range orphaned {
		block.WriteString("// - ")
		block.WriteString(method)
		block.WriteString("\n")
	}
	block.WriteString(orphanBlockEnd)
	block.WriteString("\n")

	updated = strings.TrimRight(updated, "\n") + block.String()
	return []byte(updated), updated != original
}

func removeOrphanBlock(src string) string {
	start := strings.Index(src, orphanBlockStart)
	if start < 0 {
		return src
	}
	end := strings.Index(src[start:], orphanBlockEnd)
	if end < 0 {
		return src
	}
	end += start + len(orphanBlockEnd)
	return strings.TrimRight(src[:start]+src[end:], "\n") + "\n"
}

func parseExistingMethods(src []byte, structName string) map[string]bool {
	methods := make(map[string]bool)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "existing.go", src, parser.SkipObjectResolution)
	if err != nil {
		return methods
	}

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}

		recvType := fn.Recv.List[0].Type
		switch r := recvType.(type) {
		case *ast.StarExpr:
			ident, ok := r.X.(*ast.Ident)
			if ok && ident.Name == structName {
				methods[fn.Name.Name] = true
			}
		case *ast.Ident:
			if r.Name == structName {
				methods[fn.Name.Name] = true
			}
		}
		return true
	})

	return methods
}

func pickPrimaryResponse(responses map[string]*openapi3.ResponseRef) (string, *openapi3.Response) {
	type candidate struct {
		code     string
		priority int
		numeric  int
	}

	var best *candidate
	for code, ref := range responses {
		if ref == nil || ref.Value == nil {
			continue
		}

		priority := responseCodePriority(code)
		numeric := 0
		if n, err := strconv.Atoi(code); err == nil {
			numeric = n
		}

		cur := &candidate{code: code, priority: priority, numeric: numeric}
		if best == nil || cur.priority < best.priority || (cur.priority == best.priority && cur.numeric < best.numeric) || (cur.priority == best.priority && cur.numeric == best.numeric && cur.code < best.code) {
			best = cur
		}
	}

	if best == nil {
		return "", nil
	}
	return best.code, responses[best.code].Value
}

func responseCodePriority(code string) int {
	if code == "default" {
		return 5
	}
	if len(code) > 0 && code[0] >= '1' && code[0] <= '5' {
		switch code[0] {
		case '2':
			return 0
		case '3':
			return 1
		case '1':
			return 2
		case '4':
			return 3
		case '5':
			return 4
		}
	}
	return 6
}

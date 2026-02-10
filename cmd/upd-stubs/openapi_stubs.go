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
		"encoding/json":         "",
		"log":                   "",
		"net/http":              "",
		"grpc-mock/pkg/ctxkeys": "",
		spec.GenPkgPath:         "gen",
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
	fmt.Fprintf(&buf, "}\n")

	// Methods for each operation
	for _, op := range ops {
		method := generateOpenAPIMethod(structName, spec, op)
		fmt.Fprintf(&buf, "\n%s\n", method)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Printf("failed to format %s: %v\nSource:\n%s", fileName, err, buf.String())
		return err
	}

	// Use goimports to clean up unused imports
	src, err = goimports.Process(outPath, src, nil)
	if err != nil {
		log.Printf("failed to process imports for %s: %v", fileName, err)
		return err
	}

	return os.WriteFile(outPath, src, 0o644)
}

func generateOpenAPIMethod(structName string, spec *openapiSpec, op opInfo) string {
	var buf bytes.Buffer

	methodName := op.OperationID
	if methodName == "" {
		methodName = toPascalCase(op.Method + "_" + strings.ReplaceAll(op.Path, "/", "_"))
	}

	// Get method signature from generated ServerInterface
	sig := getOpenAPIMethodSignature(spec, op)

	fmt.Fprintf(&buf, "func (h *%s) %s(%s) {\n", structName, methodName, sig.Params)
	fmt.Fprintf(&buf, "\tif h.EnableLogging {\n")
	fmt.Fprintf(&buf, "\t\treqID, _ := r.Context().Value(ctxkeys.RequestID{}).(string)\n")
	fmt.Fprintf(&buf, "\t\tlog.Printf(\"[req_id=%%s] [%s] %s\", reqID)\n", structName, methodName)
	fmt.Fprintf(&buf, "\t}\n\n")

	// Generate response based on operation
	buf.WriteString(generateOpenAPIMethodBody(spec, op))

	fmt.Fprintf(&buf, "}")
	return buf.String()
}

type methodSignature struct {
	Params string
	Return string
}

func getOpenAPIMethodSignature(spec *openapiSpec, op opInfo) methodSignature {
	var params []string
	params = append(params, "w http.ResponseWriter", "r *http.Request")

	// Add path parameters
	for _, param := range op.Operation.Parameters {
		if param.Value == nil {
			continue
		}
		if param.Value.In == "path" {
			paramName := param.Value.Name
			paramType := schemaToGoType(param.Value.Schema)
			params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
		}
	}

	// Add query params struct if exists
	if hasQueryParams(op.Operation) {
		params = append(params, fmt.Sprintf("params gen.%sParams", op.OperationID))
	}

	return methodSignature{
		Params: strings.Join(params, ", "),
		Return: "",
	}
}

func hasQueryParams(op *openapi3.Operation) bool {
	for _, param := range op.Parameters {
		if param.Value != nil && param.Value.In == "query" {
			return true
		}
	}
	return false
}

func schemaToGoType(schemaRef *openapi3.SchemaRef) string {
	if schemaRef == nil || schemaRef.Value == nil {
		return "interface{}"
	}
	schema := schemaRef.Value

	switch schema.Type.Slice()[0] {
	case "integer":
		if schema.Format == "int64" {
			return "int64"
		}
		return "int"
	case "number":
		if schema.Format == "float" {
			return "float32"
		}
		return "float64"
	case "boolean":
		return "bool"
	case "string":
		return "string"
	case "array":
		elemType := schemaToGoType(schema.Items)
		return "[]" + elemType
	default:
		return "interface{}"
	}
}

func generateOpenAPIMethodBody(spec *openapiSpec, op opInfo) string {
	var buf bytes.Buffer

	// Find successful response
	var successCode string
	var successResp *openapi3.Response
	for code, resp := range op.Operation.Responses.Map() {
		if strings.HasPrefix(code, "2") {
			successCode = code
			successResp = resp.Value
			break
		}
	}

	if successCode == "" || successResp == nil {
		buf.WriteString("\tw.WriteHeader(http.StatusOK)\n")
		return buf.String()
	}

	// Handle different response codes
	switch successCode {
	case "204":
		buf.WriteString("\tw.WriteHeader(http.StatusNoContent)\n")
	case "201":
		buf.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		buf.WriteString("\tw.WriteHeader(http.StatusCreated)\n")
		writeResponseBody(&buf, spec, successResp)
	default:
		buf.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
		writeResponseBody(&buf, spec, successResp)
	}

	return buf.String()
}

func writeResponseBody(buf *bytes.Buffer, spec *openapiSpec, resp *openapi3.Response) {
	if resp.Content == nil {
		buf.WriteString("\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n")
		return
	}

	jsonContent := resp.Content.Get("application/json")
	if jsonContent == nil || jsonContent.Schema == nil {
		buf.WriteString("\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n")
		return
	}

	schema := jsonContent.Schema
	if schema.Ref != "" {
		// Reference to a schema - generate mock based on schema name
		schemaName := filepath.Base(schema.Ref)
		buf.WriteString(fmt.Sprintf("\tjson.NewEncoder(w).Encode(gen.%s{})\n", schemaName))
		return
	}

	if schema.Value == nil {
		buf.WriteString("\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n")
		return
	}

	switch schema.Value.Type.Slice()[0] {
	case "array":
		if schema.Value.Items != nil && schema.Value.Items.Ref != "" {
			itemName := filepath.Base(schema.Value.Items.Ref)
			buf.WriteString(fmt.Sprintf("\tjson.NewEncoder(w).Encode([]gen.%s{})\n", itemName))
		} else {
			buf.WriteString("\tjson.NewEncoder(w).Encode([]interface{}{})\n")
		}
	case "object":
		buf.WriteString("\tjson.NewEncoder(w).Encode(map[string]interface{}{})\n")
	default:
		buf.WriteString("\tjson.NewEncoder(w).Encode(map[string]string{\"status\": \"ok\"})\n")
	}
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

	// Find missing methods
	var newMethods []string
	for _, op := range ops {
		methodName := op.OperationID
		if methodName == "" {
			methodName = toPascalCase(op.Method + "_" + strings.ReplaceAll(op.Path, "/", "_"))
		}
		if _, exists := existingMethods[methodName]; !exists {
			method := generateOpenAPIMethod(structName, spec, op)
			newMethods = append(newMethods, method)
		}
	}

	if len(newMethods) == 0 {
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

	return os.WriteFile(path, src, 0o644)
}

func parseExistingMethods(src []byte, structName string) map[string]bool {
	methods := make(map[string]bool)

	// Simple parsing - look for "func (h *StructName) MethodName("
	lines := strings.Split(string(src), "\n")
	prefix := fmt.Sprintf("func (h *%s)", structName)
	altPrefix := fmt.Sprintf("func (h *%s)", structName)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) || strings.HasPrefix(line, altPrefix) {
			// Extract method name
			rest := strings.TrimPrefix(line, prefix)
			rest = strings.TrimPrefix(rest, altPrefix)
			rest = strings.TrimSpace(rest)
			if idx := strings.Index(rest, "("); idx > 0 {
				methodName := rest[:idx]
				methods[methodName] = true
			}
		}
	}

	return methods
}

// sortedStringKeys returns sorted keys from a map
func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

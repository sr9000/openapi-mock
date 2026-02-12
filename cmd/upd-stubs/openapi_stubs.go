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
		"context":                  "",
		"log":                      "",
		"openapi-mock/pkg/ctxkeys": "",
		spec.GenPkgPath:            "gen",
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

	// Strict signature types
	reqType := fmt.Sprintf("gen.%sRequestObject", methodName)
	respType := fmt.Sprintf("gen.%sResponseObject", methodName)

	fmt.Fprintf(&buf, "func (h *%s) %s(ctx context.Context, request %s) (%s, error) {\n", structName, methodName, reqType, respType)
	fmt.Fprintf(&buf, "\tif h.EnableLogging {\n")
	fmt.Fprintf(&buf, "\t\treqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)\n")
	fmt.Fprintf(&buf, "\t\tlog.Printf(\"[req_id=%%s] [%s] %s\", reqID)\n", structName, methodName)
	fmt.Fprintf(&buf, "\t}\n\n")

	// Avoid unused warning for request in basic stubs
	fmt.Fprintf(&buf, "\t_ = request\n\n")

	buf.WriteString(generateOpenAPIMethodBody(spec, op, methodName))
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

func generateOpenAPIMethodBody(spec *openapiSpec, op opInfo, methodName string) string {
	var buf bytes.Buffer

	// Find response with smallest code
	var successCode string
	var successResp *openapi3.Response

	responses := op.Operation.Responses.Map()
	codes := make([]string, 0, len(responses))
	for code := range responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	if len(codes) > 0 {
		successCode = codes[0]
		successResp = responses[successCode].Value
	}

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

func responseBodyExpr(spec *openapiSpec, resp *openapi3.Response) string {
	if resp == nil || resp.Content == nil {
		return "map[string]string{\"status\": \"ok\"}"
	}

	jsonContent := resp.Content.Get("application/json")
	if jsonContent == nil || jsonContent.Schema == nil {
		return "map[string]string{\"status\": \"ok\"}"
	}

	schema := jsonContent.Schema
	if schema.Value == nil {
		// If it's a $ref, oapi-codegen will have the model in types.gen.go, but schema.Value
		// can still be nil depending on how kin-openapi provides it; keep it safe.
		return "map[string]string{\"status\": \"ok\"}"
	}

	// For strict-server JSON responses, the response wrapper type expects the *exact*
	// element type generated in server.gen.go. For many specs, list responses are
	// generated as anonymous structs, so we simply return an empty literal of the
	// wrapper type, which is always assignable.
	//
	// This keeps stubs compiling even when schemas are inline/anonymous.
	if schema.Ref == "" {
		switch schema.Value.Type.Slice()[0] {
		case "array":
			return "[]interface{}{}"
		case "object":
			return "map[string]interface{}{}"
		}
	}

	// If the schema is a $ref to a named model, try to instantiate it.
	if schema.Ref != "" {
		modelName := filepath.Base(schema.Ref)
		return fmt.Sprintf("gen.%s{}", modelName)
	}

	return "map[string]string{\"status\": \"ok\"}"
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

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const openapiModulePfx = "openapi-mock"

var (
	specsDir        = "api"
	openapiGenDir   = "internal/generated"
	openapiStubsDir = "internal/stubs"
)

// openapiSpec represents a discovered OpenAPI specification
type openapiSpec struct {
	SpecPath    string              // Path to the openapi.yaml file
	RelPath     string              // Relative path from api/ (e.g., "petstore" or "petstore/v3")
	PkgName     string              // Package name (e.g., "petstore")
	GenPkgPath  string              // Generated package import path
	StubPkgPath string              // Stubs package import path
	StrictNames map[string]bool     // StrictServerInterface method names
	Doc         *openapi3.T         // Parsed OpenAPI document
	Tags        map[string][]opInfo // Operations grouped by tag
}

// opInfo represents a single OpenAPI operation
type opInfo struct {
	OperationID string
	Method      string // GET, POST, etc.
	Path        string
	Operation   *openapi3.Operation
}

// discoverOpenAPISpecs finds all OpenAPI specs in the specs directory
func discoverOpenAPISpecs() ([]*openapiSpec, error) {
	var specs []*openapiSpec
	if _, err := os.Stat(specsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	err := filepath.Walk(specsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if name != "openapi.yaml" && name != "openapi.json" && name != "openapi.yml" {
			return nil
		}

		spec, err := loadOpenAPISpec(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}
		specs = append(specs, spec)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].RelPath < specs[j].RelPath
	})

	return specs, nil
}

func loadOpenAPISpec(path string) (*openapiSpec, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	relPath := strings.TrimPrefix(dir, specsDir+string(os.PathSeparator))
	relPath = filepath.ToSlash(relPath) // Normalize to forward slashes
	if relPath == "." || relPath == "" {
		relPath = filepath.Base(specsDir)
	}

	pkgName := sanitizeGoPackageName(filepath.Base(relPath))

	spec := &openapiSpec{
		SpecPath:    path,
		RelPath:     relPath,
		PkgName:     pkgName,
		GenPkgPath:  fmt.Sprintf("%s/%s/%s", openapiModulePfx, openapiGenDir, relPath),
		StubPkgPath: fmt.Sprintf("%s/%s/%s", openapiModulePfx, openapiStubsDir, relPath),
		StrictNames: make(map[string]bool),
		Doc:         doc,
		Tags:        make(map[string][]opInfo),
	}

	// Group operations by tag
	for pathStr, pathItem := range doc.Paths.Map() {
		for method, op := range pathItem.Operations() {
			if op == nil {
				continue
			}

			info := opInfo{
				OperationID: op.OperationID,
				Method:      method,
				Path:        pathStr,
				Operation:   op,
			}

			tagName := normalizeTagName(primaryTag(op.Tags))
			spec.Tags[tagName] = append(spec.Tags[tagName], info)
		}
	}

	// Determine operation order from StrictServerInterface in generated server code.
	// This ensures that generated stubs match the interface order (e.g. methods are sorted).
	serverGenPath := filepath.Join(openapiGenDir, relPath, "server.gen.go")
	opOrder, strictNames, err := getMethodOrderFromInterface(serverGenPath, "StrictServerInterface")
	if err != nil {
		log.Printf("warning: failed to read %s for method ordering: %v", serverGenPath, err)
	} else {
		spec.StrictNames = strictNames
	}

	// Sort operations within each tag
	for tag := range spec.Tags {
		sort.Slice(spec.Tags[tag], func(i, j int) bool {
			op1 := spec.Tags[tag][i]
			op2 := spec.Tags[tag][j]

			// We try to match the operation ID (normalized if needed).
			// If OperationID is empty (not common with strict mode), we might fall back.
			// Ideally, oapi-codegen uses OperationID as function name.
			id1 := resolveOperationMethodName(spec, op1)
			id2 := resolveOperationMethodName(spec, op2)

			idx1, ok1 := opOrder[id1]
			idx2, ok2 := opOrder[id2]

			if ok1 && ok2 {
				return idx1 < idx2
			}
			if ok1 {
				return true
			}
			if ok2 {
				return false
			}

			// Fallback: stable sort by Path then Method
			if op1.Path != op2.Path {
				return op1.Path < op2.Path
			}
			return op1.Method < op2.Method
		})
	}

	return spec, nil
}

func primaryTag(tags []string) string {
	if len(tags) == 0 {
		return "default"
	}
	return tags[0]
}

func normalizeTagName(tag string) string {
	tag = sanitizeGoIdentifier(strings.TrimSpace(tag))
	if tag == "" {
		return "default"
	}
	return tag
}

func getOperationMethodName(op opInfo) string {
	if op.OperationID != "" {
		return toPascalCase(op.OperationID)
	}
	return toPascalCase(op.Method + "_" + strings.ReplaceAll(op.Path, "/", "_"))
}

func resolveOperationMethodName(spec *openapiSpec, op opInfo) string {
	methodName := getOperationMethodName(op)
	if len(spec.StrictNames) == 0 || spec.StrictNames[methodName] {
		return methodName
	}
	methodNorm := strings.ToLower(strings.ReplaceAll(methodName, "_", ""))
	for strictName := range spec.StrictNames {
		if strings.ToLower(strings.ReplaceAll(strictName, "_", "")) == methodNorm {
			return strictName
		}
	}
	return methodName
}

// getSortedTags returns tag names sorted alphabetically
func (s *openapiSpec) getSortedTags() []string {
	tags := make([]string, 0, len(s.Tags))
	for tag := range s.Tags {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// getHandlerStructName returns the struct name for a tag's handlers
func getHandlerStructName(tag string) string {
	return toPascalCase(tag) + "Handlers"
}

// getNewHandlerFuncName returns the constructor name for a tag's handlers
func getNewHandlerFuncName(tag string) string {
	return "New" + toPascalCase(tag) + "Handlers"
}

func getMethodOrderFromInterface(filePath string, interfaceName string) (map[string]int, map[string]bool, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	order := make(map[string]int)
	names := make(map[string]bool)
	counter := 0

	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if typeSpec.Name.Name != interfaceName {
			return true
		}

		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}

		for _, method := range interfaceType.Methods.List {
			if len(method.Names) > 0 {
				methodName := method.Names[0].Name
				order[methodName] = counter
				names[methodName] = true
				counter++
			}
		}
		return false
	})

	return order, names, nil
}

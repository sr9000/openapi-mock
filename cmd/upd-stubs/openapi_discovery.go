package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

const (
	specsDir         = "specs"
	openapiGenDir    = "internal/generated"
	openapiStubsDir  = "internal/stubs"
	openapiModulePfx = "openapi-mock"
)

// openapiSpec represents a discovered OpenAPI specification
type openapiSpec struct {
	SpecPath    string              // Path to the openapi.yaml file
	RelPath     string              // Relative path from specs/ (e.g., "petstore")
	PkgName     string              // Package name (e.g., "petstore")
	GenPkgPath  string              // Generated package import path
	StubPkgPath string              // Stubs package import path
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
		if os.IsNotExist(err) {
			return nil, nil // No specs directory is OK
		}
		return nil, err
	}

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

	pkgName := strings.ReplaceAll(filepath.Base(relPath), "-", "_")

	spec := &openapiSpec{
		SpecPath:    path,
		RelPath:     relPath,
		PkgName:     pkgName,
		GenPkgPath:  fmt.Sprintf("%s/%s/%s", openapiModulePfx, openapiGenDir, relPath),
		StubPkgPath: fmt.Sprintf("%s/%s/%s", openapiModulePfx, openapiStubsDir, relPath),
		Doc:         doc,
		Tags:        make(map[string][]opInfo),
	}

	// Group operations by tag
	for pathStr, pathItem := range doc.Paths.Map() {
		for method, op := range pathItem.Operations() {
			if op == nil {
				continue
			}

			tags := op.Tags
			if len(tags) == 0 {
				tags = []string{"default"}
			}

			info := opInfo{
				OperationID: op.OperationID,
				Method:      method,
				Path:        pathStr,
				Operation:   op,
			}

			for _, tag := range tags {
				tagName := strings.ToLower(strings.ReplaceAll(tag, " ", "_"))
				spec.Tags[tagName] = append(spec.Tags[tagName], info)
			}
		}
	}

	// Determine operation order from file structure for stability
	opOrder, _ := getOperationOrder(path)

	// Sort operations within each tag
	for tag := range spec.Tags {
		sort.Slice(spec.Tags[tag], func(i, j int) bool {
			op1 := spec.Tags[tag][i]
			op2 := spec.Tags[tag][j]
			key1 := op1.Path + "|" + op1.Method
			key2 := op2.Path + "|" + op2.Method

			idx1, ok1 := opOrder[key1]
			idx2, ok2 := opOrder[key2]

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

func getOperationOrder(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var node yaml.Node
	if err := yaml.NewDecoder(f).Decode(&node); err != nil {
		return nil, err
	}

	if len(node.Content) == 0 {
		return nil, nil
	}
	root := node.Content[0]

	// Find "paths"
	var pathsNode *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "paths" {
			pathsNode = root.Content[i+1]
			break
		}
	}

	if pathsNode == nil {
		return nil, nil
	}

	order := make(map[string]int)
	counter := 0

	// pathsNode.Content contains key, value, key, value...
	for i := 0; i < len(pathsNode.Content); i += 2 {
		pathKey := pathsNode.Content[i].Value
		pathItem := pathsNode.Content[i+1]

		// pathItem.Content contains method, op, method, op...
		for j := 0; j < len(pathItem.Content); j += 2 {
			methodKey := strings.ToUpper(pathItem.Content[j].Value)
			key := pathKey + "|" + methodKey
			order[key] = counter
			counter++
		}
	}

	return order, nil
}

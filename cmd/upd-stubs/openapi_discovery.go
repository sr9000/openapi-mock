package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
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

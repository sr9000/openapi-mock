package main

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"

	goimports "golang.org/x/tools/imports"
)

const mockDocsOut = "internal/app/mock_docs_gen.go"

// generateMockDocsFile generates internal/app/mock_docs_gen.go.
func generateMockDocsFile(specs []*openapiSpec) error {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "package app\n\n")
	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"encoding/json\"\n")
	fmt.Fprintf(&buf, "\t\"sync\"\n\n")
	fmt.Fprintf(&buf, "\t\"openapi-mock/pkg/mgmt\"\n")

	type specImport struct {
		Alias string
		Path  string
	}
	imports := make([]specImport, 0, len(specs))
	for i, spec := range specs {
		aliasBase := sanitizeGoIdentifier(strings.ReplaceAll(spec.RelPath, "/", "_"))
		imports = append(imports, specImport{
			Alias: fmt.Sprintf("%sdoc%d", aliasBase, i),
			Path:  spec.GenPkgPath,
		})
	}
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})
	for _, imp := range imports {
		fmt.Fprintf(&buf, "\t%s %q\n", imp.Alias, imp.Path)
	}
	fmt.Fprintf(&buf, ")\n\n")

	fmt.Fprintf(&buf, "func MockDocs() []mgmt.MockDoc {\n")
	fmt.Fprintf(&buf, "\treturn []mgmt.MockDoc{\n")

	for i, spec := range specs {
		parts := strings.Split(spec.RelPath, "/")
		apiName := parts[0]
		apiVersion := ""
		if len(parts) > 1 {
			apiVersion = strings.Join(parts[1:], "/")
		}
		title := ""
		if spec.Doc != nil && spec.Doc.Info != nil {
			title = spec.Doc.Info.Title
		}
		aliasBase := sanitizeGoIdentifier(strings.ReplaceAll(spec.RelPath, "/", "_"))
		alias := fmt.Sprintf("%sdoc%d", aliasBase, i)

		fmt.Fprintf(&buf, "\t\tfunc() mgmt.MockDoc {\n")
		fmt.Fprintf(&buf, "\t\t\tvar once sync.Once\n")
		fmt.Fprintf(&buf, "\t\t\tvar cached []byte\n")
		fmt.Fprintf(&buf, "\t\t\tvar cachedErr error\n")
		fmt.Fprintf(&buf, "\t\t\treturn mgmt.MockDoc{\n")
		fmt.Fprintf(&buf, "\t\t\t\tAPIName: %q,\n", apiName)
		fmt.Fprintf(&buf, "\t\t\t\tAPIVersion: %q,\n", apiVersion)
		fmt.Fprintf(&buf, "\t\t\t\tTitle: %q,\n", title)
		fmt.Fprintf(&buf, "\t\t\t\tSpecJSON: func() ([]byte, error) {\n")
		fmt.Fprintf(&buf, "\t\t\t\t\tonce.Do(func() {\n")
		fmt.Fprintf(&buf, "\t\t\t\t\t\tdoc, err := %s.GetSwagger()\n", alias)
		fmt.Fprintf(&buf, "\t\t\t\t\t\tif err != nil {\n")
		fmt.Fprintf(&buf, "\t\t\t\t\t\t\tcachedErr = err\n")
		fmt.Fprintf(&buf, "\t\t\t\t\t\t\treturn\n")
		fmt.Fprintf(&buf, "\t\t\t\t\t\t}\n")
		fmt.Fprintf(&buf, "\t\t\t\t\t\tcached, cachedErr = json.Marshal(doc)\n")
		fmt.Fprintf(&buf, "\t\t\t\t\t})\n")
		fmt.Fprintf(&buf, "\t\t\t\t\treturn cached, cachedErr\n")
		fmt.Fprintf(&buf, "\t\t\t\t},\n")
		fmt.Fprintf(&buf, "\t\t\t}\n")
		fmt.Fprintf(&buf, "\t\t}(),\n")
	}

	fmt.Fprintf(&buf, "\t}\n")
	fmt.Fprintf(&buf, "}\n")

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	src, err = goimports.Process(mockDocsOut, src, nil)
	if err != nil {
		return err
	}

	return writeGeneratedFile(mockDocsOut, src)
}

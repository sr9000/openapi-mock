package main

import (
	"bytes"
	"fmt"
	"sort"
	"unicode"
)

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func toPascalCase(s string) string {
	var result []rune
	capitalizeNext := true
	for _, r := range s {
		if r == '_' || r == '-' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result = append(result, unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func writeImports(buf *bytes.Buffer, imports map[string]string) {
	fmt.Fprintf(buf, "import (\n")
	sortedImports := make([]string, 0, len(imports))
	for imp := range imports {
		sortedImports = append(sortedImports, imp)
	}
	sort.Strings(sortedImports)
	for _, imp := range sortedImports {
		alias := imports[imp]
		if alias != "" {
			fmt.Fprintf(buf, "\t%s %q\n", alias, imp)
		} else {
			fmt.Fprintf(buf, "\t%q\n", imp)
		}
	}
	fmt.Fprintf(buf, ")\n")
}

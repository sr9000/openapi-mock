package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func toSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, unicode.ToLower(r))
		} else {
			if len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
		}
	}
	return strings.Trim(string(result), "_")
}

func toPascalCase(s string) string {
	var result []rune
	capitalizeNext := true
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
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

func sanitizeGoIdentifier(name string) string {
	name = toSnakeCase(name)
	if name == "" {
		name = "value"
	}
	if isGoKeyword(name) {
		name += "_"
	}
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		name = "v_" + name
	}
	return name
}

func sanitizeGoPackageName(name string) string {
	name = sanitizeGoIdentifier(name)
	return strings.ToLower(name)
}

func isGoKeyword(s string) bool {
	_, ok := map[string]struct{}{
		"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
		"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
		"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
		"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
		"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
	}[s]
	return ok
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

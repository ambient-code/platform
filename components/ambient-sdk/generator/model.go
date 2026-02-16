package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Resource struct {
	Name              string
	Plural            string
	PathSegment       string
	Fields            []Field
	RequiredFields    []string
	PatchFields       []Field
	StatusPatchFields []Field
	HasDelete         bool
	HasPatch          bool
	HasStatusPatch    bool
	Actions           []string
}

type Field struct {
	Name       string
	GoName     string
	PythonName string
	TSName     string
	Type       string
	Format     string
	GoType     string
	PythonType string
	TSType     string
	Required   bool
	ReadOnly   bool
	JSONTag    string
}

type Spec struct {
	Resources []Resource
}

func toGoName(snakeName string) string {
	parts := strings.Split(snakeName, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if upper, ok := commonAcronyms[strings.ToUpper(part)]; ok {
			result.WriteString(upper)
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		result.WriteString(string(runes))
	}
	return result.String()
}

var commonAcronyms = map[string]string{
	"ID":   "ID",
	"URL":  "URL",
	"HTTP": "HTTP",
	"API":  "API",
	"UI":   "UI",
}

func toGoType(openAPIType, format string) string {
	switch openAPIType {
	case "string":
		if format == "date-time" {
			return "*time.Time"
		}
		return "string"
	case "integer":
		if format == "int32" {
			return "int32"
		}
		return "int"
	case "number":
		if format == "double" || format == "float" {
			return "float64"
		}
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "string"
	}
}

func toPythonType(openAPIType, format string) string {
	switch openAPIType {
	case "string":
		if format == "date-time" {
			return "Optional[datetime]"
		}
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	default:
		return "str"
	}
}

func pythonDefault(openAPIType, format string) string {
	switch openAPIType {
	case "string":
		if format == "date-time" {
			return "None"
		}
		return "\"\""
	case "integer":
		return "0"
	case "number":
		return "0.0"
	case "boolean":
		return "False"
	default:
		return "\"\""
	}
}

func jsonTag(name string, required bool) string {
	if required {
		return fmt.Sprintf("`json:\"%s\"`", name)
	}
	return fmt.Sprintf("`json:\"%s,omitempty\"`", name)
}

func toTSType(openAPIType, format string) string {
	switch openAPIType {
	case "string":
		if format == "date-time" {
			return "string"
		}
		return "string"
	case "integer":
		return "number"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

func tsDefault(openAPIType, format string) string {
	switch openAPIType {
	case "string":
		return "''"
	case "integer":
		return "0"
	case "number":
		return "0"
	case "boolean":
		return "false"
	default:
		return "''"
	}
}

func toCamelCase(snakeName string) string {
	parts := strings.Split(snakeName, "_")
	if len(parts) == 0 {
		return snakeName
	}
	var result strings.Builder
	result.WriteString(parts[0])
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		result.WriteString(string(runes))
	}
	return result.String()
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func toSnakeCase(camelCase string) string {
	var result strings.Builder
	for i, r := range camelCase {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

func pluralize(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "s") {
		return lower + "es"
	}
	if strings.HasSuffix(lower, "y") {
		return lower[:len(lower)-1] + "ies"
	}
	return lower + "s"
}

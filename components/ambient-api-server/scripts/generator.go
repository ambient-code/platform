package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/pflag"
)

/*

scripts/generator.go

This script generates basic CRUD functionality for a new Kind.

It's rude and crude, but it generates working code.

TODO: all of it can be better

*/

var (
	kind                        = "Asteroid"
	repo                        = "github.com/ambient/platform/components"
	project                     = "ambient-api-server"
	fields                      = ""
	plural                      = ""
	library                     = "github.com/openshift-online/rh-trex-ai"
	skipGenerate                = false
	openapiEndpointStart        = "# NEW ENDPOINT START"
	openapiEndpointEnd          = "# NEW ENDPOINT END"
	openApiSchemaStart          = "# NEW SCHEMA START"
	openApiSchemaEnd            = "# NEW SCHEMA END"
	openApiEndpointMatchingLine = "  # AUTO-ADD NEW PATHS"
	openApiSchemaMatchingLine   = "    # AUTO-ADD NEW SCHEMAS"
)

func init() {
	_ = flag.Set("logtostderr", "true")
	flags := pflag.CommandLine
	flags.AddGoFlagSet(flag.CommandLine)

	flags.StringVar(&kind, "kind", kind, "the name of the kind.  e.g Account or User")
	flags.StringVar(&repo, "repo", repo, "the module path of the repo (e.g. github.com/ambient/platform/components)")
	flags.StringVar(&project, "project", project, "the name of the project.  e.g ambient-api-server")
	flags.StringVar(&fields, "fields", fields, "comma-separated list of custom fields in format name:type (e.g. 'name:string,age:int,active:bool')")
	flags.StringVar(&plural, "plural", plural, "the plural form of the kind. If not provided, uses irregular plurals map or adds 's'")
	flags.StringVar(&library, "library", library, "the module path of the rh-trex-ai library (e.g. github.com/openshift-online/rh-trex-ai)")
	flags.BoolVar(&skipGenerate, "skip-generate", skipGenerate, "skip running make generate after code generation")
}

// irregularPlurals maps singular forms to their irregular plural forms
var irregularPlurals = map[string]string{
	"registry":   "registries",
	"category":   "categories",
	"company":    "companies",
	"country":    "countries",
	"family":     "families",
	"policy":     "policies",
	"directory":  "directories",
	"repository": "repositories",
	"security":   "securities",
}

func pluralize(word string) string {
	if plural != "" {
		return plural
	}

	wordLower := strings.ToLower(word)

	// Check for irregular plurals by examining word endings
	for singular, irregularPlural := range irregularPlurals {
		if strings.HasSuffix(wordLower, singular) {
			// Replace the ending with the irregular plural
			prefix := word[:len(word)-len(singular)]
			suffix := irregularPlural

			// Preserve case of the suffix based on the original word's last part
			originalSuffix := word[len(word)-len(singular):]
			if originalSuffix == strings.ToUpper(originalSuffix) {
				suffix = strings.ToUpper(irregularPlural)
			} else if len(originalSuffix) > 0 && originalSuffix[0] == strings.ToUpper(string(originalSuffix[0]))[0] {
				suffix = strings.ToUpper(string(irregularPlural[0])) + irregularPlural[1:]
			}

			return prefix + suffix
		}
	}

	return word + "s"
}

func getCmdDir() string {
	entries, err := os.ReadDir("cmd")
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return entry.Name()
		}
	}

	panic("No command directory found in cmd/")
}

func main() {
	// Parse flags
	pflag.Parse()

	// Parse custom fields
	parsedFields, err := parseFields(fields)
	if err != nil {
		panic(fmt.Sprintf("Error parsing fields: %v", err))
	}

	templates := []string{
		"api",
		"presenters",
		"dao",
		"services",
		"mock",
		"migration",
		"test",
		"test-factories",
		"testmain",
		"handlers",
		"openapi-kind",
		"plugin",
	}

	for _, nm := range templates {
		path := fmt.Sprintf("templates/generate-%s.txt", nm)
		contents, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		kindTmpl, err := template.New(nm).Parse(string(contents))
		if err != nil {
			panic(err)
		}

		kindLowerCamel := strings.ToLower(string(kind[0])) + kind[1:]
		kindPlural := pluralize(kind)
		kindPluralLower := pluralize(kindLowerCamel)
		kindPluralSnake := toSnakeCase(kindPlural)
		k := myWriter{
			Project:             project,
			ProjectPascalCase:   toPascalCase(project),
			Repo:                repo,
			Library:             library,
			Cmd:                 getCmdDir(),
			Kind:                kind,
			KindPlural:          kindPlural,
			KindLowerPlural:     kindPluralLower,
			KindLowerSingular:   kindLowerCamel,
			KindSnakeCasePlural: kindPluralSnake,
			Fields:              parsedFields,
		}

		now := time.Now()
		k.ID = fmt.Sprintf("%d%s%s%s%s", now.Year(), datePad(int(now.Month())), datePad(now.Day()), datePad(now.Hour()), datePad(now.Minute()))

		outputPaths := map[string]string{
			"generate-api":            fmt.Sprintf("plugins/%s/model.go", k.KindLowerPlural),
			"generate-presenters":     fmt.Sprintf("plugins/%s/presenter.go", k.KindLowerPlural),
			"generate-dao":            fmt.Sprintf("plugins/%s/dao.go", k.KindLowerPlural),
			"generate-handlers":       fmt.Sprintf("plugins/%s/handler.go", k.KindLowerPlural),
			"generate-migration":      fmt.Sprintf("plugins/%s/migration.go", k.KindLowerPlural),
			"generate-mock":           fmt.Sprintf("plugins/%s/mock_dao.go", k.KindLowerPlural),
			"generate-openapi-kind":   fmt.Sprintf("openapi/openapi.%s.yaml", k.KindLowerPlural),
			"generate-test-factories": fmt.Sprintf("plugins/%s/factory_test.go", k.KindLowerPlural),
			"generate-test":           fmt.Sprintf("plugins/%s/integration_test.go", k.KindLowerPlural),
			"generate-testmain":       fmt.Sprintf("plugins/%s/testmain_test.go", k.KindLowerPlural),
			"generate-services":       fmt.Sprintf("plugins/%s/service.go", k.KindLowerPlural),
			"generate-plugin":         fmt.Sprintf("plugins/%s/plugin.go", k.KindLowerPlural),
		}

		outputPath, ok := outputPaths["generate-"+nm]
		if !ok {
			panic("expected to find outputPath for " + nm)
		}

		// Create directory if it doesn't exist
		outputDir := filepath.Dir(outputPath)
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			panic(err)
		}

		f, err := os.Create(outputPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		w := bufio.NewWriter(f)
		err = kindTmpl.Execute(w, k)
		if err != nil {
			panic(err)
		}
		w.Flush()
		f.Sync()

		// Run gofmt on generated Go files
		if filepath.Ext(outputPath) == ".go" {
			gofmtCmd := exec.Command("gofmt", "-w", outputPath)
			if err := gofmtCmd.Run(); err != nil {
				fmt.Printf("Warning: gofmt failed for %s: %v\n", outputPath, err)
			}
		}

		if strings.EqualFold("generate-"+nm, "generate-openapi-kind") {
			modifyOpenapi("openapi/openapi.yaml", fmt.Sprintf("openapi/openapi.%s.yaml", k.KindLowerPlural))
		}

		// Add plugin import to main.go after generating the plugin
		if nm == "plugin" {
			addPluginImport(k)
		}

	}

	// Run make generate to regenerate OpenAPI client (skip with --skip-generate)
	if !skipGenerate {
		fmt.Println("Running make generate to regenerate OpenAPI client...")
		makeCmd := exec.Command("make", "generate")
		makeCmd.Stdout = os.Stdout
		makeCmd.Stderr = os.Stderr
		if err := makeCmd.Run(); err != nil {
			fmt.Printf("Warning: make generate failed: %v\n", err)
		}
	} else {
		fmt.Println("Skipping make generate (--skip-generate flag set)")
	}
}

func datePad(d int) string {
	if d < 10 {
		return fmt.Sprintf("0%d", d)
	}
	return fmt.Sprintf("%d", d)
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return pascal
	}
	return strings.ToLower(string(pascal[0])) + pascal[1:]
}

func parseFields(fieldsStr string) ([]Field, error) {
	if fieldsStr == "" {
		return []Field{}, nil
	}

	var fields []Field
	fieldPairs := strings.Split(fieldsStr, ",")
	for _, pair := range fieldPairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, fmt.Errorf("invalid field format: %s (expected name:type or name:type:required)", pair)
		}

		name := strings.TrimSpace(parts[0])
		fieldType := strings.TrimSpace(parts[1])
		nullable := true // Default to nullable

		// Check for :required or :optional suffix
		if len(parts) == 3 {
			modifier := strings.TrimSpace(parts[2])
			if modifier == "required" {
				nullable = false
			} else if modifier == "optional" {
				nullable = true
			} else {
				return nil, fmt.Errorf("invalid field modifier: %s (expected 'required' or 'optional')", modifier)
			}
		}

		field, err := mapFieldType(name, fieldType, nullable)
		if err != nil {
			return nil, err
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func mapFieldType(name, fieldType string, nullable bool) (Field, error) {
	goName := toPascalCase(name)
	snakeName := toSnakeCase(goName)
	camelName := toCamelCase(goName)

	field := Field{
		Name:          goName,
		Type:          fieldType,
		NameSnakeCase: snakeName,
		NameCamelCase: camelName,
		JSONTag:       fmt.Sprintf("`json:\"%s\"`", snakeName),
		Required:      !nullable,
		Nullable:      nullable,
	}

	var baseType string
	var pointerType string

	switch fieldType {
	case "string":
		baseType = "string"
		pointerType = "*string"
		field.DBType = "text"
		field.OpenAPIType = "string"
	case "int":
		baseType = "int"
		pointerType = "*int"
		field.DBType = "integer"
		field.OpenAPIType = "integer"
		field.OpenAPIFormat = "int32"
		field.NeedsIntConversion = true
	case "int64":
		baseType = "int64"
		pointerType = "*int64"
		field.DBType = "bigint"
		field.OpenAPIType = "integer"
		field.OpenAPIFormat = "int64"
	case "bool":
		baseType = "bool"
		pointerType = "*bool"
		field.DBType = "boolean"
		field.OpenAPIType = "boolean"
	case "float":
		baseType = "float64"
		pointerType = "*float64"
		field.DBType = "double precision"
		field.OpenAPIType = "number"
		field.OpenAPIFormat = "double"
	case "time":
		baseType = "time.Time"
		pointerType = "*time.Time"
		field.DBType = "timestamp"
		field.OpenAPIType = "string"
		field.OpenAPIFormat = "date-time"
	default:
		return field, fmt.Errorf("unsupported field type: %s (supported types: string, int, int64, bool, float, time)", fieldType)
	}

	// Set GoType based on nullability
	if nullable {
		field.GoType = pointerType
	} else {
		field.GoType = baseType
	}
	field.PointerType = pointerType

	return field, nil
}

type Field struct {
	Name               string
	Type               string
	GoType             string
	DBType             string
	OpenAPIType        string
	OpenAPIFormat      string
	NameSnakeCase      string
	NameCamelCase      string
	JSONTag            string
	GormTag            string
	Required           bool
	Nullable           bool
	PointerType        string
	NeedsIntConversion bool
}

type myWriter struct {
	Repo                string
	Project             string
	ProjectPascalCase   string
	Library             string
	Cmd                 string
	Kind                string
	KindPlural          string
	KindLowerPlural     string
	KindLowerSingular   string
	KindSnakeCasePlural string
	ID                  string
	Fields              []Field
}

func modifyOpenapi(mainPath string, kindPath string) {
	endpointStrings := readBetweenLines(kindPath, openapiEndpointStart, openapiEndpointEnd)
	kindFileName := strings.Split(kindPath, "/")[1]
	for _, line := range endpointStrings {
		endpointStr := strings.TrimSpace(line)
		endpointStr = strings.Replace(endpointStr, "/", "~1", -1)
		endpointStr = strings.Replace(endpointStr, ":", "", -1)
		refPath := fmt.Sprintf(`    $ref: '%s#/paths/%s'`, kindFileName, endpointStr)
		pathsLine := fmt.Sprintf("%s%s", line, refPath)
		writeAfterLine(mainPath, openApiEndpointMatchingLine, pathsLine)
	}
	schemaStrings := readBetweenLines(kindPath, openApiSchemaStart, openApiSchemaEnd)
	for _, line := range schemaStrings {
		schemaStr := strings.TrimSpace(line)
		schemaStr = strings.Replace(schemaStr, ":", "", -1)
		refPath := fmt.Sprintf(`      $ref: '%s#/components/schemas/%s'`, kindFileName, schemaStr)
		pathsLine := fmt.Sprintf("%s%s", line, refPath)
		writeAfterLine(mainPath, openApiSchemaMatchingLine, pathsLine)
	}
}

func readBetweenLines(path string, startLine string, endLine string) []string {
	readFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	readFlag := false
	var totalMatches []string
	var matchedString strings.Builder
	for fileScanner.Scan() {
		trimmed := strings.TrimSpace(fileScanner.Text())
		if trimmed == startLine {
			readFlag = true
		} else if trimmed == endLine {
			readFlag = false
			totalMatches = append(totalMatches, matchedString.String())
			matchedString.Reset()
		} else if readFlag {
			matchedString.WriteString(fileScanner.Text() + "\n")
		}
	}
	readFile.Close()
	return totalMatches
}

func writeAfterLine(path string, matchingLine string, lineToWrite string) {
	input, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	_ = strings.Replace(string(input), matchingLine, lineToWrite+"\n"+matchingLine, -1)
	output := bytes.Replace(input, []byte(matchingLine), []byte(lineToWrite+"\n"+matchingLine), -1)
	if err = os.WriteFile(path, output, 0666); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func addPluginImport(k myWriter) {
	mainFile := fmt.Sprintf("cmd/%s/main.go", k.Cmd)

	input, err := os.ReadFile(mainFile)
	if err != nil {
		panic(err)
	}

	// Check if the import already exists
	pluginImport := fmt.Sprintf(`_ "%s/%s/plugins/%s"`, k.Repo, k.Project, k.KindLowerPlural)
	if strings.Contains(string(input), pluginImport) {
		fmt.Printf("Plugin import already exists in %s\n", mainFile)
		return
	}

	// Find the import block and add the plugin import at the bottom
	importBlockStart := "import ("
	lines := strings.Split(string(input), "\n")
	var output []string

	for i, line := range lines {
		if strings.Contains(line, importBlockStart) {
			// Found the start, copy everything until closing parenthesis
			output = append(output, line)
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == ")" {
					// Insert new import before closing parenthesis
					output = append(output, "\t"+pluginImport)
					output = append(output, lines[j])
					// Copy remaining lines
					output = append(output, lines[j+1:]...)

					err = os.WriteFile(mainFile, []byte(strings.Join(output, "\n")), 0666)
					if err != nil {
						panic(err)
					}
					fmt.Printf("Added plugin import to %s\n", mainFile)
					return
				}
				output = append(output, lines[j])
			}
		}
		output = append(output, line)
	}

	panic("Could not find import block in " + mainFile)
}

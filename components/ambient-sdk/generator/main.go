package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

func main() {
	specPath := flag.String("spec", "", "path to openapi.yaml")
	goOut := flag.String("go-out", "", "output directory for Go SDK")
	pythonOut := flag.String("python-out", "", "output directory for Python SDK")
	tsOut := flag.String("ts-out", "", "output directory for TypeScript SDK")
	flag.Parse()

	if *specPath == "" {
		log.Fatal("--spec is required")
	}
	if *goOut == "" && *pythonOut == "" && *tsOut == "" {
		log.Fatal("at least one of --go-out, --python-out, or --ts-out is required")
	}

	spec, err := parseSpec(*specPath)
	if err != nil {
		log.Fatalf("parse spec: %v", err)
	}

	specHash, err := computeSpecHash(*specPath)
	if err != nil {
		log.Fatalf("compute spec hash: %v", err)
	}

	// Use relative path for spec source 
	relativeSpecPath := "../../ambient-api-server/openapi/openapi.yaml"
	
	header := GeneratedHeader{
		SpecPath:  relativeSpecPath,
		SpecHash:  specHash,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	fmt.Printf("Parsed %d resources from %s\n", len(spec.Resources), *specPath)
	for _, r := range spec.Resources {
		fmt.Printf("  %s (%s): %d fields, delete=%v\n", r.Name, r.PathSegment, len(r.Fields), r.HasDelete)
	}

	if *goOut != "" {
		if err := generateGo(spec, *goOut, header); err != nil {
			log.Fatalf("generate Go: %v", err)
		}
		fmt.Printf("Go SDK generated in %s\n", *goOut)
	}

	if *pythonOut != "" {
		if err := generatePython(spec, *pythonOut, header); err != nil {
			log.Fatalf("generate Python: %v", err)
		}
		fmt.Printf("Python SDK generated in %s\n", *pythonOut)
	}

	if *tsOut != "" {
		if err := generateTypeScript(spec, *tsOut, header); err != nil {
			log.Fatalf("generate TypeScript: %v", err)
		}
		fmt.Printf("TypeScript SDK generated in %s\n", *tsOut)
	}
}

type GeneratedHeader struct {
	SpecPath  string
	SpecHash  string
	Timestamp string
}

type goTemplateData struct {
	Header   GeneratedHeader
	Resource Resource
	Spec     *Spec
}

type pythonTemplateData struct {
	Header   GeneratedHeader
	Resource Resource
	Spec     *Spec
}

type tsTemplateData struct {
	Header   GeneratedHeader
	Resource Resource
	Spec     *Spec
}

func generateGo(spec *Spec, outDir string, header GeneratedHeader) error {
	typesDir := filepath.Join(outDir, "types")
	clientDir := filepath.Join(outDir, "client")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return err
	}

	tmplDir := filepath.Join(getTemplateDir(), "go")

	baseTmpl, err := loadTemplate(filepath.Join(tmplDir, "base.go.tmpl"))
	if err != nil {
		return fmt.Errorf("load base template: %w", err)
	}
	if err := executeTemplate(baseTmpl, filepath.Join(typesDir, "base.go"), goTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute base template: %w", err)
	}

	typesTmpl, err := loadTemplate(filepath.Join(tmplDir, "types.go.tmpl"))
	if err != nil {
		return fmt.Errorf("load types template: %w", err)
	}

	clientTmpl, err := loadTemplate(filepath.Join(tmplDir, "client.go.tmpl"))
	if err != nil {
		return fmt.Errorf("load client template: %w", err)
	}

	for _, r := range spec.Resources {
		data := goTemplateData{Header: header, Resource: r, Spec: spec}
		fileName := toSnakeCase(r.Name) + ".go"

		if err := executeTemplate(typesTmpl, filepath.Join(typesDir, fileName), data); err != nil {
			return fmt.Errorf("execute types template for %s: %w", r.Name, err)
		}

		apiFileName := toSnakeCase(r.Name) + "_api.go"
		if err := executeTemplate(clientTmpl, filepath.Join(clientDir, apiFileName), data); err != nil {
			return fmt.Errorf("execute client template for %s: %w", r.Name, err)
		}
	}

	iteratorTmpl, err := loadTemplate(filepath.Join(tmplDir, "iterator.go.tmpl"))
	if err != nil {
		return fmt.Errorf("load iterator template: %w", err)
	}
	if err := executeTemplate(iteratorTmpl, filepath.Join(clientDir, "iterator.go"), goTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute iterator template: %w", err)
	}

	listOptsTmpl, err := loadTemplate(filepath.Join(tmplDir, "list_options.go.tmpl"))
	if err != nil {
		return fmt.Errorf("load list_options template: %w", err)
	}
	if err := executeTemplate(listOptsTmpl, filepath.Join(typesDir, "list_options.go"), goTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute list_options template: %w", err)
	}

	// Generate main HTTP client
	httpClientTmpl, err := loadTemplate(filepath.Join(tmplDir, "http_client.go.tmpl"))
	if err != nil {
		return fmt.Errorf("load http_client template: %w", err)
	}
	if err := executeTemplate(httpClientTmpl, filepath.Join(clientDir, "client.go"), goTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute http_client template: %w", err)
	}

	return nil
}

func generateTypeScript(spec *Spec, outDir string, header GeneratedHeader) error {
	srcDir := filepath.Join(outDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}

	tmplDir := filepath.Join(getTemplateDir(), "ts")

	baseTmpl, err := loadTemplate(filepath.Join(tmplDir, "base.ts.tmpl"))
	if err != nil {
		return fmt.Errorf("load base template: %w", err)
	}
	if err := executeTemplate(baseTmpl, filepath.Join(srcDir, "base.ts"), tsTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute base template: %w", err)
	}

	typesTmpl, err := loadTemplate(filepath.Join(tmplDir, "types.ts.tmpl"))
	if err != nil {
		return fmt.Errorf("load types template: %w", err)
	}

	clientTmpl, err := loadTemplate(filepath.Join(tmplDir, "client.ts.tmpl"))
	if err != nil {
		return fmt.Errorf("load client template: %w", err)
	}

	for _, r := range spec.Resources {
		data := tsTemplateData{Header: header, Resource: r, Spec: spec}
		fileName := toSnakeCase(r.Name) + ".ts"

		if err := executeTemplate(typesTmpl, filepath.Join(srcDir, fileName), data); err != nil {
			return fmt.Errorf("execute types template for %s: %w", r.Name, err)
		}

		apiFileName := toSnakeCase(r.Name) + "_api.ts"
		if err := executeTemplate(clientTmpl, filepath.Join(srcDir, apiFileName), data); err != nil {
			return fmt.Errorf("execute client template for %s: %w", r.Name, err)
		}
	}

	ambientClientTmpl, err := loadTemplate(filepath.Join(tmplDir, "ambient_client.ts.tmpl"))
	if err != nil {
		return fmt.Errorf("load ambient_client template: %w", err)
	}
	if err := executeTemplate(ambientClientTmpl, filepath.Join(srcDir, "client.ts"), tsTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute ambient_client template: %w", err)
	}

	indexTmpl, err := loadTemplate(filepath.Join(tmplDir, "index.ts.tmpl"))
	if err != nil {
		return fmt.Errorf("load index template: %w", err)
	}
	if err := executeTemplate(indexTmpl, filepath.Join(srcDir, "index.ts"), tsTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute index template: %w", err)
	}

	return nil
}

func generatePython(spec *Spec, outDir string, header GeneratedHeader) error {
	pkgDir := outDir
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}

	tmplDir := filepath.Join(getTemplateDir(), "python")

	baseTmpl, err := loadTemplate(filepath.Join(tmplDir, "base.py.tmpl"))
	if err != nil {
		return fmt.Errorf("load base template: %w", err)
	}
	if err := executeTemplate(baseTmpl, filepath.Join(pkgDir, "_base.py"), pythonTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute base template: %w", err)
	}

	typesTmpl, err := loadTemplate(filepath.Join(tmplDir, "types.py.tmpl"))
	if err != nil {
		return fmt.Errorf("load types template: %w", err)
	}

	clientTmpl, err := loadTemplate(filepath.Join(tmplDir, "client.py.tmpl"))
	if err != nil {
		return fmt.Errorf("load client template: %w", err)
	}

	for _, r := range spec.Resources {
		data := pythonTemplateData{Header: header, Resource: r, Spec: spec}
		fileName := toSnakeCase(r.Name) + ".py"

		if err := executeTemplate(typesTmpl, filepath.Join(pkgDir, fileName), data); err != nil {
			return fmt.Errorf("execute types template for %s: %w", r.Name, err)
		}

		apiFileName := "_" + toSnakeCase(r.Name) + "_api.py"
		if err := executeTemplate(clientTmpl, filepath.Join(pkgDir, apiFileName), data); err != nil {
			return fmt.Errorf("execute client template for %s: %w", r.Name, err)
		}
	}

	iteratorTmpl, err := loadTemplate(filepath.Join(tmplDir, "iterator.py.tmpl"))
	if err != nil {
		return fmt.Errorf("load iterator template: %w", err)
	}
	if err := executeTemplate(iteratorTmpl, filepath.Join(pkgDir, "_iterator.py"), pythonTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute iterator template: %w", err)
	}

	// Generate main HTTP client
	httpClientTmpl, err := loadTemplate(filepath.Join(tmplDir, "http_client.py.tmpl"))
	if err != nil {
		return fmt.Errorf("load http_client template: %w", err)
	}
	if err := executeTemplate(httpClientTmpl, filepath.Join(pkgDir, "client.py"), pythonTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute http_client template: %w", err)
	}

	// Generate __init__.py
	initTmpl, err := loadTemplate(filepath.Join(tmplDir, "__init__.py.tmpl"))
	if err != nil {
		return fmt.Errorf("load __init__.py template: %w", err)
	}
	if err := executeTemplate(initTmpl, filepath.Join(pkgDir, "__init__.py"), pythonTemplateData{Header: header, Spec: spec}); err != nil {
		return fmt.Errorf("execute __init__.py template: %w", err)
	}

	return nil
}

func loadTemplate(path string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"snakeCase":     toSnakeCase,
		"lower":         strings.ToLower,
		"title":         strings.Title,
		"goName":        toGoName,
		"pythonDefault": func(f Field) string { return pythonDefault(f.Type, f.Format) },
		"isDateTime":    isDateTimeField,
		"isWritable":    func(f Field) bool { return !f.ReadOnly },
		"camelCase":     toCamelCase,
		"pluralize":     pluralize,
		"lowerFirst":    lowerFirst,
		"tsDefault":     func(f Field) string { return tsDefault(f.Type, f.Format) },
		"hasTimeImport": func(fields []Field) bool {
			for _, f := range fields {
				if f.Format == "date-time" {
					return true
				}
			}
			return false
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", path, err)
	}

	return tmpl, nil
}

func executeTemplate(tmpl *template.Template, outPath string, data interface{}) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

func computeSpecHash(specPath string) (string, error) {
	specDir := filepath.Dir(specPath)
	h := sha256.New()

	files := []string{
		specPath,
		filepath.Join(specDir, "openapi.sessions.yaml"),
		filepath.Join(specDir, "openapi.projects.yaml"),
		filepath.Join(specDir, "openapi.projectSettings.yaml"),
		filepath.Join(specDir, "openapi.users.yaml"),
	}

	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return "", fmt.Errorf("open %s: %w", f, err)
		}
		if _, err := io.Copy(h, fh); err != nil {
			fh.Close()
			return "", fmt.Errorf("read %s: %w", f, err)
		}
		fh.Close()
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func getTemplateDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "templates"
	}
	dir := filepath.Dir(exe)
	tmplDir := filepath.Join(dir, "templates")
	if _, err := os.Stat(tmplDir); err == nil {
		return tmplDir
	}
	return "templates"
}

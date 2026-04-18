package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func discoverSubSpecs(specDir string) (map[string]string, map[string]string, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, nil, fmt.Errorf("read spec dir: %w", err)
	}

	resourceFiles := map[string]string{}
	pathSegments := map[string]string{}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "openapi.yaml" {
			continue
		}
		if !strings.HasPrefix(name, "openapi.") || !strings.HasSuffix(name, ".yaml") {
			continue
		}

		subPath := filepath.Join(specDir, name)
		data, err := os.ReadFile(subPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", name, err)
		}

		var doc subSpecDoc
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", name, err)
		}

		resourceName := inferResourceName(doc.Components.Schemas)
		if resourceName == "" {
			continue
		}

		pathSeg := inferPathSegment(doc.Paths, resourceName)
		if pathSeg == "" {
			continue
		}

		resourceFiles[resourceName] = name
		pathSegments[resourceName] = pathSeg
	}

	return resourceFiles, pathSegments, nil
}

func inferResourceName(schemas map[string]interface{}) string {
	var candidates []string
	for name := range schemas {
		if strings.HasSuffix(name, "List") ||
			strings.HasSuffix(name, "PatchRequest") ||
			strings.HasSuffix(name, "StatusPatchRequest") {
			continue
		}
		candidates = append(candidates, name)
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	sort.Strings(candidates)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func inferPathSegment(paths map[string]interface{}, resourceName string) string {
	basePath := extractBasePath(paths)
	expectedLeaf := strings.ToLower(resourcePlural(resourceName))

	var collectionMatch string
	var shortest string
	for path := range paths {
		if !strings.HasPrefix(path, basePath+"/") {
			continue
		}
		rest := strings.TrimPrefix(path, basePath+"/")
		rest = strings.TrimSuffix(rest, "/")

		parts := strings.Split(rest, "/")
		leaf := parts[len(parts)-1]
		if leaf == expectedLeaf {
			if collectionMatch == "" || len(rest) < len(collectionMatch) {
				collectionMatch = rest
			}
		}

		if shortest == "" || len(rest) < len(shortest) {
			shortest = rest
		}
	}

	if collectionMatch != "" {
		return collectionMatch
	}
	return shortest
}

func inferParentPath(pathSegment string) string {
	parts := strings.Split(pathSegment, "/")
	if len(parts) <= 1 {
		return ""
	}
	lastSeg := parts[len(parts)-1]
	if strings.Contains(lastSeg, "{") {
		if len(parts) <= 2 {
			return ""
		}
		return strings.Join(parts[:len(parts)-2], "/")
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

func leafSegment(pathSegment string) string {
	parts := strings.Split(pathSegment, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.Contains(parts[i], "{") {
			return parts[i]
		}
	}
	return pathSegment
}

type openAPIDoc struct {
	Paths      map[string]interface{} `yaml:"paths"`
	Components struct {
		Schemas map[string]interface{} `yaml:"schemas"`
	} `yaml:"components"`
}

func extractBasePath(paths map[string]interface{}) string {
	for path := range paths {
		parts := strings.Split(path, "/")
		// Expect paths like /api/ambient/v1/resource or /api/ambient-api-server/v1/resource
		// Find the common prefix by looking for the version segment (v\d+)
		for i, part := range parts {
			if len(part) > 1 && part[0] == 'v' {
				allDigits := true
				for _, c := range part[1:] {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					return strings.Join(parts[:i+1], "/")
				}
			}
		}
	}
	return "/api/ambient/v1"
}

type subSpecDoc struct {
	Paths      map[string]interface{} `yaml:"paths"`
	Components struct {
		Schemas map[string]interface{} `yaml:"schemas"`
	} `yaml:"components"`
}

func parseSpec(specPath string) (*Spec, error) {
	mainData, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	var mainDoc openAPIDoc
	if err := yaml.Unmarshal(mainData, &mainDoc); err != nil {
		return nil, fmt.Errorf("parse main spec: %w", err)
	}

	specDir := filepath.Dir(specPath)

	resourceFiles, pathSegments, err := discoverSubSpecs(specDir)
	if err != nil {
		return nil, fmt.Errorf("discover sub-specs: %w", err)
	}

	var resources []Resource
	for name, file := range resourceFiles {
		subPath := filepath.Join(specDir, file)
		subData, err := os.ReadFile(subPath)
		if err != nil {
			return nil, fmt.Errorf("read sub-spec %s: %w", file, err)
		}

		var subDoc subSpecDoc
		if err := yaml.Unmarshal(subData, &subDoc); err != nil {
			return nil, fmt.Errorf("parse sub-spec %s: %w", file, err)
		}

		resource, err := extractResource(name, pathSegments[name], &subDoc)
		if err != nil {
			return nil, fmt.Errorf("extract resource %s: %w", name, err)
		}

		resources = append(resources, *resource)
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	basePath := extractBasePath(mainDoc.Paths)

	return &Spec{BasePath: basePath, Resources: resources}, nil
}

func extractResource(name, pathSegment string, doc *subSpecDoc) (*Resource, error) {
	schema, ok := doc.Components.Schemas[name]
	if !ok {
		return nil, fmt.Errorf("schema %s not found", name)
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema %s is not a map", name)
	}

	fields, requiredFields, err := extractFields(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("extract fields for %s: %w", name, err)
	}

	patchName := name + "PatchRequest"
	patchSchema, ok := doc.Components.Schemas[patchName]
	var patchFields []Field
	if ok {
		patchMap, ok := patchSchema.(map[string]interface{})
		if ok {
			patchFields, _, err = extractPatchFields(patchMap)
			if err != nil {
				return nil, fmt.Errorf("extract patch fields for %s: %w", name, err)
			}
		}
	}

	statusPatchName := name + "StatusPatchRequest"
	statusPatchSchema, ok := doc.Components.Schemas[statusPatchName]
	var statusPatchFields []Field
	hasStatusPatch := false
	if ok {
		statusPatchMap, ok := statusPatchSchema.(map[string]interface{})
		if ok {
			statusPatchFields, _, err = extractPatchFields(statusPatchMap)
			if err != nil {
				return nil, fmt.Errorf("extract status patch fields for %s: %w", name, err)
			}
			hasStatusPatch = len(statusPatchFields) > 0
		}
	}

	hasDelete := checkHasDelete(doc.Paths, pathSegment)
	hasPatch := checkHasPatch(doc.Paths, pathSegment)
	actions := detectActions(doc.Paths, pathSegment)
	parentPath := inferParentPath(pathSegment)
	isSubResource := parentPath != ""

	return &Resource{
		Name:              name,
		Plural:            resourcePlural(name),
		PathSegment:       pathSegment,
		ParentPath:        parentPath,
		Fields:            fields,
		RequiredFields:    requiredFields,
		PatchFields:       patchFields,
		StatusPatchFields: statusPatchFields,
		HasDelete:         hasDelete,
		HasPatch:          hasPatch,
		HasStatusPatch:    hasStatusPatch,
		Actions:           actions,
		IsSubResource:     isSubResource,
	}, nil
}

func resourcePlural(name string) string {
	alreadyPlural := []string{"Settings", "Data", "Info", "Metadata"}
	for _, suffix := range alreadyPlural {
		if strings.HasSuffix(name, suffix) {
			return name
		}
	}
	return name + "s"
}

func extractFields(schemaMap map[string]interface{}) ([]Field, []string, error) {
	allOf, ok := schemaMap["allOf"]
	if !ok {
		return nil, nil, fmt.Errorf("schema missing allOf")
	}

	allOfList, ok := allOf.([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("allOf is not a list")
	}

	var fields []Field
	var requiredFields []string

	for _, item := range allOfList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if _, hasRef := itemMap["$ref"]; hasRef {
			continue
		}

		if req, ok := itemMap["required"]; ok {
			if reqList, ok := req.([]interface{}); ok {
				for _, r := range reqList {
					if s, ok := r.(string); ok {
						requiredFields = append(requiredFields, s)
					}
				}
			}
		}

		props, ok := itemMap["properties"]
		if !ok {
			continue
		}

		propsMap, ok := props.(map[string]interface{})
		if !ok {
			continue
		}

		for propName, propVal := range propsMap {
			if isObjectReferenceField(propName) {
				continue
			}

			propMap, ok := propVal.(map[string]interface{})
			if !ok {
				continue
			}

			propType, _ := propMap["type"].(string)
			propFormat, _ := propMap["format"].(string)
			readOnly, _ := propMap["readOnly"].(bool)

			isRequired := false
			for _, r := range requiredFields {
				if r == propName {
					isRequired = true
					break
				}
			}

			f := Field{
				Name:       propName,
				GoName:     toGoName(propName),
				PythonName: propName,
				TSName:     toCamelCase(propName),
				Type:       propType,
				Format:     propFormat,
				GoType:     toGoType(propType, propFormat),
				PythonType: toPythonType(propType, propFormat),
				TSType:     toTSType(propType, propFormat),
				Required:   isRequired,
				ReadOnly:   readOnly,
				JSONTag:    jsonTag(propName, isRequired),
			}

			fields = append(fields, f)
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields, requiredFields, nil
}

func extractPatchFields(schemaMap map[string]interface{}) ([]Field, []string, error) {
	props, ok := schemaMap["properties"]
	if !ok {
		return nil, nil, nil
	}

	propsMap, ok := props.(map[string]interface{})
	if !ok {
		return nil, nil, nil
	}

	var fields []Field
	for propName, propVal := range propsMap {
		propMap, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}

		propType, _ := propMap["type"].(string)
		propFormat, _ := propMap["format"].(string)

		f := Field{
			Name:       propName,
			GoName:     toGoName(propName),
			PythonName: propName,
			TSName:     toCamelCase(propName),
			Type:       propType,
			Format:     propFormat,
			GoType:     toGoType(propType, propFormat),
			PythonType: toPythonType(propType, propFormat),
			TSType:     toTSType(propType, propFormat),
			Required:   false,
			JSONTag:    jsonTag(propName, false),
		}

		fields = append(fields, f)
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	return fields, nil, nil
}

func findItemPath(paths map[string]interface{}, basePath, collectionPath string) (map[string]interface{}, bool) {
	fullCollection := basePath + "/" + collectionPath
	for path, val := range paths {
		if !strings.HasPrefix(path, fullCollection+"/") {
			continue
		}
		suffix := strings.TrimPrefix(path, fullCollection+"/")
		if strings.HasPrefix(suffix, "{") && !strings.Contains(suffix, "/") {
			if pathMap, ok := val.(map[string]interface{}); ok {
				return pathMap, true
			}
		}
	}
	return nil, false
}

func checkHasPatch(paths map[string]interface{}, pathSegment string) bool {
	basePath := extractBasePath(paths)
	pathMap, ok := findItemPath(paths, basePath, pathSegment)
	if !ok {
		return false
	}
	_, hasPatch := pathMap["patch"]
	return hasPatch
}

func checkHasDelete(paths map[string]interface{}, pathSegment string) bool {
	basePath := extractBasePath(paths)
	pathMap, ok := findItemPath(paths, basePath, pathSegment)
	if !ok {
		return false
	}
	_, hasDelete := pathMap["delete"]
	return hasDelete
}

func detectActions(paths map[string]interface{}, pathSegment string) []string {
	basePath := extractBasePath(paths)
	fullCollection := basePath + "/" + pathSegment
	knownActions := []string{"start", "stop"}
	var found []string
	for _, action := range knownActions {
		for path, val := range paths {
			if !strings.HasPrefix(path, fullCollection+"/") {
				continue
			}
			if !strings.HasSuffix(path, "/"+action) {
				continue
			}
			pathMap, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			if _, hasPost := pathMap["post"]; hasPost {
				found = append(found, action)
				break
			}
		}
	}
	return found
}

var objectReferenceFields = map[string]bool{
	"id":         true,
	"kind":       true,
	"href":       true,
	"created_at": true,
	"updated_at": true,
}

func isObjectReferenceField(name string) bool {
	return objectReferenceFields[name]
}

func isDateTimeField(f Field) bool {
	return f.Format == "date-time" && strings.Contains(f.GoType, "time.Time")
}

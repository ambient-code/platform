package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type openAPIDoc struct {
	Paths      map[string]interface{} `yaml:"paths"`
	Components struct {
		Schemas map[string]interface{} `yaml:"schemas"`
	} `yaml:"components"`
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

	resourceFiles := map[string]string{
		"Session":        "openapi.sessions.yaml",
		"User":           "openapi.users.yaml",
		"Project":        "openapi.projects.yaml",
		"ProjectSettings": "openapi.projectSettings.yaml",
	}

	pathSegments := map[string]string{
		"Session":        "sessions",
		"User":           "users",
		"Project":        "projects",
		"ProjectSettings": "project_settings",
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

	return &Spec{Resources: resources}, nil
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

	return &Resource{
		Name:              name,
		Plural:            resourcePlural(name),
		PathSegment:       pathSegment,
		Fields:            fields,
		RequiredFields:    requiredFields,
		PatchFields:       patchFields,
		StatusPatchFields: statusPatchFields,
		HasDelete:         hasDelete,
		HasPatch:          hasPatch,
		HasStatusPatch:    hasStatusPatch,
		Actions:           actions,
	}, nil
}

func resourcePlural(name string) string {
	switch name {
	case "Session":
		return "Sessions"
	case "Agent":
		return "Agents"
	case "Task":
		return "Tasks"
	case "Skill":
		return "Skills"
	case "Workflow":
		return "Workflows"
	case "User":
		return "Users"
	case "WorkflowSkill":
		return "WorkflowSkills"
	case "WorkflowTask":
		return "WorkflowTasks"
	case "Project":
		return "Projects"
	case "ProjectSettings":
		return "ProjectSettings"
	case "Permission":
		return "Permissions"
	case "RepositoryRef":
		return "RepositoryRefs"
	case "ProjectKey":
		return "ProjectKeys"
	default:
		return name + "s"
	}
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

func checkHasPatch(paths map[string]interface{}, pathSegment string) bool {
	idPath := fmt.Sprintf("/api/ambient/v1/%s/{id}", pathSegment)
	pathVal, ok := paths[idPath]
	if !ok {
		return false
	}

	pathMap, ok := pathVal.(map[string]interface{})
	if !ok {
		return false
	}

	_, hasPatch := pathMap["patch"]
	return hasPatch
}

func checkHasDelete(paths map[string]interface{}, pathSegment string) bool {
	idPath := fmt.Sprintf("/api/ambient/v1/%s/{id}", pathSegment)
	pathVal, ok := paths[idPath]
	if !ok {
		return false
	}

	pathMap, ok := pathVal.(map[string]interface{})
	if !ok {
		return false
	}

	_, hasDelete := pathMap["delete"]
	return hasDelete
}

func detectActions(paths map[string]interface{}, pathSegment string) []string {
	knownActions := []string{"start", "stop"}
	var found []string
	for _, action := range knownActions {
		actionPath := fmt.Sprintf("/api/ambient/v1/%s/{id}/%s", pathSegment, action)
		pathVal, ok := paths[actionPath]
		if !ok {
			continue
		}
		pathMap, ok := pathVal.(map[string]interface{})
		if !ok {
			continue
		}
		if _, hasPost := pathMap["post"]; hasPost {
			found = append(found, action)
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

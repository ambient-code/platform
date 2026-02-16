package projects_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/projects"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func stringPtr(s string) *string { return &s }

var projectCounter int

func newProject(suffix string) (*projects.Project, error) {
	projectCounter++
	projectService := projects.Service(&environments.Environment().Services)

	project := &projects.Project{
		Name:        fmt.Sprintf("proj-%s-%d", suffix, projectCounter),
		DisplayName: stringPtr("Test Project"),
		Description: stringPtr("test-description"),
		Status:      stringPtr("active"),
	}

	result, svcErr := projectService.Create(context.Background(), project)
	if svcErr != nil {
		return nil, fmt.Errorf("projects.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newProjectList(namePrefix string, count int) ([]*projects.Project, error) {
	var items []*projects.Project
	for i := 1; i <= count; i++ {
		c, err := newProject(namePrefix)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}

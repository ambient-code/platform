package projectSettings_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/projectSettings"
	"github.com/ambient/platform/components/ambient-api-server/plugins/projects"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func stringPtr(s string) *string { return &s }

var projectCounter int

func newParentProject() (*projects.Project, error) {
	projectCounter++
	projectService := projects.Service(&environments.Environment().Services)
	result, svcErr := projectService.Create(context.Background(), &projects.Project{
		Name:        fmt.Sprintf("test-project-%d", projectCounter),
		DisplayName: stringPtr("Test Project"),
		Description: stringPtr("test-description"),
		Status:      stringPtr("active"),
	})
	if svcErr != nil {
		return nil, fmt.Errorf("projects.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newProjectSettings(id string) (*projectSettings.ProjectSettings, error) {
	psService := projectSettings.Service(&environments.Environment().Services)

	project, err := newParentProject()
	if err != nil {
		return nil, err
	}

	result, svcErr := psService.Create(context.Background(), &projectSettings.ProjectSettings{
		ProjectId:     project.ID,
		GroupAccess:   stringPtr(`[{"group_name":"admins","role":"admin"}]`),
		RunnerSecrets: stringPtr(`{"api_key":"redacted"}`),
		Repositories:  stringPtr(`[{"url":"https://github.com/test/repo","branch":"main"}]`),
	})
	if svcErr != nil {
		return nil, fmt.Errorf("projectSettings.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newProjectSettingsList(namePrefix string, count int) ([]*projectSettings.ProjectSettings, error) {
	var items []*projectSettings.ProjectSettings
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newProjectSettings(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}

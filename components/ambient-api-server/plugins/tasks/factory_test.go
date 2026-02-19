package tasks_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/tasks"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newTask(id string) (*tasks.Task, error) {
	taskService := tasks.Service(&environments.Environment().Services)

	task := &tasks.Task{
		Name:    "test-name",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
	}

	sub, err := taskService.Create(context.Background(), task)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newTaskList(namePrefix string, count int) ([]*tasks.Task, error) {
	var items []*tasks.Task
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newTask(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }

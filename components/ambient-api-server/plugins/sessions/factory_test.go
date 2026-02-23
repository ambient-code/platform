package sessions_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/sessions"
	"github.com/ambient/platform/components/ambient-api-server/plugins/users"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newUser(name string) (*users.User, error) {
	userService := users.Service(&environments.Environment().Services)
	user := &users.User{
		Username: name,
		Name:     name,
	}
	result, svcErr := userService.Create(context.Background(), user)
	if svcErr != nil {
		return nil, fmt.Errorf("users.Create: %s", svcErr.Error())
	}
	return result, nil
}

var sessionCounter int

func newSession(id string) (*sessions.Session, error) {
	sessionService := sessions.Service(&environments.Environment().Services)

	sessionCounter++
	creator, err := newUser(fmt.Sprintf("test-creator-%d", sessionCounter))
	if err != nil {
		return nil, fmt.Errorf("newUser(creator): %w", err)
	}
	assignee, err := newUser(fmt.Sprintf("test-assignee-%d", sessionCounter))
	if err != nil {
		return nil, fmt.Errorf("newUser(assignee): %w", err)
	}

	session := &sessions.Session{
		Name:            "test-name",
		RepoUrl:         stringPtr("test-repo_url"),
		Prompt:          stringPtr("test-prompt"),
		CreatedByUserId: stringPtr(creator.ID),
		AssignedUserId:  stringPtr(assignee.ID),
	}

	sub, svcErr := sessionService.Create(context.Background(), session)
	if svcErr != nil {
		return nil, fmt.Errorf("sessionService.Create: %s", svcErr.Error())
	}

	return sub, nil
}

func newSessionInProject(id string, projectId string) (*sessions.Session, error) {
	sessionService := sessions.Service(&environments.Environment().Services)

	session := &sessions.Session{
		Name:      "test-name",
		Prompt:    stringPtr("test-prompt"),
		ProjectId: stringPtr(projectId),
	}

	sub, svcErr := sessionService.Create(context.Background(), session)
	if svcErr != nil {
		return nil, fmt.Errorf("sessionService.Create: %s", svcErr.Error())
	}

	return sub, nil
}

func newSessionList(namePrefix string, count int) ([]*sessions.Session, error) {
	var items []*sessions.Session
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newSession(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string    { return &s }
func boolPtr(b bool) *bool          { return &b }
func int32Ptr(i int32) *int32       { return &i }
func float64Ptr(f float64) *float64 { return &f }

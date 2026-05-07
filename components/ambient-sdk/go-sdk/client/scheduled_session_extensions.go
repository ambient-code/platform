package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *ScheduledSessionAPI) projectPath(projectID string) string {
	return "/projects/" + url.PathEscape(projectID) + "/scheduled-sessions"
}

func (a *ScheduledSessionAPI) ListByProject(ctx context.Context, projectID string, opts *types.ListOptions) (*types.ScheduledSessionList, error) {
	var result types.ScheduledSessionList
	if err := a.client.doWithQuery(ctx, http.MethodGet, a.projectPath(projectID), nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) GetByProject(ctx context.Context, projectID, id string) (*types.ScheduledSession, error) {
	var result types.ScheduledSession
	path := a.projectPath(projectID) + "/" + url.PathEscape(id)
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) CreateInProject(ctx context.Context, projectID string, resource *types.ScheduledSession) (*types.ScheduledSession, error) {
	body, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshal scheduled session: %w", err)
	}
	var result types.ScheduledSession
	if err := a.client.do(ctx, http.MethodPost, a.projectPath(projectID), body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) UpdateInProject(ctx context.Context, projectID, id string, patch map[string]any) (*types.ScheduledSession, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshal patch: %w", err)
	}
	var result types.ScheduledSession
	path := a.projectPath(projectID) + "/" + url.PathEscape(id)
	if err := a.client.do(ctx, http.MethodPatch, path, body, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) DeleteInProject(ctx context.Context, projectID, id string) error {
	return a.client.do(ctx, http.MethodDelete, a.projectPath(projectID)+"/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (a *ScheduledSessionAPI) Suspend(ctx context.Context, projectID, id string) (*types.ScheduledSession, error) {
	var result types.ScheduledSession
	path := a.projectPath(projectID) + "/" + url.PathEscape(id) + "/suspend"
	if err := a.client.do(ctx, http.MethodPost, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) Resume(ctx context.Context, projectID, id string) (*types.ScheduledSession, error) {
	var result types.ScheduledSession
	path := a.projectPath(projectID) + "/" + url.PathEscape(id) + "/resume"
	if err := a.client.do(ctx, http.MethodPost, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) Trigger(ctx context.Context, projectID, id string) error {
	path := a.projectPath(projectID) + "/" + url.PathEscape(id) + "/trigger"
	return a.client.do(ctx, http.MethodPost, path, nil, http.StatusOK, nil)
}

func (a *ScheduledSessionAPI) Runs(ctx context.Context, projectID, id string, opts *types.ListOptions) (*types.SessionList, error) {
	var result types.SessionList
	path := a.projectPath(projectID) + "/" + url.PathEscape(id) + "/runs"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ScheduledSessionAPI) GetByName(ctx context.Context, projectID, name string) (*types.ScheduledSession, error) {
	list, err := a.ListByProject(ctx, projectID, &types.ListOptions{Search: "name = '" + name + "'"})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Name == name {
			return &list.Items[i], nil
		}
	}
	return nil, fmt.Errorf("scheduled session %q not found in project %q", name, projectID)
}

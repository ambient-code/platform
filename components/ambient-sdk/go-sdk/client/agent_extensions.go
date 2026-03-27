package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *ProjectAgentAPI) ListByProject(ctx context.Context, projectID string, opts *types.ListOptions) (*types.ProjectAgentList, error) {
	var result types.ProjectAgentList
	path := "/projects/" + url.PathEscape(projectID) + "/agents"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) GetByProject(ctx context.Context, projectID, agentID string) (*types.ProjectAgent, error) {
	var result types.ProjectAgent
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) CreateInProject(ctx context.Context, projectID string, resource *types.ProjectAgent) (*types.ProjectAgent, error) {
	body, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshal project_agent: %w", err)
	}
	var result types.ProjectAgent
	path := "/projects/" + url.PathEscape(projectID) + "/agents"
	if err := a.client.do(ctx, http.MethodPost, path, body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) UpdateInProject(ctx context.Context, projectID, agentID string, patch map[string]any) (*types.ProjectAgent, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshal patch: %w", err)
	}
	var result types.ProjectAgent
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
	if err := a.client.do(ctx, http.MethodPatch, path, body, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) DeleteInProject(ctx context.Context, projectID, agentID string) error {
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
	return a.client.do(ctx, http.MethodDelete, path, nil, http.StatusNoContent, nil)
}

func (a *ProjectAgentAPI) Ignite(ctx context.Context, projectID, agentID, prompt string) (*types.IgniteResponse, error) {
	req := types.IgniteRequest{Prompt: prompt}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal ignite request: %w", err)
	}
	var result types.IgniteResponse
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/ignite"
	if err := a.client.doMultiStatus(ctx, http.MethodPost, path, body, &result, http.StatusOK, http.StatusCreated); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) GetIgnition(ctx context.Context, projectID, agentID string) (*types.IgniteResponse, error) {
	var result types.IgniteResponse
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/ignition"
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) Sessions(ctx context.Context, projectID, agentID string, opts *types.ListOptions) (*types.SessionList, error) {
	var result types.SessionList
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/sessions"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *ProjectAgentAPI) GetInProject(ctx context.Context, projectID, agentName string) (*types.ProjectAgent, error) {
	list, err := a.ListByProject(ctx, projectID, &types.ListOptions{Search: "name = '" + agentName + "'"})
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Name == agentName {
			return &list.Items[i], nil
		}
	}
	return nil, fmt.Errorf("agent %q not found in project %q", agentName, projectID)
}

func (a *ProjectAgentAPI) ListInboxInProject(ctx context.Context, projectID, agentID string) ([]types.InboxMessage, error) {
	var result types.InboxMessageList
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox"
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (a *ProjectAgentAPI) SendInboxInProject(ctx context.Context, projectID, agentID, fromName, body string) error {
	msg := types.InboxMessage{FromName: fromName, Body: body}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal inbox message: %w", err)
	}
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox"
	return a.client.do(ctx, http.MethodPost, path, payload, http.StatusCreated, nil)
}

func (a *InboxMessageAPI) Send(ctx context.Context, projectID, agentID string, msg *types.InboxMessage) (*types.InboxMessage, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal inbox message: %w", err)
	}
	var result types.InboxMessage
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox"
	if err := a.client.do(ctx, http.MethodPost, path, body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *InboxMessageAPI) ListByAgent(ctx context.Context, projectID, agentID string, opts *types.ListOptions) (*types.InboxMessageList, error) {
	var result types.InboxMessageList
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *InboxMessageAPI) MarkRead(ctx context.Context, projectID, agentID, msgID string) error {
	patch := map[string]any{"read": true}
	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch: %w", err)
	}
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox/" + url.PathEscape(msgID)
	return a.client.do(ctx, http.MethodPatch, path, body, http.StatusOK, nil)
}

func (a *InboxMessageAPI) DeleteMessage(ctx context.Context, projectID, agentID, msgID string) error {
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox/" + url.PathEscape(msgID)
	return a.client.do(ctx, http.MethodDelete, path, nil, http.StatusNoContent, nil)
}

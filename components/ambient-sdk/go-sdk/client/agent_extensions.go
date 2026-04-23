package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *AgentAPI) ListByProject(ctx context.Context, projectID string, opts *types.ListOptions) (*types.AgentList, error) {
	var result types.AgentList
	path := "/projects/" + url.PathEscape(projectID) + "/agents"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) GetByProject(ctx context.Context, projectID, agentID string) (*types.Agent, error) {
	var result types.Agent
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) CreateInProject(ctx context.Context, projectID string, resource *types.Agent) (*types.Agent, error) {
	body, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshal agent: %w", err)
	}
	var result types.Agent
	path := "/projects/" + url.PathEscape(projectID) + "/agents"
	if err := a.client.do(ctx, http.MethodPost, path, body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) UpdateInProject(ctx context.Context, projectID, agentID string, patch map[string]any) (*types.Agent, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshal patch: %w", err)
	}
	var result types.Agent
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
	if err := a.client.do(ctx, http.MethodPatch, path, body, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) DeleteInProject(ctx context.Context, projectID, agentID string) error {
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
	return a.client.do(ctx, http.MethodDelete, path, nil, http.StatusNoContent, nil)
}

func (a *AgentAPI) Start(ctx context.Context, projectID, agentID, prompt string) (*types.StartResponse, error) {
	req := types.StartRequest{Prompt: prompt}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal start request: %w", err)
	}
	var result types.StartResponse
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/start"
	if err := a.client.doMultiStatus(ctx, http.MethodPost, path, body, &result, http.StatusOK, http.StatusCreated); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) GetStartPreview(ctx context.Context, projectID, agentID string) (*types.StartResponse, error) {
	var result types.StartResponse
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/start"
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) ListRoleBindingsByAgent(ctx context.Context, projectID, agentID string, opts *types.ListOptions) (*types.RoleBindingList, error) {
	var result types.RoleBindingList
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/role_bindings"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) Sessions(ctx context.Context, projectID, agentID string, opts *types.ListOptions) (*types.SessionList, error) {
	var result types.SessionList
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/sessions"
	if err := a.client.doWithQuery(ctx, http.MethodGet, path, nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *AgentAPI) GetInProject(ctx context.Context, projectID, agentName string) (*types.Agent, error) {
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

func (a *AgentAPI) ListInboxInProject(ctx context.Context, projectID, agentID string) ([]types.InboxMessage, error) {
	var result types.InboxMessageList
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox"
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (a *AgentAPI) SendInboxInProject(ctx context.Context, projectID, agentID, fromName, body string) error {
	msg := types.InboxMessage{FromName: fromName, Body: body}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal inbox message: %w", err)
	}
	path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID) + "/inbox"
	return a.client.do(ctx, http.MethodPost, path, payload, http.StatusCreated, nil)
}

func (a *AgentAPI) PatchLabelsInProject(ctx context.Context, projectID, agentID string, labels map[string]string) (*types.Agent, error) {
	b, err := json.Marshal(labels)
	if err != nil {
		return nil, fmt.Errorf("marshal labels: %w", err)
	}
	return a.UpdateInProject(ctx, projectID, agentID, map[string]any{"labels": string(b)})
}

func (a *AgentAPI) PatchAnnotationsInProject(ctx context.Context, projectID, agentID string, annotations map[string]string) (*types.Agent, error) {
	b, err := json.Marshal(annotations)
	if err != nil {
		return nil, fmt.Errorf("marshal annotations: %w", err)
	}
	return a.UpdateInProject(ctx, projectID, agentID, map[string]any{"annotations": string(b)})
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

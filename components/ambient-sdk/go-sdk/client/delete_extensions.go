package client

import (
	"context"
	"net/http"
	"net/url"
)

func (a *AgentAPI) Delete(ctx context.Context, id string) error {
	return a.client.do(ctx, http.MethodDelete, "/agents/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (a *RoleAPI) Delete(ctx context.Context, id string) error {
	return a.client.do(ctx, http.MethodDelete, "/roles/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (a *RoleBindingAPI) Delete(ctx context.Context, id string) error {
	return a.client.do(ctx, http.MethodDelete, "/role_bindings/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (a *SessionCheckInAPI) Delete(ctx context.Context, id string) error {
	return a.client.do(ctx, http.MethodDelete, "/session_check_ins/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

func (a *ProjectDocumentAPI) Delete(ctx context.Context, id string) error {
	return a.client.do(ctx, http.MethodDelete, "/project_documents/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

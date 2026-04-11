package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

type CredentialAPI struct {
	client *Client
}

func (c *Client) Credentials() *CredentialAPI {
	return &CredentialAPI{client: c}
}

func credentialBasePath(projectID string) string {
	return "/projects/" + url.PathEscape(projectID) + "/credentials"
}

func credentialItemPath(projectID, credID string) string {
	return credentialBasePath(projectID) + "/" + url.PathEscape(credID)
}

func (a *CredentialAPI) Create(ctx context.Context, projectID string, resource *types.Credential) (*types.Credential, error) {
	body, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshal credential: %w", err)
	}
	var result types.Credential
	if err := a.client.do(ctx, http.MethodPost, credentialBasePath(projectID), body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CredentialAPI) Get(ctx context.Context, projectID, credID string) (*types.Credential, error) {
	var result types.Credential
	if err := a.client.do(ctx, http.MethodGet, credentialItemPath(projectID, credID), nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CredentialAPI) List(ctx context.Context, projectID string, opts *types.ListOptions) (*types.CredentialList, error) {
	var result types.CredentialList
	if err := a.client.doWithQuery(ctx, http.MethodGet, credentialBasePath(projectID), nil, http.StatusOK, &result, opts); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CredentialAPI) Update(ctx context.Context, projectID, credID string, patch map[string]any) (*types.Credential, error) {
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshal patch: %w", err)
	}
	var result types.Credential
	if err := a.client.do(ctx, http.MethodPatch, credentialItemPath(projectID, credID), body, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CredentialAPI) Delete(ctx context.Context, projectID, credID string) error {
	return a.client.do(ctx, http.MethodDelete, credentialItemPath(projectID, credID), nil, http.StatusNoContent, nil)
}

func (a *CredentialAPI) GetToken(ctx context.Context, projectID, credID string) (*types.CredentialTokenResponse, error) {
	var result types.CredentialTokenResponse
	if err := a.client.do(ctx, http.MethodGet, credentialItemPath(projectID, credID)+"/token", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CredentialAPI) ListAll(ctx context.Context, projectID string, opts *types.ListOptions) *Iterator[types.Credential] {
	return NewIterator(func(page int) (*types.CredentialList, error) {
		o := *opts
		o.Page = page
		return a.List(ctx, projectID, &o)
	})
}

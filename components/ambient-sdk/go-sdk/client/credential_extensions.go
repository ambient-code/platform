package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *CredentialAPI) GetToken(ctx context.Context, id string) (*types.CredentialTokenResponse, error) {
	var result types.CredentialTokenResponse
	if err := a.client.do(ctx, http.MethodGet, "/credentials/"+url.PathEscape(id)+"/token", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *CredentialAPI) FindByName(ctx context.Context, name string) (*types.Credential, error) {
	opts := types.NewListOptions().Size(100).Build()
	opts.Search = fmt.Sprintf("name = '%s'", name)
	list, err := a.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, c := range list.Items {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("credential with name %q not found", name)
}

// CreateCompat creates a credential with project_id for backward compatibility
// with server images that predate migration 202505120001 (drop project_id column).
// Remove once all environments run the updated server.
func (a *CredentialAPI) CreateCompat(ctx context.Context, resource *types.Credential) (*types.Credential, error) {
	payload := struct {
		*types.Credential
		ProjectID string `json:"project_id,omitempty"`
	}{
		Credential: resource,
		ProjectID:  a.client.project,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal credential: %w", err)
	}
	var result types.Credential
	if err := a.client.do(ctx, http.MethodPost, "/credentials", body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

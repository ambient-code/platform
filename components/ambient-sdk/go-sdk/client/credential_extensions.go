package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *CredentialAPI) GetToken(ctx context.Context, id string) (*types.CredentialTokenResponse, error) {
	var result types.CredentialTokenResponse
	if err := a.client.do(ctx, http.MethodGet, a.basePath()+"/"+url.PathEscape(id)+"/token", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

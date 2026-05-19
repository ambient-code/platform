// Package connection creates authenticated SDK clients from CLI configuration.
package connection

import (
	"fmt"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/info"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
)

var insecureSkipTLSVerify bool

// SetInsecureSkipTLSVerify overrides TLS verification for the current process.
func SetInsecureSkipTLSVerify(v bool) {
	insecureSkipTLSVerify = v
}

// ClientFactory holds credentials for creating per-project SDK clients.
// TokenFunc is called on every ForProject to get a fresh token, enabling
// automatic refresh of short-lived OIDC tokens.
type ClientFactory struct {
	APIURL    string
	TokenFunc func() (string, error)
	Insecure  bool
}

// ForProject creates an SDK client scoped to the given project name.
// The token is fetched fresh via TokenFunc on each call, so expired
// tokens are automatically refreshed.
func (f *ClientFactory) ForProject(project string) (*sdkclient.Client, error) {
	token, err := f.TokenFunc()
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}
	opts := []sdkclient.ClientOption{
		sdkclient.WithUserAgent("acpctl/" + info.Version),
	}
	if f.Insecure {
		opts = append(opts, sdkclient.WithInsecureSkipVerify())
	}
	return sdkclient.NewClient(f.APIURL, token, project, opts...)
}

// NewClientFromConfig creates an SDK client from the saved configuration.
func NewClientFromConfig() (*sdkclient.Client, error) {
	factory, err := NewClientFactory()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	project := cfg.GetProject()
	if project == "" {
		return nil, fmt.Errorf("no project set; run 'acpctl config set project <name>' or set AMBIENT_PROJECT")
	}

	return factory.ForProject(project)
}

// NewClientFactory loads config and returns a factory for creating per-project clients.
func NewClientFactory() (*ClientFactory, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Verify we have a token at startup.
	token, err := cfg.GetTokenWithRefresh()
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	if token == "" {
		return nil, fmt.Errorf("not logged in; run 'acpctl login' first")
	}

	apiURL := cfg.GetAPIUrl()
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API URL %q: must include scheme and host (e.g. https://api.example.com)", apiURL)
	}

	return &ClientFactory{
		APIURL: apiURL,
		TokenFunc: func() (string, error) {
			return cfg.GetTokenWithRefresh()
		},
		Insecure: cfg.InsecureTLSVerify || insecureSkipTLSVerify,
	}, nil
}

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
type ClientFactory struct {
	APIURL   string
	Token    string
	Insecure bool
}

// ForProject creates an SDK client scoped to the given project name.
func (f *ClientFactory) ForProject(project string) (*sdkclient.Client, error) {
	opts := []sdkclient.ClientOption{
		sdkclient.WithUserAgent("acpctl/" + info.Version),
	}
	if f.Insecure {
		opts = append(opts, sdkclient.WithInsecureSkipVerify())
	}
	return sdkclient.NewClient(f.APIURL, f.Token, project, opts...)
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
		APIURL:   apiURL,
		Token:    token,
		Insecure: cfg.InsecureTLSVerify || insecureSkipTLSVerify,
	}, nil
}

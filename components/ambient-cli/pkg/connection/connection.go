package connection

import (
	"fmt"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/info"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
)

// NewClientFromConfig creates an SDK client from the saved configuration.
// TODO: Add TLS skip-verify support once the SDK exposes WithHTTPClient option.
// OpenShift/on-prem deployments with self-signed certs will need this.
func NewClientFromConfig() (*sdkclient.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	token := cfg.GetToken()
	if token == "" {
		return nil, fmt.Errorf("not logged in; run 'acpctl login' first")
	}

	project := cfg.GetProject()
	if project == "" {
		return nil, fmt.Errorf("no project set; run 'acpctl config set project <name>' or set AMBIENT_PROJECT")
	}

	apiURL := cfg.GetAPIUrl()
	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid API URL %q: must include scheme and host (e.g. https://api.example.com)", apiURL)
	}

	return sdkclient.NewClient(
		apiURL,
		token,
		project,
		sdkclient.WithUserAgent("acpctl/"+info.Version),
	)
}

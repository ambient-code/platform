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

// NewClientFromConfig creates an SDK client from the saved configuration.
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

	opts := []sdkclient.ClientOption{
		sdkclient.WithUserAgent("acpctl/" + info.Version),
	}
	if cfg.InsecureTLSVerify || insecureSkipTLSVerify {
		opts = append(opts, sdkclient.WithInsecureSkipVerify())
	}

	return sdkclient.NewClient(apiURL, token, project, opts...)
}

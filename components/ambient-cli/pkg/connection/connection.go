package connection

import (
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/info"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
)

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

	return sdkclient.NewClient(
		apiURL,
		token,
		project,
		sdkclient.WithUserAgent("acpctl/"+info.Version),
	)
}

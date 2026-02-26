package get

import (
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long:  "Get a configuration value by key. Valid keys: api_url, project, insecure, pager.",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	key := cmdArgs[0]
	var value string

	switch key {
	case "api_url":
		value = cfg.GetAPIUrl()
	case "project":
		value = cfg.GetProject()
	case "insecure":
		value = fmt.Sprintf("%t", cfg.Insecure)
	case "pager":
		value = cfg.Pager
	case "access_token":
		if cfg.AccessToken != "" {
			value = "[REDACTED]"
		}
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: api_url, project, insecure, pager)", key)
	}

	if value != "" {
		fmt.Fprintln(cmd.OutOrStdout(), value)
	}
	return nil
}

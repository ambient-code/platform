// Package set implements the config set subcommand for writing configuration values.
package set

import (
	"fmt"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long:  "Set a configuration value by key. Valid keys: api_url, project, pager.",
	Args:  cobra.ExactArgs(2),
	RunE:  run,
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	key := cmdArgs[0]
	value := cmdArgs[1]

	switch key {
	case "api_url":
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid URL %q: must include scheme and host (e.g. https://api.example.com)", value)
		}
		cfg.APIUrl = value
	case "project":
		cfg.Project = value
	case "pager":
		cfg.Pager = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: api_url, project, pager)", key)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
	return nil
}

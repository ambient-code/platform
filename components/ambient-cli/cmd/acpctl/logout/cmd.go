// Package logout implements the logout subcommand for clearing saved credentials.
package logout

import (
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the Ambient API server",
	Long:  "Remove saved credentials from the configuration file.",
	Args:  cobra.NoArgs,
	RunE:  run,
}

func run(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg.ClearToken()

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Logged out successfully.")
	return nil
}

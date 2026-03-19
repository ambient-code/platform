// Package agent implements subcommands for interacting with agents.
package agent

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Interact with agents",
	Long: `Interact with agents.

Examples:
  acpctl agent <id>            # (coming soon)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

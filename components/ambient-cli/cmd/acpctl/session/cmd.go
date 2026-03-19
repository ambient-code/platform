// Package session implements subcommands for interacting with sessions.
package session

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "session",
	Short: "Interact with sessions",
	Long: `Interact with agentic sessions.

Examples:
  acpctl session messages <id>               # list messages (snapshot)
  acpctl session messages <id> -f            # stream messages live
  acpctl session send <id> "Hello!"          # send a message (any event_type)
  acpctl session send <id> "Hello!"          # send a message`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(messagesCmd)
	Cmd.AddCommand(sendCmd)
}

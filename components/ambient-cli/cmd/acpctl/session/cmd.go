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
  acpctl session messages <id> -f            # live SSE stream (ends at turn end)
  acpctl session messages <id> -F            # continuous follow (Ctrl+C to stop)
  acpctl session send <id> "Hello!"          # send a message
  acpctl session send <id> "Hello!" -f       # send and stream until done
  acpctl session events <id>                 # raw AG-UI event stream`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(messagesCmd)
	Cmd.AddCommand(sendCmd)
	Cmd.AddCommand(eventsCmd)
}

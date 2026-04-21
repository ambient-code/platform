// Package ambient implements the ambient TUI dashboard subcommand.
package ambient

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
)

var Cmd = &cobra.Command{
	Use:   "ambient",
	Short: "Strategic dashboard — live view of your entire Ambient platform",
	Long: `Launches an interactive terminal dashboard for the Ambient platform.

Navigate with ↑↓ (or j/k) to switch sections:
  Cluster Pods   system pods in the ambient-code namespace
  Namespaces     all cluster namespaces (fleet-* highlighted)
  Projects       all projects via SDK
  Sessions       all sessions with phase status
  Agents         all agents with current session
  Stats          summary counts and phase breakdown

Controls:
  ↑↓ / j/k       navigate sections
  Tab            focus command bar
  Esc            unfocus command bar
  r              force refresh
  PgUp/PgDn      scroll main panel
  q / Ctrl+C     quit

Command bar accepts any shell command (kubectl, oc, acpctl, etc.)
Output streams line-by-line into the main panel.

Data refreshes automatically every 10 seconds.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		factory, err := connection.NewClientFactory()
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}

		m := tui.NewModel(client, factory)
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			return err
		}
		return nil
	},
}

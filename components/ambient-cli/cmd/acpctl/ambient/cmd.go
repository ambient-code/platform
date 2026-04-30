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
	Short: "Interactive TUI — k9s-style resource browser for the Ambient platform",
	Long: `Launches an interactive terminal UI for the Ambient platform.

Navigation (k9s-style):
  :         command mode (tab-complete resource kinds)
  /         filter mode (regex, /! inverse, /-l label)
  Enter     drill into selected resource
  Esc       back / cancel
  d         describe selected resource
  q         quit (or back from child view)
  ?         help overlay

Data refreshes automatically every 5 seconds.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		factory, err := connection.NewClientFactory()
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}

		m, err := tui.NewAppModel(factory)
		if err != nil {
			return fmt.Errorf("init TUI: %w", err)
		}
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			return err
		}
		return nil
	},
}

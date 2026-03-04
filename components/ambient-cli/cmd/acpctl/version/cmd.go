// Package version implements the version subcommand displaying build metadata.
package version

import (
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/info"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "acpctl %s (commit: %s, built: %s)\n",
			info.Version, info.Commit, info.BuildDate)
	},
}

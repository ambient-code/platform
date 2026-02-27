// Package config implements the config get and config set subcommands.
package config

import (
	configget "github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/config/get"
	configset "github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/config/set"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  "Get and set configuration values for the Ambient CLI.",
}

func init() {
	Cmd.AddCommand(configget.Cmd)
	Cmd.AddCommand(configset.Cmd)
}

package config

import (
	configget "github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/config/get"
	configset "github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/config/set"
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

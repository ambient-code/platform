package main

import (
	"fmt"
	"os"

	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/completion"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/config"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/create"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/delete"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/describe"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/get"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/login"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/logout"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/start"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/stop"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/version"
	"github.com/ambient-code/platform/components/ambient-cli/cmd/ambient/whoami"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/info"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:           "ambient",
	Short:         "Ambient Code Platform CLI",
	Long:          "Command-line interface for the Ambient Code Platform API server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       info.Version,
}

func init() {
	root.AddCommand(login.Cmd)
	root.AddCommand(logout.Cmd)
	root.AddCommand(version.Cmd)
	root.AddCommand(whoami.Cmd)
	root.AddCommand(config.Cmd)
	root.AddCommand(get.Cmd)
	root.AddCommand(create.Cmd)
	root.AddCommand(delete.Cmd)
	root.AddCommand(describe.Cmd)
	root.AddCommand(start.Cmd)
	root.AddCommand(stop.Cmd)
	root.AddCommand(completion.Cmd)
}

func main() {
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

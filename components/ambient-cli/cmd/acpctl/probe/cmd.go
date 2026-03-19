package probe

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/internal/probe"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "probe",
	Short: "Run end-to-end connectivity probe (create project → agent → ignite → stream response → cleanup)",
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	s := &probe.State{
		Client:  client,
		BaseURL: cfg.GetAPIUrl(),
		Token:   cfg.GetToken(),
		Project: cfg.GetProject(),
		Log: func(format string, a ...interface{}) {
			fmt.Fprintf(cmd.OutOrStdout(), format+"\n", a...)
		},
	}

	return probe.Run(context.Background(), s)
}

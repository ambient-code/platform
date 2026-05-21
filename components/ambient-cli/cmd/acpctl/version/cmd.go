// Package version implements the acpctl version command.
package version

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/info"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the client and server version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "Client: %s (commit: %s, built: %s)\n",
			info.Version, info.Commit, info.BuildDate)

		cfg, err := config.Load()
		if err != nil {
			return
		}
		apiURL := cfg.GetAPIUrl()
		if apiURL == "" {
			return
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		sv, err := sdkclient.FetchServerVersion(ctx, apiURL, cfg.InsecureTLSVerify)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Server: unavailable (%v)\n", err)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Server: %s (tag: %s, built: %s)\n", sv.Version, sv.GitTag, sv.BuildTime)
	},
}

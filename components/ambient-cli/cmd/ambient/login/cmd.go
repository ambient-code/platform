package login

import (
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
)

var args struct {
	token   string
	url     string
	project string
}

var Cmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to the Ambient API server",
	Long:  "Log in to the Ambient API server by providing an access token. The token is saved to the configuration file for subsequent commands.",
	Args:  cobra.NoArgs,
	RunE:  run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(&args.token, "token", "", "Access token (required)")
	flags.StringVar(&args.url, "url", "", "API server URL (default: http://localhost:8000)")
	flags.StringVar(&args.project, "project", "", "Default project name")
}

func run(cmd *cobra.Command, _ []string) error {
	if args.token == "" {
		return fmt.Errorf("--token is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg.AccessToken = args.token

	if args.url != "" {
		cfg.APIUrl = args.url
	}

	if args.project != "" {
		cfg.Project = args.project
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	location, _ := config.Location()
	fmt.Fprintf(cmd.OutOrStdout(), "Login successful. Configuration saved to %s\n", location)
	return nil
}

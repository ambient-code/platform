// Package login implements the login subcommand for saving credentials.
package login

import (
	"fmt"
	"net/url"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/spf13/cobra"
)

var args struct {
	token              string
	url                string
	project            string
	insecureSkipVerify bool
}

var Cmd = &cobra.Command{
	Use:   "login [SERVER_URL]",
	Short: "Log in to the Ambient API server",
	Long:  "Log in to the Ambient API server by providing an access token. The token is saved to the configuration file for subsequent commands.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(&args.token, "token", "", "Access token (required)")
	flags.StringVar(&args.url, "url", "", "API server URL (default: http://localhost:8000)")
	flags.StringVar(&args.project, "project", "", "Default project name")
	flags.BoolVar(&args.insecureSkipVerify, "insecure-skip-tls-verify", false, "Skip TLS certificate verification (insecure)")
}

func run(cmd *cobra.Command, positional []string) error {
	if args.token == "" {
		return fmt.Errorf("--token is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg.AccessToken = args.token

	serverURL := args.url
	if len(positional) > 0 {
		serverURL = positional[0]
	}

	if serverURL != "" {
		parsed, err := url.Parse(serverURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid URL %q: must be a valid URL with scheme and host (e.g. https://api.example.com)", serverURL)
		}
		cfg.APIUrl = serverURL
	}

	if args.project != "" {
		cfg.Project = args.project
	}

	if args.insecureSkipVerify {
		cfg.InsecureTLSVerify = true
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	location, err := config.Location()
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "Login successful. Configuration saved.")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Login successful. Configuration saved to %s\n", location)
	}

	if args.insecureSkipVerify {
		fmt.Fprintln(cmd.ErrOrStderr(), "Warning: TLS certificate verification is disabled (--insecure-skip-tls-verify)")
	}

	if exp, err := config.TokenExpiry(args.token); err == nil && !exp.IsZero() {
		if time.Until(exp) < 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: token is already expired (at %s)\n", exp.Format(time.RFC3339))
		} else if time.Until(exp) < 24*time.Hour {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: token expires soon (at %s)\n", exp.Format(time.RFC3339))
		}
	}
	return nil
}

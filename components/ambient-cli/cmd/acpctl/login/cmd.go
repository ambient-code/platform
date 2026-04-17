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
	useAuthCode        bool
	issuerURL          string
	clientID           string
	clientSecret       string
}

var Cmd = &cobra.Command{
	Use:   "login [SERVER_URL]",
	Short: "Log in to the Ambient API server",
	Long: `Log in to the Ambient API server by providing an access token or using
the browser-based OAuth2 authorization code flow against Red Hat SSO.

To log in with a static token:
  acpctl login --token <token> --url https://api.example.com

To log in via browser (OAuth2 authorization code + PKCE via Red Hat SSO):
  acpctl login --use-auth-code --url https://api.example.com`,
	Args: cobra.MaximumNArgs(1),
	RunE: run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(&args.token, "token", "", "Access token (mutually exclusive with --use-auth-code)")
	flags.StringVar(&args.url, "url", "", "API server URL (default: http://localhost:8000)")
	flags.StringVar(&args.project, "project", "", "Default project name")
	flags.BoolVar(&args.insecureSkipVerify, "insecure-skip-tls-verify", false, "Skip TLS certificate verification (insecure)")
	flags.BoolVar(&args.useAuthCode, "use-auth-code", false, "Log in via browser using OAuth2 authorization code flow (Red Hat SSO)")
	flags.StringVar(&args.issuerURL, "issuer-url", defaultIssuerURL, "OIDC issuer URL (used with --use-auth-code)")
	flags.StringVar(&args.clientID, "client-id", defaultClientID, "OAuth2 client ID (used with --use-auth-code)")
	flags.StringVar(&args.clientSecret, "client-secret", "", "OAuth2 client secret (used with --use-auth-code for confidential clients; never persisted to config)")
}

func run(cmd *cobra.Command, positional []string) error {
	if args.useAuthCode && args.token != "" {
		return fmt.Errorf("--use-auth-code and --token are mutually exclusive")
	}
	if !args.useAuthCode && args.token == "" {
		return fmt.Errorf("one of --token or --use-auth-code is required")
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

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

	var accessToken string

	if args.useAuthCode {
		tokens, err := runAuthCodeFlow(args.issuerURL, args.clientID, args.clientSecret)
		if err != nil {
			return fmt.Errorf("auth-code login: %w", err)
		}
		accessToken = tokens.AccessToken
		cfg.RefreshToken = tokens.RefreshToken
		cfg.IssuerURL = args.issuerURL
		cfg.ClientID = args.clientID
	} else {
		accessToken = args.token
		cfg.RefreshToken = ""
		cfg.IssuerURL = ""
		cfg.ClientID = ""
	}

	cfg.AccessToken = accessToken

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

	if exp, err := config.TokenExpiry(accessToken); err == nil && !exp.IsZero() {
		if time.Until(exp) < 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: token is already expired (at %s)\n", exp.Format(time.RFC3339))
		} else if time.Until(exp) < 24*time.Hour {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: token expires soon (at %s)\n", exp.Format(time.RFC3339))
		}
	}
	return nil
}

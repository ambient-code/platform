package whoami

import (
	"fmt"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current user information",
	Args:  cobra.NoArgs,
	RunE:  run,
}

func run(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	token := cfg.GetToken()
	if token == "" {
		return fmt.Errorf("not logged in; run 'acpctl login' first")
	}

	out := cmd.OutOrStdout()

	if strings.HasPrefix(token, "sha256~") {
		fmt.Fprintf(out, "Token type: OpenShift service account\n")
		fmt.Fprintf(out, "API URL:    %s\n", cfg.GetAPIUrl())
		fmt.Fprintf(out, "Project:    %s\n", cfg.GetProject())
		return nil
	}

	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(token, claims)
	if err != nil {
		fmt.Fprintf(out, "Token type: opaque\n")
		fmt.Fprintf(out, "API URL:    %s\n", cfg.GetAPIUrl())
		fmt.Fprintf(out, "Project:    %s\n", cfg.GetProject())
		return nil
	}

	if sub, ok := claims["sub"].(string); ok {
		fmt.Fprintf(out, "User:       %s\n", sub)
	}
	if email, ok := claims["email"].(string); ok {
		fmt.Fprintf(out, "Email:      %s\n", email)
	}
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		fmt.Fprintf(out, "Expires:    %s\n", expTime.Format(time.RFC3339))
	}
	fmt.Fprintf(out, "API URL:    %s\n", cfg.GetAPIUrl())
	fmt.Fprintf(out, "Project:    %s\n", cfg.GetProject())

	return nil
}

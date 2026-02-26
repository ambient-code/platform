package completion

import (
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for acpctl.

Supported shells: bash, zsh, fish, powershell

Example:
  # Bash
  acpctl completion bash > /etc/bash_completion.d/acpctl

  # Zsh
  acpctl completion zsh > "${fpath[1]}/_acpctl"

  # Fish
  acpctl completion fish > ~/.config/fish/completions/acpctl.fish`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE:      run,
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	switch cmdArgs[0] {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return cmd.Help()
	}
}

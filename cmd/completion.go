package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for bash, zsh, fish, or powershell.

To load completions:

Bash:
  $ source <(ocs completion bash)
  $ ocs completion bash > /etc/bash_completion.d/ocs

Zsh:
  $ ocs completion zsh > "${fpath[1]}/_ocs"

Fish:
  $ ocs completion fish | source
  $ ocs completion fish > ~/.config/fish/completions/ocs.fish

PowerShell:
  PS> ocs completion powershell | Out-String | Invoke-Expression
`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			default:
				return cmd.Help()
			}
		},
	}
}

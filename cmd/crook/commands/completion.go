// Package commands provides the CLI command implementations for crook.
package commands

import (
	"github.com/spf13/cobra"
)

// newCompletionCmd creates the completion subcommand for generating shell completion scripts
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for crook.

To load completions:

Bash:
  $ source <(crook completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ crook completion bash > /etc/bash_completion.d/crook
  # macOS:
  $ crook completion bash > $(brew --prefix)/etc/bash_completion.d/crook

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ crook completion zsh > "${fpath[1]}/_crook"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ crook completion fish | source

  # To load completions for each session, execute once:
  $ crook completion fish > ~/.config/fish/completions/crook.fish

PowerShell:
  PS> crook completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> crook completion powershell > crook.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(out)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			case "fish":
				return cmd.Root().GenFishCompletion(out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(out)
			default:
				return nil
			}
		},
	}

	return cmd
}

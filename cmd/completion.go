package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var desc = fmt.Sprintf(`Generate the autocompletion script for eko for the specified shell.
See each sub-command's help for details on how to use the generated script.

Bash:

  $ source <(eko completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ eko completion bash > /etc/bash_completion.d/eko
  # macOS:
  $ eko completion bash > $(brew --prefix)/etc/bash_completion.d/eko

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ eko completion zsh > "${fpath[1]}/_eko"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ eko completion fish | source

  # To load completions for each session, execute once:
  $ eko completion fish > ~/.config/fish/completions/eko.fish

PowerShell:

  PS> eko completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> eko completion powershell > eko.ps1
  # and source this file from your PowerShell profile.
`)

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "Generate completion script",
	Long:                  desc,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

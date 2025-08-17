// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for RepoBird CLI.

To enable tab completions for both 'repobird' and 'rb' commands:

Zsh:
  # IMPORTANT: The completion setup must come AFTER 'autoload -U compinit; compinit' in your ~/.zshrc
  
  # Method 1: Add to ~/.zshrc (simplest, works everywhere, includes 'rb' alias)
  $ echo 'eval "$(repobird completion zsh)"' >> ~/.zshrc
  $ source ~/.zshrc
  
  # Method 2: Static file in personal directory (faster startup)
  $ mkdir -p ~/.config/zsh/completions
  $ repobird completion zsh > ~/.config/zsh/completions/_repobird
  $ echo 'fpath=(~/.config/zsh/completions $fpath)' >> ~/.zshrc
  $ source ~/.zshrc

Bash:
  # Add to ~/.bashrc:
  $ echo 'source <(repobird completion bash)' >> ~/.bashrc
  $ echo 'complete -o default -F __start_repobird rb' >> ~/.bashrc
  $ source ~/.bashrc

  # Alternative: Install to system directory
  # Linux:
  $ repobird completion bash | sudo tee /etc/bash_completion.d/repobird
  # macOS:
  $ repobird completion bash > $(brew --prefix)/etc/bash_completion.d/repobird

Fish:
  # Install for current and future sessions:
  $ repobird completion fish > ~/.config/fish/completions/repobird.fish
  $ repobird completion fish | sed 's/repobird/rb/g' > ~/.config/fish/completions/rb.fish
  
  # Or for current session only:
  $ repobird completion fish | source

PowerShell:
  # Add to your PowerShell profile:
  PS> repobird completion powershell >> $PROFILE
  
  # Or for current session:
  PS> repobird completion powershell | Out-String | Invoke-Expression

Troubleshooting:
  - For zsh: Ensure completion setup comes AFTER 'compinit' in ~/.zshrc
  - Restart your terminal or run 'source ~/.zshrc' (or ~/.bashrc) after setup
  - Test with: repobird [TAB][TAB] or rb [TAB][TAB]
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
			// Add rb alias completion at the end
			fmt.Println("\n# Enable completion for 'rb' alias")
			fmt.Println("compdef _repobird rb")
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

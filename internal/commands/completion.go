// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const completionAliasName = "rb"

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate or install shell completion scripts",
	Long: `Generate or install shell completion scripts for RepoBird CLI.

To enable tab completions for both 'repobird' and 'rb' commands:

Recommended:
  $ repobird completion install zsh
  $ repobird completion install bash
  $ repobird completion install fish
  $ repobird completion install powershell

Preview changes before writing files:
  $ repobird completion install zsh --dry-run

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
	Args:                  cobra.NoArgs,
}

var completionGenerateCmd = &cobra.Command{
	Use:                   "generate [bash|zsh|fish|powershell]",
	Short:                 "Generate shell completion scripts",
	Aliases:               []string{"gen"},
	DisableFlagsInUseLine: true,
	ValidArgs:             completionShells(),
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		_ = writeCompletionScript(cmd.Root(), args[0], cmd.OutOrStdout())
	},
}

var completionInstallCmd = &cobra.Command{
	Use:                   "install [bash|zsh|fish|powershell]",
	Short:                 "Install shell completions for your user account",
	DisableFlagsInUseLine: true,
	ValidArgs:             completionShells(),
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("find home directory: %w", err)
		}

		return installCompletion(cmd.Root(), args[0], completionInstallOptions{
			HomeDir: home,
			Alias:   completionAliasName,
			DryRun:  dryRun,
			Out:     cmd.OutOrStdout(),
		})
	},
}

type completionInstallOptions struct {
	HomeDir string
	Alias   string
	DryRun  bool
	Out     io.Writer
}

func init() {
	completionCmd.Run = completionGenerateCmd.Run
	completionCmd.ValidArgs = completionGenerateCmd.ValidArgs
	completionCmd.Args = completionGenerateCmd.Args
	completionCmd.AddCommand(completionGenerateCmd, completionInstallCmd)
	completionInstallCmd.Flags().Bool("dry-run", false, "print installation actions without changing files")
}

// completionCmd is now added to rootCmd in root.go init() function

func completionShells() []string {
	return []string{"bash", "zsh", "fish", "powershell"}
}

func writeCompletionScript(root *cobra.Command, shell string, out io.Writer) error {
	switch shell {
	case "bash":
		return root.GenBashCompletion(out)
	case "zsh":
		if err := root.GenZshCompletion(out); err != nil {
			return err
		}
		_, err := fmt.Fprintf(out, "\n# Enable completion for '%s' alias\ncompdef _repobird %s\n", completionAliasName, completionAliasName)
		return err
	case "fish":
		return root.GenFishCompletion(out, true)
	case "powershell":
		return root.GenPowerShellCompletionWithDesc(out)
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func installCompletion(root *cobra.Command, shell string, opts completionInstallOptions) error {
	if opts.Alias == "" {
		opts.Alias = completionAliasName
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}

	switch shell {
	case "bash":
		return installBashCompletion(opts)
	case "zsh":
		return installZshCompletion(root, opts)
	case "fish":
		return installFishCompletion(root, opts)
	case "powershell":
		return installPowerShellCompletion(opts)
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func installBashCompletion(opts completionInstallOptions) error {
	path := filepath.Join(opts.HomeDir, ".bashrc")
	block := completionBlock("RepoBird CLI completions", []string{
		"source <(repobird completion bash)",
		fmt.Sprintf("complete -o default -F __start_repobird %s", opts.Alias),
	})
	return appendCompletionBlock(path, block, opts)
}

func installZshCompletion(root *cobra.Command, opts completionInstallOptions) error {
	completionDir := filepath.Join(opts.HomeDir, ".config", "zsh", "completions")
	completionPath := filepath.Join(completionDir, "_repobird")
	if err := writeGeneratedCompletion(root, "zsh", completionPath, opts); err != nil {
		return err
	}

	zshrc := filepath.Join(opts.HomeDir, ".zshrc")
	block := completionBlock("RepoBird CLI completions", []string{
		"fpath=(~/.config/zsh/completions $fpath)",
		"autoload -U compinit",
		"compinit",
	})
	return appendCompletionBlock(zshrc, block, opts)
}

func installFishCompletion(root *cobra.Command, opts completionInstallOptions) error {
	completionDir := filepath.Join(opts.HomeDir, ".config", "fish", "completions")
	repobirdPath := filepath.Join(completionDir, "repobird.fish")
	if err := writeGeneratedCompletion(root, "fish", repobirdPath, opts); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := writeCompletionScript(root, "fish", &buf); err != nil {
		return err
	}
	aliasScript := strings.ReplaceAll(buf.String(), "repobird", opts.Alias)
	return writeFile(filepath.Join(completionDir, opts.Alias+".fish"), []byte(aliasScript), opts)
}

func installPowerShellCompletion(opts completionInstallOptions) error {
	profile := os.Getenv("PROFILE")
	if profile == "" {
		profile = defaultPowerShellProfile(opts.HomeDir)
	}

	block := completionBlock("RepoBird CLI completions", []string{
		"repobird completion powershell | Out-String | Invoke-Expression",
	})
	return appendCompletionBlock(profile, block, opts)
}

func defaultPowerShellProfile(home string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	return filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
}

func writeGeneratedCompletion(root *cobra.Command, shell, path string, opts completionInstallOptions) error {
	var buf bytes.Buffer
	if err := writeCompletionScript(root, shell, &buf); err != nil {
		return err
	}
	return writeFile(path, buf.Bytes(), opts)
}

func appendCompletionBlock(path, block string, opts completionInstallOptions) error {
	if opts.DryRun {
		_, err := fmt.Fprintf(opts.Out, "Would update %s\n", path)
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if strings.Contains(string(content), block) {
		_, err := fmt.Fprintf(opts.Out, "Completion setup already present in %s\n", path)
		return err
	}

	next := strings.TrimRight(string(content), "\n")
	if next != "" {
		next += "\n\n"
	}
	next += block + "\n"

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, err = fmt.Fprintf(opts.Out, "Updated %s\n", path)
	return err
}

func writeFile(path string, content []byte, opts completionInstallOptions) error {
	if opts.DryRun {
		_, err := fmt.Fprintf(opts.Out, "Would write %s\n", path)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, err := fmt.Fprintf(opts.Out, "Wrote %s\n", path)
	return err
}

func completionBlock(title string, lines []string) string {
	return fmt.Sprintf("# %s\n%s", title, strings.Join(lines, "\n"))
}

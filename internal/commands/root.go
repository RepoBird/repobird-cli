// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/pkg/version"
)

var (
	cfg        *config.SecureConfig
	cfgFile    string
	debug      bool
	debugUser  bool
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:     "repobird",
	Short:   "CLI and TUI for RepoBird.ai - trigger AI coding agents and manage runs",
	Version: version.GetBuildInfo(),
	Long: fmt.Sprintf(`CLI and TUI (Terminal User Interface) for RepoBird.ai - trigger AI coding agents,
and monitor your AI agent runs through an interactive dashboard.

Base URL: %s
Get API Key: %s`, config.GetURLs().BaseURL, config.GetAPIKeysURL()),
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		var err error
		config.SetConfigFile(cfgFile)
		cfg, err = config.LoadSecureConfig()
		if err != nil {
			// Don't fail if config doesn't exist yet
			cfg = &config.SecureConfig{
				Config: &config.Config{},
			}
		}

		if debug {
			cfg.Debug = true
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		styler := stderrStyle()
		// Format error message for better user experience
		errorMsg := errors.FormatUserError(err)
		fmt.Fprintf(os.Stderr, "%s %s\n", styler.Error("Error:"), errorMsg)

		// Add helpful hints for common errors
		if errors.IsQuotaExceeded(err) || strings.Contains(strings.ToLower(errorMsg), "no runs remaining") {
			fmt.Fprintf(os.Stderr, "\n%s Upgrade your plan at %s\n", styler.Info("Hint:"), config.GetPricingURL())
		} else if errors.IsAuthError(err) && !strings.Contains(strings.ToLower(errorMsg), "no runs remaining") {
			fmt.Fprintf(os.Stderr, "\n%s Run 'repobird config set api-key YOUR_API_KEY' to configure authentication\n", styler.Info("Hint:"))
		} else if errors.IsNetworkError(err) {
			fmt.Fprintf(os.Stderr, "\n%s Check your internet connection and try again\n", styler.Info("Hint:"))
		}

		os.Exit(1)
	}
}

//nolint:gochecknoinits // Required for CLI root command initialization
func init() {
	// Set custom version template to show just the version info
	rootCmd.SetVersionTemplate(version.GetBuildInfo() + "\n")
	rootCmd.SetHelpFunc(coloredHelp)
	rootCmd.SetUsageFunc(coloredUsage)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/repobird/config.yaml or $HOME/.config/repobird/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVar(&debugUser, "debug-user", false, "enable debug user mode with mock data")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")

	// Add -v as shorthand for --version
	rootCmd.Flags().BoolP("version", "v", false, "version for repobird")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(newRunPresetCommand("basic"))
	rootCmd.AddCommand(newRunPresetCommand("pro"))
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(repoCmd)
	InitConfigSubcommands() // Initialize config subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(NewBulkCommand())
	rootCmd.AddCommand(examplesCmd)
	rootCmd.AddCommand(completionCmd)
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print version information",
	Run: func(cmd *cobra.Command, _ []string) {
		styler := styleFor(cmd.OutOrStdout())
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "%s %s\n", styler.Label("Version:"), version.GetVersion())
		fmt.Fprintf(out, "%s %s\n", styler.Label("Git Commit:"), version.GitCommit)
		fmt.Fprintf(out, "%s %s\n", styler.Label("Build Date:"), version.BuildDate)
		fmt.Fprintf(out, "%s %s\n", styler.Label("Go Version:"), runtime.Version())
		fmt.Fprintf(out, "%s %s/%s\n", styler.Label("OS/Arch:"), runtime.GOOS, runtime.GOARCH)
	},
}

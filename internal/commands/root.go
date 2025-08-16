package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/pkg/version"
)

var (
	cfg       *config.SecureConfig
	cfgFile   string
	debug     bool
	debugUser bool
)

var rootCmd = &cobra.Command{
	Use:   "repobird",
	Short: "CLI and TUI for RepoBird.ai - trigger AI coding agents and manage runs",
	Long: `CLI and TUI (Terminal User Interface) for RepoBird.ai - trigger AI coding agents,
submit batch runs, and monitor your AI agent runs through an interactive dashboard.

Base URL: https://repobird.ai
Get API Key: https://repobird.ai/dashboard/user-profile/api-keys`,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		var err error
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
		// Format error message for better user experience
		errorMsg := errors.FormatUserError(err)
		fmt.Fprintf(os.Stderr, "Error: %s\n", errorMsg)

		// Add helpful hints for common errors
		if errors.IsAuthError(err) {
			fmt.Fprintf(os.Stderr, "\nHint: Run 'repobird config set api-key YOUR_API_KEY' to configure authentication\n")
		} else if errors.IsQuotaExceeded(err) {
			fmt.Fprintf(os.Stderr, "\nHint: Check your usage at https://repobird.ai/dashboard\n")
		} else if errors.IsNetworkError(err) {
			fmt.Fprintf(os.Stderr, "\nHint: Check your internet connection and try again\n")
		}

		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.repobird/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVar(&debugUser, "debug-user", false, "enable debug user mode with mock data")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(NewBulkCommand())
	rootCmd.AddCommand(examplesCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.GetBuildInfo())
	},
}

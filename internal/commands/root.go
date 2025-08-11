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
	Short: "RepoBird CLI - AI-powered code generation and repository management",
	Long: `RepoBird CLI allows you to run AI-powered tasks on your repositories,
manage runs, and integrate with the RepoBird platform.`,
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
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(NewBulkCommand())
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.GetBuildInfo())
	},
}

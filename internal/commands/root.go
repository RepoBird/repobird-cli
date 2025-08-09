package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/pkg/version"
)

var (
	cfg     *config.Config
	cfgFile string
	debug   bool
)

var rootCmd = &cobra.Command{
	Use:   "repobird",
	Short: "RepoBird CLI - AI-powered code generation and repository management",
	Long: `RepoBird CLI allows you to run AI-powered tasks on your repositories,
manage runs, and integrate with the RepoBird platform.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		
		if debug {
			cfg.Debug = true
		}
		
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.repobird/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetBuildInfo())
	},
}
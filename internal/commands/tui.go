package commands

import (
	"fmt"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/mock"
	"github.com/repobird/repobird-cli/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive Terminal User Interface",
	Long: `Launch the RepoBird TUI for an interactive experience.
The TUI provides:
- Visual run management with real-time status updates
- Vim-style keybindings for efficient navigation
- Multiple views for listing, creating, and monitoring runs
- Automatic polling for active runs
- Rich terminal interface with color-coded statuses`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Check if debug-user flag is set
	if debugUser {
		// Use mock client for testing
		client := api.NewClient("mock-api-key", api.DefaultAPIURL, debug)
		mockClient := mock.NewMockClient(client)
		app := tui.NewApp(mockClient)
		return app.Run()
	}

	cfg, err := config.LoadSecureConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured. Run 'rb config set api-key YOUR_KEY' first")
	}

	client := api.NewClient(cfg.APIKey, cfg.APIURL, cfg.Debug)
	app := tui.NewApp(client)

	return app.Run()
}

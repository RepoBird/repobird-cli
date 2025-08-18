// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/mock"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/tui"
	tuiDebug "github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
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

//nolint:gochecknoinits // Required for CLI command registration
func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Check if debug-user flag is set
	if debugUser {
		// Use mock client for testing
		client := api.NewClient("mock-api-key", utils.GetAPIURL(), debug)
		mockClient := mock.NewMockClient(client)

		// Set the debug user immediately for cache initialization
		debugUserInfo := &models.UserInfo{
			Email:             "debug-user@repobird.ai",
			Name:              "Debug User",
			ID:                -1, // Negative ID for debug mode
			GithubUsername:    "debug-user",
			RemainingRuns:     100, // Deprecated
			TotalRuns:         500, // Deprecated
			RemainingProRuns:  80,
			RemainingPlanRuns: 20,
			ProTotalRuns:      400,
			PlanTotalRuns:     100,
			Tier:              "premium",
		}
		services.SetCurrentUser(debugUserInfo)

		// Log debug mode activation
		tuiDebug.LogToFilef("🎮 DEBUG MODE: Activated with mock client and debug user ID=%d 🎮\n", debugUserInfo.ID)

		app := tui.NewApp(mockClient)
		return app.Run()
	}

	cfg, err := config.LoadSecureConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	apiURL := utils.GetAPIURL(cfg.APIURL)
	client := api.NewClient(cfg.APIKey, apiURL, cfg.Debug)
	app := tui.NewApp(client)

	return app.Run()
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/utils"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify current API key",
	Long:  `Verify that your stored API key is valid and check your account status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config with API key
		secureConfig, err := config.LoadSecureConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if secureConfig.APIKey == "" {
			return errors.NoAPIKeyError()
		}

		// Verify with API
		apiURL := utils.GetAPIURL(secureConfig.APIURL)
		client := api.NewClient(secureConfig.APIKey, apiURL, secureConfig.Debug)
		userInfo, err := client.VerifyAuth()
		if err != nil {
			return fmt.Errorf("API key verification failed: %w", err)
		}

		// Set the current user for cache initialization
		services.SetCurrentUser(userInfo)

		fmt.Println("✓ API key is valid")
		fmt.Printf("  Email: %s\n", userInfo.Email)
		fmt.Printf("  Tier: %s\n", userInfo.Tier)

		fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, userInfo.ProTotalRuns)
		if userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
			fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
		}

		// Calculate reset date (assuming monthly reset)
		now := time.Now()
		nextMonth := now.AddDate(0, 1, 0)
		resetDate := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
		fmt.Printf("  Resets: %s\n", resetDate.Format("2006-01-02"))

		return nil
	},
}

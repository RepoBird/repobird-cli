package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/services"
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
		client := api.NewClient(secureConfig.APIKey, secureConfig.APIURL, secureConfig.Debug)
		userInfo, err := client.VerifyAuth()
		if err != nil {
			return fmt.Errorf("API key verification failed: %w", err)
		}

		// Set the current user for cache initialization
		services.SetCurrentUser(userInfo)

		fmt.Println("✓ API key is valid")
		fmt.Printf("  Email: %s\n", userInfo.Email)
		fmt.Printf("  Tier: %s\n", userInfo.Tier)

		// Show runs - always show Runs first, then Plan Runs
		// For Free tier, always show both lines
		if strings.Contains(strings.ToLower(userInfo.Tier), "free") {
			// Free tier - show both, Runs then Plan Runs
			// Use hardcoded defaults if totals are 0 (API didn't return them)
			proTotal := userInfo.ProTotalRuns
			planTotal := userInfo.PlanTotalRuns
			if proTotal == 0 {
				proTotal = 3 // Free tier default
			}
			if planTotal == 0 {
				planTotal = 5 // Free tier default
			}
			// Always show tier total, not extra credits
			fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, proTotal)
			fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, planTotal)
		} else {
			// Other tiers - show Runs, and Plan Runs if available
			fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, userInfo.ProTotalRuns)
			if userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
				fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
			}
		}

		// Calculate reset date (assuming monthly reset)
		now := time.Now()
		nextMonth := now.AddDate(0, 1, 0)
		resetDate := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
		fmt.Printf("  Resets: %s\n", resetDate.Format("2006-01-02"))

		return nil
	},
}

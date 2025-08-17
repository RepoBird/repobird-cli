// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/utils"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display authentication information",
	Long:  `Display information about your current authentication status and storage method.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		secureConfig, err := config.LoadSecureConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		storageInfo := secureConfig.GetStorageInfo()

		// Display storage information
		fmt.Println("Authentication Status:")
		fmt.Println()

		switch storageInfo["source"] {
		case "environment":
			fmt.Println("  Method: Environment Variable")
			fmt.Println("  Source: REPOBIRD_API_KEY")
			fmt.Println("  Security: ⚠️  Semi-secure (suitable for CI/CD)")
			fmt.Println()
			fmt.Println("  For better security in development, use 'repobird login'")

		case "system_keyring":
			fmt.Printf("  Method: System Keyring\n")
			fmt.Printf("  Type: %s\n", storageInfo["keyring_type"])
			fmt.Println("  Security: ✓ Secure")

		case "encrypted_file":
			fmt.Println("  Method: Encrypted File")
			fmt.Printf("  Location: %s\n", storageInfo["location"])
			fmt.Println("  Security: ✓ Secure (AES-256-GCM)")

		case "plain_text_config":
			fmt.Println("  Method: Plain Text Config")
			fmt.Printf("  Location: %s\n", storageInfo["location"])
			fmt.Println("  Security: ⚠️  NOT SECURE")
			fmt.Println()
			fmt.Printf("  Warning: %s\n", storageInfo["warning"])

		default:
			fmt.Println("  Status: Not configured")
			fmt.Println()
			fmt.Println("  Run 'repobird login' to configure your API key")
			return nil
		}

		// Try to get user info if API key is available
		if secureConfig.APIKey != "" {
			fmt.Println()
			apiURL := utils.GetAPIURL(secureConfig.APIURL)
			client := api.NewClient(secureConfig.APIKey, apiURL, secureConfig.Debug)
			if userInfo, err := client.VerifyAuth(); err == nil {
				// Set the current user for cache initialization
				services.SetCurrentUser(userInfo)
				fmt.Println("Account Information:")
				fmt.Printf("  Email: %s\n", userInfo.Email)
				fmt.Printf("  Tier: %s\n", userInfo.Tier)

				// Always show both for Free tier, or if either has values
				if strings.Contains(strings.ToLower(userInfo.Tier), "free") ||
					userInfo.ProTotalRuns > 0 || userInfo.RemainingProRuns > 0 {
					// Use hardcoded defaults if totals are 0 (API didn't return them)
					proTotal := userInfo.ProTotalRuns
					if proTotal == 0 && strings.Contains(strings.ToLower(userInfo.Tier), "free") {
						proTotal = 3 // Free tier default
					} else if proTotal == 0 {
						proTotal = 30 // Pro tier default
					}
					// Handle admin credits that exceed defaults
					if userInfo.RemainingProRuns > proTotal {
						proTotal = userInfo.RemainingProRuns
					}
					fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, proTotal)
				}
				if strings.Contains(strings.ToLower(userInfo.Tier), "free") ||
					userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
					// Use hardcoded defaults if totals are 0 (API didn't return them)
					planTotal := userInfo.PlanTotalRuns
					if planTotal == 0 && strings.Contains(strings.ToLower(userInfo.Tier), "free") {
						planTotal = 5 // Free tier default
					} else if planTotal == 0 {
						planTotal = 35 // Pro tier default
					}
					// Handle admin credits that exceed defaults
					if userInfo.RemainingPlanRuns > planTotal {
						planTotal = userInfo.RemainingPlanRuns
					}
					fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, planTotal)
				}

				// Calculate reset date
				now := time.Now()
				nextMonth := now.AddDate(0, 1, 0)
				resetDate := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
				fmt.Printf("  Resets: %s\n", resetDate.Format("2006-01-02"))
			} else {
				fmt.Println()
				fmt.Println("  ⚠️  Could not fetch account information")
				fmt.Println("  Run 'repobird verify' to check your API key")
			}
		}

		return nil
	},
}

// Helper to cache user info
type cachedUserInfo struct {
	UserInfo *models.UserInfo
	CachedAt time.Time
}

var userInfoCache *cachedUserInfo
var cacheTimeout = 5 * time.Minute

func getCachedUserInfo(apiKey, apiEndpoint string, debug bool) (*models.UserInfo, error) {
	// Check if cache is valid
	if userInfoCache != nil && time.Since(userInfoCache.CachedAt) < cacheTimeout {
		return userInfoCache.UserInfo, nil
	}

	// Fetch fresh data
	client := api.NewClient(apiKey, apiEndpoint, debug)
	userInfo, err := client.VerifyAuth()
	if err != nil {
		return nil, err
	}

	// Set the current user for cache initialization
	services.SetCurrentUser(userInfo)

	// Update cache
	userInfoCache = &cachedUserInfo{
		UserInfo: userInfo,
		CachedAt: time.Now(),
	}

	return userInfo, nil
}

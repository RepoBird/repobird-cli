// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
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
				printAccountUsage(userInfo)
				printAccountReset(userInfo)
			} else {
				fmt.Println()
				fmt.Println("  ⚠️  Could not fetch account information")
				fmt.Println("  Run 'repobird verify' to check your API key")
			}
		}

		return nil
	},
}

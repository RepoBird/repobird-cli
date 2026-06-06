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
		styler := stdoutStyle()

		// Display storage information
		fmt.Println(styler.Heading("Authentication Status:"))
		fmt.Println()

		switch storageInfo["source"] {
		case "environment":
			fmt.Printf("  %s Environment Variable\n", styler.Label("Method:"))
			fmt.Printf("  %s REPOBIRD_API_KEY\n", styler.Label("Source:"))
			fmt.Printf("  %s %s\n", styler.Label("Security:"), styler.Warning("⚠️  Semi-secure (suitable for CI/CD)"))
			fmt.Println()
			fmt.Printf("  %s For better security in development, use 'repobird login'\n", styler.Info("Tip:"))

		case "system_keyring":
			fmt.Printf("  %s System Keyring\n", styler.Label("Method:"))
			fmt.Printf("  %s %s\n", styler.Label("Type:"), storageInfo["keyring_type"])
			fmt.Printf("  %s %s\n", styler.Label("Security:"), styler.Success("✓ Secure"))

		case "encrypted_file":
			fmt.Printf("  %s Encrypted File\n", styler.Label("Method:"))
			fmt.Printf("  %s %s\n", styler.Label("Location:"), storageInfo["location"])
			fmt.Printf("  %s %s\n", styler.Label("Security:"), styler.Success("✓ Secure (AES-256-GCM)"))

		case "plain_text_config":
			fmt.Printf("  %s Plain Text Config\n", styler.Label("Method:"))
			fmt.Printf("  %s %s\n", styler.Label("Location:"), storageInfo["location"])
			fmt.Printf("  %s %s\n", styler.Label("Security:"), styler.Warning("⚠️  NOT SECURE"))
			fmt.Println()
			fmt.Printf("  %s %s\n", styler.Warning("Warning:"), storageInfo["warning"])

		default:
			fmt.Printf("  %s %s\n", styler.Label("Status:"), styler.Muted("Not configured"))
			fmt.Println()
			fmt.Printf("  %s Run 'repobird login' to configure your API key\n", styler.Info("Hint:"))
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
				fmt.Println(styler.Heading("Account Information:"))
				fmt.Printf("  %s %s\n", styler.Label("Email:"), userInfo.Email)
				fmt.Printf("  %s %s\n", styler.Label("Tier:"), userInfo.Tier)
				printAccountUsage(userInfo)
				printAccountReset(userInfo)
			} else {
				fmt.Println()
				fmt.Println("  " + styler.Warning("⚠️  Could not fetch account information"))
				fmt.Printf("  %s Run 'repobird verify' to check your API key\n", styler.Info("Hint:"))
			}
		}

		return nil
	},
}

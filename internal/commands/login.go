// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/utils"
)

func readAPIKeyInteractive(stdin *os.File, output io.Writer) (string, error) {
	const prompt = "Enter your API key: "

	fmt.Fprint(output, prompt)
	if term.IsTerminal(int(stdin.Fd())) {
		bytePassword, err := term.ReadPassword(int(stdin.Fd()))
		fmt.Fprintln(output)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(bytePassword)), nil
	}

	reader := bufio.NewReader(stdin)
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure your API key securely",
	Long: `Configure your RepoBird API key using secure storage.
The key will be stored in your system keyring when available,
or in an encrypted file as a fallback.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		styler := stdoutStyle()
		fmt.Println(styler.Heading("Welcome to RepoBird CLI!"))
		fmt.Printf("%s %s\n", styler.Label("Get your API key at:"), styler.URL(config.GetAPIKeysURL()))
		fmt.Println()

		// Check if API key is provided as argument (for CI/CD)
		var apiKey string
		if len(args) > 0 {
			apiKey = args[0]
		} else {
			// Interactive prompt for API key.
			maskedKey, err := readAPIKeyInteractive(os.Stdin, os.Stdout)
			if err != nil {
				return err
			} else {
				apiKey = strings.TrimSpace(maskedKey)
			}
		}

		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		// Verify the API key first
		apiURL := utils.GetAPIURL(cfg.APIURL)
		client := api.NewClient(apiKey, apiURL, cfg.Debug)
		userInfo, err := client.VerifyAuth()
		if err != nil {
			return fmt.Errorf("invalid API key: %w", err)
		}

		// Set the current user for cache initialization
		services.SetCurrentUser(userInfo)

		// Save the API key securely
		secureConfig, err := config.LoadSecureConfig()
		if err != nil {
			secureConfig = &config.SecureConfig{
				Config: &config.Config{},
			}
		}

		if err := secureConfig.SaveAPIKey(apiKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}

		// Show storage information
		storageInfo := secureConfig.GetStorageInfo()
		fmt.Println()
		fmt.Println(styler.Success("✓ API key validated and stored successfully!"))
		fmt.Printf("  %s %s\n", styler.Label("Email:"), userInfo.Email)
		fmt.Printf("  %s %s\n", styler.Label("Tier:"), userInfo.Tier)
		printAccountUsage(userInfo)

		// Display storage method
		fmt.Println()
		switch storageInfo["source"] {
		case "system_keyring":
			fmt.Printf("  %s %s (secure)\n", styler.Label("Storage:"), storageInfo["keyring_type"])
		case "encrypted_file":
			fmt.Printf("  %s Encrypted file (secure)\n", styler.Label("Storage:"))
		case "environment":
			fmt.Printf("  %s Environment variable\n", styler.Label("Storage:"))
		default:
			fmt.Printf("  %s Encrypted file (secure fallback)\n", styler.Label("Storage:"))
		}

		return nil
	},
}

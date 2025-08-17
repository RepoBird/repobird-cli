// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/services"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored API key",
	Long:  `Remove the stored API key from all secure storage locations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.LoadSecureConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Confirm logout
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Are you sure you want to logout? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		if response != "y" && response != "yes" {
			fmt.Println("Logout cancelled.")
			return nil
		}

		storage := config.NewSecureStorage()
		if err := storage.DeleteAPIKey(); err != nil {
			return fmt.Errorf("failed to remove API key: %w", err)
		}

		// Clear the current user cache
		services.ClearCurrentUser()

		fmt.Println("✓ API key removed from secure storage")
		return nil
	},
}

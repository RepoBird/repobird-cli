package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/services"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure your API key securely",
	Long: `Configure your RepoBird API key using secure storage.
The key will be stored in your system keyring when available,
or in an encrypted file as a fallback.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Welcome to RepoBird CLI!")
		fmt.Println("Get your API key at: https://repobird.ai/dashboard/user-profile/api-keys")
		fmt.Println()

		// Check if API key is provided as argument (for CI/CD)
		var apiKey string
		if len(args) > 0 {
			apiKey = args[0]
		} else {
			// Interactive prompt for API key
			fmt.Print("Enter your API key: ")

			// Read password without echoing
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				// Fallback to regular input if terminal read fails
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				apiKey = strings.TrimSpace(input)
			} else {
				apiKey = string(bytePassword)
				fmt.Println() // Add newline after hidden input
			}
		}

		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		// Verify the API key first
		client := api.NewClient(apiKey, cfg.APIURL, cfg.Debug)
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
		fmt.Println("✓ API key validated and stored successfully!")
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

		// Display storage method
		fmt.Println()
		switch storageInfo["source"] {
		case "system_keyring":
			fmt.Printf("  Storage: %s (secure)\n", storageInfo["keyring_type"])
		case "encrypted_file":
			fmt.Println("  Storage: Encrypted file (secure)")
		case "environment":
			fmt.Println("  Storage: Environment variable")
		default:
			fmt.Println("  Storage: Encrypted file (secure fallback)")
		}

		return nil
	},
}

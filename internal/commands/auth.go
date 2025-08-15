package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and API keys",
	Long:  `Manage RepoBird API authentication, including login, logout, and verification.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure your API key securely",
	Long: `Configure your RepoBird API key using secure storage.
The key will be stored in your system keyring when available,
or in an encrypted file as a fallback.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Welcome to RepoBird CLI!")
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
			fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, userInfo.ProTotalRuns)
			fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
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
			return fmt.Errorf("no API key configured. Run 'repobird auth login' first")
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
			fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, userInfo.ProTotalRuns)
			fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
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
			fmt.Println("  For better security in development, use 'repobird auth login'")

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
			fmt.Println("  Run 'repobird auth login' to configure your API key")
			return nil
		}

		// Try to get user info if API key is available
		if secureConfig.APIKey != "" {
			fmt.Println()
			client := api.NewClient(secureConfig.APIKey, secureConfig.APIURL, secureConfig.Debug)
			if userInfo, err := client.VerifyAuth(); err == nil {
				// Set the current user for cache initialization
				services.SetCurrentUser(userInfo)
				fmt.Println("Account Information:")
				fmt.Printf("  Email: %s\n", userInfo.Email)
				fmt.Printf("  Tier: %s\n", userInfo.Tier)
				
				// Always show both for Free tier, or if either has values
				if strings.Contains(strings.ToLower(userInfo.Tier), "free") || 
				   userInfo.ProTotalRuns > 0 || userInfo.RemainingProRuns > 0 {
					fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, userInfo.ProTotalRuns)
				}
				if strings.Contains(strings.ToLower(userInfo.Tier), "free") || 
				   userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
					fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
				}

				// Calculate reset date
				now := time.Now()
				nextMonth := now.AddDate(0, 1, 0)
				resetDate := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
				fmt.Printf("  Resets: %s\n", resetDate.Format("2006-01-02"))
			} else {
				fmt.Println()
				fmt.Println("  ⚠️  Could not fetch account information")
				fmt.Println("  Run 'repobird auth verify' to check your API key")
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

func init() {
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(verifyCmd)
	authCmd.AddCommand(infoCmd)
}

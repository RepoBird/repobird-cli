package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/utils"
)

const (
	configKeyAPIKey = "api-key"
	configKeyAPIURL = "api-url"
	configKeyDebug  = "debug"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage RepoBird configuration",
	Long:  `Manage RepoBird CLI configuration including API keys and endpoints.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// Use secure config for API key handling
		secureCfg, err := config.LoadSecureConfig()
		if err != nil {
			secureCfg = &config.SecureConfig{
				Config: &config.Config{},
			}
		}

		switch key {
		case configKeyAPIKey, "api_key":
			// Use secure storage for API key
			if err := secureCfg.SaveAPIKey(value); err != nil {
				return fmt.Errorf("failed to save API key securely: %w", err)
			}

			// Get storage info to show user where it's stored
			info := secureCfg.GetStorageInfo()
			fmt.Println("✓ API key configured successfully")
			if source, ok := info["source"].(string); ok {
				switch source {
				case "system_keyring":
					if keyringType, ok := info["keyring_type"].(string); ok {
						fmt.Printf("  Stored securely in: %s\n", keyringType)
					}
				case "encrypted_file":
					fmt.Println("  Stored securely in: encrypted file")
				default:
					fmt.Printf("  Storage method: %s\n", source)
				}
			}

		case configKeyAPIURL, "api_url":
			secureCfg.APIURL = value
			fmt.Printf("API URL set to: %s\n", value)
			if err := config.SaveConfig(secureCfg.Config); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

		case configKeyDebug:
			secureCfg.Debug = value == "true"
			fmt.Printf("Debug mode: %v\n", secureCfg.Debug)
			if err := config.SaveConfig(secureCfg.Config); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		secureCfg, err := config.LoadSecureConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(args) == 0 {
			// Show all configuration
			fmt.Printf("API URL: %s\n", secureCfg.APIURL)
			if secureCfg.APIKey != "" {
				fmt.Printf("API Key: %s\n", utils.MaskAPIKey(secureCfg.APIKey))
			} else {
				fmt.Println("API Key: (not set)")
			}
			fmt.Printf("Debug: %v\n", secureCfg.Debug)

			// Show storage info
			info := secureCfg.GetStorageInfo()
			if source, ok := info["source"].(string); ok && source != "not_found" {
				fmt.Printf("\nAPI Key Storage:\n")
				fmt.Printf("  Method: %s\n", source)
				if secure, ok := info["secure"].(bool); ok {
					if secure {
						fmt.Println("  Security: ✓ Secure")
					} else {
						fmt.Println("  Security: ⚠ Not secure")
						if warning, ok := info["warning"].(string); ok {
							fmt.Printf("  Warning: %s\n", warning)
						}
					}
				}
				if keyringType, ok := info["keyring_type"].(string); ok {
					fmt.Printf("  Type: %s\n", keyringType)
				}
				if location, ok := info["location"].(string); ok {
					fmt.Printf("  Location: %s\n", location)
				}
			}
			return nil
		}

		key := args[0]
		switch key {
		case configKeyAPIKey, "api_key":
			if secureCfg.APIKey != "" {
				fmt.Println(utils.MaskAPIKey(secureCfg.APIKey))
			} else {
				fmt.Println("(not set)")
			}
		case configKeyAPIURL, "api_url":
			fmt.Println(secureCfg.APIURL)
		case configKeyDebug:
			fmt.Println(secureCfg.Debug)
		case "storage", "storage-info":
			// Show detailed storage information
			info := secureCfg.GetStorageInfo()
			fmt.Println("API Key Storage Information:")
			for k, v := range info {
				fmt.Printf("  %s: %v\n", k, v)
			}
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		return nil
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete [key]",
	Short: "Delete a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		switch key {
		case configKeyAPIKey, "api_key":
			storage := config.NewSecureStorage()
			if err := storage.DeleteAPIKey(); err != nil {
				return fmt.Errorf("failed to delete API key: %w", err)
			}

			fmt.Println("✓ API key deleted from all storage locations")
			return nil

		default:
			return fmt.Errorf("cannot delete configuration key: %s", key)
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configDeleteCmd)
}

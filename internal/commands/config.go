// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

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

const availableKeysHelp = `
Available keys:
  api-key    API authentication key
  api-url    API endpoint URL
  debug      Enable debug output (true/false)`

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage RepoBird configuration",
	Long: `Manage RepoBird CLI configuration including API keys and endpoints.

Available configuration keys:
  api-key    API authentication key (stored securely)
  api-url    API endpoint URL (default: https://repobird.ai)
  debug      Enable debug output (true/false)

Storage locations:
  Config file: ~/.repobird/config.yaml
  API key:     Secure storage (system keyring or encrypted file)
  Cache:       ~/.config/repobird/cache/

Examples:
  repobird config get                      # Show all configuration
  repobird config set api-key YOUR_KEY     # Set API key
  repobird config set api-url https://...  # Set custom API endpoint
  repobird config delete api-key           # Remove API key`,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long:  `Set a configuration value.` + availableKeysHelp,
	Example: `  repobird config set api-key YOUR_KEY
  repobird config set api-url https://repobird.ai
  repobird config set debug true`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("accepts 2 arg(s), received %d%s", len(args), availableKeysHelp)
		}
		return nil
	},
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
			return fmt.Errorf("unknown configuration key: %s%s", key, availableKeysHelp)
		}

		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long: `Get a configuration value.

Available keys:
  api-key       API authentication key (masked)
  api-url       API endpoint URL
  debug         Debug output setting
  storage-info  API key storage information

If no key is specified, shows all configuration values.`,
	Example: `  repobird config get           # Show all configuration
  repobird config get api-key    # Show masked API key
  repobird config get api-url    # Show API endpoint URL
  repobird config get debug      # Show debug setting
  repobird config get storage-info  # Show API key storage details`,
	Args: cobra.MaximumNArgs(1),
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
			return fmt.Errorf("unknown configuration key: %s%s", key, availableKeysHelp)
		}

		return nil
	},
}

var configDeleteCmd = &cobra.Command{
	Use:   "delete [key]",
	Short: "Delete a configuration value",
	Long: `Delete a configuration value.

Available keys:
  api-key    API authentication key (removes from all storage locations)

Note: Only the API key can be deleted. Other settings can be changed with 'config set'.`,
	Example: `  repobird config delete api-key`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("accepts 1 arg, received %d\n\nOnly 'api-key' can be deleted", len(args))
		}
		return nil
	},
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
			return fmt.Errorf("cannot delete configuration key: %s\n\nOnly 'api-key' can be deleted", key)
		}
	},
}

// InitConfigSubcommands adds subcommands to configCmd
func InitConfigSubcommands() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configDeleteCmd)
}

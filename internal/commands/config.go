package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/config"
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

		cfg, err := config.LoadConfig()
		if err != nil {
			cfg = &config.Config{}
		}

		switch key {
		case configKeyAPIKey, "api_key":
			cfg.APIKey = value
			fmt.Println("API key configured successfully")
		case configKeyAPIURL, "api_url":
			cfg.APIURL = value
			fmt.Printf("API URL set to: %s\n", value)
		case configKeyDebug:
			cfg.Debug = value == "true"
			fmt.Printf("Debug mode: %v\n", cfg.Debug)
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(args) == 0 {
			fmt.Printf("API URL: %s\n", cfg.APIURL)
			if cfg.APIKey != "" {
				fmt.Printf("API Key: %s...%s\n", cfg.APIKey[:4], cfg.APIKey[len(cfg.APIKey)-4:])
			} else {
				fmt.Println("API Key: (not set)")
			}
			fmt.Printf("Debug: %v\n", cfg.Debug)
			return nil
		}

		key := args[0]
		switch key {
		case configKeyAPIKey, "api_key":
			if cfg.APIKey != "" {
				fmt.Printf("%s...%s\n", cfg.APIKey[:4], cfg.APIKey[len(cfg.APIKey)-4:])
			} else {
				fmt.Println("(not set)")
			}
		case configKeyAPIURL, "api_url":
			fmt.Println(cfg.APIURL)
		case configKeyDebug:
			fmt.Println(cfg.Debug)
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

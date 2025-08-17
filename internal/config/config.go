// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
	APIURL string `mapstructure:"api_url"`
	Debug  bool   `mapstructure:"debug"`
}

var (
	defaultConfig = Config{
		APIURL: "https://repobird.ai",
		Debug:  false,
	}
)

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".repobird")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			// Log error but continue - config can still work from current directory
			fmt.Fprintf(os.Stderr, "Warning: failed to create config directory %s: %v\n", configDir, err)
		}
		viper.AddConfigPath(configDir)
	}

	viper.AddConfigPath(".")

	viper.SetEnvPrefix("REPOBIRD")
	viper.AutomaticEnv()

	viper.SetDefault("api_url", defaultConfig.APIURL)
	viper.SetDefault("debug", defaultConfig.Debug)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	if config.APIKey == "" && os.Getenv(EnvAPIKey) != "" {
		config.APIKey = os.Getenv(EnvAPIKey)
	}

	if config.APIURL == "" {
		config.APIURL = defaultConfig.APIURL
	}

	if os.Getenv(EnvAPIURL) != "" {
		config.APIURL = os.Getenv(EnvAPIURL)
	}

	return &config, nil
}

func SaveConfig(config *Config) error {
	viper.Set("api_key", config.APIKey)
	viper.Set("api_url", config.APIURL)
	viper.Set("debug", config.Debug)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".repobird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

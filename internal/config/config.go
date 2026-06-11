// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
	APIURL string `mapstructure:"api_url"`
	Debug  bool   `mapstructure:"debug"`
	Color  string `mapstructure:"color"`
}

var (
	defaultConfig = Config{
		APIURL: "https://api.repobird.ai",
		Debug:  false,
		Color:  "auto",
	}
	configFileOverride string
)

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configFileOverride != "" {
		viper.SetConfigFile(configFileOverride)
	} else {
		configDir := ConfigDir()
		if err := os.MkdirAll(configDir, 0755); err != nil {
			// Log error but continue - config can still work from current directory
			fmt.Fprintf(os.Stderr, "Warning: failed to create config directory %s: %v\n", configDir, err)
		}
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(LegacyConfigDir())
		viper.AddConfigPath(".")
	}

	viper.SetEnvPrefix("REPOBIRD")
	viper.AutomaticEnv()

	viper.SetDefault("api_url", defaultConfig.APIURL)
	viper.SetDefault("debug", defaultConfig.Debug)
	viper.SetDefault("color", defaultConfig.Color)

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

	if os.Getenv(EnvColor) != "" {
		config.Color = os.Getenv(EnvColor)
	}
	config.Color = normalizeColor(config.Color)

	return &config, nil
}

func SaveConfig(config *Config) error {
	viper.Set("api_key", "")
	viper.Set("api_url", config.APIURL)
	viper.Set("debug", config.Debug)
	viper.Set("color", normalizeColor(config.Color))

	configDir := ConfigDir()
	configFile := filepath.Join(configDir, "config.yaml")
	if configFileOverride != "" {
		configFile = configFileOverride
		configDir = filepath.Dir(configFile)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func SetConfigFile(path string) {
	configFileOverride = path
}

func ConfigDir() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "repobird")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "repobird")
	}

	return filepath.Join(homeDir, ".config", "repobird")
}

func LegacyConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".repobird")
	}

	return filepath.Join(homeDir, ".repobird")
}

func normalizeColor(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "always":
		return "always"
	case "never", "off", "false", "disabled", "none":
		return "never"
	default:
		return defaultConfig.Color
	}
}

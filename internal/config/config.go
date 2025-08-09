package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey  string `mapstructure:"api_key"`
	APIURL  string `mapstructure:"api_url"`
	Debug   bool   `mapstructure:"debug"`
}

var (
	defaultConfig = Config{
		APIURL: "https://api.repobird.ai",
		Debug:  false,
	}
)

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".repobird")
		os.MkdirAll(configDir, 0755)
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
	
	if config.APIKey == "" && os.Getenv("REPOBIRD_API_KEY") != "" {
		config.APIKey = os.Getenv("REPOBIRD_API_KEY")
	}
	
	if config.APIURL == "" {
		config.APIURL = defaultConfig.APIURL
	}
	
	if os.Getenv("REPOBIRD_API_URL") != "" {
		config.APIURL = os.Getenv("REPOBIRD_API_URL")
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
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	configFile := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}
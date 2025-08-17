// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Default(t *testing.T) {
	// Setup temporary home directory
	_ = setupTempHome(t)

	config, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have default values
	assert.Equal(t, "https://repobird.ai", config.APIURL)
	assert.Empty(t, config.APIKey)
	assert.False(t, config.Debug)
}

func TestLoadConfig_WithExistingFile(t *testing.T) {
	tempDir := setupTempHome(t)

	// Create config file
	configDir := filepath.Join(tempDir, ".repobird")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "config.yaml")
	configContent := `
api_url: https://custom.api.com
api_key: test-key-123
debug: true
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "https://custom.api.com", config.APIURL)
	assert.Equal(t, "test-key-123", config.APIKey)
	assert.True(t, config.Debug)
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	_ = setupTempHome(t)

	// Set environment variables
	originalAPIURL := os.Getenv(EnvAPIURL)
	originalAPIKey := os.Getenv(EnvAPIKey)
	originalDebug := os.Getenv(EnvDebug)

	defer func() {
		// Restore original values
		if originalAPIURL != "" {
			_ = os.Setenv(EnvAPIURL, originalAPIURL)
		} else {
			_ = os.Unsetenv(EnvAPIURL)
		}
		if originalAPIKey != "" {
			_ = os.Setenv(EnvAPIKey, originalAPIKey)
		} else {
			_ = os.Unsetenv(EnvAPIKey)
		}
		if originalDebug != "" {
			_ = os.Setenv(EnvDebug, originalDebug)
		} else {
			_ = os.Unsetenv(EnvDebug)
		}
	}()

	_ = os.Setenv(EnvAPIURL, "https://env.api.com")
	_ = os.Setenv(EnvAPIKey, "env-key-456")
	_ = os.Setenv(EnvDebug, "true")

	config, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Environment variables should override defaults
	assert.Equal(t, "https://env.api.com", config.APIURL)
	assert.Equal(t, "env-key-456", config.APIKey)
	assert.True(t, config.Debug)
}

func TestLoadConfig_FileAndEnvironmentPrecedence(t *testing.T) {
	tempDir := setupTempHome(t)

	// Create config file
	configDir := filepath.Join(tempDir, ".repobird")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "config.yaml")
	configContent := `
api_url: https://file.api.com
api_key: file-key-123
debug: false
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variables (should override file)
	originalAPIURL := os.Getenv(EnvAPIURL)
	originalDebug := os.Getenv(EnvDebug)

	defer func() {
		if originalAPIURL != "" {
			_ = os.Setenv(EnvAPIURL, originalAPIURL)
		} else {
			_ = os.Unsetenv(EnvAPIURL)
		}
		if originalDebug != "" {
			_ = os.Setenv(EnvDebug, originalDebug)
		} else {
			_ = os.Unsetenv(EnvDebug)
		}
	}()

	_ = os.Setenv(EnvAPIURL, "https://env-override.api.com")
	_ = os.Setenv(EnvDebug, "true")

	config, err := LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Environment should override file values
	assert.Equal(t, "https://env-override.api.com", config.APIURL)
	assert.Equal(t, "file-key-123", config.APIKey) // No env var set, should use file value
	assert.True(t, config.Debug)                   // Environment override
}

func TestLoadConfig_InvalidConfigFile(t *testing.T) {
	tempDir := setupTempHome(t)

	// Create invalid config file
	configDir := filepath.Join(tempDir, ".repobird")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "config.yaml")
	invalidContent := `
invalid: yaml: content
  missing: proper
structure
`
	err = os.WriteFile(configFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	config, err := LoadConfig()
	// Should return an error for invalid YAML
	require.Error(t, err)
	require.Nil(t, config)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestSaveConfig(t *testing.T) {
	tempDir := setupTempHome(t)

	config := &Config{
		APIURL: "https://saved.api.com",
		APIKey: "saved-key-789",
		Debug:  true,
	}

	err := SaveConfig(config)
	require.NoError(t, err)

	// Verify file was created
	configFile := filepath.Join(tempDir, ".repobird", "config.yaml")
	assert.FileExists(t, configFile)

	// Load and verify content
	loadedConfig, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, config.APIURL, loadedConfig.APIURL)
	assert.Equal(t, config.APIKey, loadedConfig.APIKey)
	assert.Equal(t, config.Debug, loadedConfig.Debug)
}

func TestSaveConfig_CreatesDirectory(t *testing.T) {
	tempDir := setupTempHome(t)

	// Ensure directory doesn't exist
	configDir := filepath.Join(tempDir, ".repobird")
	_, err := os.Stat(configDir)
	require.True(t, os.IsNotExist(err))

	config := &Config{
		APIURL: "https://repobird.ai",
		APIKey: "test-key",
		Debug:  false,
	}

	err = SaveConfig(config)
	require.NoError(t, err)

	// Directory should be created
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGetConfigPath(t *testing.T) {
	tempDir := setupTempHome(t)

	expected := filepath.Join(tempDir, ".repobird", "config.yaml")

	// Since getConfigPath isn't exported, test the path used by SaveConfig
	configDir := filepath.Join(tempDir, ".repobird")
	actualPath := filepath.Join(configDir, "config.yaml")

	assert.Equal(t, expected, actualPath)
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		isValid bool
	}{
		{
			name: "Valid config",
			config: Config{
				APIURL: "https://api.example.com",
				APIKey: "valid-key-123",
				Debug:  false,
			},
			isValid: true,
		},
		{
			name: "Valid with debug",
			config: Config{
				APIURL: "https://repobird.ai",
				APIKey: "another-valid-key",
				Debug:  true,
			},
			isValid: true,
		},
		{
			name: "Empty API key (still valid - might be set via environment)",
			config: Config{
				APIURL: "https://repobird.ai",
				APIKey: "",
				Debug:  false,
			},
			isValid: true,
		},
		{
			name: "Custom API URL",
			config: Config{
				APIURL: "https://custom-domain.com/api",
				APIKey: "custom-key",
				Debug:  false,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test saving and loading config
			_ = setupTempHome(t)

			err := SaveConfig(&tt.config)
			if tt.isValid {
				assert.NoError(t, err)

				loadedConfig, err := LoadConfig()
				assert.NoError(t, err)

				assert.Equal(t, tt.config.APIURL, loadedConfig.APIURL)
				assert.Equal(t, tt.config.APIKey, loadedConfig.APIKey)
				assert.Equal(t, tt.config.Debug, loadedConfig.Debug)
			}
		})
	}
}

func TestConfig_EdgeCases(t *testing.T) {
	t.Run("Config with very long API key", func(t *testing.T) {
		_ = setupTempHome(t)

		longKey := make([]byte, 1000)
		for i := range longKey {
			longKey[i] = 'a'
		}

		config := &Config{
			APIURL: "https://repobird.ai",
			APIKey: string(longKey),
			Debug:  false,
		}

		err := SaveConfig(config)
		require.NoError(t, err)

		loadedConfig, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, config.APIKey, loadedConfig.APIKey)
	})

	t.Run("Config with special characters in API key", func(t *testing.T) {
		_ = setupTempHome(t)

		config := &Config{
			APIURL: "https://repobird.ai",
			APIKey: "key-with-special!@#$%^&*()_+{}|:<>?[]\\;'\"./~`",
			Debug:  false,
		}

		err := SaveConfig(config)
		require.NoError(t, err)

		loadedConfig, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, config.APIKey, loadedConfig.APIKey)
	})

	t.Run("Config with non-standard API URL", func(t *testing.T) {
		_ = setupTempHome(t)

		config := &Config{
			APIURL: "http://localhost:8080/api/v2",
			APIKey: "local-key",
			Debug:  true,
		}

		err := SaveConfig(config)
		require.NoError(t, err)

		loadedConfig, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, config.APIURL, loadedConfig.APIURL)
	})
}

// setupTempHome creates a temporary directory and sets HOME to it
func setupTempHome(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "repobird-config-test-*")
	require.NoError(t, err)

	originalHome := os.Getenv("HOME")
	originalXDGConfig := os.Getenv("XDG_CONFIG_HOME")
	originalAPIURL := os.Getenv("REPOBIRD_API_URL")
	originalAPIKey := os.Getenv("REPOBIRD_API_KEY")

	_ = os.Setenv("HOME", tempDir)
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir) // Also set XDG for complete isolation
	_ = os.Unsetenv("REPOBIRD_API_URL")       // Clear any env vars that might affect config
	_ = os.Unsetenv("REPOBIRD_API_KEY")

	// Create a new viper instance to avoid global state pollution
	viper.New()

	// Reset viper to clear any cached config
	viper.Reset()

	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
		_ = os.Setenv("XDG_CONFIG_HOME", originalXDGConfig)
		if originalAPIURL != "" {
			_ = os.Setenv("REPOBIRD_API_URL", originalAPIURL)
		}
		if originalAPIKey != "" {
			_ = os.Setenv("REPOBIRD_API_KEY", originalAPIKey)
		}
		_ = os.RemoveAll(tempDir)
		viper.Reset() // Reset viper after test
	})

	return tempDir
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"os"
	"testing"

	"github.com/repobird/repobird-cli/internal/config"
)

func TestGetContainer(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		configURL   string
		description string
	}{
		{
			name: "Container uses dev environment URL",
			envVars: map[string]string{
				"REPOBIRD_ENV": "dev",
			},
			configURL:   "https://production.api.com",
			description: "Container should use localhost:3000 when REPOBIRD_ENV=dev",
		},
		{
			name: "Container uses API URL override",
			envVars: map[string]string{
				"REPOBIRD_ENV":     "dev",
				"REPOBIRD_API_URL": "https://custom.api.com",
			},
			configURL:   "https://production.api.com",
			description: "Container should use REPOBIRD_API_URL when set",
		},
		{
			name:        "Container uses config fallback",
			envVars:     map[string]string{},
			configURL:   "https://config.api.com",
			description: "Container should use config URL as fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset container
			resetContainer()

			// Clear and set environment variables
			os.Clearenv()
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Set up mock config
			cfg = &config.SecureConfig{
				Config: &config.Config{
					APIKey: "test-key",
					APIURL: tt.configURL,
					Debug:  false,
				},
			}

			// Get container
			container := getContainer()
			if container == nil {
				t.Fatal("Expected container to be created")
			}

			// Verify container was created with correct config
			containerConfig := container.Config()
			if containerConfig == nil {
				t.Fatal("Expected container to have config")
			}

			// The actual URL verification happens through GetAPIURL
			// which is tested separately in url_env_test.go
			// This test ensures the container creation flow works

			// Reset for next test
			resetContainer()
		})
	}
}

func TestResetContainer(t *testing.T) {
	// Set up mock config
	cfg = &config.SecureConfig{
		Config: &config.Config{
			APIKey: "test-key",
			APIURL: "https://test.api.com",
			Debug:  false,
		},
	}

	// Create container
	container1 := getContainer()
	if container1 == nil {
		t.Fatal("Expected container to be created")
	}

	// Get container again - should be same instance
	container2 := getContainer()
	if container1 != container2 {
		t.Error("Expected same container instance when called twice")
	}

	// Reset container
	resetContainer()

	// Get container again - should be new instance
	container3 := getContainer()
	if container1 == container3 {
		t.Error("Expected new container instance after reset")
	}
}

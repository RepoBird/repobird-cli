// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"os"
	"testing"
)

func TestGetAPIURL(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		configFallback string
		expectedURL    string
		description    string
	}{
		// Basic scenarios
		{
			name:        "Default production URL",
			envVars:     map[string]string{},
			expectedURL: "https://repobird.ai",
			description: "Should return production URL when no env vars set",
		},
		{
			name: "Dev environment lowercase",
			envVars: map[string]string{
				"REPOBIRD_ENV": "dev",
			},
			expectedURL: "http://localhost:3000",
			description: "Should return localhost:3000 when REPOBIRD_ENV=dev",
		},
		{
			name: "Dev environment uppercase",
			envVars: map[string]string{
				"REPOBIRD_ENV": "DEV",
			},
			expectedURL: "http://localhost:3000",
			description: "Should handle uppercase DEV environment",
		},
		{
			name: "Development environment full word",
			envVars: map[string]string{
				"REPOBIRD_ENV": "development",
			},
			expectedURL: "http://localhost:3000",
			description: "Should return localhost:3000 when REPOBIRD_ENV=development",
		},
		{
			name: "Development environment mixed case",
			envVars: map[string]string{
				"REPOBIRD_ENV": "Development",
			},
			expectedURL: "http://localhost:3000",
			description: "Should handle mixed case Development",
		},
		
		// Priority testing
		{
			name: "API URL override takes precedence over dev env",
			envVars: map[string]string{
				"REPOBIRD_ENV":     "dev",
				"REPOBIRD_API_URL": "https://custom.api.com",
			},
			expectedURL: "https://custom.api.com",
			description: "REPOBIRD_API_URL should override REPOBIRD_ENV",
		},
		{
			name: "API URL without dev env",
			envVars: map[string]string{
				"REPOBIRD_API_URL": "https://staging.repobird.ai",
			},
			expectedURL: "https://staging.repobird.ai",
			description: "Should use REPOBIRD_API_URL when set",
		},
		{
			name: "API URL overrides config fallback",
			envVars: map[string]string{
				"REPOBIRD_API_URL": "https://api-override.com",
			},
			configFallback: "https://config-fallback.com",
			expectedURL:    "https://api-override.com",
			description:    "REPOBIRD_API_URL should override config fallback",
		},
		
		// Localhost scenarios
		{
			name: "Localhost with custom port",
			envVars: map[string]string{
				"REPOBIRD_API_URL": "http://localhost:8080",
			},
			expectedURL: "http://localhost:8080",
			description: "Should allow custom localhost ports via API_URL",
		},
		{
			name: "127.0.0.1 address",
			envVars: map[string]string{
				"REPOBIRD_API_URL": "http://127.0.0.1:3000",
			},
			expectedURL: "http://127.0.0.1:3000",
			description: "Should support 127.0.0.1 addresses",
		},
		
		// Empty and edge cases
		{
			name: "Empty API URL with dev env",
			envVars: map[string]string{
				"REPOBIRD_ENV":     "dev",
				"REPOBIRD_API_URL": "",
			},
			expectedURL: "http://localhost:3000",
			description: "Empty REPOBIRD_API_URL should not override dev env",
		},
		{
			name: "Production environment explicit",
			envVars: map[string]string{
				"REPOBIRD_ENV": "production",
			},
			expectedURL: "https://repobird.ai",
			description: "Production env should use default production URL",
		},
		{
			name: "Staging environment",
			envVars: map[string]string{
				"REPOBIRD_ENV": "staging",
			},
			expectedURL: "https://repobird.ai",
			description: "Unknown environments should default to production",
		},
		
		// Config fallback scenarios
		{
			name:           "Config fallback used when no env vars",
			envVars:        map[string]string{},
			configFallback: "https://custom-config.api.com",
			expectedURL:    "https://custom-config.api.com",
			description:    "Config fallback should be used when no env vars set",
		},
		{
			name: "Dev env overrides config fallback",
			envVars: map[string]string{
				"REPOBIRD_ENV": "dev",
			},
			configFallback: "https://custom-config.api.com",
			expectedURL:    "http://localhost:3000",
			description:    "Dev environment should override config fallback",
		},
		{
			name:           "Empty config fallback ignored",
			envVars:        map[string]string{},
			configFallback: "",
			expectedURL:    "https://repobird.ai",
			description:    "Empty config fallback should be ignored",
		},
		{
			name: "All three sources with priority",
			envVars: map[string]string{
				"REPOBIRD_ENV":     "dev",
				"REPOBIRD_API_URL": "https://api-url-override.com",
			},
			configFallback: "https://config-fallback.com",
			expectedURL:    "https://api-url-override.com",
			description:    "Should follow priority: API_URL > ENV > config > default",
		},
		
		// HTTPS and HTTP handling
		{
			name: "HTTP URL in dev",
			envVars: map[string]string{
				"REPOBIRD_API_URL": "http://dev.api.com",
			},
			expectedURL: "http://dev.api.com",
			description: "Should preserve HTTP protocol",
		},
		{
			name: "HTTPS URL in dev",
			envVars: map[string]string{
				"REPOBIRD_ENV":     "dev",
				"REPOBIRD_API_URL": "https://secure.dev.api.com",
			},
			expectedURL: "https://secure.dev.api.com",
			description: "Should preserve HTTPS protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv("REPOBIRD_ENV")
			os.Unsetenv("REPOBIRD_API_URL")

			// Set test environment variables
			for key, value := range tt.envVars {
				if value != "" {
					t.Setenv(key, value)
				}
			}

			// Test the function
			var got string
			if tt.configFallback != "" {
				got = GetAPIURL(tt.configFallback)
			} else {
				got = GetAPIURL()
			}
			if got != tt.expectedURL {
				t.Errorf("GetAPIURL() = %v, want %v\nDescription: %s", got, tt.expectedURL, tt.description)
			}
		})
	}
}

func TestGetRepoBirdBaseURLEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		expectedURL string
	}{
		{
			name:        "Default production",
			envVar:      "",
			expectedURL: "https://repobird.ai",
		},
		{
			name:        "Dev environment",
			envVar:      "dev",
			expectedURL: "http://localhost:3000",
		},
		{
			name:        "Development environment",
			envVar:      "development",
			expectedURL: "http://localhost:3000",
		},
		{
			name:        "Production explicit",
			envVar:      "production",
			expectedURL: "https://repobird.ai",
		},
		{
			name:        "Unknown environment",
			envVar:      "staging",
			expectedURL: "https://repobird.ai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set environment variable
			os.Unsetenv("REPOBIRD_ENV")
			if tt.envVar != "" {
				t.Setenv("REPOBIRD_ENV", tt.envVar)
			}

			// Test the function
			got := getRepoBirdBaseURL()
			if got != tt.expectedURL {
				t.Errorf("getRepoBirdBaseURL() = %v, want %v", got, tt.expectedURL)
			}
		})
	}
}
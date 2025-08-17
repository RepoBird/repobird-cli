// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"strings"
)

// URLs provides centralized URL management for RepoBird
type URLs struct {
	BaseURL      string
	DashboardURL string
	PricingURL   string
	ReposURL     string
	SettingsURL  string
	APIKeysURL   string
}

// GetURLs returns the appropriate URLs based on the current API configuration
func GetURLs() *URLs {
	baseURL := getBaseURL()

	return &URLs{
		BaseURL:      baseURL,
		DashboardURL: baseURL + "/dashboard",
		PricingURL:   baseURL + "/pricing",
		ReposURL:     baseURL + "/repos",
		SettingsURL:  baseURL + "/settings/api",
		APIKeysURL:   baseURL + "/dashboard/user-profile/api-keys",
	}
}

// getBaseURL determines the base URL from the API URL
func getBaseURL() string {
	// Check if dev environment
	if os.Getenv("REPOBIRD_ENV") == "dev" {
		return "http://localhost:3000"
	}

	// Check environment variable for API URL
	apiURL := os.Getenv("REPOBIRD_API_URL")
	if apiURL == "" {
		apiURL = "https://repobird.ai"
	}

	// Handle localhost and development environments
	if strings.Contains(apiURL, "localhost") || strings.Contains(apiURL, "127.0.0.1") {
		// For local development, use localhost:3000 for frontend
		if strings.Contains(apiURL, ":8080") {
			return "http://localhost:3000"
		}
		return apiURL
	}

	if strings.Contains(apiURL, "ngrok") {
		// Ngrok URLs typically don't have a separate frontend
		return apiURL
	}

	// For production and staging, use the standard URL
	return "https://repobird.ai"
}

// GetPricingURL is a convenience function for getting the pricing URL
func GetPricingURL() string {
	return GetURLs().PricingURL
}

// GetDashboardURL is a convenience function for getting the dashboard URL
func GetDashboardURL() string {
	return GetURLs().DashboardURL
}

// GetAPIKeysURL is a convenience function for getting the API keys URL
func GetAPIKeysURL() string {
	return GetURLs().APIKeysURL
}

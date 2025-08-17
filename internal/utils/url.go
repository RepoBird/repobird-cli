// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// URL pattern matching - covers HTTP/HTTPS URLs
var urlPattern = regexp.MustCompile(`https?://[^\s]+`)

// IsURL checks if a string contains a valid URL
func IsURL(text string) bool {
	if text == "" {
		return false
	}

	// Check if the entire string is a URL
	if _, err := url.Parse(text); err == nil && (strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")) {
		return true
	}

	// Check if the text contains a URL
	return urlPattern.MatchString(text)
}

// ExtractURL extracts the first URL from a string
func ExtractURL(text string) string {
	if text == "" {
		return ""
	}

	// If the entire string is a URL, return it
	if _, err := url.Parse(text); err == nil && (strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")) {
		return strings.TrimSpace(text)
	}

	// Extract the first URL from the text
	matches := urlPattern.FindString(text)
	if matches != "" {
		return strings.TrimSpace(matches)
	}

	return ""
}

// OpenURL opens a URL in the default browser
func OpenURL(urlStr string) error {
	if urlStr == "" {
		return nil
	}

	// Extract URL if the string contains other text
	cleanURL := ExtractURL(urlStr)
	if cleanURL == "" {
		return nil
	}

	return openURLSilent(cleanURL)
}

// openURLSilent opens a URL while suppressing stderr to prevent GTK theme warnings
func openURLSilent(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux, freebsd, openbsd, netbsd, etc.
		cmd = exec.Command("xdg-open", url)
	}

	// Suppress stderr to prevent GTK theme warnings from cluttering the terminal
	cmd.Stderr = nil
	// Also suppress stdout to keep it clean
	cmd.Stdout = nil

	return cmd.Run()
}

// ContainsURL checks if a field label typically contains URLs
func ContainsURL(fieldLabel string) bool {
	labelLower := strings.ToLower(fieldLabel)
	return strings.Contains(labelLower, "url") ||
		strings.Contains(labelLower, "link") ||
		strings.Contains(labelLower, "pr") ||
		strings.Contains(labelLower, "pull request")
}

// getRepoBirdBaseURL returns the base URL for RepoBird frontend based on environment
func getRepoBirdBaseURL() string {
	env := os.Getenv("REPOBIRD_ENV")
	switch strings.ToLower(env) {
	case "dev", "development":
		return "http://localhost:3000"
	default: // production
		return "https://repobird.ai"
	}
}

// GenerateRepoBirdURL generates a RepoBird URL for a given run ID
func GenerateRepoBirdURL(runID string) string {
	if runID == "" || runID == "null" {
		return ""
	}
	baseURL := getRepoBirdBaseURL()
	return baseURL + "/repos/issue-runs/" + runID
}

// GetAPIURL returns the appropriate API URL based on environment
// Priority: REPOBIRD_API_URL > REPOBIRD_ENV=dev > configFallback > default
func GetAPIURL(configFallback ...string) string {
	// Debug output if needed
	if os.Getenv("REPOBIRD_DEBUG_API_URL") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG GetAPIURL] REPOBIRD_ENV=%q, REPOBIRD_API_URL=%q, fallback=%v\n",
			os.Getenv("REPOBIRD_ENV"), os.Getenv("REPOBIRD_API_URL"), configFallback)
	}

	// Check REPOBIRD_API_URL first - this always takes precedence
	if apiURL := os.Getenv("REPOBIRD_API_URL"); apiURL != "" {
		if os.Getenv("REPOBIRD_DEBUG_API_URL") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG GetAPIURL] Using REPOBIRD_API_URL: %s\n", apiURL)
		}
		return apiURL
	}

	// Check REPOBIRD_ENV for dev mode
	env := os.Getenv("REPOBIRD_ENV")
	if strings.ToLower(env) == "dev" || strings.ToLower(env) == "development" {
		if os.Getenv("REPOBIRD_DEBUG_API_URL") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG GetAPIURL] Using dev mode: http://localhost:3000\n")
		}
		return "http://localhost:3000"
	}

	// Use config fallback if provided
	if len(configFallback) > 0 && configFallback[0] != "" {
		if os.Getenv("REPOBIRD_DEBUG_API_URL") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG GetAPIURL] Using config fallback: %s\n", configFallback[0])
		}
		return configFallback[0]
	}

	// Default to production
	if os.Getenv("REPOBIRD_DEBUG_API_URL") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG GetAPIURL] Using default: https://repobird.ai\n")
	}
	return "https://repobird.ai"
}

// IsNonEmptyNumber checks if a string contains a non-empty number (digits only)
func IsNonEmptyNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

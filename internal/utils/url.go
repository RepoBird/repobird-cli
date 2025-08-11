package utils

import (
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

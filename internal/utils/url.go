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

	return browser.OpenURL(cleanURL)
}

// ContainsURL checks if a field label typically contains URLs
func ContainsURL(fieldLabel string) bool {
	labelLower := strings.ToLower(fieldLabel)
	return strings.Contains(labelLower, "url") ||
		strings.Contains(labelLower, "link") ||
		strings.Contains(labelLower, "pr") ||
		strings.Contains(labelLower, "pull request")
}

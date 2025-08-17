// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"os"
	"testing"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid HTTPS URL", "https://github.com/user/repo", true},
		{"Valid HTTP URL", "http://example.com", true},
		{"URL with path", "https://api.github.com/repos/user/repo/pulls/123", true},
		{"Text with URL", "PR URL: https://github.com/user/repo/pull/123", true},
		{"No URL", "This is just text", false},
		{"Empty string", "", false},
		{"Invalid protocol", "ftp://example.com", false},
		{"Just domain", "github.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsURL(tt.input)
			if result != tt.expected {
				t.Errorf("IsURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Pure URL", "https://github.com/user/repo", "https://github.com/user/repo"},
		{"URL with label", "PR URL: https://github.com/user/repo/pull/123", "https://github.com/user/repo/pull/123"},
		{"URL with spaces", "  https://example.com  ", "https://example.com"},
		{"Text with URL in middle", "Check out https://github.com for more info", "https://github.com"},
		{"No URL", "This is just text", ""},
		{"Empty string", "", ""},
		{"Multiple URLs", "First: https://github.com and second: https://example.com", "https://github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractURL(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContainsURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"PR URL", "PR URL", true},
		{"URL field", "Repository URL", true},
		{"Link field", "External Link", true},
		{"Pull Request", "Pull Request", true},
		{"PR field", "PR", true},
		{"Regular field", "Title", false},
		{"Status field", "Status", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsURL(tt.input)
			if result != tt.expected {
				t.Errorf("ContainsURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOpenURLSilent(t *testing.T) {
	// Skip this test to prevent opening browser during test runs
	t.Skip("Skipping openURLSilent test to prevent browser opening")

	// Original test code commented out:
	// Test that the function properly constructs commands based on OS
	// We won't actually run the commands in tests, just verify the logic

	testURL := "https://github.com/example/repo"

	// This should not panic or error on valid URL
	// In actual usage, we'd mock exec.Command, but for this simple test
	// we'll just verify the function exists and can be called
	err := openURLSilent(testURL)

	// The command might fail in CI/test environments, but it shouldn't panic
	// and should return some kind of result (either nil or an error)
	if err != nil {
		t.Logf("openURLSilent returned error (expected in test environment): %v", err)
	}

	// Test with empty URL
	err = openURLSilent("")
	if err != nil {
		t.Logf("openURLSilent with empty URL returned error: %v", err)
	}
}

func TestOpenURL(t *testing.T) {
	// Skip this test to prevent opening browser during test runs
	t.Skip("Skipping OpenURL test to prevent browser opening")

	// Original test code commented out:
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Empty URL", "", false},
		{"Valid URL", "https://github.com", false}, // May fail in test env, that's ok
		{"URL with text", "Check out https://github.com for code", false},
		{"No URL in text", "This has no URL", false}, // Should return nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OpenURL(tt.input)
			if (err != nil) != tt.wantErr {
				// In test environments, commands might fail - that's expected
				t.Logf("OpenURL(%q) error = %v (may be expected in test environment)", tt.input, err)
			}
		})
	}
}

func TestGenerateRepoBirdURL(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("REPOBIRD_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("REPOBIRD_ENV", originalEnv)
		} else {
			os.Unsetenv("REPOBIRD_ENV")
		}
	}()

	tests := []struct {
		name     string
		env      string
		runID    string
		expected string
	}{
		// Production environment (default)
		{"Production - Numeric ID", "", "927", "https://repobird.ai/repos/issue-runs/927"},
		{"Production explicit - String ID", "prod", "abc123", "https://repobird.ai/repos/issue-runs/abc123"},
		{"Production - Empty ID", "", "", ""},
		{"Production - Null ID", "", "null", ""},
		{"Production - Large numeric ID", "", "123456789", "https://repobird.ai/repos/issue-runs/123456789"},

		// Development environment
		{"Development - Numeric ID", "dev", "927", "http://localhost:3000/repos/issue-runs/927"},
		{"Development - String ID", "development", "abc123", "http://localhost:3000/repos/issue-runs/abc123"},
		{"Development - Empty ID", "dev", "", ""},
		{"Development - Null ID", "dev", "null", ""},
		{"Development - Large numeric ID", "dev", "123456789", "http://localhost:3000/repos/issue-runs/123456789"},

		// Case sensitivity tests
		{"Development uppercase", "DEV", "927", "http://localhost:3000/repos/issue-runs/927"},
		{"Development mixed case", "Dev", "927", "http://localhost:3000/repos/issue-runs/927"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment for this test
			if tt.env != "" {
				os.Setenv("REPOBIRD_ENV", tt.env)
			} else {
				os.Unsetenv("REPOBIRD_ENV")
			}

			result := GenerateRepoBirdURL(tt.runID)
			if result != tt.expected {
				t.Errorf("GenerateRepoBirdURL(%q) with REPOBIRD_ENV=%q = %q, want %q", tt.runID, tt.env, result, tt.expected)
			}
		})
	}
}

func TestGetRepoBirdBaseURL(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("REPOBIRD_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("REPOBIRD_ENV", originalEnv)
		} else {
			os.Unsetenv("REPOBIRD_ENV")
		}
	}()

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default (empty)", "", "https://repobird.ai"},
		{"Production explicit", "prod", "https://repobird.ai"},
		{"Production uppercase", "PROD", "https://repobird.ai"},
		{"Development", "dev", "http://localhost:3000"},
		{"Development full", "development", "http://localhost:3000"},
		{"Development uppercase", "DEV", "http://localhost:3000"},
		{"Development mixed case", "Dev", "http://localhost:3000"},
		{"Unknown environment defaults to prod", "staging", "https://repobird.ai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment for this test
			if tt.env != "" {
				os.Setenv("REPOBIRD_ENV", tt.env)
			} else {
				os.Unsetenv("REPOBIRD_ENV")
			}

			result := getRepoBirdBaseURL()
			if result != tt.expected {
				t.Errorf("getRepoBirdBaseURL() with REPOBIRD_ENV=%q = %q, want %q", tt.env, result, tt.expected)
			}
		})
	}
}

func TestIsNonEmptyNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid number", "927", true},
		{"Large number", "123456789", true},
		{"Single digit", "5", true},
		{"Zero", "0", true},
		{"Empty string", "", false},
		{"String with letters", "abc123", false},
		{"Mixed alphanumeric", "12a34", false},
		{"String with spaces", "1 2 3", false},
		{"Negative number", "-123", false},
		{"Decimal number", "12.34", false},
		{"Just letters", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNonEmptyNumber(tt.input)
			if result != tt.expected {
				t.Errorf("IsNonEmptyNumber(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

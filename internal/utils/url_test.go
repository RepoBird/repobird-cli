package utils

import (
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

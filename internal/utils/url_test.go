// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"
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

// mockCommandExecutor captures command execution details for testing
type mockCommandExecutor struct {
	commands []struct {
		name string
		args []string
	}
	returnError error
}

func (m *mockCommandExecutor) Run(ctx context.Context, name string, args ...string) error {
	m.commands = append(m.commands, struct {
		name string
		args []string
	}{name: name, args: args})
	return m.returnError
}

func TestOpenURLSilent(t *testing.T) {
	// Save original executor and restore after test
	originalExecutor := cmdExecutor
	defer func() { cmdExecutor = originalExecutor }()

	testURL := "https://github.com/example/repo"

	tests := []struct {
		name        string
		url         string
		os          string
		wantCmd     string
		wantArgs    []string
		returnError error
		wantError   bool
	}{
		{
			name:     "macOS",
			url:      testURL,
			os:       "darwin",
			wantCmd:  "open",
			wantArgs: []string{testURL},
		},
		{
			name:     "Windows",
			url:      testURL,
			os:       "windows",
			wantCmd:  "rundll32",
			wantArgs: []string{"url.dll,FileProtocolHandler", testURL},
		},
		{
			name:     "Linux",
			url:      testURL,
			os:       "linux",
			wantCmd:  "xdg-open",
			wantArgs: []string{testURL},
		},
		{
			name:        "Command fails",
			url:         testURL,
			os:          runtime.GOOS,
			returnError: errors.New("command failed"),
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock executor
			mock := &mockCommandExecutor{returnError: tt.returnError}
			cmdExecutor = mock

			// Temporarily override runtime.GOOS if needed
			if tt.os != "" && tt.os != runtime.GOOS {
				// For this test, we'll simulate the behavior by checking
				// what command would be used on different platforms
				// Note: We can't actually change runtime.GOOS, so we test
				// with the current OS and verify the mock captures the right command
				if runtime.GOOS != tt.os {
					t.Skipf("Skipping %s test on %s", tt.os, runtime.GOOS)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := openURLSilent(ctx, tt.url)
			if (err != nil) != tt.wantError {
				t.Errorf("openURLSilent() error = %v, wantError %v", err, tt.wantError)
			}

			// Verify the correct command was called (only if we're on the right OS)
			if tt.wantCmd != "" && !tt.wantError && runtime.GOOS == tt.os {
				if len(mock.commands) != 1 {
					t.Fatalf("Expected 1 command call, got %d", len(mock.commands))
				}
				if mock.commands[0].name != tt.wantCmd {
					t.Errorf("Expected command %q, got %q", tt.wantCmd, mock.commands[0].name)
				}
				if len(mock.commands[0].args) != len(tt.wantArgs) {
					t.Errorf("Expected %d args, got %d", len(tt.wantArgs), len(mock.commands[0].args))
				} else {
					for i, arg := range tt.wantArgs {
						if mock.commands[0].args[i] != arg {
							t.Errorf("Expected arg[%d] = %q, got %q", i, arg, mock.commands[0].args[i])
						}
					}
				}
			}
		})
	}

	// Test with empty URL should not call any command
	t.Run("Empty URL", func(t *testing.T) {
		mock := &mockCommandExecutor{}
		cmdExecutor = mock

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := openURLSilent(ctx, "")
		if err == nil {
			// The function should still be called but with empty URL
			if len(mock.commands) != 1 {
				t.Fatalf("Expected 1 command call with empty URL, got %d", len(mock.commands))
			}
		}
	})
}

func TestOpenURL(t *testing.T) {
	// Save original executor and restore after test
	originalExecutor := cmdExecutor
	defer func() { cmdExecutor = originalExecutor }()

	tests := []struct {
		name        string
		input       string
		wantCalled  bool
		wantURL     string
		returnError error
		wantErr     bool
	}{
		{
			name:       "Empty URL",
			input:      "",
			wantCalled: false,
			wantErr:    false,
		},
		{
			name:       "Valid URL",
			input:      "https://github.com",
			wantCalled: true,
			wantURL:    "https://github.com",
			wantErr:    false,
		},
		{
			name:       "URL with text",
			input:      "Check out https://github.com for code",
			wantCalled: true,
			wantURL:    "https://github.com",
			wantErr:    false,
		},
		{
			name:       "No URL in text",
			input:      "This has no URL",
			wantCalled: false,
			wantErr:    false,
		},
		{
			name:        "Command execution error",
			input:       "https://example.com",
			wantCalled:  true,
			wantURL:     "https://example.com",
			returnError: errors.New("failed to open"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock executor
			mock := &mockCommandExecutor{returnError: tt.returnError}
			cmdExecutor = mock

			err := OpenURLWithTimeout(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			// Check if the command was called as expected
			if tt.wantCalled {
				if len(mock.commands) != 1 {
					t.Fatalf("Expected command to be called once, got %d calls", len(mock.commands))
				}
				// Verify the URL passed to the command
				if len(mock.commands[0].args) > 0 {
					// The URL is typically the last argument
					actualURL := mock.commands[0].args[len(mock.commands[0].args)-1]
					if actualURL != tt.wantURL {
						t.Errorf("Expected URL %q, got %q", tt.wantURL, actualURL)
					}
				}
			} else {
				if len(mock.commands) != 0 {
					t.Errorf("Expected no command calls, got %d", len(mock.commands))
				}
			}
		})
	}
}

func TestGenerateRepoBirdURL(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("REPOBIRD_ENV")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("REPOBIRD_ENV", originalEnv)
		} else {
			_ = os.Unsetenv("REPOBIRD_ENV")
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
				_ = os.Setenv("REPOBIRD_ENV", tt.env)
			} else {
				_ = os.Unsetenv("REPOBIRD_ENV")
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
			_ = os.Setenv("REPOBIRD_ENV", originalEnv)
		} else {
			_ = os.Unsetenv("REPOBIRD_ENV")
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
				_ = os.Setenv("REPOBIRD_ENV", tt.env)
			} else {
				_ = os.Unsetenv("REPOBIRD_ENV")
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

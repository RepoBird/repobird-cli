// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


//go:build integration
// +build integration

package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		contains []string
	}{
		{
			name:     "version command",
			args:     []string{"version"},
			wantExit: 0,
			contains: []string{"Version:", "Git Commit:", "Build Date:"},
		},
		{
			name:     "version with --help",
			args:     []string{"version", "--help"},
			wantExit: 0,
			contains: []string{"Print version information"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunCommand(t, tt.args...)
			AssertExitCode(t, result, tt.wantExit)

			for _, expected := range tt.contains {
				AssertContains(t, result.Stdout, expected)
			}
		})
	}
}

// TestHelpCommand tests the help command
func TestHelpCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "main help",
			args:     []string{"help"},
			contains: []string{"CLI and TUI", "Available Commands:", "run", "status", "config", "login", "verify"},
		},
		{
			name:     "help flag",
			args:     []string{"--help"},
			contains: []string{"CLI and TUI", "Available Commands:"},
		},
		{
			name:     "help for run",
			args:     []string{"help", "run"},
			contains: []string{"Create one or more runs from", "JSON", "YAML", "--dry-run"},
		},
		{
			name:     "help for status",
			args:     []string{"help", "status"},
			contains: []string{"Check the status", "--follow", "--json"},
		},
		{
			name:     "help for config",
			args:     []string{"help", "config"},
			contains: []string{"Manage RepoBird CLI configuration", "set", "get", "delete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunCommand(t, tt.args...)
			AssertSuccess(t, result)

			for _, expected := range tt.contains {
				AssertContains(t, result.Stdout, expected)
			}
		})
	}
}

// TestConfigCommands tests configuration management
func TestConfigCommands(t *testing.T) {
	// Create isolated environment
	homeDir := SetupTestConfig(t)
	env := map[string]string{
		"HOME":            homeDir,
		"XDG_CONFIG_HOME": filepath.Join(homeDir, ".config"),
	}

	t.Run("set and get API key", func(t *testing.T) {
		// Set API key
		result := RunCommandWithEnv(t, env, "config", "set", "api-key", "MY_TEST_KEY")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "API key configured successfully")

		// Get API key
		result = RunCommandWithEnv(t, env, "config", "get", "api-key")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "MY_T")
	})

	t.Run("show configuration", func(t *testing.T) {
		// The 'list' command doesn't exist, use 'get' to verify the key is set
		result := RunCommandWithEnv(t, env, "config", "get", "api-key")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "MY_T")
	})

	t.Run("delete configuration", func(t *testing.T) {
		t.Skip("Skipping delete test due to keyring access issues in test environment")
		// The delete command requires keyring access which may not be available in CI
		// This would need proper mocking or a test-specific keyring implementation
	})

	t.Run("config path", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "config", "path")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, ".repobird")
	})
}

// TestAuthCommands tests authentication commands
func TestAuthCommands(t *testing.T) {
	env, mockServer := SetupTestEnv(t)
	defer mockServer.Close()

	t.Run("verify with valid key", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "verify")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "valid")
	})

	t.Run("info", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "info")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Email:")
		AssertContains(t, result.Stdout, "Tier:")
	})

	t.Run("verify without API key", func(t *testing.T) {
		delete(env, "REPOBIRD_API_KEY")
		result := RunCommandWithEnv(t, env, "verify")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "API key not configured")
	})
}

// TestRunCommand tests the run command with --dry-run
func TestRunCommand(t *testing.T) {
	env, mockServer := SetupTestEnv(t)
	defer mockServer.Close()

	// Create test task files
	tmpDir := t.TempDir()

	validTask := `{
		"prompt": "Test task",
		"repository": "test/repo",
		"source": "main",
		"target": "feature/test",
		"runType": "run",
		"title": "Test Run"
	}`
	taskFile := CreateTestFile(t, tmpDir, "task.json", validTask)

	invalidTask := `{ invalid json }`
	invalidFile := CreateTestFile(t, tmpDir, "invalid.json", invalidTask)

	yamlTask := `
prompt: Test YAML task
repository: test/repo
source: main
target: feature/test
runType: run
title: Test YAML Run
`
	yamlFile := CreateTestFile(t, tmpDir, "task.yaml", yamlTask)

	t.Run("run with valid JSON and --dry-run", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "run", taskFile, "--dry-run")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Validation successful")
	})

	t.Run("run with valid YAML and --dry-run", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "run", yamlFile, "--dry-run")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Validation successful")
	})

	t.Run("run with invalid JSON", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "run", invalidFile, "--dry-run")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "invalid")
	})

	t.Run("run with non-existent file", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "run", "/nonexistent/file.json", "--dry-run")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "no such file")
	})
}

// TestStatusCommand tests the status command
func TestStatusCommand(t *testing.T) {
	env, mockServer := SetupTestEnv(t)
	defer mockServer.Close()

	t.Run("status list all runs", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "status")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "12345")
		AssertContains(t, result.Stdout, "67890")
		AssertContains(t, result.Stdout, "DONE")
		AssertContains(t, result.Stdout, "RUNNING")
	})

	t.Run("status specific run", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "status", "12345")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "12345")
		AssertContains(t, result.Stdout, "DONE")
		AssertContains(t, result.Stdout, "test/repo")
	})

	t.Run("status non-existent run", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "status", "99999")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "not found")
	})

	t.Run("status with JSON output", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "status", "--json")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, `"id":`)
		AssertContains(t, result.Stdout, `"status":`)
	})
}

// TestBulkCommands tests bulk operations with --dry-run
func TestBulkCommands(t *testing.T) {
	env, mockServer := SetupTestEnv(t)
	defer mockServer.Close()

	// Create bulk config file
	tmpDir := t.TempDir()
	bulkConfig := `{
		"repository": "test/repo",
		"source": "main",
		"runType": "run",
		"runs": [
			{
				"prompt": "Task 1",
				"target": "feature/task1",
				"title": "First task"
			},
			{
				"prompt": "Task 2",
				"target": "feature/task2",
				"title": "Second task"
			}
		]
	}`
	bulkFile := CreateTestFile(t, tmpDir, "bulk.json", bulkConfig)

	t.Run("bulk run with --dry-run", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "bulk", bulkFile, "--dry-run")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Configuration valid")
	})

	t.Run("bulk with invalid file", func(t *testing.T) {
		// Try to run bulk with non-existent file
		result := RunCommandWithEnv(t, env, "bulk", "/nonexistent/bulk.json", "--dry-run")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "failed to read file")
	})
}

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	mockServer := NewMockServer(t)
	defer mockServer.Close()

	homeDir := SetupTestConfig(t)

	t.Run("REPOBIRD_API_KEY override", func(t *testing.T) {
		env := map[string]string{
			"HOME":             homeDir,
			"REPOBIRD_API_URL": mockServer.URL,
			"REPOBIRD_API_KEY": "TEST_KEY",
		}

		result := RunCommandWithEnv(t, env, "verify")
		AssertSuccess(t, result)
	})

	t.Run("REPOBIRD_API_URL override", func(t *testing.T) {
		env := map[string]string{
			"HOME":             homeDir,
			"REPOBIRD_API_URL": "http://localhost:9999", // Non-existent
			"REPOBIRD_API_KEY": "TEST_KEY",
			"REPOBIRD_TIMEOUT": "1s", // Short timeout
		}

		_ = RunCommandWithEnv(t, env, "status")
		// Command may succeed but won't connect to the server
		// Just verify it doesn't crash
	})

	t.Run("REPOBIRD_DEBUG mode", func(t *testing.T) {
		env := map[string]string{
			"HOME":           homeDir,
			"REPOBIRD_DEBUG": "true",
		}

		result := RunCommandWithEnv(t, env, "version")
		AssertSuccess(t, result)
		// Debug mode might add extra output
	})
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	env, mockServer := SetupTestEnv(t)
	defer mockServer.Close()

	t.Run("invalid command", func(t *testing.T) {
		result := RunCommand(t, "invalidcommand")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "unknown command")
	})

	t.Run("missing required arguments", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "run") // Missing file argument
		// When run without args and no stdin, it shows help and exits successfully
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Create one or more runs from")
	})

	t.Run("rate limiting", func(t *testing.T) {
		// Skip rate limiting test as mock server doesn't implement it
		t.Skip("Rate limiting not implemented in mock server")
	})

	t.Run("server error", func(t *testing.T) {
		// The mock server only fails once then resets, so we need to set it immediately before use
		mockServer.SetFailNext(true)
		result := RunCommandWithEnv(t, env, "verify")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "Error")
	})
}

// TestCompletionCommand tests shell completion generation
func TestCompletionCommand(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(fmt.Sprintf("completion for %s", shell), func(t *testing.T) {
			result := RunCommand(t, "completion", shell)
			AssertSuccess(t, result)

			// Check for shell-specific patterns
			switch shell {
			case "bash":
				AssertContains(t, result.Stdout, "complete")
			case "zsh":
				AssertContains(t, result.Stdout, "compdef")
			case "fish":
				AssertContains(t, result.Stdout, "complete -c repobird")
			case "powershell":
				AssertContains(t, result.Stdout, "Register-ArgumentCompleter")
			}
		})
	}
}

// TestTUICommand tests basic TUI launching (can't test interaction)
func TestTUICommand(t *testing.T) {
	env, mockServer := SetupTestEnv(t)
	defer mockServer.Close()

	t.Run("tui help", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "tui", "--help")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Launch the RepoBird TUI")
	})

	// Can't fully test interactive TUI, but can verify it starts
	// A real TUI test would need a PTY and terminal emulation
}

// TestGoldenFiles tests output against golden files
func TestGoldenFiles(t *testing.T) {
	update := GetUpdateFlag()

	t.Run("version output", func(t *testing.T) {
		result := RunCommand(t, "version")
		AssertSuccess(t, result)

		// Normalize version output for golden comparison
		lines := strings.Split(result.Stdout, "\n")
		normalized := ""
		for _, line := range lines {
			if strings.HasPrefix(line, "Version:") {
				normalized += "Version: X.X.X\n"
			} else if strings.HasPrefix(line, "Git Commit:") {
				normalized += "Git Commit: XXXXX\n"
			} else if strings.HasPrefix(line, "Build Date:") {
				normalized += "Build Date: XXXXX\n"
			} else if strings.HasPrefix(line, "Go Version:") {
				normalized += "Go Version: X.X.X\n"
			} else if strings.HasPrefix(line, "OS/Arch:") {
				normalized += "OS/Arch: X/X\n"
			}
		}

		goldenPath := filepath.Join("testdata", "golden", "version.txt")
		CompareGolden(t, normalized, goldenPath, update)
	})

	t.Run("help output", func(t *testing.T) {
		result := RunCommand(t, "help")
		AssertSuccess(t, result)

		// Help output should be stable
		goldenPath := filepath.Join("testdata", "golden", "help.txt")
		CompareGolden(t, result.Stdout, goldenPath, update)
	})
}

// TestPerformance tests command performance
func TestPerformance(t *testing.T) {
	t.Run("version command speed", func(t *testing.T) {
		result := RunCommand(t, "version")
		AssertSuccess(t, result)

		if result.Duration > 1*time.Second {
			t.Errorf("Version command took too long: %v", result.Duration)
		}
	})

	t.Run("help command speed", func(t *testing.T) {
		result := RunCommand(t, "help")
		AssertSuccess(t, result)

		if result.Duration > 1*time.Second {
			t.Errorf("Help command took too long: %v", result.Duration)
		}
	})
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/config"
)

func ensureRunTestConfig() {
	if cfg == nil {
		cfg = &config.SecureConfig{
			Config: &config.Config{},
		}
	}
}

func TestRunCommand_WithFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupConfig   func()
		expectError   bool
		errorContains string
		expectDryRun  bool
	}{
		{
			name: "Valid minimal flags",
			args: []string{"-r", "test/repo", "-p", "Fix the bug", "--dry-run"},
			setupConfig: func() {
				cfg.APIKey = "test-key"
			},
			expectError:  false,
			expectDryRun: true,
		},
		{
			name: "Valid with all optional flags",
			args: []string{
				"--repo", "test/repo",
				"--prompt", "Fix authentication",
				"--source", "main",
				"--target", "fix/auth",
				"--title", "Auth Fix",
				"--run-type", "plan",
				"--context", "Users can't login",
				"--dry-run",
			},
			setupConfig: func() {
				cfg.APIKey = "test-key"
			},
			expectError:  false,
			expectDryRun: true,
		},
		{
			name: "Missing prompt flag",
			args: []string{"-r", "test/repo"},
			setupConfig: func() {
				cfg.APIKey = "test-key"
			},
			expectError:   true,
			errorContains: "missing required flag: --prompt (-p) is required when --repo is specified",
		},
		{
			name: "Missing repo flag",
			args: []string{"-p", "Fix the bug"},
			setupConfig: func() {
				cfg.APIKey = "test-key"
			},
			expectError:   true,
			errorContains: "missing required flag: --repo (-r) is required when --prompt is specified",
		},
		{
			name: "No API key configured",
			args: []string{"-r", "test/repo", "-p", "Fix bug"},
			setupConfig: func() {
				cfg.APIKey = ""
			},
			expectError:   true,
			errorContains: "API key not configured",
		},
		{
			name: "Empty repo value",
			args: []string{"-r", "", "-p", "Fix bug"},
			setupConfig: func() {
				cfg.APIKey = "test-key"
			},
			expectError:   true,
			errorContains: "missing required flag: --repo (-r) is required when --prompt is specified",
		},
		{
			name: "Empty prompt value",
			args: []string{"-r", "test/repo", "-p", ""},
			setupConfig: func() {
				cfg.APIKey = "test-key"
			},
			expectError:   true,
			errorContains: "missing required flag: --prompt (-p) is required when --repo is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureRunTestConfig()
			// Save original config and flags
			originalAPIKey := cfg.APIKey
			defer func() {
				cfg.APIKey = originalAPIKey
				// Reset flags after test
				repo = ""
				prompt = ""
				source = ""
				target = ""
				baseBranch = ""
				outputMode = ""
				outputBranch = ""
				prTargetBranch = ""
				outputBranchPolicy = ""
				title = ""
				runType = ""
				contextFlag = ""
				dryRun = false
				follow = false
			}()

			// Setup config for test
			if tt.setupConfig != nil {
				tt.setupConfig()
			}

			// Execute command directly without creating new command
			// This simulates what happens when the command is run from CLI
			err := runCmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Capture stdout temporarily
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute the command function
			cmdErr := runCommand(runCmd, []string{})

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check error expectations
			if tt.expectError {
				assert.Error(t, cmdErr)
				if tt.errorContains != "" && cmdErr != nil {
					assert.Contains(t, cmdErr.Error(), tt.errorContains,
						"Error should contain expected message")
				}
			} else {
				if tt.expectDryRun {
					// For dry-run, check output contains expected content
					assert.Contains(t, output, "Validation successful")
					assert.Contains(t, output, "Run would be created with")
				}
			}
		})
	}
}

func TestRunCommand_ValidationWithFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectInJSON map[string]string
	}{
		{
			name: "Default run type when not specified",
			args: []string{"-r", "test/repo", "-p", "Fix bug", "--dry-run"},
			expectInJSON: map[string]string{
				"RunType":        "run",
				"RepositoryName": "test/repo",
				"Prompt":         "Fix bug",
			},
		},
		{
			name: "Plan type explicitly set",
			args: []string{"-r", "test/repo", "-p", "Fix bug", "--run-type", "plan", "--dry-run"},
			expectInJSON: map[string]string{
				"RunType":        "plan",
				"RepositoryName": "test/repo",
				"Prompt":         "Fix bug",
			},
		},
		{
			name: "Basic preset selects DeepSeek model",
			args: []string{"-r", "test/repo", "-p", "Fix bug", "--basic", "--dry-run"},
			expectInJSON: map[string]string{
				"RunType":        "basic",
				"RepositoryName": "test/repo",
				"Prompt":         "Fix bug",
				"OpenCodeModel":  "openrouter/deepseek/deepseek-v4-flash",
			},
		},
		{
			name: "Pro preset selects Kimi model",
			args: []string{"-r", "test/repo", "-p", "Fix bug", "--pro", "--dry-run"},
			expectInJSON: map[string]string{
				"RunType":        "pro",
				"RepositoryName": "test/repo",
				"Prompt":         "Fix bug",
				"OpenCodeModel":  "openrouter/moonshotai/kimi-k2.6",
			},
		},
		{
			name: "All optional fields populated",
			args: []string{
				"-r", "owner/repo", "-p", "Task prompt",
				"--source", "develop",
				"--target", "feature/new",
				"--title", "Task Title",
				"--context", "Additional context",
				"--dry-run",
			},
			expectInJSON: map[string]string{
				"RepositoryName": "owner/repo",
				"Prompt":         "Task prompt",
				"SourceBranch":   "develop",
				"TargetBranch":   "feature/new",
				"Title":          "Task Title",
				"Context":        "Additional context",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureRunTestConfig()
			// Setup config with API key
			originalAPIKey := cfg.APIKey
			cfg.APIKey = "test-key"
			defer func() {
				cfg.APIKey = originalAPIKey
				// Reset flags
				repo = ""
				prompt = ""
				source = ""
				target = ""
				baseBranch = ""
				outputMode = ""
				outputBranch = ""
				prTargetBranch = ""
				outputBranchPolicy = ""
				title = ""
				runType = ""
				contextFlag = ""
				basicRun = false
				proRun = false
				dryRun = false
			}()

			// Parse flags
			err := runCmd.ParseFlags(tt.args)
			require.NoError(t, err)

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute command
			cmdErr := runCommand(runCmd, []string{})

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			require.NoError(t, cmdErr)
			assert.Contains(t, output, "Validation successful")

			// Extract JSON from output
			jsonStart := strings.Index(output, "{")
			jsonEnd := strings.LastIndex(output, "}") + 1
			require.True(t, jsonStart >= 0 && jsonEnd > jsonStart, "JSON output not found")

			jsonStr := output[jsonStart:jsonEnd]
			var result map[string]interface{}
			err = json.Unmarshal([]byte(jsonStr), &result)
			require.NoError(t, err, "Failed to parse JSON output")

			// Verify expected fields
			for key, expectedVal := range tt.expectInJSON {
				actualVal, exists := result[key]
				assert.True(t, exists, fmt.Sprintf("Field %s not found in output", key))
				assert.Equal(t, expectedVal, fmt.Sprintf("%v", actualVal),
					fmt.Sprintf("Field %s has unexpected value", key))
			}
		})
	}
}

func TestRunCommand_BranchOnlyFlag(t *testing.T) {
	ensureRunTestConfig()
	originalAPIKey := cfg.APIKey
	cfg.APIKey = "test-key"
	defer func() {
		cfg.APIKey = originalAPIKey
		repo = ""
		prompt = ""
		source = ""
		target = ""
		baseBranch = ""
		outputMode = ""
		outputBranch = ""
		prTargetBranch = ""
		outputBranchPolicy = ""
		title = ""
		runType = ""
		contextFlag = ""
		branchOnly = false
		dryRun = false
	}()

	err := runCmd.ParseFlags([]string{
		"-r", "owner/repo",
		"-p", "Push commits to a branch",
		"--target", "automation/maintenance",
		"--branch-only",
		"--dry-run",
	})
	require.NoError(t, err)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdErr := runCommand(runCmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, cmdErr)
	assert.Contains(t, output, "Validation successful")

	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}") + 1
	require.True(t, jsonStart >= 0 && jsonEnd > jsonStart, "JSON output not found")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output[jsonStart:jsonEnd]), &result)
	require.NoError(t, err)
	assert.Equal(t, true, result["BranchOnly"])
	assert.Equal(t, "branch", result["OutputMode"])
	assert.Equal(t, "automation/maintenance", result["OutputBranch"])
}

func TestRunCommand_BranchOutputFields(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]interface{}
	}{
		{
			name: "PR mode defaults target to base branch",
			args: []string{
				"-r", "owner/repo",
				"-p", "Open a PR",
				"--base-branch", "main",
				"--dry-run",
			},
			expected: map[string]interface{}{
				"BaseBranch":     "main",
				"OutputMode":     "pr",
				"PRTargetBranch": "main",
			},
		},
		{
			name: "branch-only maps legacy target to output branch",
			args: []string{
				"-r", "owner/repo",
				"-p", "Push without PR",
				"--source", "main",
				"--target", "automation/docs",
				"--branch-only",
				"--dry-run",
			},
			expected: map[string]interface{}{
				"BaseBranch":   "main",
				"OutputMode":   "branch",
				"OutputBranch": "automation/docs",
				"BranchOnly":   true,
			},
		},
		{
			name: "explicit reuse policy is preserved",
			args: []string{
				"-r", "owner/repo",
				"-p", "Reuse branch",
				"--base-branch", "main",
				"--output-mode", "branch",
				"--output-branch", "automation/docs",
				"--output-branch-policy", "reuse",
				"--dry-run",
			},
			expected: map[string]interface{}{
				"BaseBranch":         "main",
				"OutputMode":         "branch",
				"OutputBranch":       "automation/docs",
				"OutputBranchPolicy": "reuse",
				"BranchOnly":         true,
			},
		},
		{
			name: "legacy source aliases base branch",
			args: []string{
				"-r", "owner/repo",
				"-p", "Legacy config",
				"--source", "develop",
				"--dry-run",
			},
			expected: map[string]interface{}{
				"BaseBranch":     "develop",
				"SourceBranch":   "develop",
				"OutputMode":     "pr",
				"PRTargetBranch": "develop",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureRunTestConfig()
			originalAPIKey := cfg.APIKey
			cfg.APIKey = "test-key"
			defer func() {
				cfg.APIKey = originalAPIKey
				repo = ""
				prompt = ""
				source = ""
				target = ""
				baseBranch = ""
				outputMode = ""
				outputBranch = ""
				prTargetBranch = ""
				outputBranchPolicy = ""
				title = ""
				runType = ""
				contextFlag = ""
				branchOnly = false
				dryRun = false
			}()

			err := runCmd.ParseFlags(tt.args)
			require.NoError(t, err)

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			cmdErr := runCommand(runCmd, []string{})

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			require.NoError(t, cmdErr)
			jsonStart := strings.Index(output, "{")
			jsonEnd := strings.LastIndex(output, "}") + 1
			require.True(t, jsonStart >= 0 && jsonEnd > jsonStart, "JSON output not found")

			var result map[string]interface{}
			err = json.Unmarshal([]byte(output[jsonStart:jsonEnd]), &result)
			require.NoError(t, err)
			for key, expected := range tt.expected {
				assert.Equal(t, expected, result[key], key)
			}
		})
	}
}

func TestRunPresetCommand_BranchOnlyFlag(t *testing.T) {
	ensureRunTestConfig()
	originalAPIKey := cfg.APIKey
	cfg.APIKey = "test-key"
	defer func() {
		cfg.APIKey = originalAPIKey
		repo = ""
		prompt = ""
		source = ""
		target = ""
		baseBranch = ""
		outputMode = ""
		outputBranch = ""
		prTargetBranch = ""
		outputBranchPolicy = ""
		title = ""
		runType = ""
		contextFlag = ""
		branchOnly = false
		dryRun = false
	}()

	cmd := newRunPresetCommand("basic")
	err := cmd.ParseFlags([]string{
		"-r", "owner/repo",
		"--target", "automation/maintenance",
		"--branch-only",
		"--dry-run",
	})
	require.NoError(t, err)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdErr := runCommandWithPreset(cmd, []string{"Push commits to a branch"}, "basic")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, cmdErr)
	assert.Contains(t, output, "Validation successful")

	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}") + 1
	require.True(t, jsonStart >= 0 && jsonEnd > jsonStart, "JSON output not found")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output[jsonStart:jsonEnd]), &result)
	require.NoError(t, err)
	assert.Equal(t, true, result["BranchOnly"])
}

func TestRunCommand_RejectsConflictingPresetFlags(t *testing.T) {
	ensureRunTestConfig()
	originalAPIKey := cfg.APIKey
	cfg.APIKey = "test-key"
	defer func() {
		cfg.APIKey = originalAPIKey
		repo = ""
		prompt = ""
		source = ""
		target = ""
		title = ""
		runType = ""
		contextFlag = ""
		basicRun = false
		proRun = false
		dryRun = false
	}()

	err := runCmd.ParseFlags([]string{"-r", "test/repo", "-p", "Fix bug", "--basic", "--pro", "--dry-run"})
	require.NoError(t, err)

	cmdErr := runCommand(runCmd, []string{})
	require.Error(t, cmdErr)
	assert.Contains(t, cmdErr.Error(), "--basic and --pro cannot be used together")
}

func TestRunPresetCommand_UsesPromptArgument(t *testing.T) {
	ensureRunTestConfig()
	originalAPIKey := cfg.APIKey
	cfg.APIKey = "test-key"
	defer func() {
		cfg.APIKey = originalAPIKey
		repo = ""
		prompt = ""
		source = ""
		target = ""
		title = ""
		runType = ""
		contextFlag = ""
		basicRun = false
		proRun = false
		dryRun = false
	}()

	cmd := newRunPresetCommand("pro")
	err := cmd.ParseFlags([]string{"-r", "test/repo", "--dry-run"})
	require.NoError(t, err)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmdErr := cmd.RunE(cmd, []string{"Fix the bug"})
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, cmdErr)
	assert.Contains(t, output, "Validation successful")
	assert.Contains(t, output, `"RunType": "pro"`)
	assert.Contains(t, output, `"OpenCodeModel": "openrouter/moonshotai/kimi-k2.6"`)
	assert.Contains(t, output, "Model: Kimi K2.6")
}

func TestRunPresetCommand_AutoDetectsRepository(t *testing.T) {
	ensureRunTestConfig()
	originalAPIKey := cfg.APIKey
	cfg.APIKey = "test-key"
	defer func() {
		cfg.APIKey = originalAPIKey
		repo = ""
		prompt = ""
		source = ""
		target = ""
		title = ""
		runType = ""
		contextFlag = ""
		basicRun = false
		proRun = false
		dryRun = false
		resetContainer()
	}()

	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })
	require.NoError(t, os.Chdir(tempDir))

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/acme/webapp.git").Run())

	cmd := newRunPresetCommand("basic")
	err = cmd.ParseFlags([]string{"--dry-run"})
	require.NoError(t, err)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmdErr := cmd.RunE(cmd, []string{"Fix the bug"})
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, cmdErr)
	assert.Contains(t, output, "Auto-detected repository: acme/webapp")
	assert.Contains(t, output, `"RepositoryName": "acme/webapp"`)
	assert.Contains(t, output, `"RunType": "basic"`)
	assert.Contains(t, output, "Model: DeepSeek V4 Flash")
}

func TestRunCommand_ErrorMessages(t *testing.T) {
	// Test specific error message scenarios
	tests := []struct {
		name            string
		setupFunc       func()
		args            []string
		expectedMessage string
	}{
		{
			name: "No API key error",
			setupFunc: func() {
				cfg.APIKey = ""
			},
			args:            []string{"-r", "repo", "-p", "prompt"},
			expectedMessage: "API key not configured",
		},
		{
			name: "Only repo provided",
			setupFunc: func() {
				cfg.APIKey = "test"
			},
			args:            []string{"--repo", "myrepo"},
			expectedMessage: "missing required flag: --prompt (-p) is required when --repo is specified",
		},
		{
			name: "Only prompt provided",
			setupFunc: func() {
				cfg.APIKey = "test"
			},
			args:            []string{"--prompt", "do something"},
			expectedMessage: "missing required flag: --repo (-r) is required when --prompt is specified",
		},
		{
			name: "Empty string for repo",
			setupFunc: func() {
				cfg.APIKey = "test"
			},
			args:            []string{"-r", "", "-p", "task"},
			expectedMessage: "missing required flag: --repo (-r) is required when --prompt is specified",
		},
		{
			name: "Empty string for prompt",
			setupFunc: func() {
				cfg.APIKey = "test"
			},
			args:            []string{"-r", "repo", "-p", ""},
			expectedMessage: "missing required flag: --prompt (-p) is required when --repo is specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureRunTestConfig()
			// Save original config
			originalAPIKey := cfg.APIKey
			defer func() {
				cfg.APIKey = originalAPIKey
				// Reset flags
				repo = ""
				prompt = ""
				source = ""
				target = ""
				title = ""
				runType = ""
				contextFlag = ""
				dryRun = false
			}()

			// Setup for test
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			// Parse flags
			err := runCmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Execute command
			cmdErr := runCommand(runCmd, []string{})
			require.Error(t, cmdErr)
			assert.Contains(t, cmdErr.Error(), tt.expectedMessage,
				"Error message should be specific and helpful")
		})
	}
}

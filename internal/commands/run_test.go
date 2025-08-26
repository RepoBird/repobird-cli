// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_WithFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupConfig    func()
		expectError    bool
		errorContains  string
		expectDryRun   bool
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
			// Save original config and flags
			originalAPIKey := cfg.APIKey
			defer func() {
				cfg.APIKey = originalAPIKey
				// Reset flags after test
				repo = ""
				prompt = ""
				source = ""
				target = ""
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
		name          string
		args          []string
		expectInJSON  map[string]string
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
				title = ""
				runType = ""
				contextFlag = ""
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
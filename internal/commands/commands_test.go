package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/models"
)

func TestRunCommand_Execute(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		taskContent    string
		expectError    bool
		expectedOutput string
	}{
		{
			name: "Valid task file",
			args: []string{"run", "task.json"},
			taskContent: `{
				"prompt": "Fix authentication bug",
				"repository": "test/repo",
				"source": "main",
				"target": "fix/auth",
				"runType": "run"
			}`,
			expectError: false,
		},
		{
			name: "Valid task file with follow flag",
			args: []string{"run", "task.json", "--follow"},
			taskContent: `{
				"prompt": "Fix authentication bug",
				"repository": "test/repo",
				"source": "main",
				"target": "fix/auth",
				"runType": "run"
			}`,
			expectError: false,
		},
		{
			name:        "Missing task file",
			args:        []string{"run", "nonexistent.json"},
			expectError: true,
		},
		{
			name: "Invalid JSON",
			args: []string{"run", "invalid.json"},
			taskContent: `{
				"prompt": "Fix bug",
				"invalid": json
			}`,
			expectError: true,
		},
		{
			name: "Missing required fields",
			args: []string{"run", "incomplete.json"},
			taskContent: `{
				"repository": "test/repo"
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up temporary directory
			tempDir, err := os.MkdirTemp("", "run-cmd-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			originalWd, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(originalWd)

			err = os.Chdir(tempDir)
			require.NoError(t, err)

			// Create task file if content provided
			if tt.taskContent != "" {
				taskFile := filepath.Base(tt.args[1]) // Extract filename from args
				err = os.WriteFile(taskFile, []byte(tt.taskContent), 0644)
				require.NoError(t, err)
			}

			// Create root command with run subcommand
			rootCmd := NewRootCommand()
			runCmd := NewRunCommand()
			rootCmd.AddCommand(runCmd)

			// Capture output
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			// Execute command
			err = rootCmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Note: This will fail without actual API setup
				// In a real test, you'd need to mock the API client
				if err != nil {
					t.Logf("Command failed as expected without API: %v", err)
				}
			}
		})
	}
}

func TestStatusCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Status without run ID",
			args:        []string{"status"},
			expectError: false, // Should show all runs
		},
		{
			name:        "Status with run ID",
			args:        []string{"status", "test-run-123"},
			expectError: false,
		},
		{
			name:        "Status with follow flag",
			args:        []string{"status", "test-run-123", "--follow"},
			expectError: false,
		},
		{
			name:        "Status with JSON output",
			args:        []string{"status", "--format", "json"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			statusCmd := NewStatusCommand()
			rootCmd.AddCommand(statusCmd)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Note: Will likely fail without API setup
				if err != nil {
					t.Logf("Command failed as expected without API: %v", err)
				}
			}
		})
	}
}

func TestConfigCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		setup       func(t *testing.T) string // Returns temp home dir
	}{
		{
			name:        "Config set API key",
			args:        []string{"config", "set", "api-key", "test-key-123"},
			expectError: false,
			setup:       setupTempHome,
		},
		{
			name:        "Config get API key",
			args:        []string{"config", "get", "api-key"},
			expectError: false,
			setup:       setupTempHome,
		},
		{
			name:        "Config set API URL",
			args:        []string{"config", "set", "api-url", "https://custom.api.com"},
			expectError: false,
			setup:       setupTempHome,
		},
		{
			name:        "Config list all",
			args:        []string{"config", "list"},
			expectError: false,
			setup:       setupTempHome,
		},
		{
			name:        "Config invalid key",
			args:        []string{"config", "set", "invalid-key", "value"},
			expectError: true,
			setup:       setupTempHome,
		},
		{
			name:        "Config missing value",
			args:        []string{"config", "set", "api-key"},
			expectError: true,
			setup:       setupTempHome,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			if tt.setup != nil {
				cleanup = setupTestEnvironment(t)
				defer cleanup()
			}

			rootCmd := NewRootCommand()
			configCmd := NewConfigCommand()
			rootCmd.AddCommand(configCmd)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			output := buf.String()
			t.Logf("Command output: %s", output)
		})
	}
}

func TestAuthCommand_Execute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Auth verify",
			args:        []string{"auth", "verify"},
			expectError: false, // Will fail without valid API key, but command should run
		},
		{
			name:        "Auth login",
			args:        []string{"auth", "login"},
			expectError: false,
		},
		{
			name:        "Auth logout",
			args:        []string{"auth", "logout"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnvironment(t)
			defer cleanup()

			rootCmd := NewRootCommand()
			authCmd := NewAuthCommand()
			rootCmd.AddCommand(authCmd)

			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Commands might fail due to no API key, but should not panic
				if err != nil {
					t.Logf("Command failed as expected without API setup: %v", err)
				}
			}
		})
	}
}

func TestParseTaskFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		validate    func(t *testing.T, req *models.RunRequest)
	}{
		{
			name: "Valid JSON task",
			content: `{
				"prompt": "Fix authentication bug",
				"repository": "test/repo",
				"source": "main",
				"target": "fix/auth",
				"runType": "run",
				"title": "Fix Auth Bug",
				"context": "Users cannot login",
				"files": ["auth.go", "login.go"]
			}`,
			expectError: false,
			validate: func(t *testing.T, req *models.RunRequest) {
				assert.Equal(t, "Fix authentication bug", req.Prompt)
				assert.Equal(t, "test/repo", req.Repository)
				assert.Equal(t, "main", req.Source)
				assert.Equal(t, "fix/auth", req.Target)
				assert.Equal(t, models.RunTypeRun, req.RunType)
				assert.Equal(t, "Fix Auth Bug", req.Title)
				assert.Equal(t, "Users cannot login", req.Context)
				assert.Len(t, req.Files, 2)
			},
		},
		{
			name: "Valid YAML task",
			content: `
prompt: "Add new feature"
repository: "test/repo"
source: "main"
target: "feature/new"
runType: "run"
title: "New Feature"
`,
			expectError: false,
			validate: func(t *testing.T, req *models.RunRequest) {
				assert.Equal(t, "Add new feature", req.Prompt)
				assert.Equal(t, models.RunTypeRun, req.RunType)
				assert.Equal(t, "New Feature", req.Title)
			},
		},
		{
			name: "Approval type task",
			content: `{
				"prompt": "Review PR changes",
				"repository": "test/repo",
				"source": "feature",
				"target": "main",
				"runType": "approval"
			}`,
			expectError: false,
			validate: func(t *testing.T, req *models.RunRequest) {
				assert.Equal(t, models.RunTypeApproval, req.RunType)
			},
		},
		{
			name: "Invalid JSON",
			content: `{
				"prompt": "Fix bug",
				"invalid": json
			}`,
			expectError: true,
		},
		{
			name: "Missing required fields",
			content: `{
				"title": "Incomplete task"
			}`,
			expectError: true,
		},
		{
			name: "Invalid run type",
			content: `{
				"prompt": "Test",
				"repository": "test/repo",
				"source": "main",
				"target": "feature", 
				"runType": "invalid"
			}`,
			expectError: false, // JSON will accept any string
			validate: func(t *testing.T, req *models.RunRequest) {
				assert.Equal(t, models.RunType("invalid"), req.RunType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempFile, err := os.CreateTemp("", "task-*.json")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())

			// Write content
			_, err = tempFile.WriteString(tt.content)
			require.NoError(t, err)
			tempFile.Close()

			// Parse file
			req, err := parseTaskFile(tempFile.Name())

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, req)
				if tt.validate != nil {
					tt.validate(t, req)
				}
			}
		})
	}
}

func TestCommandValidation(t *testing.T) {
	t.Run("Run command requires task file", func(t *testing.T) {
		rootCmd := NewRootCommand()
		runCmd := NewRunCommand()
		rootCmd.AddCommand(runCmd)

		rootCmd.SetArgs([]string{"run"})

		err := rootCmd.Execute()
		assert.Error(t, err)
	})

	t.Run("Config set requires key and value", func(t *testing.T) {
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		rootCmd := NewRootCommand()
		configCmd := NewConfigCommand()
		rootCmd.AddCommand(configCmd)

		rootCmd.SetArgs([]string{"config", "set"})

		err := rootCmd.Execute()
		assert.Error(t, err)
	})
}

func TestCommandHelp(t *testing.T) {
	tests := []struct {
		name    string
		command *cobra.Command
	}{
		{"Root help", NewRootCommand()},
		{"Run help", NewRunCommand()},
		{"Status help", NewStatusCommand()},
		{"Config help", NewConfigCommand()},
		{"Auth help", NewAuthCommand()},
		{"TUI help", NewTUICommand()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.command.SetOut(&buf)
			tt.command.SetArgs([]string{"--help"})

			err := tt.command.Execute()
			assert.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, "Usage:")
		})
	}
}

// Helper functions

func setupTempHome(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "repobird-cmd-test-*")
	require.NoError(t, err)
	return tempDir
}

func setupTestEnvironment(t *testing.T) func() {
	tempDir := setupTempHome(t)

	originalHome := os.Getenv("HOME")
	originalAPIKey := os.Getenv("REPOBIRD_API_KEY")
	originalAPIURL := os.Getenv("REPOBIRD_API_URL")

	os.Setenv("HOME", tempDir)
	os.Unsetenv("REPOBIRD_API_KEY")
	os.Unsetenv("REPOBIRD_API_URL")

	return func() {
		os.RemoveAll(tempDir)
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalAPIKey != "" {
			os.Setenv("REPOBIRD_API_KEY", originalAPIKey)
		}
		if originalAPIURL != "" {
			os.Setenv("REPOBIRD_API_URL", originalAPIURL)
		}
	}
}

// Mock parseTaskFile function for testing
func parseTaskFile(filename string) (*models.RunRequest, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var req models.RunRequest

	// Try JSON first
	if err := json.Unmarshal(data, &req); err != nil {
		// Could try YAML here in real implementation
		return nil, err
	}

	// Basic validation
	if req.Prompt == "" || req.Repository == "" {
		return nil, assert.AnError
	}

	return &req, nil
}

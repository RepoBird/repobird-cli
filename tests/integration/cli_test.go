// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const ansiEscape = "\x1b["

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

	t.Run("config help shows XDG storage", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "config", "--help")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, ".config/repobird")
	})
}

func TestXDGAPIKeyStorageAndLegacyMigration(t *testing.T) {
	homeDir := SetupTestConfig(t)
	xdgConfigHome := filepath.Join(homeDir, ".config")
	legacyDir := filepath.Join(homeDir, ".repobird")
	xdgAPIKeyPath := filepath.Join(xdgConfigHome, "repobird", ".api_key.enc")

	env := map[string]string{
		"HOME":               homeDir,
		"XDG_CONFIG_HOME":    xdgConfigHome,
		"REPOBIRD_TEST_MODE": "true",
		"NO_COLOR":           "true",
	}

	t.Run("config set stores API key under XDG config", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "config", "set", "api-key", "MY_TEST_KEY")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "API key configured successfully")

		if _, err := os.Stat(xdgAPIKeyPath); err != nil {
			t.Fatalf("expected XDG encrypted API key at %s: %v", xdgAPIKeyPath, err)
		}
		if _, err := os.Stat(filepath.Join(legacyDir, ".api_key.enc")); !os.IsNotExist(err) {
			t.Fatalf("expected no legacy encrypted API key, got err=%v", err)
		}
	})

	t.Run("legacy plain text key is read and migrated to XDG", func(t *testing.T) {
		legacyHome := SetupTestConfig(t)
		legacyXDGHome := filepath.Join(legacyHome, ".config")
		legacyConfigDir := filepath.Join(legacyHome, ".repobird")
		if err := os.MkdirAll(legacyConfigDir, 0755); err != nil {
			t.Fatalf("failed to create legacy config dir: %v", err)
		}
		legacyConfigPath := filepath.Join(legacyConfigDir, "config.yaml")
		if err := os.WriteFile(legacyConfigPath, []byte("api_key: LEGACY_TEST_KEY\napi_url: https://repobird.ai\n"), 0600); err != nil {
			t.Fatalf("failed to write legacy config: %v", err)
		}

		legacyEnv := map[string]string{
			"HOME":               legacyHome,
			"XDG_CONFIG_HOME":    legacyXDGHome,
			"REPOBIRD_TEST_MODE": "true",
			"NO_COLOR":           "true",
		}

		result := RunCommandWithEnv(t, legacyEnv, "config", "get", "api-key")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "LEGA")

		migratedPath := filepath.Join(legacyXDGHome, "repobird", ".api_key.enc")
		if _, err := os.Stat(migratedPath); err != nil {
			t.Fatalf("expected migrated XDG encrypted API key at %s: %v", migratedPath, err)
		}

		legacyConfig, err := os.ReadFile(legacyConfigPath)
		if err != nil {
			t.Fatalf("failed to read legacy config after migration: %v", err)
		}
		if strings.Contains(string(legacyConfig), "api_key:") {
			t.Fatalf("expected legacy plain text API key to be removed, got:\n%s", legacyConfig)
		}
	})
}

func TestBinaryColorPolicy(t *testing.T) {
	homeDir := SetupTestConfig(t)
	env := map[string]string{
		"HOME":            homeDir,
		"XDG_CONFIG_HOME": filepath.Join(homeDir, ".config"),
		"NO_COLOR":        "true",
		"REPOBIRD_COLOR":  "always",
	}

	t.Run("REPOBIRD_COLOR always colors non-TTY help despite NO_COLOR", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "help")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, ansiEscape)
	})

	t.Run("config color never disables non-TTY version color", func(t *testing.T) {
		configEnv := map[string]string{
			"HOME":            homeDir,
			"XDG_CONFIG_HOME": filepath.Join(homeDir, ".config"),
			"NO_COLOR":        "",
		}
		result := RunCommandWithEnv(t, configEnv, "config", "set", "color", "never")
		AssertSuccess(t, result)

		result = RunCommandWithEnv(t, configEnv, "version")
		AssertSuccess(t, result)
		AssertNotContains(t, result.Stdout, ansiEscape)
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

	t.Run("run sends prompt-risk acknowledgement to API", func(t *testing.T) {
		result := RunCommandWithEnv(t, env,
			"run",
			"--repo", "test/repo",
			"--prompt", "Test task",
			"--acknowledge-prompt-risk",
		)
		AssertSuccess(t, result)

		var createBody map[string]interface{}
		for _, req := range mockServer.Requests() {
			if req.Method == "POST" && req.Path == "/api/v1/runs" {
				if err := json.Unmarshal(req.Body, &createBody); err != nil {
					t.Fatalf("failed to unmarshal create-run body: %v\n%s", err, req.Body)
				}
			}
		}
		if createBody == nil {
			t.Fatal("expected create-run request")
		}
		if got, ok := createBody["acknowledgePromptRisk"].(bool); !ok || !got {
			t.Fatalf("expected acknowledgePromptRisk=true in create-run request, got %#v", createBody["acknowledgePromptRisk"])
		}
	})
}

func TestReasoningEffortCompatibility(t *testing.T) {
	t.Run("no-preference flag request preserves the legacy payload", func(t *testing.T) {
		env, mockServer := SetupTestEnv(t)
		defer mockServer.Close()

		result := RunCommandWithEnv(t, env,
			"run",
			"--repo", "test/repo",
			"--prompt", "Test task",
			"--run-type", "run",
			"--acknowledge-prompt-risk",
		)
		AssertSuccess(t, result)

		request := lastCreateRunRequest(t, mockServer.Requests())
		if got := request["runType"]; got != "run" {
			t.Fatalf("expected runType compatibility, got %#v", got)
		}
		for _, field := range []string{"reasoningEffort", "opencodeModel", "opencodeProvider"} {
			if _, ok := request[field]; ok {
				t.Fatalf("expected omitted %s in no-preference request, got %#v", field, request[field])
			}
		}
	})

	t.Run("no-preference task files retain supported formats and JSON output", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := []struct {
			name    string
			content string
		}{
			{
				name: "task.json",
				content: `{
					"prompt": "JSON compatibility task",
					"repository": "test/repo",
					"source": "main",
					"target": "feature/json",
					"runType": "run"
				}`,
			},
			{
				name: "task.yaml",
				content: `prompt: YAML compatibility task
repository: test/repo
source: main
target: feature/yaml
runType: run
`,
			},
		}

		for _, file := range files {
			t.Run(file.name, func(t *testing.T) {
				path := CreateTestFile(t, tmpDir, file.name, file.content)
				homeDir := t.TempDir()
				result := RunCommandWithEnv(t, map[string]string{
					"HOME":               homeDir,
					"XDG_CONFIG_HOME":    filepath.Join(homeDir, ".config"),
					"REPOBIRD_API_KEY":   "TEST_KEY",
					"REPOBIRD_TEST_MODE": "true",
					"NO_COLOR":           "true",
				}, "--json", "run", path, "--dry-run")
				AssertSuccess(t, result)
				assertStdoutIsSingleJSONValue(t, result)

				var output struct {
					Schema  string         `json:"schema"`
					Request map[string]any `json:"request"`
				}
				if err := json.Unmarshal([]byte(result.Stdout), &output); err != nil {
					t.Fatalf("failed to decode dry-run JSON: %v", err)
				}
				if output.Schema != "repobird.run.dry_run.v1" {
					t.Fatalf("unexpected dry-run schema: %q", output.Schema)
				}
				for _, field := range []string{"reasoningEffort", "opencodeModel", "opencodeProvider"} {
					if _, ok := output.Request[field]; ok {
						t.Fatalf("expected omitted %s in no-preference dry-run JSON", field)
					}
				}
			})
		}
	})

	t.Run("existing command aliases remain compatible", func(t *testing.T) {
		versionResult := RunCommand(t, "v")
		AssertSuccess(t, versionResult)
		AssertContains(t, versionResult.Stdout, "Version:")

		env, mockServer := SetupTestEnv(t)
		defer mockServer.Close()
		statusResult := RunCommandWithEnv(t, env, "st", "12345")
		AssertSuccess(t, statusResult)
		AssertContains(t, statusResult.Stdout, "12345")
		AssertContains(t, statusResult.Stdout, "DONE")
		AssertNotContains(t, statusResult.Stdout, "Model:")
		AssertNotContains(t, statusResult.Stdout, "Reasoning effort:")
	})

	t.Run("legacy and partial run responses do not invent policy values", func(t *testing.T) {
		tests := []struct {
			name            string
			policyFields    string
			expectedModel   string
			expectedEffort  string
			expectModel     bool
			expectReasoning bool
		}{
			{name: "legacy response"},
			{
				name:          "model-only response",
				policyFields:  `,"model":"future-provider/model-v99"`,
				expectedModel: "future-provider/model-v99",
				expectModel:   true,
			},
			{
				name:            "reasoning-only response",
				policyFields:    `,"reasoningEffort":"future-ultra"`,
				expectedEffort:  "future-ultra",
				expectReasoning: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer TEST_KEY" {
						t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
						http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
						return
					}
					if r.Method != http.MethodGet || r.URL.Path != "/api/v1/runs/legacy-run" {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, `{"data":{"id":"legacy-run","status":"DONE","repository":"test/repo","repositoryName":"test/repo","source":"main","target":"feature/test","runType":"run","title":"Legacy Run","createdAt":"2026-08-15T07:00:00Z","updatedAt":"2026-08-15T07:01:00Z"%s}}`, tt.policyFields)
				}))
				defer server.Close()

				homeDir := t.TempDir()
				env := map[string]string{
					"HOME":               homeDir,
					"XDG_CONFIG_HOME":    filepath.Join(homeDir, ".config"),
					"REPOBIRD_API_URL":   server.URL,
					"REPOBIRD_API_KEY":   "TEST_KEY",
					"REPOBIRD_TEST_MODE": "true",
					"NO_COLOR":           "true",
				}

				human := RunCommandWithEnv(t, env, "status", "legacy-run")
				AssertSuccess(t, human)
				AssertContains(t, human.Stdout, "Legacy Run")
				if tt.expectModel {
					AssertContains(t, human.Stdout, tt.expectedModel)
				} else {
					AssertNotContains(t, human.Stdout, "Model:")
				}
				if tt.expectReasoning {
					AssertContains(t, human.Stdout, tt.expectedEffort)
				} else {
					AssertNotContains(t, human.Stdout, "Reasoning effort:")
				}

				machine := RunCommandWithEnv(t, env, "status", "legacy-run", "--json")
				AssertSuccess(t, machine)
				assertStdoutIsSingleJSONValue(t, machine)
				var output map[string]any
				if err := json.Unmarshal([]byte(machine.Stdout), &output); err != nil {
					t.Fatalf("failed to decode status JSON: %v", err)
				}
				model, hasModel := output["model"]
				if hasModel != tt.expectModel || (hasModel && model != tt.expectedModel) {
					t.Fatalf("unexpected model JSON field: %#v", model)
				}
				effort, hasEffort := output["reasoningEffort"]
				if hasEffort != tt.expectReasoning || (hasEffort && effort != tt.expectedEffort) {
					t.Fatalf("unexpected reasoningEffort JSON field: %#v", effort)
				}
			})
		}
	})

	t.Run("service rejection keeps the established error and exit path", func(t *testing.T) {
		var createAttempts atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs" {
				http.NotFound(w, r)
				return
			}
			createAttempts.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"code":"INVALID_REASONING_EFFORT","message":"Reasoning effort is not supported"}}`)
		}))
		defer server.Close()

		homeDir := t.TempDir()
		result := RunCommandWithEnv(t, map[string]string{
			"HOME":               homeDir,
			"XDG_CONFIG_HOME":    filepath.Join(homeDir, ".config"),
			"REPOBIRD_API_URL":   server.URL,
			"REPOBIRD_API_KEY":   "TEST_KEY",
			"REPOBIRD_TEST_MODE": "true",
			"NO_COLOR":           "true",
		}, "run", "--repo", "test/repo", "--prompt", "Test task", "--acknowledge-prompt-risk")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "Reasoning effort is not supported")
		if got := createAttempts.Load(); got != 1 {
			t.Fatalf("expected one create attempt without fallback or retry, got %d", got)
		}
	})
}

func lastCreateRunRequest(t *testing.T, requests []RecordedRequest) map[string]any {
	t.Helper()
	for i := len(requests) - 1; i >= 0; i-- {
		request := requests[i]
		if request.Method != http.MethodPost || request.Path != "/api/v1/runs" {
			continue
		}
		var body map[string]any
		if err := json.Unmarshal(request.Body, &body); err != nil {
			t.Fatalf("failed to decode create-run request: %v\n%s", err, request.Body)
		}
		return body
	}
	t.Fatal("expected create-run request")
	return nil
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
		assertStdoutIsSingleJSONValue(t, result)
		AssertContains(t, result.Stdout, `"id":`)
		AssertContains(t, result.Stdout, `"status":`)
	})

	t.Run("root JSON flag with status output", func(t *testing.T) {
		rootEnv, rootMockServer := SetupTestEnv(t)
		defer rootMockServer.Close()

		result := RunCommandWithEnv(t, rootEnv, "--json", "status")
		AssertSuccess(t, result)
		assertStdoutIsSingleJSONValue(t, result)
		AssertContains(t, result.Stdout, `"id":`)
		AssertContains(t, result.Stdout, `"status":`)
	})

	t.Run("status shows credits instead of zero legacy run quotas", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "status")
		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Credits:")
		AssertContains(t, result.Stdout, "42.75 available")
		AssertContains(t, result.Stdout, "1.25 reserved")
		AssertNotContains(t, result.Stdout, "Runs:")
		AssertNotContains(t, result.Stdout, "Plan Runs:")
	})

	t.Run("status list sends page and limit query parameters", func(t *testing.T) {
		result := RunCommandWithEnv(t, env, "status", "--limit", "7")
		AssertSuccess(t, result)

		var seenRunsRequest bool
		for _, req := range mockServer.Requests() {
			if req.Method == "GET" && req.Path == "/api/v1/runs" {
				seenRunsRequest = true
				if strings.Contains(req.RawQuery, "page=1") && strings.Contains(req.RawQuery, "limit=7") {
					return
				}
			}
		}
		if seenRunsRequest {
			t.Fatal("expected one list-runs request to include page=1 and limit=7")
		}
		t.Fatal("expected list-runs request")
	})
}

func assertStdoutIsSingleJSONValue(t *testing.T, result *CommandResult) {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(result.Stdout))
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
	if err := decoder.Decode(&value); err != io.EOF {
		t.Fatalf("stdout contains trailing data after first JSON value: %v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
}

// TestBulkCommands tests that legacy bulk operations are disabled.
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
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "bulk runs are currently unavailable")
	})

	t.Run("bulk with invalid file", func(t *testing.T) {
		// Try to run bulk with non-existent file
		result := RunCommandWithEnv(t, env, "bulk", "/nonexistent/bulk.json", "--dry-run")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "bulk runs are currently unavailable")
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

	t.Run("completion installer dry run", func(t *testing.T) {
		homeDir := t.TempDir()
		result := RunCommandWithEnv(t, map[string]string{
			"HOME": homeDir,
		}, "completion", "install", "bash", "--dry-run")

		AssertSuccess(t, result)
		AssertContains(t, result.Stdout, "Would update")
		AssertContains(t, result.Stdout, filepath.Join(homeDir, ".bashrc"))
	})
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

	t.Run("examples command", func(t *testing.T) {
		result := RunCommand(t, "examples")
		AssertSuccess(t, result)

		// Examples output should be stable
		goldenPath := filepath.Join("testdata", "golden", "examples.txt")
		CompareGolden(t, result.Stdout, goldenPath, update)
	})

	t.Run("examples schema run", func(t *testing.T) {
		result := RunCommand(t, "examples", "schema", "run")
		AssertSuccess(t, result)

		// Schema output should be stable
		goldenPath := filepath.Join("testdata", "golden", "examples-schema-run.txt")
		CompareGolden(t, result.Stdout, goldenPath, update)
	})

	t.Run("examples schema bulk disabled", func(t *testing.T) {
		result := RunCommand(t, "examples", "schema", "bulk")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "bulk runs are currently unavailable")
	})
}

// TestExamplesGenerate tests the examples generate commands
func TestExamplesGenerate(t *testing.T) {
	t.Run("generate json example", func(t *testing.T) {
		result := RunCommand(t, "examples", "generate", "run", "-f", "json")
		AssertSuccess(t, result)

		// Extract JSON from output (skip the description line)
		lines := strings.Split(result.Stdout, "\n")
		var jsonStart int
		for i, line := range lines {
			if strings.TrimSpace(line) == "{" {
				jsonStart = i
				break
			}
		}

		if jsonStart == 0 {
			t.Fatal("Could not find JSON in output")
		}

		jsonContent := strings.Join(lines[jsonStart:], "\n")

		// Output should be valid JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
			t.Errorf("Generated JSON is not valid: %v\nContent: %s", err, jsonContent)
		}

		// Check required fields
		if _, ok := data["repository"]; !ok {
			t.Error("Generated JSON missing required 'repository' field")
		}
		if _, ok := data["prompt"]; !ok {
			t.Error("Generated JSON missing required 'prompt' field")
		}
	})

	t.Run("generate yaml example", func(t *testing.T) {
		result := RunCommand(t, "examples", "generate", "run", "-f", "yaml")
		AssertSuccess(t, result)

		// Check that output contains required YAML fields
		if !strings.Contains(result.Stdout, "repository:") {
			t.Error("Generated YAML missing 'repository' field")
		}
		if !strings.Contains(result.Stdout, "prompt:") {
			t.Error("Generated YAML missing 'prompt' field")
		}
	})

	t.Run("generate markdown example", func(t *testing.T) {
		result := RunCommand(t, "examples", "generate", "run", "-f", "md")
		AssertSuccess(t, result)

		// Skip the description line and check for markdown frontmatter
		lines := strings.Split(result.Stdout, "\n")
		var mdStart int
		for i, line := range lines {
			if line == "---" {
				mdStart = i
				break
			}
		}

		if mdStart == 0 {
			t.Error("Generated markdown missing frontmatter")
			return
		}

		mdContent := strings.Join(lines[mdStart:], "\n")
		if !strings.HasPrefix(mdContent, "---\n") {
			t.Error("Generated markdown missing frontmatter delimiter")
		}
		if !strings.Contains(mdContent, "repository:") {
			t.Error("Generated markdown missing 'repository' in frontmatter")
		}
	})

	t.Run("generate minimal example", func(t *testing.T) {
		result := RunCommand(t, "examples", "generate", "minimal", "-f", "json")
		AssertSuccess(t, result)

		// Extract JSON from output
		lines := strings.Split(result.Stdout, "\n")
		var jsonStart int
		for i, line := range lines {
			if strings.TrimSpace(line) == "{" {
				jsonStart = i
				break
			}
		}

		if jsonStart == 0 {
			t.Fatal("Could not find JSON in output")
		}

		jsonContent := strings.Join(lines[jsonStart:], "\n")

		// Output should be valid JSON with only required fields
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
			t.Errorf("Generated minimal JSON is not valid: %v\nContent: %s", err, jsonContent)
		}

		// Should have exactly 2 fields (repository and prompt)
		if len(data) != 2 {
			t.Errorf("Minimal example should have exactly 2 fields, got %d", len(data))
		}
	})

	t.Run("generate bulk example", func(t *testing.T) {
		result := RunCommand(t, "examples", "generate", "bulk")
		AssertFailure(t, result)
		AssertContains(t, result.Stderr, "bulk runs are currently unavailable")
	})

	t.Run("generate with output file", func(t *testing.T) {
		// Create temp directory for test
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "test-config.json")

		result := RunCommand(t, "examples", "generate", "minimal", "-o", outputPath)
		AssertSuccess(t, result)

		// Check file was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputPath)
		}

		// Check file contents
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(content, &data); err != nil {
			t.Errorf("Output file contains invalid JSON: %v", err)
		}
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

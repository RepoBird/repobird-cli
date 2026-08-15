// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/models"
)

func TestPrintRunCreateJSONPreservesExplicitReasoningEffort(t *testing.T) {
	var output bytes.Buffer
	err := printRunCreateJSON(&output, &domain.Run{}, domain.CreateRunRequest{
		Prompt:          "Fix auth",
		RepositoryName:  "acme/webapp",
		RunType:         "basic",
		ReasoningEffort: "future-ultra",
	})
	require.NoError(t, err)

	var result map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	request, ok := result["request"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "future-ultra", request["reasoningEffort"])
}

func TestRunJSONOutputsPreserveReturnedReasoningPolicy(t *testing.T) {
	tests := []struct {
		name  string
		print func(io.Writer, *domain.Run) error
	}{
		{
			name: "create",
			print: func(out io.Writer, run *domain.Run) error {
				return printRunCreateJSON(out, run, domain.CreateRunRequest{})
			},
		},
		{
			name: "wait",
			print: func(out io.Writer, run *domain.Run) error {
				return printRunWaitJSON(out, run, domain.CreateRunRequest{}, ExitCodeSuccess, false, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("present", func(t *testing.T) {
				var output bytes.Buffer
				err := tt.print(&output, &domain.Run{
					ID:              "run-123",
					Model:           "future-provider/model-v99",
					ReasoningEffort: "future-ultra",
				})
				require.NoError(t, err)

				var result map[string]any
				require.NoError(t, json.Unmarshal(output.Bytes(), &result))
				run, ok := result["run"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "future-provider/model-v99", run["model"])
				require.Equal(t, "future-ultra", run["reasoningEffort"])
				require.NotContains(t, run, "opencodeModel")
			})

			t.Run("absent", func(t *testing.T) {
				var output bytes.Buffer
				require.NoError(t, tt.print(&output, &domain.Run{ID: "run-123"}))

				var result map[string]any
				require.NoError(t, json.Unmarshal(output.Bytes(), &result))
				run, ok := result["run"].(map[string]any)
				require.True(t, ok)
				require.NotContains(t, run, "model")
				require.NotContains(t, run, "reasoningEffort")
			})
		})
	}
}

func TestProcessSingleRunJSONCreateOutputIsMachineReadable(t *testing.T) {
	ensureRunTestConfig()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("REPOBIRD_API_URL", "")
	t.Setenv("REPOBIRD_ENV", "prod")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/runs", r.URL.Path)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 123,
				"publicId":           "run_123e4567-e89b-12d3-a456-426614174000",
				"status":             "QUEUED",
				"baseBranch":         "main",
				"outputMode":         "pull_request",
				"outputBranch":       "repobird/fix-auth",
				"prTargetBranch":     "release",
				"outputBranchPolicy": "create",
			},
		})
	}))
	defer server.Close()

	originalAPIKey := cfg.APIKey
	originalAPIURL := cfg.APIURL
	originalDryRun := dryRun
	originalFollow := follow
	originalJSONOutput := jsonOutput
	cfg.APIKey = "test-key"
	cfg.APIURL = server.URL
	dryRun = false
	follow = false
	jsonOutput = true
	resetContainer()
	defer func() {
		cfg.APIKey = originalAPIKey
		cfg.APIURL = originalAPIURL
		dryRun = originalDryRun
		follow = originalFollow
		jsonOutput = originalJSONOutput
		resetContainer()
	}()

	output := captureRunStdout(t, func() {
		err := processSingleRun(&models.RunConfig{
			Prompt:         "Fix auth",
			Repository:     "acme/webapp",
			BaseBranch:     "main",
			OutputMode:     "pr",
			PRTargetBranch: "release",
			RunType:        "run",
		}, "")
		require.NoError(t, err)
	})

	require.NotContains(t, output, "Creating run")
	require.NotContains(t, output, "Run created successfully")

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, "repobird.run.create.v1", result["schema"])
	require.Equal(t, "run.create", result["operation"])
	require.Equal(t, true, result["success"])
	require.Equal(t, "https://repobird.ai/repos/issue-runs/run_123e4567-e89b-12d3-a456-426614174000", result["url"])

	run, ok := result["run"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "123", run["id"])
	require.Equal(t, "run_123e4567-e89b-12d3-a456-426614174000", run["publicId"])
	require.Equal(t, "queued", run["status"])
	require.Equal(t, "acme/webapp", run["repositoryName"])
	require.Equal(t, "main", run["baseBranch"])
	require.Equal(t, "pull_request", run["outputMode"])
	require.Equal(t, "repobird/fix-auth", run["outputBranch"])
	require.Equal(t, "release", run["prTargetBranch"])
	require.Equal(t, "create", run["outputBranchPolicy"])

	request, ok := result["request"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, request, "reasoningEffort")
}

func TestProcessSingleRunJSONDryRunOutputIsMachineReadable(t *testing.T) {
	ensureRunTestConfig()

	originalDryRun := dryRun
	originalJSONOutput := jsonOutput
	dryRun = true
	jsonOutput = true
	defer func() {
		dryRun = originalDryRun
		jsonOutput = originalJSONOutput
	}()

	output := captureRunStdout(t, func() {
		err := processSingleRun(&models.RunConfig{
			Prompt:          "Fix auth",
			Repository:      "acme/webapp",
			RunType:         "basic",
			ReasoningEffort: "max",
		}, "")
		require.NoError(t, err)
	})

	require.NotContains(t, output, "Validation successful")

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, "repobird.run.dry_run.v1", result["schema"])
	require.Equal(t, "run.dry_run", result["operation"])
	require.Equal(t, true, result["valid"])

	request, ok := result["request"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Fix auth", request["prompt"])
	require.Equal(t, "acme/webapp", request["repositoryName"])
	require.Equal(t, "basic", request["runType"])
	require.Equal(t, "opencode", request["agent"])
	require.Equal(t, "max", request["reasoningEffort"])
	require.NotContains(t, request, "opencodeModel")
	require.NotContains(t, request, "opencodeProvider")
}

func TestProcessSingleRunJSONDryRunOmitsUnspecifiedPolicy(t *testing.T) {
	ensureRunTestConfig()

	originalDryRun := dryRun
	originalJSONOutput := jsonOutput
	dryRun = true
	jsonOutput = true
	defer func() {
		dryRun = originalDryRun
		jsonOutput = originalJSONOutput
	}()

	output := captureRunStdout(t, func() {
		err := processSingleRun(&models.RunConfig{
			Prompt:     "Fix auth",
			Repository: "acme/webapp",
			RunType:    "basic",
		}, "")
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	request, ok := result["request"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, request, "reasoningEffort")
	require.NotContains(t, request, "opencodeModel")
	require.NotContains(t, request, "opencodeProvider")
}

func TestProcessSingleRunHumanDryRunDoesNotAdvertiseClientPolicy(t *testing.T) {
	ensureRunTestConfig()

	originalDryRun := dryRun
	originalJSONOutput := jsonOutput
	dryRun = true
	jsonOutput = false
	defer func() {
		dryRun = originalDryRun
		jsonOutput = originalJSONOutput
	}()

	output := captureRunStdout(t, func() {
		err := processSingleRun(&models.RunConfig{
			Prompt:     "Fix auth",
			Repository: "acme/webapp",
			RunType:    "basic",
		}, "")
		require.NoError(t, err)
	})

	require.Contains(t, output, "Run type: Basic")
	require.NotContains(t, output, "Model:")
	require.NotContains(t, output, "Reasoning effort:")
}

func TestProcessSingleRunRejectsJSONFollowBeforeCreate(t *testing.T) {
	ensureRunTestConfig()

	originalDryRun := dryRun
	originalFollow := follow
	originalJSONOutput := jsonOutput
	dryRun = false
	follow = true
	jsonOutput = true
	defer func() {
		dryRun = originalDryRun
		follow = originalFollow
		jsonOutput = originalJSONOutput
	}()

	err := processSingleRun(&models.RunConfig{
		Prompt:     "Fix auth",
		Repository: "acme/webapp",
		RunType:    "run",
	}, "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "--follow cannot be used with --json")
}

func TestProcessSingleRunPrintsCanonicalCreateResponseFields(t *testing.T) {
	ensureRunTestConfig()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("REPOBIRD_API_URL", "")
	t.Setenv("REPOBIRD_ENV", "prod")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/runs", r.URL.Path)

		var request map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "acme/webapp", request["repositoryName"])

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":                 123,
				"publicId":           "run_123e4567-e89b-12d3-a456-426614174000",
				"status":             "QUEUED",
				"baseBranch":         "main",
				"outputMode":         "pull_request",
				"outputBranch":       "repobird/fix-auth",
				"prTargetBranch":     "release",
				"outputBranchPolicy": "create",
			},
		})
	}))
	defer server.Close()

	originalAPIKey := cfg.APIKey
	originalAPIURL := cfg.APIURL
	originalDryRun := dryRun
	originalFollow := follow
	cfg.APIKey = "test-key"
	cfg.APIURL = server.URL
	dryRun = false
	follow = false
	resetContainer()
	defer func() {
		cfg.APIKey = originalAPIKey
		cfg.APIURL = originalAPIURL
		dryRun = originalDryRun
		follow = originalFollow
		resetContainer()
	}()

	output := captureRunStdout(t, func() {
		err := processSingleRun(&models.RunConfig{
			Prompt:         "Fix auth",
			Repository:     "acme/webapp",
			BaseBranch:     "main",
			OutputMode:     "pr",
			PRTargetBranch: "release",
			RunType:        "run",
		}, "")
		require.NoError(t, err)
	})

	require.Contains(t, output, "Run created successfully!")
	require.Contains(t, output, "Run ID: 123")
	require.Contains(t, output, "Public ID: run_123e4567-e89b-12d3-a456-426614174000")
	require.Contains(t, output, "Status: QUEUED")
	require.Contains(t, output, "Repository: acme/webapp")
	require.Contains(t, output, "Base branch: main")
	require.Contains(t, output, "Output branch: repobird/fix-auth")
	require.Contains(t, output, "PR target branch: release")
	require.Contains(t, output, "Output mode: pull_request")
	require.Contains(t, output, "Output branch policy: create")
	require.Contains(t, output, "URL: https://repobird.ai/repos/issue-runs/run_123e4567-e89b-12d3-a456-426614174000")
	require.NotContains(t, output, "Source:")
}

func TestProcessSingleRunBlocksRecentDuplicateSubmission(t *testing.T) {
	ensureRunTestConfig()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("REPOBIRD_API_URL", "")
	t.Setenv("REPOBIRD_ENV", "prod")

	postCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postCount++
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":     123,
				"status": "QUEUED",
			},
		})
	}))
	defer server.Close()

	originalAPIKey := cfg.APIKey
	originalAPIURL := cfg.APIURL
	originalDryRun := dryRun
	originalFollow := follow
	originalForceRun := forceRun
	cfg.APIKey = "test-key"
	cfg.APIURL = server.URL
	dryRun = false
	follow = false
	forceRun = false
	resetContainer()
	defer func() {
		cfg.APIKey = originalAPIKey
		cfg.APIURL = originalAPIURL
		dryRun = originalDryRun
		follow = originalFollow
		forceRun = originalForceRun
		resetContainer()
	}()

	runConfig := &models.RunConfig{
		Prompt:     "Fix auth",
		Repository: "acme/webapp",
		RunType:    "run",
	}

	captureRunStdout(t, func() {
		require.NoError(t, processSingleRun(runConfig, ""))
	})

	err := processSingleRun(runConfig, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "identical run submitted")
	require.Contains(t, err.Error(), "--force")
	require.Equal(t, 1, postCount)
}

func TestProcessSingleRunAllowsRecentDuplicateWithForce(t *testing.T) {
	ensureRunTestConfig()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("REPOBIRD_API_URL", "")
	t.Setenv("REPOBIRD_ENV", "prod")

	postCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postCount++
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":     postCount,
				"status": "QUEUED",
			},
		})
	}))
	defer server.Close()

	originalAPIKey := cfg.APIKey
	originalAPIURL := cfg.APIURL
	originalDryRun := dryRun
	originalFollow := follow
	originalForceRun := forceRun
	cfg.APIKey = "test-key"
	cfg.APIURL = server.URL
	dryRun = false
	follow = false
	forceRun = false
	resetContainer()
	defer func() {
		cfg.APIKey = originalAPIKey
		cfg.APIURL = originalAPIURL
		dryRun = originalDryRun
		follow = originalFollow
		forceRun = originalForceRun
		resetContainer()
	}()

	runConfig := &models.RunConfig{
		Prompt:     "Fix auth",
		Repository: "acme/webapp",
		RunType:    "run",
	}

	captureRunStdout(t, func() {
		require.NoError(t, processSingleRun(runConfig, ""))
	})

	forceRun = true
	captureRunStdout(t, func() {
		require.NoError(t, processSingleRun(runConfig, ""))
	})

	require.Equal(t, 2, postCount)
}

func captureRunStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

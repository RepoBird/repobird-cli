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

	"github.com/repobird/repobird-cli/internal/models"
)

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

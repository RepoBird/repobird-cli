// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/models"
)

func TestProcessSingleRunWaitJSONSuccessPrintsFinalResult(t *testing.T) {
	server := newRunWaitTestServer(t, []map[string]any{
		{
			"id":             123,
			"publicId":       "run_public",
			"status":         "DONE",
			"repositoryName": "acme/webapp",
			"prUrl":          "https://github.com/acme/webapp/pull/1",
		},
	}, nil)
	defer server.Close()

	restore := configureRunWaitTest(t, server.URL)
	defer restore()

	output := captureRunStdout(t, func() {
		err := processSingleRun(waitTestConfig(), "")
		require.NoError(t, err)
	})

	var result runWaitJSONOutput
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, "repobird.run.wait.v1", result.Schema)
	require.Equal(t, "run.wait", result.Operation)
	require.True(t, result.Success)
	require.Equal(t, ExitCodeSuccess, result.ExitCode)
	require.False(t, result.TimedOut)
	require.Equal(t, "123", result.Run.ID)
	require.Equal(t, "completed", result.Run.Status)
	require.Equal(t, "https://github.com/acme/webapp/pull/1", result.Run.PullRequestURL)
}

func TestProcessSingleRunWaitJSONTerminalFailureReturnsRunFailedExitCode(t *testing.T) {
	server := newRunWaitTestServer(t, []map[string]any{
		{
			"id":             123,
			"publicId":       "run_public",
			"status":         "FAILED",
			"repositoryName": "acme/webapp",
			"error":          "tests failed",
		},
	}, nil)
	defer server.Close()

	restore := configureRunWaitTest(t, server.URL)
	defer restore()

	output := captureRunStdout(t, func() {
		err := processSingleRun(waitTestConfig(), "")
		require.Error(t, err)
		require.Equal(t, ExitCodeRunFailed, exitCodeForError(err))
		require.Contains(t, err.Error(), "tests failed")
	})

	var result runWaitJSONOutput
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, "repobird.run.wait.v1", result.Schema)
	require.False(t, result.Success)
	require.Equal(t, ExitCodeRunFailed, result.ExitCode)
	require.False(t, result.TimedOut)
	require.Equal(t, "123", result.Run.ID)
	require.Equal(t, "failed", result.Run.Status)
	require.Equal(t, "tests failed", result.Error)
}

func TestProcessSingleRunWaitJSONTimeoutReturnsTimeoutExitCode(t *testing.T) {
	server := newRunWaitTestServer(t, []map[string]any{
		{
			"id":             123,
			"publicId":       "run_public",
			"status":         "PROCESSING",
			"repositoryName": "acme/webapp",
		},
	}, nil)
	defer server.Close()

	restore := configureRunWaitTest(t, server.URL)
	defer restore()
	waitTimeout = 20 * time.Millisecond

	output := captureRunStdout(t, func() {
		err := processSingleRun(waitTestConfig(), "")
		require.Error(t, err)
		require.Equal(t, ExitCodeTimeout, exitCodeForError(err))
	})

	var result runWaitJSONOutput
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, "repobird.run.wait.v1", result.Schema)
	require.False(t, result.Success)
	require.Equal(t, ExitCodeTimeout, result.ExitCode)
	require.True(t, result.TimedOut)
	require.Equal(t, "123", result.Run.ID)
	require.Equal(t, "running", result.Run.Status)
}

func TestProcessSingleRunWaitCreateErrorMapsQuotaExitCode(t *testing.T) {
	server := newRunWaitTestServer(t, nil, map[string]any{
		"error":   "NO_RUNS_REMAINING",
		"message": "No credits remaining",
		"details": map[string]any{
			"tier":      "pro",
			"limit":     10,
			"remaining": 0,
		},
	})
	defer server.Close()

	restore := configureRunWaitTest(t, server.URL)
	defer restore()

	err := processSingleRun(waitTestConfig(), "")
	require.Error(t, err)
	require.Equal(t, ExitCodeQuota, exitCodeForError(err))
}

func newRunWaitTestServer(t *testing.T, statuses []map[string]any, createError map[string]any) *httptest.Server {
	t.Helper()

	getCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs":
			if createError != nil {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(createError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":             123,
					"publicId":       "run_public",
					"status":         "QUEUED",
					"repositoryName": "acme/webapp",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs/123":
			require.NotEmpty(t, statuses)
			index := getCount
			if index >= len(statuses) {
				index = len(statuses) - 1
			}
			getCount++
			_ = json.NewEncoder(w).Encode(map[string]any{"data": statuses[index]})
		default:
			http.NotFound(w, r)
		}
	}))
}

func configureRunWaitTest(t *testing.T, apiURL string) func() {
	t.Helper()

	ensureRunTestConfig()
	originalAPIKey := cfg.APIKey
	originalAPIURL := cfg.APIURL
	originalDryRun := dryRun
	originalFollow := follow
	originalWait := wait
	originalJSONOutput := jsonOutput
	originalWaitTimeout := waitTimeout
	originalWaitPollInterval := waitPollInterval
	originalForceRun := forceRun
	originalIdempotencyKey := idempotencyKey

	cfg.APIKey = "test-key"
	cfg.APIURL = apiURL
	dryRun = false
	follow = false
	wait = true
	jsonOutput = true
	waitTimeout = time.Second
	waitPollInterval = 10 * time.Millisecond
	forceRun = true
	idempotencyKey = ""
	resetContainer()

	return func() {
		cfg.APIKey = originalAPIKey
		cfg.APIURL = originalAPIURL
		dryRun = originalDryRun
		follow = originalFollow
		wait = originalWait
		jsonOutput = originalJSONOutput
		waitTimeout = originalWaitTimeout
		waitPollInterval = originalWaitPollInterval
		forceRun = originalForceRun
		idempotencyKey = originalIdempotencyKey
		resetContainer()
	}
}

func waitTestConfig() *models.RunConfig {
	return &models.RunConfig{
		Prompt:     "Fix auth",
		Repository: "acme/webapp",
		RunType:    "run",
	}
}

func TestExitCodeForErrorClassifiesAuthQuotaRunFailureAndTimeout(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: ExitCodeSuccess},
		{name: "generic", err: errors.New("boom"), want: ExitCodeGeneric},
		{name: "auth message", err: errors.New("API key not configured"), want: ExitCodeAuth},
		{name: "quota message", err: errors.New("no runs remaining"), want: ExitCodeQuota},
		{name: "run failed", err: newExitError(ExitCodeRunFailed, "run failed"), want: ExitCodeRunFailed},
		{name: "timeout", err: newExitError(ExitCodeTimeout, "timed out"), want: ExitCodeTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, exitCodeForError(tt.err))
		})
	}
}

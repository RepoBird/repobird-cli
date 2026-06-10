// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWaitShellExitCodeContract(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		createBody map[string]any
		getBody    map[string]any
		wantExit   int
		wantJSON   map[string]any
	}{
		{
			name:       "successful terminal state exits zero",
			statusCode: http.StatusCreated,
			createBody: runWaitSmokeRun("QUEUED", ""),
			getBody:    runWaitSmokeRun("DONE", ""),
			wantExit:   0,
			wantJSON: map[string]any{
				"schema":   "repobird.run.wait.v1",
				"success":  true,
				"exitCode": float64(0),
				"timedOut": false,
				"status":   "completed",
			},
		},
		{
			name:       "failed terminal state exits four",
			statusCode: http.StatusCreated,
			createBody: runWaitSmokeRun("QUEUED", ""),
			getBody:    runWaitSmokeRun("FAILED", "tests failed"),
			wantExit:   4,
			wantJSON: map[string]any{
				"schema":   "repobird.run.wait.v1",
				"success":  false,
				"exitCode": float64(4),
				"timedOut": false,
				"status":   "failed",
				"error":    "tests failed",
			},
		},
		{
			name:       "timeout exits five",
			statusCode: http.StatusCreated,
			createBody: runWaitSmokeRun("QUEUED", ""),
			getBody:    runWaitSmokeRun("PROCESSING", ""),
			wantExit:   5,
			wantJSON: map[string]any{
				"schema":   "repobird.run.wait.v1",
				"success":  false,
				"exitCode": float64(5),
				"timedOut": true,
				"status":   "running",
			},
		},
		{
			name:       "auth error exits two",
			statusCode: http.StatusUnauthorized,
			createBody: map[string]any{"error": "INVALID_API_KEY", "message": "Invalid API key"},
			wantExit:   2,
		},
		{
			name:       "quota error exits three",
			statusCode: http.StatusForbidden,
			createBody: map[string]any{"error": "NO_RUNS_REMAINING", "message": "No credits remaining"},
			wantExit:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newRunWaitSmokeServer(t, tt.statusCode, tt.createBody, tt.getBody)
			defer server.Close()

			homeDir := SetupTestConfig(t)
			taskFile := CreateTestFile(t, t.TempDir(), "task.json", `{
				"prompt": "Smoke test wait exit codes",
				"repository": "test/repo",
				"runType": "run"
			}`)
			env := map[string]string{
				"HOME":               homeDir,
				"XDG_CONFIG_HOME":    filepath.Join(homeDir, ".config"),
				"REPOBIRD_API_URL":   server.URL,
				"REPOBIRD_API_KEY":   "TEST_KEY",
				"REPOBIRD_TEST_MODE": "true",
				"NO_COLOR":           "true",
			}

			result := RunCommandWithEnv(t, env, "run", taskFile, "--wait", "--json", "--timeout", "20ms", "--force")
			AssertExitCode(t, result, tt.wantExit)

			if tt.wantJSON == nil {
				return
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(result.Stdout), &got); err != nil {
				t.Fatalf("stdout is not one JSON object: %v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
			}
			if strings.Count(strings.TrimSpace(result.Stdout), "\n{") > 0 {
				t.Fatalf("expected exactly one JSON object on stdout, got:\n%s", result.Stdout)
			}
			for key, want := range tt.wantJSON {
				if got[key] != want {
					t.Fatalf("JSON field %s = %#v, want %#v\nstdout:\n%s\nstderr:\n%s", key, got[key], want, result.Stdout, result.Stderr)
				}
			}
		})
	}
}

func newRunWaitSmokeServer(t *testing.T, createStatus int, createBody, getBody map[string]any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs":
			w.WriteHeader(createStatus)
			if createStatus >= http.StatusBadRequest {
				_ = json.NewEncoder(w).Encode(createBody)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": createBody})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs/123":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": getBody})
		default:
			http.NotFound(w, r)
		}
	}))
}

func runWaitSmokeRun(status, message string) map[string]any {
	run := map[string]any{
		"id":             123,
		"publicId":       "run_public",
		"status":         status,
		"repositoryName": "test/repo",
		"prUrl":          "https://github.com/test/repo/pull/1",
	}
	if message != "" {
		run["error"] = message
	}
	return run
}

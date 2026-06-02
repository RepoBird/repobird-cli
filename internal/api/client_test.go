// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

const (
	testRunID     = "test-123"
	httpMethodGET = "GET"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		baseURL     string
		debug       bool
		expectedURL string
	}{
		{
			name:        "Default URL",
			apiKey:      "test-key",
			baseURL:     "",
			debug:       false,
			expectedURL: DefaultAPIURL,
		},
		{
			name:        "Custom URL",
			apiKey:      "test-key",
			baseURL:     "https://custom.api.com",
			debug:       true,
			expectedURL: "https://custom.api.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.baseURL, tt.debug)
			if client.baseURL != tt.expectedURL {
				t.Errorf("expected baseURL %s, got %s", tt.expectedURL, client.baseURL)
			}
			if client.apiKey != tt.apiKey {
				t.Errorf("expected apiKey %s, got %s", tt.apiKey, client.apiKey)
			}
			if client.debug != tt.debug {
				t.Errorf("expected debug %v, got %v", tt.debug, client.debug)
			}
		})
	}
}

func TestGetRepositorySupportsBranchDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repositories/repo_123" {
			t.Errorf("expected path /api/v1/repositories/repo_123, got %s", r.URL.Path)
		}
		if r.Method != httpMethodGET {
			t.Errorf("expected GET, got %s", r.Method)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                    1,
			"name":                  "RepoBird CLI",
			"repoName":              "repobird-cli",
			"repoOwner":             "RepoBird",
			"defaultBranch":         "main",
			"defaultBaseBranch":     "develop",
			"defaultPrTargetBranch": "release",
			"defaultOutputBranch":   "automation/repobird",
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	repo, err := client.GetRepository("repo_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.FullName() != "RepoBird/repobird-cli" {
		t.Errorf("expected full name RepoBird/repobird-cli, got %s", repo.FullName())
	}
	if repo.DefaultBaseBranch == nil || *repo.DefaultBaseBranch != "develop" {
		t.Fatalf("expected default base branch develop, got %#v", repo.DefaultBaseBranch)
	}
	if repo.DefaultPRTargetBranch == nil || *repo.DefaultPRTargetBranch != "release" {
		t.Fatalf("expected default PR target branch release, got %#v", repo.DefaultPRTargetBranch)
	}
	if repo.DefaultOutputBranch == nil || *repo.DefaultOutputBranch != "automation/repobird" {
		t.Fatalf("expected default output branch automation/repobird, got %#v", repo.DefaultOutputBranch)
	}
}

func TestUpdateRepositoryDefaultsSendsSetAndClearPayload(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repositories/42" {
			t.Errorf("expected path /api/v1/repositories/42, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                    42,
			"name":                  "webapp",
			"repoName":              "webapp",
			"repoOwner":             "acme",
			"defaultBranch":         "main",
			"defaultBaseBranch":     "develop",
			"defaultPrTargetBranch": nil,
			"defaultOutputBranch":   nil,
		})
	}))
	defer server.Close()

	baseBranch := "develop"
	client := NewClient("test-key", server.URL, false)
	_, err := client.UpdateRepositoryDefaults("42", models.RepositoryDefaultsUpdate{
		DefaultBaseBranch:        &baseBranch,
		ClearDefaultPRTarget:     true,
		ClearDefaultOutputBranch: true,
		ClearDefaultBaseBranch:   false,
		DefaultPRTargetBranch:    nil,
		DefaultOutputBranch:      nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := requestBody["defaultBaseBranch"]; got != "develop" {
		t.Errorf("expected defaultBaseBranch develop, got %#v", got)
	}
	if got, ok := requestBody["defaultPrTargetBranch"]; !ok || got != nil {
		t.Errorf("expected defaultPrTargetBranch null, got %#v (present=%v)", got, ok)
	}
	if got, ok := requestBody["defaultOutputBranch"]; !ok || got != nil {
		t.Errorf("expected defaultOutputBranch null, got %#v (present=%v)", got, ok)
	}
}

func TestUpdateRepositoryDefaultsRequiresAChange(t *testing.T) {
	client := NewClient("test-key", "https://example.test", false)
	_, err := client.UpdateRepositoryDefaults("42", models.RepositoryDefaultsUpdate{})
	if err == nil || !strings.Contains(err.Error(), "at least one repository default") {
		t.Fatalf("expected missing change error, got %v", err)
	}
}

func TestCreateRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointRuns {
			t.Errorf("expected path /api/v1/runs, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		var req models.RunRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		resp := models.RunResponse{
			ID:         testRunID,
			Status:     models.StatusQueued,
			Repository: req.Repository,
			Source:     req.Source,
			Target:     req.Target,
			Prompt:     req.Prompt,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	req := &models.RunRequest{
		Prompt:     "Test prompt",
		Repository: "test/repo",
		Source:     "main",
		Target:     "feature",
		RunType:    models.RunTypeRun,
	}

	resp, err := client.CreateRun(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != testRunID {
		t.Errorf("expected ID %s, got %s", testRunID, resp.ID)
	}
	if resp.Status != models.StatusQueued {
		t.Errorf("expected status QUEUED, got %s", resp.Status)
	}
}

func TestGetRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != RunDetailsURL(testRunID) {
			t.Errorf("expected path /api/v1/runs/%s, got %s", testRunID, r.URL.Path)
		}
		if r.Method != httpMethodGET {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := models.RunResponse{
			ID:         testRunID,
			Status:     models.StatusProcessing,
			Repository: "test/repo",
			Source:     "main",
			Target:     "feature",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	resp, err := client.GetRun(testRunID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != testRunID {
		t.Errorf("expected ID %s, got %s", testRunID, resp.ID)
	}
	if resp.Status != models.StatusProcessing {
		t.Errorf("expected status PROCESSING, got %s", resp.Status)
	}
}

func TestListRuns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != httpMethodGET {
			t.Errorf("expected GET, got %s", r.Method)
		}

		runs := []*models.RunResponse{
			{
				ID:         "test-1",
				Status:     models.StatusDone,
				Repository: "test/repo",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			{
				ID:         "test-2",
				Status:     models.StatusFailed,
				Repository: "test/repo",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(runs)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	runs, err := client.ListRunsLegacy(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
	if runs[0].ID != "test-1" {
		t.Errorf("expected first run ID test-1, got %s", runs[0].ID)
	}
}

func TestVerifyAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != EndpointAuthVerify {
			t.Errorf("expected path /api/v1/auth/verify, got %s", r.URL.Path)
		}
		if r.Method != httpMethodGET {
			t.Errorf("expected GET, got %s", r.Method)
		}

		userInfo := models.UserInfo{
			Email:         "test@example.com",
			RemainingRuns: 5,
			TotalRuns:     10,
			Tier:          "pro",
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(userInfo)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	userInfo, err := client.VerifyAuth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if userInfo.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", userInfo.Email)
	}
	if userInfo.RemainingRuns != 5 {
		t.Errorf("expected 5 remaining runs, got %d", userInfo.RemainingRuns)
	}
	if userInfo.Tier != "pro" {
		t.Errorf("expected tier pro, got %s", userInfo.Tier)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Invalid API key"))
	}))
	defer server.Close()

	client := NewClient("bad-key", server.URL, false)
	_, err := client.GetRun(testRunID)
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
	expectedMsg := "authentication failed: Invalid API key (http_401)"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: got %q, want %q", err.Error(), expectedMsg)
	}
}

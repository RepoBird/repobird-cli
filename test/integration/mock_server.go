// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockServer represents a mock RepoBird API server for testing
type MockServer struct {
	*httptest.Server
	mu           sync.RWMutex
	runs         map[string]*MockRun
	bulkRuns     map[string]*MockBulkRun
	apiKeys      map[string]bool
	rateLimits   map[string]int
	failNext     bool
	responseTime time.Duration
}

// MockRun represents a mock run object
type MockRun struct {
	ID             int       `json:"id"`
	Status         string    `json:"status"`
	Repository     string    `json:"repository,omitempty"`     // Legacy field
	RepositoryName string    `json:"repositoryName,omitempty"` // New API field
	Title          string    `json:"title"`
	RunType        string    `json:"runType"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	PrURL          string    `json:"prUrl,omitempty"`
	CommandLogURL  string    `json:"commandLogUrl,omitempty"`
	Errors         []string  `json:"errors,omitempty"`
	Source         string    `json:"source,omitempty"`
	Target         string    `json:"target,omitempty"`
}

// MockBulkRun represents a mock bulk run batch
type MockBulkRun struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	RunIDs    []int      `json:"runIds"`
	CreatedAt time.Time  `json:"createdAt"`
	Runs      []*MockRun `json:"runs"`
}

// MockUser represents a mock user object
type MockUser struct {
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	Name           string   `json:"name"`
	GithubUsername string   `json:"githubUsername"`
	Tier           MockTier `json:"tier"`
}

// MockTier represents a mock tier object
type MockTier struct {
	Name                string `json:"name"`
	RemainingProRuns    int    `json:"remainingProRuns"`
	RemainingPlanRuns   int    `json:"remainingPlanRuns"`
	LastPeriodResetDate string `json:"lastPeriodResetDate"`
}

// NewMockServer creates a new mock API server
func NewMockServer(t *testing.T) *MockServer {
	ms := &MockServer{
		runs:       make(map[string]*MockRun),
		bulkRuns:   make(map[string]*MockBulkRun),
		apiKeys:    make(map[string]bool),
		rateLimits: make(map[string]int),
	}

	// Add some default valid API keys
	ms.apiKeys["TEST_KEY"] = true
	ms.apiKeys["VALID_KEY"] = true

	// Add some default mock runs
	ms.runs["12345"] = &MockRun{
		ID:             12345,
		Status:         "DONE",
		Repository:     "test/repo", // Include both for compatibility
		RepositoryName: "test/repo",
		Title:          "Test Run",
		RunType:        "run",
		Source:         "main",
		Target:         "feature/test",
		CreatedAt:      time.Now().Add(-1 * time.Hour),
		UpdatedAt:      time.Now().Add(-30 * time.Minute),
		PrURL:          "https://github.com/test/repo/pull/1",
	}

	ms.runs["67890"] = &MockRun{
		ID:             67890,
		Status:         "RUNNING",
		Repository:     "test/another-repo", // Include both for compatibility
		RepositoryName: "test/another-repo",
		Title:          "Another Test Run",
		RunType:        "plan",
		Source:         "main",
		Target:         "feature/plan",
		CreatedAt:      time.Now().Add(-10 * time.Minute),
		UpdatedAt:      time.Now(),
	}

	// Create the test server
	server := httptest.NewServer(http.HandlerFunc(ms.handler))
	ms.Server = server

	return ms
}

// handler routes requests to appropriate handlers
func (ms *MockServer) handler(w http.ResponseWriter, r *http.Request) {
	// Add response delay if configured
	ms.mu.RLock()
	responseTime := ms.responseTime
	ms.mu.RUnlock()
	if responseTime > 0 {
		time.Sleep(responseTime)
	}

	// Check for forced failure
	ms.mu.Lock()
	shouldFail := ms.failNext
	if shouldFail {
		ms.failNext = false
	}
	ms.mu.Unlock()

	if shouldFail {
		http.Error(w, "Forced failure", http.StatusInternalServerError)
		return
	}

	// Check authentication
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, `{"error": "AUTH_REQUIRED", "message": "Authentication required"}`, http.StatusUnauthorized)
		return
	}

	apiKey := strings.TrimPrefix(auth, "Bearer ")
	if !ms.isValidAPIKey(apiKey) {
		http.Error(w, `{"error": "INVALID_API_KEY", "message": "Invalid API key"}`, http.StatusUnauthorized)
		return
	}

	// Check rate limiting
	if ms.isRateLimited(apiKey) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, `{"error": "RATE_LIMITED", "message": "Rate limit exceeded"}`, http.StatusTooManyRequests)
		return
	}

	// Route to appropriate handler
	switch {
	case r.URL.Path == "/api/v1/auth/verify" && r.Method == "GET":
		ms.handleAuthVerify(w, r)
	case r.URL.Path == "/api/v1/users/me" && r.Method == "GET":
		ms.handleUserInfo(w, r)
	case r.URL.Path == "/api/v1/runs" && r.Method == "POST":
		ms.handleCreateRun(w, r)
	case r.URL.Path == "/api/v1/runs" && r.Method == "GET":
		ms.handleListRuns(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/runs/") && r.Method == "GET":
		ms.handleGetRun(w, r)
	case r.URL.Path == "/api/v1/bulk/runs" && r.Method == "POST":
		ms.handleCreateBulkRun(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/bulk/") && r.Method == "GET":
		ms.handleGetBulkRun(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleAuthVerify handles auth verification requests
func (ms *MockServer) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   true,
		"message": "API key is valid",
	})
}

// handleUserInfo handles user info requests
func (ms *MockServer) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	user := MockUser{
		ID:             "user_test123",
		Email:          "test@example.com",
		Name:           "Test User",
		GithubUsername: "testuser",
		Tier: MockTier{
			Name:                "pro",
			RemainingProRuns:    45,
			RemainingPlanRuns:   10,
			LastPeriodResetDate: time.Now().Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// handleCreateRun handles run creation requests
func (ms *MockServer) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error": "INVALID_PAYLOAD", "message": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Check for dry run
	if dryRun, ok := payload["dryRun"].(bool); ok && dryRun {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Dry run mode - no run created",
			"valid":   true,
			"payload": payload,
		})
		return
	}

	// Create a new mock run
	runID := fmt.Sprintf("%d", time.Now().Unix())
	repoName := getStringField(payload, "repository")
	if repoName == "" {
		repoName = getStringField(payload, "repositoryName")
	}
	run := &MockRun{
		ID:             int(time.Now().Unix()),
		Status:         "QUEUED",
		Repository:     repoName, // Include both for compatibility
		RepositoryName: repoName,
		Title:          getStringField(payload, "title"),
		RunType:        getStringField(payload, "runType"),
		Source:         getStringField(payload, "source"),
		Target:         getStringField(payload, "target"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ms.mu.Lock()
	ms.runs[runID] = run
	ms.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// handleListRuns handles listing runs
func (ms *MockServer) handleListRuns(w http.ResponseWriter, r *http.Request) {
	ms.mu.RLock()
	runs := make([]*MockRun, 0, len(ms.runs))
	for _, run := range ms.runs {
		runs = append(runs, run)
	}
	ms.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

// handleGetRun handles getting a specific run
func (ms *MockServer) handleGetRun(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/runs/")

	ms.mu.RLock()
	run, exists := ms.runs[runID]
	ms.mu.RUnlock()

	if !exists {
		http.Error(w, `{"error": "RUN_NOT_FOUND", "message": "Run not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// handleCreateBulkRun handles bulk run creation
func (ms *MockServer) handleCreateBulkRun(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error": "INVALID_PAYLOAD", "message": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Check for dry run
	if dryRun, ok := payload["dryRun"].(bool); ok && dryRun {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Dry run mode - no bulk run created",
			"valid":   true,
			"payload": payload,
		})
		return
	}

	// Create a mock bulk run
	batchID := fmt.Sprintf("batch_%d", time.Now().Unix())
	bulkRun := &MockBulkRun{
		ID:        batchID,
		Status:    "PENDING",
		RunIDs:    []int{12345, 67890},
		CreatedAt: time.Now(),
		Runs:      []*MockRun{},
	}

	ms.mu.Lock()
	ms.bulkRuns[batchID] = bulkRun
	ms.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bulkRun)
}

// handleGetBulkRun handles getting bulk run status
func (ms *MockServer) handleGetBulkRun(w http.ResponseWriter, r *http.Request) {
	batchID := strings.TrimPrefix(r.URL.Path, "/api/v1/bulk/")

	ms.mu.RLock()
	bulkRun, exists := ms.bulkRuns[batchID]
	ms.mu.RUnlock()

	if !exists {
		http.Error(w, `{"error": "BATCH_NOT_FOUND", "message": "Bulk run batch not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bulkRun)
}

// Helper methods

func (ms *MockServer) isValidAPIKey(key string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.apiKeys[key]
}

func (ms *MockServer) isRateLimited(key string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	count := ms.rateLimits[key]
	ms.rateLimits[key]++

	// Rate limit after 10 requests for testing
	return count >= 10
}

func (ms *MockServer) SetFailNext(fail bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.failNext = fail
}

func (ms *MockServer) SetResponseTime(duration time.Duration) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.responseTime = duration
}

func (ms *MockServer) AddAPIKey(key string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.apiKeys[key] = true
}

func (ms *MockServer) ResetRateLimits() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.rateLimits = make(map[string]int)
}

func getStringField(data map[string]interface{}, field string) string {
	if val, ok := data[field].(string); ok {
		return val
	}
	return ""
}

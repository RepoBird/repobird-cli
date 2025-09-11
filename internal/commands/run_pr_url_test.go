// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/domain"
)

// mockRunService implements domain.RunService for testing
type mockRunService struct {
	runs map[string]*domain.Run
	waitForCompletionFunc func(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error)
}

func (m *mockRunService) CreateRun(ctx context.Context, req domain.CreateRunRequest) (*domain.Run, error) {
	run := &domain.Run{
		ID:             "test-run-123",
		Status:         domain.StatusQueued,
		RepositoryName: req.RepositoryName,
		SourceBranch:   req.SourceBranch,
		TargetBranch:   req.TargetBranch,
		Prompt:         req.Prompt,
		RunType:        req.RunType,
		Title:          req.Title,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if m.runs == nil {
		m.runs = make(map[string]*domain.Run)
	}
	m.runs[run.ID] = run
	return run, nil
}

func (m *mockRunService) GetRun(ctx context.Context, id string) (*domain.Run, error) {
	if run, ok := m.runs[id]; ok {
		return run, nil
	}
	return nil, fmt.Errorf("run not found")
}

func (m *mockRunService) ListRuns(ctx context.Context, opts domain.ListOptions) ([]*domain.Run, error) {
	var runs []*domain.Run
	for _, run := range m.runs {
		runs = append(runs, run)
	}
	return runs, nil
}

func (m *mockRunService) WaitForCompletion(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error) {
	if m.waitForCompletionFunc != nil {
		return m.waitForCompletionFunc(ctx, id, callback)
	}
	// Default behavior - return completed run without PR URL
	if run, ok := m.runs[id]; ok {
		run.Status = domain.StatusCompleted
		return run, nil
	}
	return nil, fmt.Errorf("run not found")
}

func TestFollowRunStatus_WithPRURL(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func() *mockRunService
		expectPRURL    bool
		expectedOutput []string
	}{
		{
			name: "Completed run with PR URL",
			setupMock: func() *mockRunService {
				mock := &mockRunService{
					runs: make(map[string]*domain.Run),
				}
				// Create initial run
				mock.runs["test-123"] = &domain.Run{
					ID:     "test-123",
					Status: domain.StatusQueued,
				}
				// Mock WaitForCompletion to return completed status
				mock.waitForCompletionFunc = func(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error) {
					// Simulate status updates
					if callback != nil {
						callback(domain.StatusQueued, "")
						callback(domain.StatusRunning, "Processing...")
						callback(domain.StatusCompleted, "")
					}
					// Return completed run without PR URL (will be fetched by GetRun)
					return &domain.Run{
						ID:     id,
						Status: domain.StatusCompleted,
					}, nil
				}
				// Update run with PR URL for when GetRun is called
				mock.runs["test-123"] = &domain.Run{
					ID:             "test-123",
					Status:         domain.StatusCompleted,
					PullRequestURL: "https://github.com/test/repo/pull/123",
				}
				return mock
			},
			expectPRURL: true,
			expectedOutput: []string{
				"Run completed with status: DONE",
				"Pull Request: https://github.com/test/repo/pull/123",
			},
		},
		{
			name: "Completed run without PR URL",
			setupMock: func() *mockRunService {
				mock := &mockRunService{
					runs: make(map[string]*domain.Run),
				}
				mock.runs["test-123"] = &domain.Run{
					ID:     "test-123",
					Status: domain.StatusQueued,
				}
				mock.waitForCompletionFunc = func(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error) {
					// Update the run in the mock to completed status
					if run, ok := mock.runs[id]; ok {
						run.Status = domain.StatusCompleted
						run.PullRequestURL = "" // No PR URL
					}
					return &domain.Run{
						ID:             id,
						Status:         domain.StatusCompleted,
						PullRequestURL: "", // No PR URL
					}, nil
				}
				return mock
			},
			expectPRURL: false,
			expectedOutput: []string{
				"Run completed with status: DONE",
			},
		},
		{
			name: "Failed run",
			setupMock: func() *mockRunService {
				mock := &mockRunService{
					runs: make(map[string]*domain.Run),
				}
				mock.runs["test-123"] = &domain.Run{
					ID:     "test-123",
					Status: domain.StatusQueued,
				}
				mock.waitForCompletionFunc = func(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error) {
					return &domain.Run{
						ID:     id,
						Status: domain.StatusFailed,
						Error:  "Build failed",
					}, nil
				}
				return mock
			},
			expectPRURL: false,
			expectedOutput: []string{
				"Run failed: Build failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock service
			mockService := tt.setupMock()

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute function
			err := followRunStatus(mockService, "test-123")

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check no error
			require.NoError(t, err)

			// Verify expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected, 
					"Output should contain expected message: %s", expected)
			}

			// Verify PR URL presence/absence
			if tt.expectPRURL {
				assert.Contains(t, output, "Pull Request:")
				assert.Contains(t, output, "github.com")
			} else {
				// Should not contain PR URL if not expected
				if !strings.Contains(output, "failed") {
					assert.NotContains(t, output, "Pull Request:")
				}
			}
		})
	}
}

func TestFollowRunStatus_DebugOutput(t *testing.T) {
	// Save original debug state
	originalDebug := debug
	originalEnv := os.Getenv("REPOBIRD_DEBUG_LOG")
	defer func() {
		debug = originalDebug
		os.Setenv("REPOBIRD_DEBUG_LOG", originalEnv)
	}()

	// Enable debug via environment variable
	os.Setenv("REPOBIRD_DEBUG_LOG", "1")

	// Setup mock
	mockService := &mockRunService{
		runs: make(map[string]*domain.Run),
	}
	mockService.runs["test-123"] = &domain.Run{
		ID:             "test-123",
		Status:         domain.StatusCompleted,
		PullRequestURL: "https://github.com/test/repo/pull/456",
	}
	mockService.waitForCompletionFunc = func(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error) {
		return &domain.Run{
			ID:             id,
			Status:         domain.StatusCompleted,
			PullRequestURL: "",
		}, nil
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute function
	err := followRunStatus(mockService, "test-123")

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)

	// Verify debug output is present
	assert.Contains(t, output, "DEBUG:")
	assert.Contains(t, output, "WaitForCompletion returned")
	assert.Contains(t, output, "Run completed, fetching full details")
	assert.Contains(t, output, "Fetched full details")
}

func TestFormatStatusForDisplay(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{domain.StatusCompleted, "DONE"},
		{domain.StatusQueued, "QUEUED"},
		{domain.StatusRunning, "PROCESSING"},
		{domain.StatusFailed, "FAILED"},
		{domain.StatusCancelled, "CANCELLED"},
		{domain.StatusCreated, "CREATED"},
		{"unknown", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatStatusForDisplay(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{3665 * time.Second, "1h1m5s"},
		{2*time.Hour + 30*time.Minute + 45*time.Second, "2h30m45s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
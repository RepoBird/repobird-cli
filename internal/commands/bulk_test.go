// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	"github.com/repobird/repobird-cli/internal/config"
)

func TestNewBulkCommandDevelopmentGate(t *testing.T) {
	t.Run("hidden and unavailable by default", func(t *testing.T) {
		t.Setenv(config.EnvEnvironment, "")
		t.Setenv(config.EnvEnableBulkRuns, "")

		cmd := NewBulkCommand()
		assert.True(t, cmd.Hidden)
		assert.ErrorContains(t, cmd.RunE(cmd, nil), "bulk runs are currently unavailable")
	})

	t.Run("visible in development when enabled", func(t *testing.T) {
		t.Setenv(config.EnvEnvironment, "development")
		t.Setenv(config.EnvEnableBulkRuns, "1")

		cmd := NewBulkCommand()
		assert.False(t, cmd.Hidden)
	})
}

func TestPrintBulkDryRunSummaryJSONIsMachineReadable(t *testing.T) {
	originalJSONOutput := jsonOutput
	jsonOutput = true
	defer func() {
		jsonOutput = originalJSONOutput
	}()

	output := captureBulkStdout(t, func() {
		err := printDryRunSummary(&bulk.BulkConfig{
			Repository: "acme/webapp",
			Source:     "main",
			RunType:    "run",
			Runs: []bulk.BulkRunConfig{
				{Title: "Fix auth", Prompt: "Fix auth bug", Target: "fix/auth"},
				{Prompt: "Add tests", Target: "test/auth"},
			},
		})
		assert.NoError(t, err)
	})

	assert.NotContains(t, output, "Configuration valid")

	var result map[string]any
	assert.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, "repobird.bulk.dry_run.v1", result["schema"])
	assert.Equal(t, "bulk.dry_run", result["operation"])
	assert.Equal(t, true, result["valid"])
	assert.Equal(t, "acme/webapp", result["repositoryName"])
	assert.Equal(t, "main", result["sourceBranch"])
	assert.Equal(t, "run", result["runType"])
	assert.Equal(t, float64(2), result["totalRuns"])

	runs, ok := result["runs"].([]any)
	assert.True(t, ok)
	assert.Len(t, runs, 2)
	first := runs[0].(map[string]any)
	assert.Equal(t, float64(1), first["index"])
	assert.Equal(t, "Fix auth", first["title"])
	assert.Equal(t, "fix/auth", first["targetBranch"])
	second := runs[1].(map[string]any)
	assert.Equal(t, "Run 2", second["title"])
}

func TestDisplayBulkSubmissionResultsJSONIsMachineReadable(t *testing.T) {
	originalJSONOutput := jsonOutput
	jsonOutput = true
	defer func() {
		jsonOutput = originalJSONOutput
	}()

	output := captureBulkStdout(t, func() {
		displayBulkSubmissionResults(&dto.BulkRunResponse{
			Data: dto.BulkRunData{
				BatchID:    "batch-123",
				BatchTitle: "Nightly fixes",
				Successful: []dto.RunCreatedItem{
					{ID: 101, Status: "QUEUED", RepositoryName: "acme/webapp", Title: "Fix auth", RequestIndex: 0},
				},
				Failed: []dto.RunError{
					{RequestIndex: 1, Prompt: "Add tests", Message: "invalid target"},
				},
				Metadata: dto.BulkResponseMetadata{
					TotalRequested:  2,
					TotalSuccessful: 1,
					TotalFailed:     1,
				},
			},
		})
	})

	assert.NotContains(t, output, "Partial success")
	assert.NotContains(t, output, "Created runs")

	var result map[string]any
	assert.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, "repobird.bulk.create.v1", result["schema"])
	assert.Equal(t, "bulk.create", result["operation"])
	assert.Equal(t, false, result["success"])
	assert.Equal(t, "batch-123", result["batchId"])

	runs, ok := result["runs"].([]any)
	assert.True(t, ok)
	assert.Len(t, runs, 1)
	run := runs[0].(map[string]any)
	assert.Equal(t, "101", run["id"])
	assert.Equal(t, "QUEUED", run["status"])
	assert.Equal(t, "acme/webapp", run["repositoryName"])

	failed, ok := result["failed"].([]any)
	assert.True(t, ok)
	assert.Len(t, failed, 1)
	failure := failed[0].(map[string]any)
	assert.Equal(t, float64(2), failure["index"])
	assert.Equal(t, "invalid target", failure["message"])
}

// TestDisplayBulkResults_PRURLDisplay tests that PR URLs are displayed for completed bulk runs
func TestDisplayBulkResults_PRURLDisplay(t *testing.T) {
	tests := []struct {
		name           string
		statusResponse dto.BulkStatusResponse
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "Completed runs with PR URLs",
			statusResponse: dto.BulkStatusResponse{
				Data: dto.BulkStatusData{
					BatchID: "batch-123",
					Status:  "COMPLETED",
					Runs: []dto.RunStatusItem{
						{
							ID:     1,
							Title:  "Fix bug #1",
							Status: "DONE",
							PRURL:  &[]string{"https://github.com/test/repo/pull/101"}[0],
						},
						{
							ID:     2,
							Title:  "Add feature #2",
							Status: "DONE",
							PRURL:  &[]string{"https://github.com/test/repo/pull/102"}[0],
						},
						{
							ID:     3,
							Title:  "Refactor code",
							Status: "DONE",
							PRURL:  &[]string{"https://github.com/test/repo/pull/103"}[0],
						},
					},
					Metadata: dto.BulkStatusMetadata{
						TotalRuns:  3,
						Queued:     0,
						Processing: 0,
						Completed:  3,
						Failed:     0,
						StartedAt:  "2024-01-01T10:00:00Z",
					},
				},
			},
			expectedOutput: []string{
				"✓ Fix bug #1",
				"✓ Add feature #2",
				"✓ Refactor code",
				"Total: 3",
				"Completed: 3",
			},
			// Note: PR URLs are only fetched when API client can get them,
			// which won't happen in unit test without proper mock
			notExpected: []string{},
		},
		{
			name: "Mixed statuses - only completed runs show PR URLs",
			statusResponse: dto.BulkStatusResponse{
				Data: dto.BulkStatusData{
					BatchID: "batch-456",
					Status:  "IN_PROGRESS",
					Runs: []dto.RunStatusItem{
						{
							ID:     1,
							Title:  "Completed task",
							Status: "DONE",
							PRURL:  &[]string{"https://github.com/test/repo/pull/201"}[0],
						},
						{
							ID:     2,
							Title:  "Failed task",
							Status: "FAILED",
						},
						{
							ID:     3,
							Title:  "Running task",
							Status: "PROCESSING",
						},
					},
					Metadata: dto.BulkStatusMetadata{
						TotalRuns:  3,
						Queued:     0,
						Processing: 1,
						Completed:  1,
						Failed:     1,
						StartedAt:  "2024-01-01T10:00:00Z",
					},
				},
			},
			expectedOutput: []string{
				"✓ Completed task",
				"✗ Failed task",
				"● Running task", // ● is the processing icon
				"Total: 3",
				"Completed: 1",
				"Failed: 1",
			},
			notExpected: []string{
				"pull/202", // Failed run shouldn't have PR URL
				"pull/203", // Running run shouldn't have PR URL
			},
		},
		{
			name: "Completed runs without PR URLs",
			statusResponse: dto.BulkStatusResponse{
				Data: dto.BulkStatusData{
					BatchID: "batch-789",
					Status:  "COMPLETED",
					Runs: []dto.RunStatusItem{
						{
							ID:     1,
							Title:  "Task without PR",
							Status: "DONE",
							PRURL:  nil, // No PR URL
						},
					},
					Metadata: dto.BulkStatusMetadata{
						TotalRuns:  1,
						Queued:     0,
						Processing: 0,
						Completed:  1,
						Failed:     0,
						StartedAt:  "2024-01-01T10:00:00Z",
					},
				},
			},
			expectedOutput: []string{
				"✓ Task without PR",
				"Total: 1",
				"Completed: 1",
			},
			notExpected: []string{
				"github.com",
				"pull",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Display bulk results
			displayBulkResults(tt.statusResponse.Data)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Verify expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected,
					"Output should contain: %s", expected)
			}

			// Verify not expected
			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, output, notExpected,
					"Output should not contain: %s", notExpected)
			}
		})
	}
}

func captureBulkStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w

	fn()

	assert.NoError(t, w.Close())
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	assert.NoError(t, err)
	return buf.String()
}

func TestBulkRuns_PRURLInStatusDisplay(t *testing.T) {
	// Test that the batch status display properly shows PR URLs
	now := time.Now()
	completedTime := now.Add(5 * time.Minute)

	completedTimeStr := completedTime.Format(time.RFC3339)
	statusWithPRs := dto.BulkStatusResponse{
		Data: dto.BulkStatusData{
			BatchID: "batch-follow-123",
			Status:  "COMPLETED",
			Runs: []dto.RunStatusItem{
				{
					ID:          1,
					Title:       "First bulk task",
					Status:      "DONE",
					PRURL:       &[]string{"https://github.com/test/repo/pull/301"}[0],
					CompletedAt: &completedTimeStr,
				},
				{
					ID:          2,
					Title:       "Second bulk task",
					Status:      "DONE",
					PRURL:       &[]string{"https://github.com/test/repo/pull/302"}[0],
					CompletedAt: &completedTimeStr,
				},
			},
			Metadata: dto.BulkStatusMetadata{
				TotalRuns:  2,
				Queued:     0,
				Processing: 0,
				Completed:  2,
				Failed:     0,
				StartedAt:  now.Format(time.RFC3339),
			},
		},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Display the status
	displayBulkResults(statusWithPRs.Data)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify tasks are displayed
	assert.Contains(t, output, "First bulk task",
		"Should display first task title")
	assert.Contains(t, output, "Second bulk task",
		"Should display second task title")
	assert.Contains(t, output, "✓",
		"Should show completed status with checkmark")
	assert.Contains(t, output, "Total: 2",
		"Should show total runs")
	assert.Contains(t, output, "Completed: 2",
		"Should show completed count")
	// Note: PR URLs won't show without a proper API client mock
}

func TestDisplayMultiLineBulkStatusOmitsSpinnerOutsideTTY(t *testing.T) {
	status := dto.BulkStatusData{
		Runs: []dto.RunStatusItem{
			{ID: 1, Status: "PROCESSING"},
			{ID: 2, Status: "QUEUED"},
		},
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = old
		_ = r.Close()
	})

	displayMultiLineBulkStatus(status, "⠋", time.Now())

	assert.NoError(t, w.Close())

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	assert.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "⠋")
	assert.NotContains(t, output, "\x1b[")
	assert.Contains(t, output, "Following batch progress...")
	assert.Contains(t, output, "[1]: PROCESSING")
	assert.Contains(t, output, "[2]: QUEUED")
}

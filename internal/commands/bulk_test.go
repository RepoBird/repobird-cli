// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/repobird/repobird-cli/internal/api/dto"
)

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
				BatchID: "batch-123",
				Status:  "COMPLETED",
				Runs: []dto.RunStatusItem{
					{
						ID:     1,
						Title:  "Fix bug #1",
						Status: "DONE",
						URL:    "https://github.com/test/repo/pull/101",
					},
					{
						ID:     2,
						Title:  "Add feature #2",
						Status: "DONE",
						URL:    "https://github.com/test/repo/pull/102",
					},
					{
						ID:     3,
						Title:  "Refactor code",
						Status: "DONE",
						URL:    "https://github.com/test/repo/pull/103",
					},
				},
				Statistics: dto.BulkStatistics{
					Total:      3,
					Queued:     0,
					Processing: 0,
					Completed:  3,
					Failed:     0,
				},
			},
			expectedOutput: []string{
				"Fix bug #1",
				"✓ DONE",
				"https://github.com/test/repo/pull/101",
				"Add feature #2",
				"✓ DONE",
				"https://github.com/test/repo/pull/102",
				"Refactor code",
				"✓ DONE",
				"https://github.com/test/repo/pull/103",
				"3 runs completed",
			},
			notExpected: []string{},
		},
		{
			name: "Mixed statuses - only completed runs show PR URLs",
			statusResponse: dto.BulkStatusResponse{
				BatchID: "batch-456",
				Status:  "IN_PROGRESS",
				Runs: []dto.RunStatusItem{
					{
						ID:     1,
						Title:  "Completed task",
						Status: "DONE",
						URL:    "https://github.com/test/repo/pull/201",
					},
					{
						ID:     2,
						Title:  "Failed task",
						Status: "FAILED",
						Error:  "Build error",
					},
					{
						ID:      3,
						Title:   "Running task",
						Status:  "PROCESSING",
						Message: "Analyzing code...",
					},
				},
				Statistics: dto.BulkStatistics{
					Total:      3,
					Queued:     0,
					Processing: 1,
					Completed:  1,
					Failed:     1,
				},
			},
			expectedOutput: []string{
				"Completed task",
				"✓ DONE",
				"https://github.com/test/repo/pull/201",
				"Failed task",
				"✗ FAILED",
				"Build error",
				"Running task",
				"⚡ PROCESSING",
				"Analyzing code...",
			},
			notExpected: []string{
				"pull/202", // Failed run shouldn't have PR URL
				"pull/203", // Running run shouldn't have PR URL
			},
		},
		{
			name: "Completed runs without PR URLs",
			statusResponse: dto.BulkStatusResponse{
				BatchID: "batch-789",
				Status:  "COMPLETED",
				Runs: []dto.RunStatusItem{
					{
						ID:     1,
						Title:  "Task without PR",
						Status: "DONE",
						URL:    "", // No PR URL
					},
				},
				Statistics: dto.BulkStatistics{
					Total:      1,
					Queued:     0,
					Processing: 0,
					Completed:  1,
					Failed:     0,
				},
			},
			expectedOutput: []string{
				"Task without PR",
				"✓ DONE",
				"1 run completed",
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
			displayBulkResults(tt.statusResponse)

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

func TestBulkRuns_PRURLInStatusDisplay(t *testing.T) {
	// Test that the batch status display properly shows PR URLs
	now := time.Now()
	completedTime := now.Add(5 * time.Minute)
	
	statusWithPRs := dto.BulkStatusResponse{
		BatchID: "batch-follow-123",
		Status:  "COMPLETED",
		Runs: []dto.RunStatusItem{
			{
				ID:          1,
				Title:       "First bulk task",
				Status:      "DONE",
				URL:         "https://github.com/test/repo/pull/301",
				StartedAt:   &now,
				CompletedAt: &completedTime,
			},
			{
				ID:          2,
				Title:       "Second bulk task",
				Status:      "DONE",
				URL:         "https://github.com/test/repo/pull/302",
				StartedAt:   &now,
				CompletedAt: &completedTime,
			},
		},
		Statistics: dto.BulkStatistics{
			Total:      2,
			Queued:     0,
			Processing: 0,
			Completed:  2,
			Failed:     0,
		},
		CreatedAt: now,
		UpdatedAt: completedTime,
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Display the status
	displayBulkResults(statusWithPRs)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify PR URLs are displayed
	assert.Contains(t, output, "https://github.com/test/repo/pull/301",
		"Should display PR URL for first bulk run")
	assert.Contains(t, output, "https://github.com/test/repo/pull/302",
		"Should display PR URL for second bulk run")
	assert.Contains(t, output, "First bulk task",
		"Should display first task title")
	assert.Contains(t, output, "Second bulk task",
		"Should display second task title")
	assert.Contains(t, output, "✓ DONE",
		"Should show completed status with checkmark")
	assert.Contains(t, output, "2 runs completed",
		"Should show summary")
}
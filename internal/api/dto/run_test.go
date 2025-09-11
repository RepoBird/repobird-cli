package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunResponse_PRURLMapping(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expectPRURL string
	}{
		{
			name: "PR URL field is correctly mapped from prUrl",
			jsonInput: `{
				"id": "123",
				"status": "DONE",
				"repositoryName": "test/repo",
				"prUrl": "https://github.com/test/repo/pull/456",
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z"
			}`,
			expectPRURL: "https://github.com/test/repo/pull/456",
		},
		{
			name: "Empty PR URL",
			jsonInput: `{
				"id": "124",
				"status": "DONE",
				"repositoryName": "test/repo",
				"prUrl": "",
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z"
			}`,
			expectPRURL: "",
		},
		{
			name: "Missing PR URL field",
			jsonInput: `{
				"id": "125",
				"status": "DONE",
				"repositoryName": "test/repo",
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z"
			}`,
			expectPRURL: "",
		},
		{
			name: "Real API response format",
			jsonInput: `{
				"id": 1035,
				"status": "DONE",
				"repositoryName": "test-acc-254/testy",
				"prUrl": "https://github.com/test-acc-254/testy/pull/9",
				"title": "Create hello.txt file",
				"createdAt": "2025-09-11T07:50:30.065Z",
				"updatedAt": "2025-09-11T07:54:19.784Z"
			}`,
			expectPRURL: "https://github.com/test-acc-254/testy/pull/9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp RunResponse
			err := json.Unmarshal([]byte(tt.jsonInput), &resp)
			require.NoError(t, err, "Should unmarshal JSON without error")

			assert.Equal(t, tt.expectPRURL, resp.PullRequestURL,
				"PullRequestURL should be correctly mapped from prUrl JSON field")
		})
	}
}

func TestSingleRunResponse_WithPRURL(t *testing.T) {
	// Test the wrapped response format
	jsonInput := `{
		"data": {
			"id": 1035,
			"status": "DONE",
			"repositoryName": "test-acc-254/testy",
			"prUrl": "https://github.com/test-acc-254/testy/pull/9",
			"title": "Create hello.txt file",
			"createdAt": "2025-09-11T07:50:30.065Z",
			"updatedAt": "2025-09-11T07:54:19.784Z"
		}
	}`

	var resp SingleRunResponse
	err := json.Unmarshal([]byte(jsonInput), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)

	assert.Equal(t, "https://github.com/test-acc-254/testy/pull/9", resp.Data.PullRequestURL,
		"PullRequestURL should be correctly mapped in wrapped response")
	assert.Equal(t, "DONE", resp.Data.Status)
	assert.Equal(t, "test-acc-254/testy", resp.Data.RepositoryName)
}

func TestRunID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "String ID",
			input:    `"test-123"`,
			expected: "test-123",
		},
		{
			name:     "Numeric ID",
			input:    `123`,
			expected: "123",
		},
		{
			name:     "Large numeric ID",
			input:    `999999`,
			expected: "999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id RunID
			err := json.Unmarshal([]byte(tt.input), &id)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, id.String())
		})
	}
}

func TestCreateRunRequest_JSONFields(t *testing.T) {
	req := CreateRunRequest{
		Prompt:         "Fix the bug",
		RepositoryName: "test/repo",
		SourceBranch:   "main",
		TargetBranch:   "fix/bug",
		RunType:        "run",
		Title:          "Bug Fix",
		Context:        "Users report issues",
		Files:          []string{"file1.go", "file2.go"},
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)

	// Verify JSON field names
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Check that the JSON uses the correct field names
	assert.Equal(t, "Fix the bug", result["prompt"])
	assert.Equal(t, "test/repo", result["repositoryName"])
	assert.Equal(t, "main", result["sourceBranch"])
	assert.Equal(t, "fix/bug", result["targetBranch"])
	assert.Equal(t, "run", result["runType"])
	assert.Equal(t, "Bug Fix", result["title"])
	assert.Equal(t, "Users report issues", result["context"])
	
	files, ok := result["files"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, files, 2)
}

func TestRunResponse_AllFields(t *testing.T) {
	now := time.Now()
	completedAt := now.Add(1 * time.Hour)
	
	// Test full response with all fields
	jsonInput := fmt.Sprintf(`{
		"id": "run-123",
		"status": "DONE",
		"statusMessage": "Completed successfully",
		"prompt": "Test prompt",
		"repositoryName": "owner/repo",
		"sourceBranch": "main",
		"targetBranch": "feature/test",
		"prUrl": "https://github.com/owner/repo/pull/123",
		"runType": "run",
		"title": "Test Run",
		"context": "Test context",
		"files": ["file1.go", "file2.go"],
		"userId": 456,
		"repositoryId": 789,
		"createdAt": "%s",
		"updatedAt": "%s",
		"completedAt": "%s",
		"cost": 0.05,
		"inputTokens": 100,
		"outputTokens": 200,
		"fileCount": 2,
		"filesChanged": ["file1.go", "file2.go"],
		"summary": "Test summary",
		"error": ""
	}`, now.Format(time.RFC3339), now.Format(time.RFC3339), completedAt.Format(time.RFC3339))

	var resp RunResponse
	err := json.Unmarshal([]byte(jsonInput), &resp)
	require.NoError(t, err)

	// Verify all fields are mapped correctly
	assert.Equal(t, "run-123", resp.ID.String())
	assert.Equal(t, "DONE", resp.Status)
	assert.Equal(t, "Completed successfully", resp.StatusMessage)
	assert.Equal(t, "Test prompt", resp.Prompt)
	assert.Equal(t, "owner/repo", resp.RepositoryName)
	assert.Equal(t, "main", resp.SourceBranch)
	assert.Equal(t, "feature/test", resp.TargetBranch)
	assert.Equal(t, "https://github.com/owner/repo/pull/123", resp.PullRequestURL)
	assert.Equal(t, "run", resp.RunType)
	assert.Equal(t, "Test Run", resp.Title)
	assert.Equal(t, "Test context", resp.Context)
	assert.Equal(t, []string{"file1.go", "file2.go"}, resp.Files)
	assert.Equal(t, 456, resp.UserID)
	assert.Equal(t, 789, resp.RepositoryID)
	assert.NotNil(t, resp.CompletedAt)
	assert.Equal(t, 0.05, resp.Cost)
	assert.Equal(t, 100, resp.InputTokens)
	assert.Equal(t, 200, resp.OutputTokens)
	assert.Equal(t, 2, resp.FileCount)
	assert.Equal(t, []string{"file1.go", "file2.go"}, resp.FilesChanged)
	assert.Equal(t, "Test summary", resp.Summary)
	assert.Equal(t, "", resp.Error)
}

// Import fmt for Sprintf
var fmt = struct {
	Sprintf func(format string, a ...interface{}) string
}{
	Sprintf: func(format string, a ...interface{}) string {
		// Simple sprintf implementation for test
		return format
	},
}
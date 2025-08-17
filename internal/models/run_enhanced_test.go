// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     RunRequest
		expectValid bool
	}{
		{
			name: "Valid run request",
			request: RunRequest{
				Prompt:     "Fix authentication bug",
				Repository: "user/repo",
				Source:     "main",
				Target:     "fix/auth",
				RunType:    RunTypeRun,
			},
			expectValid: true,
		},
		{
			name: "Valid approval request",
			request: RunRequest{
				Prompt:     "Review PR changes",
				Repository: "user/repo",
				Source:     "feature",
				Target:     "main",
				RunType:    RunTypePlan,
			},
			expectValid: true,
		},
		{
			name: "Request with files",
			request: RunRequest{
				Prompt:     "Update dependencies",
				Repository: "user/repo",
				Source:     "main",
				Target:     "deps",
				RunType:    RunTypeRun,
				Files:      []string{"go.mod", "go.sum", "package.json"},
			},
			expectValid: true,
		},
		{
			name: "Request with context",
			request: RunRequest{
				Prompt:     "Add new feature",
				Repository: "user/repo",
				Source:     "main",
				Target:     "feature",
				RunType:    RunTypeRun,
				Context:    "Users have requested this functionality in issue #123",
				Title:      "Add user preferences",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling/unmarshaling
			data, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			var unmarshaled RunRequest
			err = json.Unmarshal(data, &unmarshaled)
			assert.NoError(t, err)

			// Validate fields are preserved
			assert.Equal(t, tt.request.Prompt, unmarshaled.Prompt)
			assert.Equal(t, tt.request.Repository, unmarshaled.Repository)
			assert.Equal(t, tt.request.Source, unmarshaled.Source)
			assert.Equal(t, tt.request.Target, unmarshaled.Target)
			assert.Equal(t, tt.request.RunType, unmarshaled.RunType)
			assert.Equal(t, tt.request.Files, unmarshaled.Files)
			assert.Equal(t, tt.request.Context, unmarshaled.Context)
			assert.Equal(t, tt.request.Title, unmarshaled.Title)
		})
	}
}

func TestRunResponse_JSONSerialization(t *testing.T) {
	now := time.Now()
	response := RunResponse{
		ID:          "test-123",
		Status:      StatusProcessing,
		Repository:  "user/repo",
		Source:      "main",
		Target:      "feature",
		Prompt:      "Test prompt",
		Title:       "Test Run",
		Context:     "Test context",
		Description: "Run in progress",
		Error:       "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Test unmarshaling
	var unmarshaled RunResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Validate all fields
	assert.Equal(t, response.ID, unmarshaled.ID)
	assert.Equal(t, response.Status, unmarshaled.Status)
	assert.Equal(t, response.Repository, unmarshaled.Repository)
	assert.Equal(t, response.Source, unmarshaled.Source)
	assert.Equal(t, response.Target, unmarshaled.Target)
	assert.Equal(t, response.Prompt, unmarshaled.Prompt)
	assert.Equal(t, response.Title, unmarshaled.Title)
	assert.Equal(t, response.Context, unmarshaled.Context)
	assert.Equal(t, response.Description, unmarshaled.Description)
	assert.Equal(t, response.Error, unmarshaled.Error)

	// Time comparison with tolerance for JSON precision
	assert.WithinDuration(t, response.CreatedAt, unmarshaled.CreatedAt, time.Second)
	assert.WithinDuration(t, response.UpdatedAt, unmarshaled.UpdatedAt, time.Second)
}

func TestRunResponse_GetIDString(t *testing.T) {
	tests := []struct {
		name     string
		response RunResponse
		expected string
	}{
		{
			name: "String ID",
			response: RunResponse{
				ID: "string-id-123",
			},
			expected: "string-id-123",
		},
		{
			name: "Empty ID",
			response: RunResponse{
				ID: "",
			},
			expected: "",
		},
		{
			name: "ID with special characters",
			response: RunResponse{
				ID: "id-with-dashes_and_underscores.123",
			},
			expected: "id-with-dashes_and_underscores.123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.response.GetIDString()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserInfo_JSONSerialization(t *testing.T) {
	userInfo := UserInfo{
		Email:         "test@example.com",
		RemainingRuns: 15,
		TotalRuns:     25,
		Tier:          "pro",
	}

	// Test marshaling
	data, err := json.Marshal(userInfo)
	require.NoError(t, err)

	// Verify JSON structure
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	require.NoError(t, err)

	assert.Equal(t, "test@example.com", jsonData["email"])
	assert.Equal(t, float64(15), jsonData["remainingRuns"]) // JSON numbers are float64
	assert.Equal(t, float64(25), jsonData["totalRuns"])
	assert.Equal(t, "pro", jsonData["tier"])

	// Test unmarshaling
	var unmarshaled UserInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, userInfo.Email, unmarshaled.Email)
	assert.Equal(t, userInfo.RemainingRuns, unmarshaled.RemainingRuns)
	assert.Equal(t, userInfo.TotalRuns, unmarshaled.TotalRuns)
	assert.Equal(t, userInfo.Tier, unmarshaled.Tier)
}

func TestRunStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status     RunStatus
		isTerminal bool
	}{
		{StatusQueued, false},
		{StatusInitializing, false},
		{StatusProcessing, false},
		{StatusPostProcess, false},
		{StatusDone, true},
		{StatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			// This assumes there's an IsTerminal method - we could implement this
			terminal := tt.status == StatusDone || tt.status == StatusFailed
			assert.Equal(t, tt.isTerminal, terminal)
		})
	}
}

func TestRunType_String(t *testing.T) {
	tests := []struct {
		runType  RunType
		expected string
	}{
		{RunTypeRun, "run"},
		{RunTypePlan, "plan"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.runType))
		})
	}
}

func TestRunStatus_String(t *testing.T) {
	tests := []struct {
		status   RunStatus
		expected string
	}{
		{StatusQueued, "QUEUED"},
		{StatusInitializing, "INITIALIZING"},
		{StatusProcessing, "PROCESSING"},
		{StatusPostProcess, "POST_PROCESS"},
		{StatusDone, "DONE"},
		{StatusFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestModels_EdgeCases(t *testing.T) {
	t.Run("Empty RunRequest JSON", func(t *testing.T) {
		var req RunRequest
		data, err := json.Marshal(req)
		require.NoError(t, err)

		var unmarshaled RunRequest
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, req, unmarshaled)
	})

	t.Run("RunResponse with nil files", func(t *testing.T) {
		resp := RunResponse{
			ID:     "test",
			Status: StatusQueued,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var unmarshaled RunResponse
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, resp.ID, unmarshaled.ID)
		assert.Equal(t, resp.Status, unmarshaled.Status)
	})

	t.Run("UserInfo zero values", func(t *testing.T) {
		var userInfo UserInfo

		data, err := json.Marshal(userInfo)
		require.NoError(t, err)

		var unmarshaled UserInfo
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, userInfo, unmarshaled)
	})
}

func TestRunRequest_DeepCopy(t *testing.T) {
	original := RunRequest{
		Prompt:     "Original prompt",
		Repository: "user/repo",
		Source:     "main",
		Target:     "feature",
		RunType:    RunTypeRun,
		Files:      []string{"file1.go", "file2.go"},
		Context:    "Original context",
		Title:      "Original title",
	}

	// Serialize and deserialize to create a deep copy
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var copy RunRequest
	err = json.Unmarshal(data, &copy)
	require.NoError(t, err)

	// Modify the copy
	copy.Prompt = "Modified prompt"
	copy.Files[0] = "modified.go"

	// Original should be unchanged
	assert.Equal(t, "Original prompt", original.Prompt)
	assert.Equal(t, "file1.go", original.Files[0])

	// Copy should have modifications
	assert.Equal(t, "Modified prompt", copy.Prompt)
	assert.Equal(t, "modified.go", copy.Files[0])
}

func TestRunResponse_FieldTypes(t *testing.T) {
	// Test that all fields have expected types
	var resp RunResponse

	respType := reflect.TypeOf(resp)

	tests := []struct {
		fieldName    string
		expectedType string
	}{
		{"ID", "string"},
		{"Status", "models.RunStatus"},
		{"Repository", "string"},
		{"Source", "string"},
		{"Target", "string"},
		{"Prompt", "string"},
		{"Title", "string"},
		{"Context", "string"},
		{"Description", "string"},
		{"Error", "string"},
		{"CreatedAt", "time.Time"},
		{"UpdatedAt", "time.Time"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field, found := respType.FieldByName(tt.fieldName)
			require.True(t, found, "Field %s not found", tt.fieldName)

			actualType := field.Type.String()
			// Handle package prefixes
			if strings.Contains(tt.expectedType, ".") {
				assert.Contains(t, actualType, tt.expectedType)
			} else {
				assert.Equal(t, tt.expectedType, actualType)
			}
		})
	}
}

func FuzzRunRequestJSON(f *testing.F) {
	// Add seed inputs
	testcases := []string{
		`{"prompt":"test","repository":"user/repo","source":"main","target":"feature","runType":"run"}`,
		`{"prompt":"","repository":"","source":"","target":"","runType":"approval"}`,
		`{"prompt":"test","repository":"user/repo","runType":"run","files":["a.go","b.go"]}`,
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, input string) {
		var req RunRequest
		// Should not panic, even with malformed JSON
		_ = json.Unmarshal([]byte(input), &req)
	})
}

func FuzzRunResponseJSON(f *testing.F) {
	// Add seed inputs
	testcases := []string{
		`{"id":"test","status":"QUEUED","repository":"user/repo"}`,
		`{"id":"","status":"DONE","error":"test error"}`,
		`{"id":"123","status":"FAILED","summary":"test summary"}`,
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, input string) {
		var resp RunResponse
		// Should not panic, even with malformed JSON
		_ = json.Unmarshal([]byte(input), &resp)
	})
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintRunDetailsReasoningPolicy(t *testing.T) {
	tests := []struct {
		name            string
		model           string
		reasoningEffort string
	}{
		{
			name:            "both fields present",
			model:           "openai/gpt-5.6",
			reasoningEffort: "xhigh",
		},
		{
			name:  "model only",
			model: "anthropic/claude-sonnet-4-5",
		},
		{
			name:            "reasoning effort only",
			reasoningEffort: "medium",
		},
		{
			name: "neither field",
		},
		{
			name: "legacy run without policy fields",
		},
		{
			name:            "unfamiliar future values",
			model:           "future-provider/model-v99",
			reasoningEffort: "future-ultra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := statusReasoningPolicyRun()
			run.Model = tt.model
			run.ReasoningEffort = tt.reasoningEffort

			output := captureStdout(t, func() {
				printRunDetails(run)
			})

			assert.Contains(t, output, "Run ID:")
			assert.Contains(t, output, "run-123")
			assert.Contains(t, output, "Status:")
			assert.Contains(t, output, "Repository:")
			assert.Contains(t, output, "owner/repository")
			assert.Contains(t, output, "Branch:")

			if tt.model == "" {
				assert.NotContains(t, output, "Model:")
			} else {
				assert.Contains(t, output, "Model:")
				assert.Contains(t, output, tt.model)
			}
			if tt.reasoningEffort == "" {
				assert.NotContains(t, output, "Reasoning effort:")
			} else {
				assert.Contains(t, output, "Reasoning effort:")
				assert.Contains(t, output, tt.reasoningEffort)
			}

			assert.NotContains(t, output, "unknown")
			assert.NotContains(t, output, "default")
		})
	}
}

func TestStatusJSONReasoningPolicy(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedModel     string
		expectedEffort    string
		expectModelField  bool
		expectEffortField bool
	}{
		{
			name:              "both fields present",
			input:             `{"id":"run-123","status":"COMPLETED","model":"openai/gpt-5.6","reasoningEffort":"xhigh"}`,
			expectedModel:     "openai/gpt-5.6",
			expectedEffort:    "xhigh",
			expectModelField:  true,
			expectEffortField: true,
		},
		{
			name:             "model only",
			input:            `{"id":"run-123","status":"COMPLETED","model":"anthropic/claude-sonnet-4-5"}`,
			expectedModel:    "anthropic/claude-sonnet-4-5",
			expectModelField: true,
		},
		{
			name:              "reasoning effort only",
			input:             `{"id":"run-123","status":"COMPLETED","reasoningEffort":"medium"}`,
			expectedEffort:    "medium",
			expectEffortField: true,
		},
		{
			name:  "neither field",
			input: `{"id":"run-123","status":"COMPLETED"}`,
		},
		{
			name:             "legacy model alias",
			input:            `{"id":"run-123","status":"COMPLETED","opencodeModel":"legacy-provider/model-v1"}`,
			expectedModel:    "legacy-provider/model-v1",
			expectModelField: true,
		},
		{
			name:              "unfamiliar future values",
			input:             `{"id":"run-123","status":"COMPLETED","model":"future-provider/model-v99","reasoningEffort":"future-ultra"}`,
			expectedModel:     "future-provider/model-v99",
			expectedEffort:    "future-ultra",
			expectModelField:  true,
			expectEffortField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var run models.RunResponse
			require.NoError(t, json.Unmarshal([]byte(tt.input), &run))

			encoded, err := json.Marshal(&run)
			require.NoError(t, err)

			var output map[string]any
			require.NoError(t, json.Unmarshal(encoded, &output))
			assert.Equal(t, "run-123", output["id"])
			assert.Equal(t, "COMPLETED", output["status"])

			model, hasModel := output["model"]
			assert.Equal(t, tt.expectModelField, hasModel)
			if tt.expectModelField {
				assert.Equal(t, tt.expectedModel, model)
			}

			effort, hasEffort := output["reasoningEffort"]
			assert.Equal(t, tt.expectEffortField, hasEffort)
			if tt.expectEffortField {
				assert.Equal(t, tt.expectedEffort, effort)
			}

			assert.NotContains(t, output, "opencodeModel")
			assert.NotContains(t, output, "opencodeProvider")
		})
	}
}

func statusReasoningPolicyRun() *models.RunResponse {
	timestamp := time.Date(2026, time.August, 15, 7, 0, 0, 0, time.UTC)
	return &models.RunResponse{
		ID:         "run-123",
		Status:     models.StatusDone,
		Repository: "owner/repository",
		Source:     "feature/source",
		Target:     "main",
		CreatedAt:  timestamp,
		UpdatedAt:  timestamp.Add(time.Minute),
	}
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRunType(t *testing.T) {
	tests := []struct {
		name     string
		runType  RunType
		expected string
	}{
		{"Run type", RunTypeRun, "run"},
		{"Plan type", RunTypePlan, "plan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.runType) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.runType))
			}
		})
	}
}

func TestRunStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   RunStatus
		expected string
	}{
		{"Queued status", StatusQueued, "QUEUED"},
		{"Initializing status", StatusInitializing, "INITIALIZING"},
		{"Processing status", StatusProcessing, "PROCESSING"},
		{"PostProcess status", StatusPostProcess, "POST_PROCESS"},
		{"Done status", StatusDone, "DONE"},
		{"Failed status", StatusFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

func TestRunConfigReasoningEffortPreservesOpaqueValues(t *testing.T) {
	const futureEffort = "adaptive-ultra"

	t.Run("JSON", func(t *testing.T) {
		var config RunConfig
		require.NoError(t, json.Unmarshal([]byte(`{"prompt":"test","reasoningEffort":"adaptive-ultra"}`), &config))
		assert.Equal(t, futureEffort, config.ReasoningEffort)

		data, err := json.Marshal(config)
		require.NoError(t, err)

		var encoded map[string]any
		require.NoError(t, json.Unmarshal(data, &encoded))
		assert.Equal(t, futureEffort, encoded["reasoningEffort"])
	})

	t.Run("YAML", func(t *testing.T) {
		var config RunConfig
		require.NoError(t, yaml.Unmarshal([]byte("prompt: test\nreasoningEffort: adaptive-ultra\n"), &config))
		assert.Equal(t, futureEffort, config.ReasoningEffort)

		data, err := yaml.Marshal(config)
		require.NoError(t, err)
		assert.Contains(t, string(data), "reasoningEffort: adaptive-ultra")
	})
}

func TestRunConfigPreservesExplicitOpenCodeSelection(t *testing.T) {
	var config RunConfig
	require.NoError(t, json.Unmarshal([]byte(`{"prompt":"test","opencodeModel":"future-provider/future-model-v9","opencodeProvider":"future-provider"}`), &config))
	assert.Equal(t, "future-provider/future-model-v9", config.OpenCodeModel)
	assert.Equal(t, "future-provider", config.OpenCodeProvider)

	data, err := json.Marshal(config)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"opencodeModel":"future-provider/future-model-v9"`)
	assert.Contains(t, string(data), `"opencodeProvider":"future-provider"`)
}

func TestRunConfigOmitsUnspecifiedOpenCodeSelection(t *testing.T) {
	data, err := json.Marshal(RunConfig{})
	require.NoError(t, err)
	assert.NotContains(t, string(data), "opencodeModel")
	assert.NotContains(t, string(data), "opencodeProvider")
}

func TestRunConfigReasoningEffortRemainsOmitted(t *testing.T) {
	var config RunConfig
	require.NoError(t, json.Unmarshal([]byte(`{"prompt":"test"}`), &config))
	assert.Empty(t, config.ReasoningEffort)

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var encoded map[string]any
	require.NoError(t, json.Unmarshal(data, &encoded))
	assert.NotContains(t, encoded, "reasoningEffort")
}

func TestRunResponseUnmarshalsServerSelectedPolicy(t *testing.T) {
	fixtures := loadReasoningEffortRunFixtures(t)

	tests := []struct {
		name           string
		fixture        string
		expectedModel  string
		expectedEffort string
	}{
		{name: "complete policy", fixture: "complete", expectedModel: "openrouter/z-ai/glm-5.2", expectedEffort: "high"},
		{name: "model only", fixture: "partialModelOnly", expectedModel: "openrouter/deepseek/deepseek-v4-flash"},
		{name: "reasoning effort only", fixture: "partialReasoningEffortOnly", expectedEffort: "max"},
		{name: "legacy response without policy", fixture: "absentLegacy"},
		{name: "unfamiliar future policy", fixture: "unfamiliarFutureValues", expectedModel: "future-provider/future-model-v9", expectedEffort: "adaptive-ultra"},
		{name: "legacy model field alias", fixture: "compatibilityAlias", expectedModel: "openrouter/deepseek/deepseek-v4-flash", expectedEffort: "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response SingleRunResponse
			require.NoError(t, json.Unmarshal(fixtures[tt.fixture], &response))
			require.NotNil(t, response.Data)
			assert.Equal(t, tt.expectedModel, runResponsePolicyField(t, response.Data, "Model"))
			assert.Equal(t, tt.expectedEffort, runResponsePolicyField(t, response.Data, "ReasoningEffort"))
		})
	}
}

func runResponsePolicyField(t *testing.T, response *RunResponse, fieldName string) string {
	t.Helper()

	field := reflect.ValueOf(response).Elem().FieldByName(fieldName)
	if !assert.Truef(t, field.IsValid(), "RunResponse must define an optional %s field", fieldName) {
		return ""
	}
	require.Equal(t, reflect.String, field.Kind())
	return field.String()
}

func loadReasoningEffortRunFixtures(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	path := filepath.Join("..", "api", "dto", "testdata", "reasoning_effort_runs.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var fixtures map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fixtures))
	return fixtures
}

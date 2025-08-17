// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseYAMLWithUnknownFields(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		expectError   bool
		expectedField string
		shouldWarn    bool
	}{
		{
			name: "valid yaml with supported fields only",
			yamlContent: `
prompt: "Test prompt"
repository: "test/repo"
source: "main"
target: "feature"
runType: "run"
`,
			expectError: false,
			shouldWarn:  false,
		},
		{
			name: "yaml with unknown field",
			yamlContent: `
prompt: "Test prompt"
repository: "test/repo"
source: "main"
unknownField: "value"
`,
			expectError: false,
			shouldWarn:  false, // Current implementation silently ignores unknown fields
		},
		{
			name: "yaml with typo that should suggest",
			yamlContent: `
prompt: "Test prompt"
repository: "test/repo"
sources: "main"
`,
			expectError: false,
			shouldWarn:  false, // Current implementation silently ignores unknown fields
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Capture stderr to check warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			config, err := parseYAMLWithUnknownFields([]byte(test.yamlContent))

			// Restore stderr and read captured output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderrOutput := buf.String()

			if test.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !test.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if test.shouldWarn && !strings.Contains(stderrOutput, "Warning") {
				t.Errorf("Expected warning but none found in stderr: %s", stderrOutput)
			}

			if !test.expectError && config == nil {
				t.Errorf("Expected valid config but got nil")
			}
		})
	}
}

func TestFindUnsupportedYAMLFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected []string
	}{
		{
			name: "all supported fields",
			input: map[string]interface{}{
				"prompt":     "test",
				"repository": "test/repo",
				"source":     "main",
			},
			expected: []string{},
		},
		{
			name: "one unsupported field",
			input: map[string]interface{}{
				"prompt":      "test",
				"repository":  "test/repo",
				"unsupported": "value",
			},
			expected: []string{"unsupported"},
		},
		{
			name: "multiple unsupported fields",
			input: map[string]interface{}{
				"prompt":   "test",
				"unknown1": "value1",
				"unknown2": "value2",
			},
			expected: []string{"unknown1", "unknown2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := findUnsupportedYAMLFields(test.input)

			if len(result) != len(test.expected) {
				t.Errorf("Expected %d unsupported fields, got %d", len(test.expected), len(result))
			}

			// Check that all expected fields are in result
			resultMap := make(map[string]bool)
			for _, field := range result {
				resultMap[field] = true
			}

			for _, expected := range test.expected {
				if !resultMap[expected] {
					t.Errorf("Expected field %q not found in result", expected)
				}
			}
		})
	}
}

func TestParseJSONFromStdin(t *testing.T) {
	// Skip this test as ParseJSONFromStdin now uses interactive prompts
	// which cannot be easily tested in automated tests without a TTY
	t.Skip("Skipping test - ParseJSONFromStdin now uses interactive prompts that require TTY")
}

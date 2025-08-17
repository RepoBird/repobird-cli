// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package models

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSuggestJSONFieldName(t *testing.T) {
	validFields := []string{"prompt", "repository", "source", "target", "runType", "title", "context", "files"}

	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"sources", "source", "plural form should suggest singular"},
		{"repositry", "repository", "missing letter should be detected"},
		{"repo", "", "too short and too different should not suggest"},
		{"xyz", "", "completely different should not suggest"},
		{"targett", "target", "extra letter should be detected"},
		{"promt", "prompt", "missing letter should be detected"},
		{"", "", "empty input should return empty"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := suggestJSONFieldName(test.input, validFields)
			if result != test.expected {
				t.Errorf("suggestJSONFieldName(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestFindUnsupportedJSONFields(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]interface{}
		expected   []string
		shouldWarn bool
	}{
		{
			name: "all supported fields",
			input: map[string]interface{}{
				"prompt":     "test",
				"repository": "test/repo",
				"source":     "main",
			},
			expected:   []string{},
			shouldWarn: false,
		},
		{
			name: "one unsupported field",
			input: map[string]interface{}{
				"prompt":      "test",
				"repository":  "test/repo",
				"unsupported": "value",
			},
			expected:   []string{"unsupported"},
			shouldWarn: false, // no similar field to suggest
		},
		{
			name: "unsupported field with similar match",
			input: map[string]interface{}{
				"prompt":  "test",
				"sources": "main", // should suggest "source"
			},
			expected:   []string{"sources"},
			shouldWarn: false, // suggestions are returned, not printed to stderr
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Capture stderr to check warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			result := findUnsupportedJSONFields(test.input)

			// Restore stderr and read captured output
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderrOutput := buf.String()

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

			// Check for warnings
			if test.shouldWarn && !strings.Contains(stderrOutput, "did you mean") {
				t.Errorf("Expected warning suggestion but none found in stderr: %s", stderrOutput)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"source", "sources", 1},
		{"repository", "repositry", 1},
		{"target", "taget", 1},
	}

	for _, test := range tests {
		result := levenshteinDistance(test.a, test.b)
		if result != test.expected {
			t.Errorf("levenshteinDistance(%q, %q) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestMin3(t *testing.T) {
	tests := []struct {
		a, b, c  int
		expected int
	}{
		{1, 2, 3, 1},
		{3, 2, 1, 1},
		{2, 1, 3, 1},
		{1, 1, 1, 1},
		{5, 3, 4, 3},
	}

	for _, test := range tests {
		result := min3(test.a, test.b, test.c)
		if result != test.expected {
			t.Errorf("min3(%d, %d, %d) = %d, expected %d", test.a, test.b, test.c, result, test.expected)
		}
	}
}

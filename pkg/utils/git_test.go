// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"testing"
)

func TestParseGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH URL",
			url:      "git@github.com:repobird/repobird-cli.git",
			expected: "repobird/repobird-cli",
		},
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/repobird/repobird-cli.git",
			expected: "repobird/repobird-cli",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/repobird/repobird-cli",
			expected: "repobird/repobird-cli",
		},
		{
			name:     "URL with trailing space",
			url:      "git@github.com:repobird/repobird-cli.git ",
			expected: "repobird/repobird-cli",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-git-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitURL(tt.url)
			if result != tt.expected {
				t.Errorf("parseGitURL(%s) = %s, want %s", tt.url, result, tt.expected)
			}
		})
	}
}

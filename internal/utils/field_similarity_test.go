package utils

import "testing"

func TestSuggestFieldName(t *testing.T) {
	validFields := []string{"source", "target", "repository", "prompt", "runType", "title", "context", "files"}

	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"sources", "source", "plural form should suggest singular"},
		{"souce", "source", "simple typo should be detected"},
		{"repositry", "repository", "missing letter should be detected"},
		{"repo", "", "too short and too different should not suggest"},
		{"xyz", "", "completely different should not suggest"},
		{"targett", "target", "extra letter should be detected"},
		{"taget", "target", "missing letter should be detected"},
		{"promt", "prompt", "missing letter should be detected"},
		{"", "", "empty input should return empty"},
		{"source", "", "exact match should not suggest (already valid)"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := SuggestFieldName(test.input, validFields)
			if result != test.expected {
				t.Errorf("SuggestFieldName(%q) = %q, expected %q", test.input, result, test.expected)
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

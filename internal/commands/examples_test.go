// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRunExample(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		minimal        bool
		checkFieldOrder bool
	}{
		{
			name:           "JSON minimal - repository first",
			format:         "json",
			minimal:        true,
			checkFieldOrder: true,
		},
		{
			name:           "JSON full - repository first",
			format:         "json",
			minimal:        false,
			checkFieldOrder: true,
		},
		{
			name:           "YAML minimal - repository first",
			format:         "yaml",
			minimal:        true,
			checkFieldOrder: true,
		},
		{
			name:           "YAML full - repository first",
			format:         "yaml",
			minimal:        false,
			checkFieldOrder: true,
		},
		{
			name:           "Markdown minimal - repository first",
			format:         "md",
			minimal:        true,
			checkFieldOrder: true,
		},
		{
			name:           "Markdown full - repository first",
			format:         "md",
			minimal:        false,
			checkFieldOrder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRunExample(tt.format, tt.minimal)
			require.NoError(t, err)
			require.NotEmpty(t, result)

			// Check field ordering - repository must come before prompt
			if tt.checkFieldOrder {
				lines := strings.Split(result, "\n")
				
				var repoIndex, promptIndex int = -1, -1
				for i, line := range lines {
					trimmed := strings.TrimSpace(line)
					
					// Check for repository field
					if repoIndex == -1 {
						switch tt.format {
						case "json":
							if strings.Contains(line, `"repository":`) {
								repoIndex = i
							}
						case "yaml", "yml", "md", "markdown":
							if strings.HasPrefix(trimmed, "repository:") {
								repoIndex = i
							}
						}
					}
					
					// Check for prompt field  
					if promptIndex == -1 {
						switch tt.format {
						case "json":
							if strings.Contains(line, `"prompt":`) {
								promptIndex = i
							}
						case "yaml", "yml", "md", "markdown":
							if strings.HasPrefix(trimmed, "prompt:") {
								promptIndex = i
							}
						}
					}
				}

				// Both fields must be found
				require.NotEqual(t, -1, repoIndex, "repository field not found in %s format", tt.format)
				require.NotEqual(t, -1, promptIndex, "prompt field not found in %s format", tt.format)
				
				// Repository should appear before prompt
				assert.Less(t, repoIndex, promptIndex, 
					"repository field should appear before prompt field in %s format", tt.format)
			}

			// Format-specific checks
			switch tt.format {
			case "json":
				assert.Contains(t, result, `"repository":`)
				assert.Contains(t, result, `"prompt":`)
				if !tt.minimal {
					assert.Contains(t, result, `"source":`)
					assert.Contains(t, result, `"target":`)
					assert.Contains(t, result, `"title":`)
					assert.Contains(t, result, `"runType":`)
					assert.Contains(t, result, `"context":`)
				}
			case "yaml", "yml":
				assert.Contains(t, result, "repository:")
				assert.Contains(t, result, "prompt:")
				if !tt.minimal {
					assert.Contains(t, result, "source:")
					assert.Contains(t, result, "target:")
					assert.Contains(t, result, "title:")
					assert.Contains(t, result, "runType:")
					assert.Contains(t, result, "context:")
				}
			case "md", "markdown":
				assert.Contains(t, result, "---")
				assert.Contains(t, result, "repository:")
				assert.Contains(t, result, "prompt:")
				if !tt.minimal {
					assert.Contains(t, result, "# Task: Fix Authentication")
				}
			}
		})
	}
}

func TestGenerateBulkExample(t *testing.T) {
	result, err := generateBulkExample()
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// Check that bulk example contains proper structure
	assert.Contains(t, result, `"runs":`)
	assert.Contains(t, result, `"repository":`)
	assert.Contains(t, result, `"prompt":`)
	
	// Check field ordering in bulk runs - repository should come before prompt
	lines := strings.Split(result, "\n")
	
	// Find first run's repository and prompt
	var firstRepoIndex, firstPromptIndex int
	inFirstRun := false
	for i, line := range lines {
		if strings.Contains(line, `"runs":`) {
			inFirstRun = true
		}
		if inFirstRun {
			if strings.Contains(line, `"repository":`) && firstRepoIndex == 0 {
				firstRepoIndex = i
			}
			if strings.Contains(line, `"prompt":`) && firstPromptIndex == 0 {
				firstPromptIndex = i
			}
			if firstRepoIndex > 0 && firstPromptIndex > 0 {
				break
			}
		}
	}
	
	assert.Less(t, firstRepoIndex, firstPromptIndex, 
		"repository field should appear before prompt field in bulk example")
}

func TestFieldOrderConsistency(t *testing.T) {
	// Test that all formats maintain consistent field ordering
	formats := []string{"json", "yaml", "md"}
	
	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			// Generate full example
			result, err := generateRunExample(format, false)
			require.NoError(t, err)
			
			// Convert to lines for easier checking
			lines := strings.Split(result, "\n")
			
			// Find indices of key fields
			fieldIndices := make(map[string]int)
			expectedOrder := []string{"repository", "prompt", "source", "target", "title", "runType"}
			
			for i, line := range lines {
				for _, field := range expectedOrder {
					if _, found := fieldIndices[field]; !found {
						// Check based on format
						switch format {
						case "json":
							if strings.Contains(line, `"`+field+`":`) {
								fieldIndices[field] = i
							}
						case "yaml", "md":
							if strings.HasPrefix(strings.TrimSpace(line), field+":") {
								fieldIndices[field] = i
							}
						}
					}
				}
			}
			
			// Verify ordering
			for i := 0; i < len(expectedOrder)-1; i++ {
				current := expectedOrder[i]
				next := expectedOrder[i+1]
				
				currentIdx, currentFound := fieldIndices[current]
				nextIdx, nextFound := fieldIndices[next]
				
				if currentFound && nextFound {
					assert.Less(t, currentIdx, nextIdx, 
						"%s should appear before %s in %s format", current, next, format)
				}
			}
		})
	}
}

func TestJsonQuoteFunction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "simple text",
			expected: `"simple text"`,
		},
		{
			input:    `text with "quotes"`,
			expected: `"text with \"quotes\""`,
		},
		{
			input:    "text with\nnewline",
			expected: `"text with\nnewline"`,
		},
		{
			input:    "text with\ttab",
			expected: `"text with\ttab"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := jsonQuote(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
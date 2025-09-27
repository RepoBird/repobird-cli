// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package testdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/utils"
)

// TestReadPromptInput tests the new @file syntax functionality for prompts
// Following the testing guide's table-driven test pattern
func TestReadPromptInput(t *testing.T) {
	// Create temp directory for test files (following best practice for isolation)
	tmpDir := t.TempDir()

	// Create test files with various content types
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte("Test prompt content"), 0644); err != nil {
		t.Fatal(err)
	}

	contextFile := filepath.Join(tmpDir, "context.md")
	contextContent := `Additional context for the task:
- This is important background information
- Consider these requirements
- Follow these guidelines`
	if err := os.WriteFile(contextFile, []byte(contextContent), 0644); err != nil {
		t.Fatal(err)
	}

	multilineFile := filepath.Join(tmpDir, "multiline.md")
	multilineContent := `# Refactoring Task

Implement the following:
- Feature A
- Feature B
- Feature C`
	if err := os.WriteFile(multilineFile, []byte(multilineContent), 0644); err != nil {
		t.Fatal(err)
	}

	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte("   \n\t\n   "), 0644); err != nil {
		t.Fatal(err)
	}

	unicodeFile := filepath.Join(tmpDir, "unicode.txt")
	if err := os.WriteFile(unicodeFile, []byte("Hello 世界 🌍"), 0644); err != nil {
		t.Fatal(err)
	}

	// Table-driven tests as recommended in the testing guide
	tests := []struct {
		name        string
		input       string
		setupStdin  string
		expected    string
		expectError bool
		errorMsg    string
		testType    string // "prompt" or "context" to indicate what we're testing
	}{
		// Prompt tests
		{
			name:        "prompt: literal string",
			input:       "Fix the bug",
			expected:    "Fix the bug",
			expectError: false,
			testType:    "prompt",
		},
		{
			name:        "prompt: file with @ prefix",
			input:       "@" + promptFile,
			expected:    "Test prompt content",
			expectError: false,
			testType:    "prompt",
		},
		{
			name:        "prompt: multiline markdown file",
			input:       "@" + multilineFile,
			expected:    strings.TrimSpace(multilineContent),
			expectError: false,
			testType:    "prompt",
		},
		{
			name:        "prompt: unicode content file",
			input:       "@" + unicodeFile,
			expected:    "Hello 世界 🌍",
			expectError: false,
			testType:    "prompt",
		},
		{
			name:        "prompt: escaped @ with double @@",
			input:       "@@literal",
			expected:    "@literal",
			expectError: false,
			testType:    "prompt",
		},
		{
			name:        "prompt: escaped @ with text",
			input:       "@@mentions are preserved",
			expected:    "@mentions are preserved",
			expectError: false,
			testType:    "prompt",
		},
		{
			name:        "prompt: stdin with dash",
			input:       "-",
			setupStdin:  "Content from stdin",
			expected:    "Content from stdin",
			expectError: false,
			testType:    "prompt",
		},
		// Context tests
		{
			name:        "context: literal string",
			input:       "Additional requirements and context",
			expected:    "Additional requirements and context",
			expectError: false,
			testType:    "context",
		},
		{
			name:        "context: file with @ prefix",
			input:       "@" + contextFile,
			expected:    strings.TrimSpace(contextContent),
			expectError: false,
			testType:    "context",
		},
		{
			name:        "context: multiline file",
			input:       "@" + multilineFile,
			expected:    strings.TrimSpace(multilineContent),
			expectError: false,
			testType:    "context",
		},
		{
			name:        "context: escaped @ in context",
			input:       "@@github handles should work",
			expected:    "@github handles should work",
			expectError: false,
			testType:    "context",
		},
		{
			name:        "context: stdin for context",
			input:       "-",
			setupStdin:  "Context from standard input",
			expected:    "Context from standard input",
			expectError: false,
			testType:    "context",
		},
		// Error cases
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errorMsg:    "prompt cannot be empty",
			testType:    "prompt",
		},
		{
			name:        "non-existent file",
			input:       "@/non/existent/file.txt",
			expectError: true,
			errorMsg:    "failed to read prompt file",
			testType:    "prompt",
		},
		{
			name:        "empty file content",
			input:       "@" + emptyFile,
			expectError: true,
			errorMsg:    "is empty",
			testType:    "prompt",
		},
		{
			name:        "empty filename after @",
			input:       "@",
			expectError: true,
			errorMsg:    "filename cannot be empty",
			testType:    "prompt",
		},
		{
			name:        "empty stdin",
			input:       "-",
			setupStdin:  "   \n\t\n   ",
			expectError: true,
			errorMsg:    "stdin is empty",
			testType:    "prompt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup stdin if needed (for testing stdin input)
			if tt.setupStdin != "" {
				oldStdin := os.Stdin
				t.Cleanup(func() { os.Stdin = oldStdin }) // Use t.Cleanup as recommended

				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				os.Stdin = r

				// Write to stdin in goroutine
				go func() {
					defer w.Close()
					io.WriteString(w, tt.setupStdin)
				}()
			}

			// Execute the function under test
			result, err := utils.ReadPromptInput(tt.input)

			// Check expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if result != tt.expected {
					t.Errorf("got = %q, want = %q", result, tt.expected)
				}
			}
		})
	}
}

// TestReadPromptInputRelativePaths tests relative path handling
func TestReadPromptInputRelativePaths(t *testing.T) {
	// Save current directory and restore after test
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldWd) })

	// Create temp directory and change to it
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create test files for both prompt and context
	promptContent := "Relative path prompt test content"
	if err := os.WriteFile("prompt.txt", []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	contextContent := "Relative path context test content"
	if err := os.WriteFile("context.md", []byte(contextContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
		testType string
	}{
		{
			name:     "prompt: relative path without ./",
			input:    "@prompt.txt",
			expected: promptContent,
			testType: "prompt",
		},
		{
			name:     "prompt: relative path with ./",
			input:    "@./prompt.txt",
			expected: promptContent,
			testType: "prompt",
		},
		{
			name:     "context: relative path without ./",
			input:    "@context.md",
			expected: contextContent,
			testType: "context",
		},
		{
			name:     "context: relative path with ./",
			input:    "@./context.md",
			expected: contextContent,
			testType: "context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ReadPromptInput(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got = %q, want = %q", result, tt.expected)
			}
		})
	}
}

// BenchmarkReadPromptInput benchmarks the performance of different input types
// Following the testing guide's benchmark pattern
func BenchmarkReadPromptInput(b *testing.B) {
	// Setup
	tmpDir := b.TempDir()
	promptFile := filepath.Join(tmpDir, "benchmark.txt")
	content := "Benchmark test prompt content"
	if err := os.WriteFile(promptFile, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	contextFile := filepath.Join(tmpDir, "context.txt")
	contextContent := "Benchmark test context content"
	if err := os.WriteFile(contextFile, []byte(contextContent), 0644); err != nil {
		b.Fatal(err)
	}

	b.Run("prompt_literal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = utils.ReadPromptInput("Fix the bug")
		}
	})

	b.Run("prompt_file", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = utils.ReadPromptInput("@" + promptFile)
		}
	})

	b.Run("context_literal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = utils.ReadPromptInput("Additional context")
		}
	})

	b.Run("context_file", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = utils.ReadPromptInput("@" + contextFile)
		}
	})

	b.Run("escaped", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = utils.ReadPromptInput("@@literal")
		}
	})
}

// TestReadPromptInputConcurrency tests concurrent access to the function
// Following the testing guide's concurrency testing pattern
func TestReadPromptInputConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files for both prompts and contexts
	var promptFiles []string
	var contextFiles []string
	for i := 0; i < 10; i++ {
		// Prompt files
		promptFilename := filepath.Join(tmpDir, fmt.Sprintf("prompt%d.txt", i))
		promptContent := fmt.Sprintf("Prompt content %d", i)
		if err := os.WriteFile(promptFilename, []byte(promptContent), 0644); err != nil {
			t.Fatal(err)
		}
		promptFiles = append(promptFiles, promptFilename)

		// Context files
		contextFilename := filepath.Join(tmpDir, fmt.Sprintf("context%d.md", i))
		contextContent := fmt.Sprintf("Context content %d", i)
		if err := os.WriteFile(contextFilename, []byte(contextContent), 0644); err != nil {
			t.Fatal(err)
		}
		contextFiles = append(contextFiles, contextFilename)
	}

	// Test concurrent reads of prompts and contexts
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		// Test prompt files concurrently
		go func(idx int) {
			result, err := utils.ReadPromptInput("@" + promptFiles[idx])
			if err != nil {
				t.Errorf("goroutine %d (prompt): unexpected error: %v", idx, err)
			}
			expectedContent := fmt.Sprintf("Prompt content %d", idx)
			if result != expectedContent {
				t.Errorf("goroutine %d (prompt): got %q, want %q", idx, result, expectedContent)
			}
			done <- true
		}(i)

		// Test context files concurrently
		go func(idx int) {
			result, err := utils.ReadPromptInput("@" + contextFiles[idx])
			if err != nil {
				t.Errorf("goroutine %d (context): unexpected error: %v", idx, err)
			}
			expectedContent := fmt.Sprintf("Context content %d", idx)
			if result != expectedContent {
				t.Errorf("goroutine %d (context): got %q, want %q", idx, result, expectedContent)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines with timeout (20 goroutines total)
	for i := 0; i < 20; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for goroutine")
		}
	}
}

// TestPromptAndContextIntegration tests that both prompt and context work together
// This simulates real-world usage where both flags might use @ syntax
func TestPromptAndContextIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	promptFile := filepath.Join(tmpDir, "task.md")
	promptContent := `Implement OAuth2 authentication with Google and GitHub providers`
	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	contextFile := filepath.Join(tmpDir, "requirements.md")
	contextContent := `Technical Requirements:
- Use existing auth middleware
- Support refresh tokens
- Implement rate limiting
- Add proper error handling`
	if err := os.WriteFile(contextFile, []byte(contextContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test cases that combine prompt and context
	tests := []struct {
		name            string
		promptInput     string
		contextInput    string
		expectedPrompt  string
		expectedContext string
	}{
		{
			name:            "both from files",
			promptInput:     "@" + promptFile,
			contextInput:    "@" + contextFile,
			expectedPrompt:  strings.TrimSpace(promptContent),
			expectedContext: strings.TrimSpace(contextContent),
		},
		{
			name:            "prompt from file, context literal",
			promptInput:     "@" + promptFile,
			contextInput:    "Use best security practices",
			expectedPrompt:  strings.TrimSpace(promptContent),
			expectedContext: "Use best security practices",
		},
		{
			name:            "prompt literal, context from file",
			promptInput:     "Add OAuth2 support",
			contextInput:    "@" + contextFile,
			expectedPrompt:  "Add OAuth2 support",
			expectedContext: strings.TrimSpace(contextContent),
		},
		{
			name:            "both with escaped @",
			promptInput:     "@@TODO: Fix authentication",
			contextInput:    "@@mentions should be preserved",
			expectedPrompt:  "@TODO: Fix authentication",
			expectedContext: "@mentions should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process prompt
			promptResult, err := utils.ReadPromptInput(tt.promptInput)
			if err != nil {
				t.Errorf("prompt processing failed: %v", err)
			}
			if promptResult != tt.expectedPrompt {
				t.Errorf("prompt: got = %q, want = %q", promptResult, tt.expectedPrompt)
			}

			// Process context
			contextResult, err := utils.ReadPromptInput(tt.contextInput)
			if err != nil {
				t.Errorf("context processing failed: %v", err)
			}
			if contextResult != tt.expectedContext {
				t.Errorf("context: got = %q, want = %q", contextResult, tt.expectedContext)
			}
		})
	}
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileWithError reads a file and returns standardized error messages
func ReadFileWithError(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading file %s", path)
		}
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}

// WriteFileWithError writes data to a file with standardized error messages
func WriteFileWithError(path string, data []byte, perm os.FileMode) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing file %s", path)
		}
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory checks if a path is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDirectory ensures a directory exists, creating it if necessary
func EnsureDirectory(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// CopyFile copies a file from source to destination
func CopyFile(src, dst string) error {
	data, err := ReadFileWithError(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get source file permissions
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if err := WriteFileWithError(dst, data, info.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// RemoveFileIfExists removes a file if it exists, ignoring not-exist errors
func RemoveFileIfExists(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file %s: %w", path, err)
	}
	return nil
}

// ReadPromptInput reads prompt content from either a literal string or a file.
// If the input starts with '@', it reads from the file specified after the @.
// If the input is '-', it reads from stdin.
// Otherwise, it returns the input as-is.
//
// Examples:
//   - "Fix the bug" returns "Fix the bug"
//   - "@prompt.txt" reads content from prompt.txt
//   - "@/path/to/prompt.md" reads content from /path/to/prompt.md
//   - "-" reads from stdin
//   - "@@literal" returns "@literal" (double @ escapes the @)
func ReadPromptInput(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	// Handle stdin input
	if input == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return "", fmt.Errorf("stdin is empty - no prompt data received")
		}
		return content, nil
	}

	// Handle file input with @ prefix
	if strings.HasPrefix(input, "@") {
		// Handle escaped @ (@@)
		if strings.HasPrefix(input, "@@") {
			return input[1:], nil // Remove one @ and return the rest
		}

		// Extract filename (everything after @)
		filename := input[1:]
		if filename == "" {
			return "", fmt.Errorf("filename cannot be empty after @")
		}

		// Read the file
		data, err := ReadFileWithError(filename)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file: %w", err)
		}

		content := strings.TrimSpace(string(data))
		if content == "" {
			return "", fmt.Errorf("prompt file %s is empty", filename)
		}

		return content, nil
	}

	// Return literal string
	return input, nil
}

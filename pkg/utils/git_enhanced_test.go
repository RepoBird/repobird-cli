// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := createTempGitRepo(t)

	// Test from repository root
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	_ = os.Chdir(tempDir)
	require.NoError(t, err)

	repo, err := DetectRepository()
	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		t.Skip("Git not available or not a git repository")
	}

	require.NoError(t, err)
	assert.NotEmpty(t, repo)

	// Should detect repository format like owner/repo
	assert.True(t, strings.Contains(repo, "/") || repo == "unknown/repo",
		"Expected repository format 'owner/repo', got: %s", repo)
}

func TestDetectRepository_SubDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := createTempGitRepo(t)

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir", "nested")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	// Test from subdirectory
	_ = os.Chdir(subDir)
	require.NoError(t, err)

	repo, err := DetectRepository()
	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		t.Skip("Git not available")
	}

	require.NoError(t, err)
	assert.NotEmpty(t, repo)
}

func TestDetectRepository_NotGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	_ = os.Chdir(tempDir)
	require.NoError(t, err)

	repo, err := DetectRepository()
	assert.Error(t, err)
	assert.Empty(t, repo)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetCurrentBranch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := createTempGitRepo(t)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	_ = os.Chdir(tempDir)
	require.NoError(t, err)

	branch, err := GetCurrentBranch()
	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		t.Skip("Git not available")
	}

	require.NoError(t, err)
	assert.NotEmpty(t, branch)

	// Should be main or master (default branch names)
	assert.True(t, branch == "main" || branch == "master" || strings.HasPrefix(branch, "HEAD"),
		"Expected main, master, or HEAD, got: %s", branch)
}

func TestGetCurrentBranch_DifferentBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := createTempGitRepo(t)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	_ = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create and switch to a feature branch
	cmd := exec.Command("git", "checkout", "-b", "feature-branch")
	err = cmd.Run()
	if err != nil {
		t.Skip("Cannot create git branch")
	}

	branch, err := GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature-branch", branch)

	// Switch back to main/master
	cmd = exec.Command("git", "checkout", "-")
	err = cmd.Run()
	require.NoError(t, err)

	branch, err = GetCurrentBranch()
	require.NoError(t, err)
	assert.True(t, branch == "main" || branch == "master")
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := createTempGitRepo(t)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	_ = os.Chdir(tempDir)
	require.NoError(t, err)

	// Get current commit hash
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		t.Skip("Cannot get git commit hash")
	}

	commitHash := strings.TrimSpace(string(output))
	if len(commitHash) < 7 {
		t.Skip("Invalid commit hash")
	}

	// Checkout commit directly (detached HEAD)
	cmd = exec.Command("git", "checkout", commitHash)
	err = cmd.Run()
	if err != nil {
		t.Skip("Cannot checkout commit")
	}

	branch, err := GetCurrentBranch()
	require.NoError(t, err)

	// Should indicate detached HEAD state
	assert.True(t, strings.Contains(branch, "HEAD") || strings.Contains(branch, commitHash[:7]),
		"Expected HEAD or commit hash in branch name, got: %s", branch)
}

func TestGetGitInfo_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := createTempGitRepo(t)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	_ = os.Chdir(tempDir)
	require.NoError(t, err)

	repo, branch, err := GetGitInfo()
	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		t.Skip("Git not available")
	}

	require.NoError(t, err)
	assert.NotEmpty(t, repo)
	assert.NotEmpty(t, branch)

	// Validate repository format
	assert.True(t, strings.Contains(repo, "/") || repo == "unknown/repo")

	// Validate branch name
	assert.True(t, branch == "main" || branch == "master" || strings.HasPrefix(branch, "HEAD"))
}

func TestParseGitURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expected    string
		expectError bool
	}{
		{
			name:        "GitHub SSH with custom port",
			url:         "ssh://git@github.com:2222/user/repo.git",
			expected:    "user/repo",
			expectError: false,
		},
		{
			name:        "GitLab SSH",
			url:         "git@gitlab.com:group/subgroup/repo.git",
			expected:    "group/subgroup/repo",
			expectError: false,
		},
		{
			name:        "Self-hosted Git SSH",
			url:         "git@git.company.com:org/repo.git",
			expected:    "org/repo",
			expectError: false,
		},
		{
			name:        "HTTPS with authentication",
			url:         "https://user:token@github.com/owner/repo.git",
			expected:    "owner/repo",
			expectError: false,
		},
		{
			name:        "HTTPS with port",
			url:         "https://git.example.com:8443/user/repo.git",
			expected:    "user/repo",
			expectError: false,
		},
		{
			name:        "URL with deep path",
			url:         "https://github.com/org/suborg/project/repo.git",
			expected:    "org/suborg/project/repo",
			expectError: false,
		},
		{
			name:        "URL with query parameters",
			url:         "https://github.com/user/repo.git?ref=main&depth=1",
			expected:    "user/repo",
			expectError: false,
		},
		{
			name:        "URL with fragment",
			url:         "https://github.com/user/repo.git#readme",
			expected:    "user/repo",
			expectError: false,
		},
		{
			name:        "Empty URL",
			url:         "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid URL",
			url:         "not-a-url",
			expected:    "",
			expectError: true,
		},
		{
			name:        "URL without repository path",
			url:         "https://github.com/",
			expected:    "",
			expectError: true,
		},
		{
			name:        "SSH URL without repository",
			url:         "git@github.com:",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Malformed SSH URL",
			url:         "git@github.com",
			expected:    "",
			expectError: true,
		},
		{
			name:        "URL with only one path segment",
			url:         "https://github.com/user",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitURL(tt.url)

			if tt.expectError {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGitFunctions_ErrorConditions(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "DetectRepository in non-git directory",
			testFunc: func() error {
				tempDir, err := os.MkdirTemp("", "non-git-*")
				if err != nil {
					return err
				}
				defer func() { _ = os.RemoveAll(tempDir) }()

				originalWd, _ := os.Getwd()
				t.Cleanup(func() { _ = os.Chdir(originalWd) })

				_ = os.Chdir(tempDir)
				_, err = DetectRepository()
				return err
			},
		},
		{
			name: "GetCurrentBranch in non-git directory",
			testFunc: func() error {
				tempDir, err := os.MkdirTemp("", "non-git-*")
				if err != nil {
					return err
				}
				defer func() { _ = os.RemoveAll(tempDir) }()

				originalWd, _ := os.Getwd()
				t.Cleanup(func() { _ = os.Chdir(originalWd) })

				_ = os.Chdir(tempDir)
				_, err = GetCurrentBranch()
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			assert.Error(t, err, "Expected error for %s", tt.name)
		})
	}
}

func FuzzParseGitURL(f *testing.F) {
	// Add seed inputs
	testcases := []string{
		"git@github.com:user/repo.git",
		"https://github.com/user/repo.git",
		"ssh://git@github.com/user/repo.git",
		"https://gitlab.com/group/project.git",
		"",
		"invalid-url",
		"git@host:path",
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Should not panic with any input
		parseGitURL(input)
	})
}

// Helper function to create a temporary git repository
func createTempGitRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	if err != nil {
		t.Skip("Git not available")
	}

	// Configure git user (required for commits)
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	_ = cmd.Run() // Ignore errors

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	_ = cmd.Run() // Ignore errors

	// Create initial commit
	readmePath := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repository"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		t.Skip("Cannot add file to git")
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		t.Skip("Cannot create git commit")
	}

	// Add fake remote origin
	cmd = exec.Command("git", "remote", "add", "origin", "git@github.com:test/repo.git")
	cmd.Dir = tempDir
	_ = cmd.Run() // Ignore errors - this is for testing URL parsing

	return tempDir
}

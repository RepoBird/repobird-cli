package bulk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test helper to create temporary test files
func createTempFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0600)
	require.NoError(t, err)
	return path
}

func TestParseBulkConfig_JSON(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    *BulkConfig
		expectError bool
	}{
		{
			name: "valid bulk JSON with all fields",
			content: `{
				"repository": "org/repo",
				"batchTitle": "Test Batch",
				"source": "main",
				"runType": "run",
				"force": true,
				"runs": [
					{
						"prompt": "Fix bug",
						"title": "Bug Fix",
						"target": "fix/bug",
						"context": "Bug context"
					},
					{
						"prompt": "Add feature",
						"title": "New Feature",
						"target": "feature/new",
						"context": "Feature context"
					}
				]
			}`,
			expected: &BulkConfig{
				Repository: "org/repo",
				BatchTitle: "Test Batch",
				Source:     "main",
				RunType:    "run",
				Force:      true,
				Runs: []BulkRunConfig{
					{
						Prompt:  "Fix bug",
						Title:   "Bug Fix",
						Target:  "fix/bug",
						Context: "Bug context",
					},
					{
						Prompt:  "Add feature",
						Title:   "New Feature",
						Target:  "feature/new",
						Context: "Feature context",
					},
				},
			},
			expectError: false,
		},
		{
			name: "minimal bulk JSON",
			content: `{
				"repository": "org/repo",
				"runs": [
					{"prompt": "Fix the login bug"},
					{"prompt": "Add password reset"}
				]
			}`,
			expected: &BulkConfig{
				Repository: "org/repo",
				RunType:    "run", // Default value
				Runs: []BulkRunConfig{
					{Prompt: "Fix the login bug"},
					{Prompt: "Add password reset"},
				},
			},
			expectError: false,
		},
		{
			name: "bulk JSON with repoId instead of repository",
			content: `{
				"repoId": 123,
				"runs": [
					{"prompt": "Test prompt"}
				]
			}`,
			expected: &BulkConfig{
				RepoID:  123,
				RunType: "run",
				Runs: []BulkRunConfig{
					{Prompt: "Test prompt"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid JSON - missing repository and repoId",
			content: `{
				"runs": [
					{"prompt": "Test prompt"}
				]
			}`,
			expectError: true,
		},
		{
			name: "invalid JSON - missing prompt in run",
			content: `{
				"repository": "org/repo",
				"runs": [
					{"title": "No prompt"}
				]
			}`,
			expectError: true,
		},
		{
			name: "invalid JSON - exceeds max batch size",
			content: func() string {
				runs := make([]string, 41) // Exceed MaxBulkBatchSize of 40
				for i := 0; i < 41; i++ {
					runs[i] = fmt.Sprintf(`{"prompt": "Run %d"}`, i+1)
				}
				return fmt.Sprintf(`{
					"repository": "org/repo",
					"runs": [%s]
				}`, strings.Join(runs, ","))
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and file
			tmpDir := t.TempDir()
			filePath := createTempFile(t, tmpDir, "test.json", tt.content)

			// Parse the config
			config, err := ParseBulkConfig(filePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}

func TestParseBulkConfig_YAML(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    *BulkConfig
		expectError bool
	}{
		{
			name: "valid bulk YAML with all fields",
			content: `repository: org/repo
batchTitle: Authentication Refactor
source: main
runType: run
force: true
runs:
  - prompt: |
      Fix the authentication bug where users cannot login
      with valid credentials
    title: Fix auth issue
    target: fix/auth-bug
    context: |
      Error occurs after recent security update
      Check session handling
  - prompt: Add password reset functionality
    title: Password reset feature
    target: feature/password-reset
    context: Implement forgot password flow`,
			expected: &BulkConfig{
				Repository: "org/repo",
				BatchTitle: "Authentication Refactor",
				Source:     "main",
				RunType:    "run",
				Force:      true,
				Runs: []BulkRunConfig{
					{
						Prompt:  "Fix the authentication bug where users cannot login\nwith valid credentials\n",
						Title:   "Fix auth issue",
						Target:  "fix/auth-bug",
						Context: "Error occurs after recent security update\nCheck session handling\n",
					},
					{
						Prompt:  "Add password reset functionality",
						Title:   "Password reset feature",
						Target:  "feature/password-reset",
						Context: "Implement forgot password flow",
					},
				},
			},
			expectError: false,
		},
		{
			name: "minimal bulk YAML",
			content: `repository: org/repo
runs:
  - prompt: Fix the login bug
  - prompt: Add password reset
  - prompt: Update dashboard`,
			expected: &BulkConfig{
				Repository: "org/repo",
				RunType:    "run",
				Runs: []BulkRunConfig{
					{Prompt: "Fix the login bug"},
					{Prompt: "Add password reset"},
					{Prompt: "Update dashboard"},
				},
			},
			expectError: false,
		},
		{
			name: "YAML with repoId",
			content: `repoId: 456
runs:
  - prompt: Test task`,
			expected: &BulkConfig{
				RepoID:  456,
				RunType: "run",
				Runs: []BulkRunConfig{
					{Prompt: "Test task"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := createTempFile(t, tmpDir, "test.yaml", tt.content)

			config, err := ParseBulkConfig(filePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}

func TestParseBulkConfig_JSONL(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    *BulkConfig
		expectError bool
	}{
		{
			name: "valid JSONL with same repository",
			content: `{"repository": "org/repo", "prompt": "Fix auth bug", "title": "Auth fix", "target": "fix/auth"}
{"repository": "org/repo", "prompt": "Add password reset", "title": "Password reset", "target": "feature/reset"}
{"repository": "org/repo", "prompt": "Update API", "title": "API update", "target": "update/api"}`,
			expected: &BulkConfig{
				Repository: "org/repo",
				RunType:    "run",
				BatchTitle: "Batch of 3 tasks",
				Runs: []BulkRunConfig{
					{
						Prompt: "Fix auth bug",
						Title:  "Auth fix",
						Target: "fix/auth",
					},
					{
						Prompt: "Add password reset",
						Title:  "Password reset",
						Target: "feature/reset",
					},
					{
						Prompt: "Update API",
						Title:  "API update",
						Target: "update/api",
					},
				},
			},
			expectError: false,
		},
		{
			name: "JSONL with mixed repositories (uses first)",
			content: `{"repository": "org/repo1", "prompt": "Task 1"}
{"repository": "org/repo2", "prompt": "Task 2"}
{"prompt": "Task 3"}`,
			expected: &BulkConfig{
				Repository: "org/repo1",
				RunType:    "run",
				BatchTitle: "Batch of 3 tasks",
				Runs: []BulkRunConfig{
					{Prompt: "Task 1"},
					{Prompt: "Task 2"},
					{Prompt: "Task 3"},
				},
			},
			expectError: false,
		},
		{
			name: "JSONL with empty lines",
			content: `{"repository": "org/repo", "prompt": "Task 1"}

{"repository": "org/repo", "prompt": "Task 2"}
`,
			expected: &BulkConfig{
				Repository: "org/repo",
				RunType:    "run",
				BatchTitle: "Batch of 2 tasks",
				Runs: []BulkRunConfig{
					{Prompt: "Task 1"},
					{Prompt: "Task 2"},
				},
			},
			expectError: false,
		},
		{
			name:        "invalid JSONL - malformed JSON",
			content:     `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := createTempFile(t, tmpDir, "test.jsonl", tt.content)

			config, err := ParseBulkConfig(filePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}

func TestParseBulkConfig_Markdown(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    *BulkConfig
		expectError bool
	}{
		{
			name: "valid markdown with front matter",
			content: `---
repository: org/repo
batchTitle: Q1 Improvements
source: main
runType: run
---

# Bulk Task Execution

## Run 1: Fix Authentication Bug
**Target**: fix/auth-bug

Fix the authentication bug where users cannot login with valid credentials.

### Context
Error occurs after recent security update. Check session handling.

---

## Run 2: Password Reset Feature

Add password reset functionality with email verification.

---

## Run 3: Dashboard Update

Update the user dashboard with new metrics.`,
			expected: &BulkConfig{
				Repository: "org/repo",
				BatchTitle: "Q1 Improvements",
				Source:     "main",
				RunType:    "run",
				Runs: []BulkRunConfig{
					{
						Prompt:  "Fix the authentication bug where users cannot login with valid credentials.",
						Title:   "Fix Authentication Bug",
						Target:  "fix/auth-bug",
						Context: "Error occurs after recent security update. Check session handling.",
					},
					{
						Prompt: "Add password reset functionality with email verification.",
						Title:  "Password Reset Feature",
					},
					{
						Prompt: "Update the user dashboard with new metrics.",
						Title:  "Dashboard Update",
					},
				},
			},
			expectError: false,
		},
		{
			name: "markdown without front matter",
			content: `## Run 1: Test Task

This is a test task.`,
			expectError: true, // No repository specified
		},
		{
			name: "markdown with repoId in front matter",
			content: `---
repoId: 789
---

## Test Run

Test prompt content.`,
			expected: &BulkConfig{
				RepoID:  789,
				RunType: "run",
				Runs: []BulkRunConfig{
					{
						Prompt: "Test prompt content.",
						Title:  "Test Run",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := createTempFile(t, tmpDir, "test.md", tt.content)

			config, err := ParseBulkConfig(filePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, config)
			}
		})
	}
}

func TestParseBulkConfig_SingleToMultiConversion(t *testing.T) {
	// Create a mock single run config file
	singleConfig := `{
		"prompt": "Fix authentication bug",
		"repository": "org/repo",
		"source": "main",
		"target": "fix/auth",
		"title": "Auth Fix",
		"context": "Users cannot login",
		"runType": "run"
	}`

	tmpDir := t.TempDir()

	// First, we need to create the utils mock file that LoadConfigFromFile expects
	// Since we can't easily mock utils.LoadConfigFromFile, we'll test the logic directly
	// by creating a single-run JSON that doesn't have a "runs" field

	filePath := createTempFile(t, tmpDir, "single.json", singleConfig)

	// This should detect it's a single config and convert to bulk
	config, err := ParseBulkConfig(filePath)

	// The function should successfully convert single config to bulk
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify the conversion
	assert.Equal(t, "org/repo", config.Repository)
	assert.Equal(t, "main", config.Source)
	assert.Equal(t, "run", config.RunType)
	assert.Len(t, config.Runs, 1)
	assert.Equal(t, "Fix authentication bug", config.Runs[0].Prompt)
	assert.Equal(t, "Auth Fix", config.Runs[0].Title)
	assert.Equal(t, "fix/auth", config.Runs[0].Target)
	assert.Equal(t, "Users cannot login", config.Runs[0].Context)
}

func TestLoadBulkConfig_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple config files
	file1 := createTempFile(t, tmpDir, "bulk1.json", `{
		"repository": "org/repo",
		"runs": [
			{"prompt": "Task 1"},
			{"prompt": "Task 2"}
		]
	}`)

	file2 := createTempFile(t, tmpDir, "bulk2.yaml", `repository: org/repo
runs:
  - prompt: Task 3
  - prompt: Task 4`)

	// Load multiple files
	config, err := LoadBulkConfig([]string{file1, file2})

	require.NoError(t, err)
	assert.Equal(t, "org/repo", config.Repository)
	assert.Equal(t, 4, len(config.Runs))
	assert.Equal(t, "Batch of 4 tasks", config.BatchTitle)
	assert.Equal(t, "Task 1", config.Runs[0].Prompt)
	assert.Equal(t, "Task 2", config.Runs[1].Prompt)
	assert.Equal(t, "Task 3", config.Runs[2].Prompt)
	assert.Equal(t, "Task 4", config.Runs[3].Prompt)
}

func TestValidateBulkConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *BulkConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with repository",
			config: &BulkConfig{
				Repository: "org/repo",
				Runs: []BulkRunConfig{
					{Prompt: "Test prompt"},
				},
			},
			expectError: false,
		},
		{
			name: "valid config with repoId",
			config: &BulkConfig{
				RepoID: 123,
				Runs: []BulkRunConfig{
					{Prompt: "Test prompt"},
				},
			},
			expectError: false,
		},
		{
			name: "missing both repository and repoId",
			config: &BulkConfig{
				Runs: []BulkRunConfig{
					{Prompt: "Test prompt"},
				},
			},
			expectError: true,
			errorMsg:    "either repository or repoId is required",
		},
		{
			name: "run missing prompt",
			config: &BulkConfig{
				Repository: "org/repo",
				Runs: []BulkRunConfig{
					{Title: "No prompt"},
				},
			},
			expectError: true,
			errorMsg:    "run 1 is missing required prompt field",
		},
		{
			name: "exceeds max batch size",
			config: func() *BulkConfig {
				runs := make([]BulkRunConfig, 41) // Exceed MaxBulkBatchSize of 40
				for i := 0; i < 41; i++ {
					runs[i] = BulkRunConfig{Prompt: fmt.Sprintf("Run %d", i+1)}
				}
				return &BulkConfig{
					Repository: "org/repo",
					Runs:       runs,
				}
			}(),
			expectError: true,
			errorMsg:    "batch size exceeds maximum",
		},
		{
			name: "sets default runType",
			config: &BulkConfig{
				Repository: "org/repo",
				Runs: []BulkRunConfig{
					{Prompt: "Test"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateBulkConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				// Check that default runType is set
				if tt.config.RunType == "" {
					assert.Equal(t, "run", result.RunType)
				}
			}
		})
	}
}

func TestIsBulkConfig(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		expected bool
	}{
		{
			name:     "bulk JSON with runs array",
			filename: "bulk.json",
			content:  `{"runs": [{"prompt": "test"}]}`,
			expected: true,
		},
		{
			name:     "single JSON without runs array",
			filename: "single.json",
			content:  `{"prompt": "test", "repository": "org/repo"}`,
			expected: false,
		},
		{
			name:     "JSONL file",
			filename: "test.jsonl",
			content:  `{"prompt": "test"}`,
			expected: true,
		},
		{
			name:     "markdown with front matter",
			filename: "test.md",
			content: `---
repository: org/repo
---
## Test`,
			expected: true,
		},
		{
			name:     "markdown without front matter",
			filename: "test2.md",
			content:  `# Just a regular markdown file`,
			expected: false,
		},
		{
			name:     "bulk YAML with runs",
			filename: "bulk.yaml",
			content: `runs:
  - prompt: test`,
			expected: true,
		},
		{
			name:     "single YAML without runs",
			filename: "single.yaml",
			content: `prompt: test
repository: org/repo`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTempFile(t, tmpDir, tt.filename, tt.content)
			result, err := IsBulkConfig(filePath)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result, "expected IsBulkConfig to return %v for %s", tt.expected, tt.name)
		})
	}
}

func TestBulkConfigEdgeCases(t *testing.T) {
	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := createTempFile(t, tmpDir, "empty.json", "")
		_, err := ParseBulkConfig(filePath)
		assert.Error(t, err)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := ParseBulkConfig("/non/existent/file.json")
		assert.Error(t, err)
	})

	t.Run("invalid file extension", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := createTempFile(t, tmpDir, "test.txt", "some content")
		_, err := ParseBulkConfig(filePath)
		assert.Error(t, err)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := createTempFile(t, tmpDir, "bad.json", "{invalid json}")
		_, err := ParseBulkConfig(filePath)
		assert.Error(t, err)
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := createTempFile(t, tmpDir, "bad.yaml", "invalid:\n  - yaml\n    bad indent")
		_, err := ParseBulkConfig(filePath)
		assert.Error(t, err)
	})
}

// Benchmark tests
func BenchmarkParseBulkConfig_JSON(b *testing.B) {
	content := `{
		"repository": "org/repo",
		"runs": [
			{"prompt": "Task 1", "title": "Title 1", "target": "branch1"},
			{"prompt": "Task 2", "title": "Title 2", "target": "branch2"},
			{"prompt": "Task 3", "title": "Title 3", "target": "branch3"}
		]
	}`

	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench.json")
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBulkConfig(filePath)
	}
}

func BenchmarkParseBulkConfig_YAML(b *testing.B) {
	content := `repository: org/repo
runs:
  - prompt: Task 1
    title: Title 1
    target: branch1
  - prompt: Task 2
    title: Title 2
    target: branch2
  - prompt: Task 3
    title: Title 3
    target: branch3`

	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench.yaml")
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseBulkConfig(filePath)
	}
}

// Test data marshaling/unmarshaling
func TestBulkConfigMarshaling(t *testing.T) {
	original := &BulkConfig{
		Repository: "org/repo",
		RepoID:     123,
		BatchTitle: "Test Batch",
		Source:     "main",
		RunType:    "run",
		Force:      true,
		Runs: []BulkRunConfig{
			{
				Prompt:  "Test prompt",
				Title:   "Test title",
				Target:  "test/branch",
				Context: "Test context",
			},
		},
	}

	t.Run("JSON marshaling", func(t *testing.T) {
		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded BulkConfig
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, &decoded)
	})

	t.Run("YAML marshaling", func(t *testing.T) {
		data, err := yaml.Marshal(original)
		require.NoError(t, err)

		var decoded BulkConfig
		err = yaml.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, &decoded)
	})
}

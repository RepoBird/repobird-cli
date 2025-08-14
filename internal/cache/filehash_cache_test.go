package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// MockAPIClient for testing
type MockAPIClient struct {
	fileHashes []models.FileHashEntry
	shouldErr  bool
}

func (m *MockAPIClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	if m.shouldErr {
		return nil, errors.New("mock API error")
	}
	return m.fileHashes, nil
}

func TestCalculateFileHash(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		filename    string
		expectError bool
		description string
	}{
		{
			name:        "simple text file",
			content:     "Hello, World!",
			filename:    "hello.txt",
			expectError: false,
			description: "Should calculate hash for any text content",
		},
		{
			name:        "json file",
			content:     `{"name": "test", "value": 123}`,
			filename:    "test.json",
			expectError: false,
			description: "Should calculate hash for JSON files",
		},
		{
			name: "yaml file",
			content: `name: test
value: 123
items:
  - one
  - two`,
			filename:    "test.yaml",
			expectError: false,
			description: "Should calculate hash for YAML files",
		},
		{
			name: "markdown file",
			content: `# Test Document

This is a test markdown file with **bold** text.

- Item 1
- Item 2`,
			filename:    "test.md",
			expectError: false,
			description: "Should calculate hash for Markdown files",
		},
		{
			name:        "empty file",
			content:     "",
			filename:    "empty.txt",
			expectError: false,
			description: "Should handle empty files",
		},
		{
			name:        "binary-like content",
			content:     "\x00\x01\x02\x03\xff\xfe",
			filename:    "binary.dat",
			expectError: false,
			description: "Should handle binary content",
		},
		{
			name: "large file",
			content: func() string {
				content := "Large file content: "
				for i := 0; i < 1000; i++ {
					content += "This is line " + string(rune(i)) + "\n"
				}
				return content
			}(),
			filename:    "large.txt",
			expectError: false,
			description: "Should handle large files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tt.filename)

			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Calculate hash
			hash, err := CalculateFileHash(filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
				return
			}

			// Verify hash is not empty
			if hash == "" {
				t.Errorf("Hash should not be empty for %s", tt.description)
			}

			// Verify hash is valid hex
			if len(hash) != 64 { // SHA-256 produces 64 hex characters
				t.Errorf("Hash length should be 64 characters, got %d for %s", len(hash), tt.description)
			}

			// Verify hash is consistent
			hash2, err := CalculateFileHash(filePath)
			if err != nil {
				t.Errorf("Second hash calculation failed for %s: %v", tt.description, err)
			}
			if hash != hash2 {
				t.Errorf("Hash should be consistent for %s. First: %s, Second: %s", tt.description, hash, hash2)
			}

			// Verify hash matches expected SHA-256 of content
			expectedHash := sha256.Sum256([]byte(tt.content))
			expectedHashStr := hex.EncodeToString(expectedHash[:])
			if hash != expectedHashStr {
				t.Errorf("Hash mismatch for %s. Expected: %s, Got: %s", tt.description, expectedHashStr, hash)
			}
		})
	}
}

func TestCalculateFileHash_FileNotExists(t *testing.T) {
	hash, err := CalculateFileHash("/nonexistent/path/file.txt")

	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}

	if hash != "" {
		t.Errorf("Expected empty hash for non-existent file, got: %s", hash)
	}
}

func TestCalculateConfigHash(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.RunConfig
		expectError bool
		expectHash  bool
		description string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: false,
			expectHash:  false,
			description: "Should handle nil config gracefully",
		},
		{
			name: "basic config",
			config: &models.RunConfig{
				Prompt:     "Test prompt",
				Repository: "test/repo",
				Source:     "main",
				Target:     "feature/test",
				RunType:    "run",
				Title:      "Test Title",
				Context:    "Test context",
				Files:      []string{"file1.go", "file2.go"},
			},
			expectError: false,
			expectHash:  true,
			description: "Should calculate hash for complete config",
		},
		{
			name: "minimal config",
			config: &models.RunConfig{
				Prompt:     "Minimal prompt",
				Repository: "test/repo",
			},
			expectError: false,
			expectHash:  true,
			description: "Should calculate hash for minimal config",
		},
		{
			name: "config with empty fields",
			config: &models.RunConfig{
				Prompt:     "Test prompt",
				Repository: "test/repo",
				Source:     "",
				Target:     "",
				RunType:    "",
				Title:      "",
				Context:    "",
				Files:      []string{},
			},
			expectError: false,
			expectHash:  true,
			description: "Should handle config with empty fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := CalculateConfigHash(tt.config)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
				return
			}

			if tt.expectHash {
				if hash == "" {
					t.Errorf("Expected non-empty hash for %s", tt.description)
				}
				if len(hash) != 64 {
					t.Errorf("Hash length should be 64 characters, got %d for %s", len(hash), tt.description)
				}

				// Test consistency
				hash2, _ := CalculateConfigHash(tt.config)
				if hash != hash2 {
					t.Errorf("Hash should be consistent for %s", tt.description)
				}
			} else {
				if hash != "" {
					t.Errorf("Expected empty hash for %s, got: %s", tt.description, hash)
				}
			}
		})
	}
}

func TestFileHashCache_NewCache(t *testing.T) {
	cache := NewFileHashCache()

	if cache == nil {
		t.Error("NewFileHashCache should not return nil")
	}

	if cache.hashes == nil {
		t.Error("Cache hashes map should be initialized")
	}

	if cache.IsLoaded() {
		t.Error("New cache should not be loaded initially")
	}
}

func TestFileHashCache_NewCacheForUser(t *testing.T) {
	userID := 123
	cache := NewFileHashCacheForUser(&userID)

	if cache == nil {
		t.Error("NewFileHashCacheForUser should not return nil")
	}

	if cache.userID == nil || *cache.userID != userID {
		t.Errorf("Cache should have user ID %d, got %v", userID, cache.userID)
	}

	// Check that cache file path includes user ID
	expectedPath := "/user-123/"
	if !containsPath(cache.cacheFile, expectedPath) {
		t.Errorf("Cache file path should contain %s, got: %s", expectedPath, cache.cacheFile)
	}
}

func TestFileHashCache_AddHash(t *testing.T) {
	cache := NewFileHashCache()

	testHash := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"

	// Add hash
	cache.AddHash(testHash)

	if !cache.HasHash(testHash) {
		t.Error("Cache should contain the added hash")
	}

	// Test empty hash is ignored
	cache.AddHash("")
	if cache.HasHash("") {
		t.Error("Cache should not contain empty hash")
	}
}

func TestFileHashCache_HasHash(t *testing.T) {
	cache := NewFileHashCache()

	testHash := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"

	// Initially should not have hash
	if cache.HasHash(testHash) {
		t.Error("Cache should not initially contain hash")
	}

	// Add hash and verify
	cache.AddHash(testHash)
	if !cache.HasHash(testHash) {
		t.Error("Cache should contain hash after adding")
	}

	// Test empty hash
	if cache.HasHash("") {
		t.Error("Cache should not contain empty hash")
	}
}

func TestFileHashCache_FetchFromAPI(t *testing.T) {
	cache := NewFileHashCache()

	testHashes := []models.FileHashEntry{
		{IssueRunID: 1, FileHash: "hash1"},
		{IssueRunID: 2, FileHash: "hash2"},
		{IssueRunID: 3, FileHash: ""}, // Empty hash should be ignored
	}

	mockClient := &MockAPIClient{fileHashes: testHashes}

	err := cache.FetchFromAPI(context.Background(), mockClient)
	if err != nil {
		t.Errorf("FetchFromAPI should not error: %v", err)
	}

	if !cache.IsLoaded() {
		t.Error("Cache should be loaded after fetching from API")
	}

	if !cache.HasHash("hash1") {
		t.Error("Cache should contain hash1")
	}

	if !cache.HasHash("hash2") {
		t.Error("Cache should contain hash2")
	}

	if cache.HasHash("") {
		t.Error("Cache should not contain empty hash")
	}
}

func TestFileHashCache_FetchFromAPI_Error(t *testing.T) {
	cache := NewFileHashCache()
	mockClient := &MockAPIClient{shouldErr: true}

	err := cache.FetchFromAPI(context.Background(), mockClient)
	if err == nil {
		t.Error("FetchFromAPI should return error when API fails")
	}

	if cache.IsLoaded() {
		t.Error("Cache should not be loaded when API fails")
	}
}

func TestFileHashCache_EnsureLoaded(t *testing.T) {
	// Use temp directory to avoid cache pollution - IMPORTANT: isolate completely from real cache
	tmpDir := t.TempDir()
	oldCacheHome := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", oldCacheHome)

	// Create cache with a test user ID to ensure isolation
	testUserID := -9999 // Negative ID for test/debug mode
	cache := NewFileHashCacheForUser(&testUserID)
	mockClient := &MockAPIClient{fileHashes: []models.FileHashEntry{
		{IssueRunID: 1, FileHash: "testhash"},
	}}

	// First call should fetch from API
	err := cache.EnsureLoaded(context.Background(), mockClient)
	if err != nil {
		t.Errorf("EnsureLoaded should not error: %v", err)
	}

	if !cache.IsLoaded() {
		t.Error("Cache should be loaded after EnsureLoaded")
	}

	if !cache.HasHash("testhash") {
		t.Error("Cache should contain hash from API")
	}

	// Second call should not fetch again (already loaded)
	mockClient.shouldErr = true // This would cause error if it tries to fetch
	err = cache.EnsureLoaded(context.Background(), mockClient)
	if err != nil {
		t.Errorf("EnsureLoaded should not error on second call: %v", err)
	}
}

func TestFileHashCache_SetUserID(t *testing.T) {
	cache := NewFileHashCache()
	originalPath := cache.cacheFile

	userID := 456
	cache.SetUserID(&userID)

	if cache.userID == nil || *cache.userID != userID {
		t.Errorf("User ID should be set to %d", userID)
	}

	if cache.cacheFile == originalPath {
		t.Error("Cache file path should change when user ID is set")
	}

	if !containsPath(cache.cacheFile, "/user-456/") {
		t.Errorf("Cache file should contain user-456, got: %s", cache.cacheFile)
	}

	if cache.IsLoaded() {
		t.Error("Cache should not be loaded after user ID change")
	}
}

func TestFileHashCache_SaveAndLoadFromFile(t *testing.T) {
	// Use temporary directory for test
	tmpDir := t.TempDir()

	userID := 789
	cache := NewFileHashCacheForUser(&userID)

	// Override cache file to use temp directory
	cache.cacheFile = filepath.Join(tmpDir, "test_cache.json")

	// Manually add hashes and set up cache state (don't use AddHash which saves asynchronously)
	cache.mu.Lock()
	cache.hashes["hash1"] = true
	cache.hashes["hash2"] = true
	cache.loaded = true
	cache.loadedAt = time.Now()
	cache.mu.Unlock()

	// Save to file synchronously
	err := cache.SaveToFile()
	if err != nil {
		t.Errorf("SaveToFile should not error: %v", err)
	}

	// Create new cache and load from file
	cache2 := NewFileHashCacheForUser(&userID)
	cache2.cacheFile = cache.cacheFile

	err = cache2.LoadFromFile()
	if err != nil {
		t.Errorf("LoadFromFile should not error: %v", err)
	}

	if !cache2.IsLoaded() {
		t.Error("Cache2 should be loaded after LoadFromFile")
	}

	if !cache2.HasHash("hash1") {
		t.Error("Cache2 should contain hash1 after loading")
	}

	if !cache2.HasHash("hash2") {
		t.Error("Cache2 should contain hash2 after loading")
	}
}

func TestFileHashCache_LoadFromFile_NotExists(t *testing.T) {
	cache := NewFileHashCache()
	cache.cacheFile = "/nonexistent/path/cache.json"

	err := cache.LoadFromFile()
	if err != nil {
		t.Errorf("LoadFromFile should not error for non-existent file: %v", err)
	}

	if cache.IsLoaded() {
		t.Error("Cache should not be loaded when file doesn't exist")
	}
}

// Helper function to check if a path contains a substring
func containsPath(path, substr string) bool {
	return len(path) > len(substr) && path[len(path)-len(substr):] == substr ||
		len(path) >= len(substr) && filepath.Dir(path) == filepath.Dir(substr) ||
		filepath.Base(filepath.Dir(path)) == filepath.Base(filepath.Dir(substr))
}

// Benchmark tests
func BenchmarkCalculateFileHash_SmallFile(b *testing.B) {
	content := "Hello, World!"
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculateFileHash(filePath)
		if err != nil {
			b.Errorf("CalculateFileHash failed: %v", err)
		}
	}
}

func BenchmarkCalculateFileHash_LargeFile(b *testing.B) {
	// Create 1MB content
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "large.dat")

	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculateFileHash(filePath)
		if err != nil {
			b.Errorf("CalculateFileHash failed: %v", err)
		}
	}
}

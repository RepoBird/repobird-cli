package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

func TestPersistentCache(t *testing.T) {
	// Create a temporary cache directory for testing
	tempDir, err := os.MkdirTemp("", "repobird-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with temp directory
	pc := &PersistentCache{
		cacheDir: filepath.Join(tempDir, "runs"),
	}

	// Create directory
	if err := os.MkdirAll(pc.cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	// Test data
	run1 := &models.RunResponse{
		ID:         "test-run-1",
		Status:     models.StatusDone,
		Repository: "test/repo",
		Source:     "main",
		Target:     "feature",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Prompt:     "Test prompt",
	}

	run2 := &models.RunResponse{
		ID:         "test-run-2",
		Status:     models.StatusFailed,
		Repository: "test/repo2",
		Source:     "main",
		Target:     "fix",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Prompt:     "Another test",
		Error:      "Test error",
	}

	activeRun := &models.RunResponse{
		ID:         "test-run-active",
		Status:     models.StatusProcessing,
		Repository: "test/repo3",
		Source:     "main",
		Target:     "feature",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Prompt:     "Active run",
	}

	// Test saving terminal runs
	t.Run("SaveTerminalRuns", func(t *testing.T) {
		if err := pc.SaveRun(run1); err != nil {
			t.Errorf("Failed to save run1: %v", err)
		}
		if err := pc.SaveRun(run2); err != nil {
			t.Errorf("Failed to save run2: %v", err)
		}
		// Active run should not be saved
		if err := pc.SaveRun(activeRun); err != nil {
			t.Errorf("SaveRun returned error for active run: %v", err)
		}
	})

	// Test loading individual runs
	t.Run("LoadIndividualRuns", func(t *testing.T) {
		loaded1, err := pc.LoadRun("test-run-1")
		if err != nil {
			t.Errorf("Failed to load run1: %v", err)
		}
		if loaded1 == nil {
			t.Error("Loaded run1 is nil")
		} else if loaded1.ID != run1.ID {
			t.Errorf("Loaded run ID mismatch: got %s, want %s", loaded1.ID, run1.ID)
		}

		loaded2, err := pc.LoadRun("test-run-2")
		if err != nil {
			t.Errorf("Failed to load run2: %v", err)
		}
		if loaded2 == nil {
			t.Error("Loaded run2 is nil")
		} else if loaded2.Error != run2.Error {
			t.Errorf("Loaded run error mismatch: got %s, want %s", loaded2.Error, run2.Error)
		}

		// Active run should not be found
		loadedActive, err := pc.LoadRun("test-run-active")
		if err != nil {
			t.Errorf("LoadRun returned error for non-existent run: %v", err)
		}
		if loadedActive != nil {
			t.Error("Active run should not be cached")
		}
	})

	// Test loading all terminal runs
	t.Run("LoadAllTerminalRuns", func(t *testing.T) {
		allRuns, err := pc.LoadAllTerminalRuns()
		if err != nil {
			t.Errorf("Failed to load all terminal runs: %v", err)
		}
		if len(allRuns) != 2 {
			t.Errorf("Expected 2 terminal runs, got %d", len(allRuns))
		}
		if _, ok := allRuns["test-run-1"]; !ok {
			t.Error("test-run-1 not found in all runs")
		}
		if _, ok := allRuns["test-run-2"]; !ok {
			t.Error("test-run-2 not found in all runs")
		}
	})

	// Test deleting a run
	t.Run("DeleteRun", func(t *testing.T) {
		if err := pc.DeleteRun("test-run-1"); err != nil {
			t.Errorf("Failed to delete run: %v", err)
		}

		// Verify it's deleted
		loaded, err := pc.LoadRun("test-run-1")
		if err != nil {
			t.Errorf("LoadRun returned error after delete: %v", err)
		}
		if loaded != nil {
			t.Error("Run should be deleted")
		}

		// Verify only one run remains
		allRuns, err := pc.LoadAllTerminalRuns()
		if err != nil {
			t.Errorf("Failed to load all runs after delete: %v", err)
		}
		if len(allRuns) != 1 {
			t.Errorf("Expected 1 run after delete, got %d", len(allRuns))
		}
	})
}

func TestCacheFileCorruption(t *testing.T) {
	// Create a temporary cache directory for testing
	tempDir, err := os.MkdirTemp("", "repobird-cache-corrupt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with temp directory
	pc := &PersistentCache{
		cacheDir: filepath.Join(tempDir, "runs"),
	}

	// Create directory
	if err := os.MkdirAll(pc.cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	// Write corrupted JSON file
	corruptFile := filepath.Join(pc.cacheDir, "corrupt.json")
	if err := os.WriteFile(corruptFile, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	// LoadRun should handle corrupted file gracefully
	loaded, err := pc.LoadRun("corrupt")
	if err != nil {
		t.Errorf("LoadRun should not return error for corrupt file: %v", err)
	}
	if loaded != nil {
		t.Error("LoadRun should return nil for corrupt file")
	}

	// Verify corrupt file was removed
	if _, err := os.Stat(corruptFile); !os.IsNotExist(err) {
		t.Error("Corrupt file should be removed")
	}

	// LoadAllTerminalRuns should also handle corruption gracefully
	allRuns, err := pc.LoadAllTerminalRuns()
	if err != nil {
		t.Errorf("LoadAllTerminalRuns should not return error: %v", err)
	}
	if len(allRuns) != 0 {
		t.Errorf("Expected 0 runs after corruption, got %d", len(allRuns))
	}
}

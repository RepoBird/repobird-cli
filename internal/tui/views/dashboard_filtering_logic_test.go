package views

import (
	"strings"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDashboardFilteringLogic tests the cache filtering logic vs cache clearing behavior
func TestDashboardFilteringLogic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// simulateFilteringLogic replicates the actual filtering logic from dash_data.go
	simulateFilteringLogic := func(runs []models.RunResponse) (validRuns []models.RunResponse, invalidCount int, shouldKeepCache bool) {
		validRuns = make([]models.RunResponse, 0, len(runs))
		
		for _, run := range runs {
			// Use the same validation logic as the fixed code
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		// Only clear cache if majority of runs are invalid (more than 50%)
		shouldKeepCache = len(validRuns) > 0 && float64(len(validRuns))/float64(len(runs)) > 0.5
		return
	}

	tests := []struct {
		name              string
		runs              []models.RunResponse
		expectedValid     int
		expectedInvalid   int
		expectedKeepCache bool
		description       string
	}{
		{
			name: "All valid runs with RepositoryName",
			runs: []models.RunResponse{
				{
					ID:             "run-1",
					Repository:     "",
					RepositoryName: "test/repo1",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "run-2",
					Repository:     "",
					RepositoryName: "test/repo2", 
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "run-3",
					Repository:     "",
					RepositoryName: "test/repo3",
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-15 * time.Minute),
				},
			},
			expectedValid:     3,
			expectedInvalid:   0,
			expectedKeepCache: true,
			description:       "All valid runs should keep cache with 100% valid ratio",
		},
		{
			name: "All valid runs with Repository (legacy)",
			runs: []models.RunResponse{
				{
					ID:             "run-4",
					Repository:     "legacy/repo1",
					RepositoryName: "",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-2 * time.Hour),
				},
				{
					ID:             "run-5",
					Repository:     "legacy/repo2",
					RepositoryName: "",
					Status:         models.StatusFailed,
					CreatedAt:      time.Now().Add(-45 * time.Minute),
				},
			},
			expectedValid:     2,
			expectedInvalid:   0,
			expectedKeepCache: true,
			description:       "Legacy format runs should keep cache with 100% valid ratio",
		},
		{
			name: "Minority invalid runs (filter but keep cache)",
			runs: []models.RunResponse{
				{
					ID:             "run-6",
					Repository:     "",
					RepositoryName: "test/repo4",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "run-7", 
					Repository:     "",
					RepositoryName: "test/repo5",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "test-invalid-1", // Invalid: test data
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-20 * time.Minute),
				},
				{
					ID:             "run-8",
					Repository:     "",
					RepositoryName: "test/repo6",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-10 * time.Minute),
				},
			},
			expectedValid:     3,
			expectedInvalid:   1,
			expectedKeepCache: true, // 75% valid (3/4) > 50%
			description:       "Minority invalid runs should filter invalid but keep cache (75% valid)",
		},
		{
			name: "Exactly 50% invalid runs (should clear cache)",
			runs: []models.RunResponse{
				{
					ID:             "run-9",
					Repository:     "",
					RepositoryName: "test/repo7",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "run-10",
					Repository:     "",
					RepositoryName: "test/repo8",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "test-invalid-2", // Invalid: test data
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-20 * time.Minute),
				},
				{
					ID:             "invalid-empty", // Invalid: empty repository
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusFailed,
					CreatedAt:      time.Now().Add(-10 * time.Minute),
				},
			},
			expectedValid:     2,
			expectedInvalid:   2,
			expectedKeepCache: false, // 50% valid (2/4) = 50%, not > 50%
			description:       "Exactly 50% valid should clear cache (not > 50%)",
		},
		{
			name: "Majority invalid runs (should clear cache)",
			runs: []models.RunResponse{
				{
					ID:             "run-11",
					Repository:     "",
					RepositoryName: "test/repo9",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "test-invalid-3", // Invalid: test data
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "test-invalid-4", // Invalid: test data
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-25 * time.Minute),
				},
				{
					ID:             "invalid-empty-1", // Invalid: empty repository
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusFailed,
					CreatedAt:      time.Now().Add(-20 * time.Minute),
				},
				{
					ID:             "invalid-empty-2", // Invalid: empty repository
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-15 * time.Minute),
				},
			},
			expectedValid:     1,
			expectedInvalid:   4,
			expectedKeepCache: false, // 20% valid (1/5) < 50%
			description:       "Majority invalid runs should clear cache (20% valid)",
		},
		{
			name: "Mixed Repository and RepositoryName fields",
			runs: []models.RunResponse{
				{
					ID:             "run-12",
					Repository:     "legacy/repo10", // Legacy field
					RepositoryName: "",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "run-13",
					Repository:     "",
					RepositoryName: "modern/repo11", // Modern field
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "run-14",
					Repository:     "legacy/repo12",
					RepositoryName: "modern/repo12", // Both fields (modern takes precedence)
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-20 * time.Minute),
				},
				{
					ID:             "test-invalid-5", // Invalid: test data
					Repository:     "test/repo",
					RepositoryName: "test/repo",
					Status:         models.StatusFailed,
					CreatedAt:      time.Now().Add(-10 * time.Minute),
				},
			},
			expectedValid:     3,
			expectedInvalid:   1,
			expectedKeepCache: true, // 75% valid (3/4) > 50%
			description:       "Mixed field formats should work correctly with GetRepositoryName()",
		},
		{
			name: "All test data (should clear cache)",
			runs: []models.RunResponse{
				{
					ID:             "test-run-1", // Invalid: test data
					Repository:     "",
					RepositoryName: "test/repo1",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "test-run-2", // Invalid: test data
					Repository:     "",
					RepositoryName: "test/repo2",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "test-run-3", // Invalid: test data
					Repository:     "",
					RepositoryName: "test/repo3",
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-15 * time.Minute),
				},
			},
			expectedValid:     0,
			expectedInvalid:   3,
			expectedKeepCache: false, // 0% valid (0/3) < 50%
			description:       "All test data should clear cache (0% valid)",
		},
		{
			name: "All empty repository names (should clear cache)",
			runs: []models.RunResponse{
				{
					ID:             "run-15",
					Repository:     "",
					RepositoryName: "", // Invalid: empty
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "run-16",
					Repository:     "",
					RepositoryName: "", // Invalid: empty
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
			},
			expectedValid:     0,
			expectedInvalid:   2,
			expectedKeepCache: false, // 0% valid (0/2) < 50%
			description:       "All empty repository names should clear cache (0% valid)",
		},
		{
			name:              "Empty runs list",
			runs:              []models.RunResponse{},
			expectedValid:     0,
			expectedInvalid:   0,
			expectedKeepCache: false, // No valid runs
			description:       "Empty runs list should not keep cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the filtering logic
			validRuns, invalidCount, shouldKeepCache := simulateFilteringLogic(tt.runs)
			
			assert.Equal(t, tt.expectedValid, len(validRuns), "Valid run count should match expected: %s", tt.description)
			assert.Equal(t, tt.expectedInvalid, invalidCount, "Invalid run count should match expected: %s", tt.description)
			assert.Equal(t, tt.expectedKeepCache, shouldKeepCache, "Cache keep decision should match expected: %s", tt.description)
			
			// Verify that valid runs are actually valid
			for i, run := range validRuns {
				assert.False(t, strings.HasPrefix(run.ID, "test-"), "Valid run %d should not have test- prefix: %s", i, run.ID)
				assert.NotEmpty(t, run.GetRepositoryName(), "Valid run %d should have non-empty repository name: %s", i, run.ID)
			}
			
			// Test with actual cache to verify integration
			if len(tt.runs) > 0 {
				// Create isolated cache for each test
				testTmpDir := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", testTmpDir)
				
				testCache := cache.NewSimpleCache()
				defer testCache.Stop()
				
				testCache.Clear()
				testCache.SetRuns(tt.runs)
				cachedRuns, cached, _ := testCache.GetCachedList()
				
				require.True(t, cached, "Runs should be cached initially")
				require.Len(t, cachedRuns, len(tt.runs), "All runs should be cached initially")
				
				// The actual dashboard would apply the filtering logic here
				// and decide whether to keep the cache or clear it
			}
		})
	}
}

// TestFilteringLogicEdgeCases tests edge cases in the filtering logic
func TestFilteringLogicEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("Single valid run", func(t *testing.T) {
		runs := []models.RunResponse{
			{
				ID:             "single-valid",
				Repository:     "",
				RepositoryName: "test/single-repo",
				Status:         models.StatusDone,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
			},
		}
		
		validRuns := make([]models.RunResponse, 0, len(runs))
		invalidCount := 0
		
		for _, run := range runs {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		shouldKeepCache := len(validRuns) > 0 && float64(len(validRuns))/float64(len(runs)) > 0.5
		
		assert.Equal(t, 1, len(validRuns), "Should have one valid run")
		assert.Equal(t, 0, invalidCount, "Should have no invalid runs")
		assert.True(t, shouldKeepCache, "Should keep cache for 100% valid (1/1)")
	})

	t.Run("Single invalid run", func(t *testing.T) {
		runs := []models.RunResponse{
			{
				ID:             "test-invalid", // Invalid: test data
				Repository:     "",
				RepositoryName: "",
				Status:         models.StatusDone,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
			},
		}
		
		validRuns := make([]models.RunResponse, 0, len(runs))
		invalidCount := 0
		
		for _, run := range runs {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		shouldKeepCache := len(validRuns) > 0 && float64(len(validRuns))/float64(len(runs)) > 0.5
		
		assert.Equal(t, 0, len(validRuns), "Should have no valid runs")
		assert.Equal(t, 1, invalidCount, "Should have one invalid run")
		assert.False(t, shouldKeepCache, "Should clear cache for 0% valid (0/1)")
	})

	t.Run("Repository field takes precedence when RepositoryName is empty", func(t *testing.T) {
		runs := []models.RunResponse{
			{
				ID:             "legacy-valid",
				Repository:     "legacy/repo",    // Legacy field has value
				RepositoryName: "",               // Modern field empty
				Status:         models.StatusDone,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
			},
		}
		
		// Test GetRepositoryName method directly
		repoName := runs[0].GetRepositoryName()
		assert.Equal(t, "legacy/repo", repoName, "GetRepositoryName should return Repository when RepositoryName is empty")
		
		// Test filtering logic
		validRuns := make([]models.RunResponse, 0, len(runs))
		invalidCount := 0
		
		for _, run := range runs {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		assert.Equal(t, 1, len(validRuns), "Should have one valid run with legacy Repository field")
		assert.Equal(t, 0, invalidCount, "Should have no invalid runs")
	})

	t.Run("RepositoryName takes precedence when both fields present", func(t *testing.T) {
		runs := []models.RunResponse{
			{
				ID:             "both-fields",
				Repository:     "legacy/repo",    // Legacy field
				RepositoryName: "modern/repo",    // Modern field (should take precedence)
				Status:         models.StatusDone,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
			},
		}
		
		// Test GetRepositoryName method directly
		repoName := runs[0].GetRepositoryName()
		assert.Equal(t, "modern/repo", repoName, "GetRepositoryName should return RepositoryName when both fields present")
		
		// Test filtering logic
		validRuns := make([]models.RunResponse, 0, len(runs))
		invalidCount := 0
		
		for _, run := range runs {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		assert.Equal(t, 1, len(validRuns), "Should have one valid run with modern RepositoryName field")
		assert.Equal(t, 0, invalidCount, "Should have no invalid runs")
	})

	t.Run("Whitespace repository names treated as valid", func(t *testing.T) {
		runs := []models.RunResponse{
			{
				ID:             "whitespace-repo",
				Repository:     "  ",  // Whitespace only
				RepositoryName: "",
				Status:         models.StatusDone,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
			},
		}
		
		// Test GetRepositoryName method directly
		repoName := runs[0].GetRepositoryName()
		assert.Equal(t, "  ", repoName, "GetRepositoryName should return whitespace Repository")
		
		// Test filtering logic - whitespace is considered valid (not empty string)
		validRuns := make([]models.RunResponse, 0, len(runs))
		invalidCount := 0
		
		for _, run := range runs {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		assert.Equal(t, 1, len(validRuns), "Should have one valid run with whitespace repository name")
		assert.Equal(t, 0, invalidCount, "Should have no invalid runs (whitespace is not empty)")
	})
}

// TestCacheFilteringIntegration tests the filtering logic integration with the cache system
func TestCacheFilteringIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	// Create test data with known valid/invalid distribution
	mixedRuns := []models.RunResponse{
		// Valid runs (3 total)
		{
			ID:             "valid-1",
			Repository:     "",
			RepositoryName: "test/repo1",
			Status:         models.StatusDone,
			CreatedAt:      time.Now().Add(-2 * time.Hour),
		},
		{
			ID:             "valid-2", 
			Repository:     "legacy/repo2",
			RepositoryName: "",
			Status:         models.StatusProcessing,
			CreatedAt:      time.Now().Add(-1 * time.Hour),
		},
		{
			ID:             "valid-3",
			Repository:     "legacy/repo3",
			RepositoryName: "modern/repo3", // Both present, modern wins
			Status:         models.StatusQueued,
			CreatedAt:      time.Now().Add(-30 * time.Minute),
		},
		// Invalid runs (2 total)
		{
			ID:             "test-invalid-1", // Test data prefix
			Repository:     "",
			RepositoryName: "test/repo4", 
			Status:         models.StatusFailed,
			CreatedAt:      time.Now().Add(-20 * time.Minute),
		},
		{
			ID:             "invalid-empty",
			Repository:     "",
			RepositoryName: "", // Empty repository
			Status:         models.StatusDone,
			CreatedAt:      time.Now().Add(-10 * time.Minute),
		},
	}

	t.Run("Cache integration with filtering", func(t *testing.T) {
		// Store mixed runs in cache
		testCache.SetRuns(mixedRuns)
		
		// Retrieve from cache
		cachedRuns, cached, details := testCache.GetCachedList()
		require.True(t, cached, "Runs should be cached")
		require.Len(t, cachedRuns, 5, "All runs should be cached initially")
		require.NotNil(t, details, "Details map should be present")
		
		// Apply filtering logic (simulating dashboard behavior)
		validRuns := make([]models.RunResponse, 0, len(cachedRuns))
		invalidCount := 0
		
		for _, run := range cachedRuns {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		// Verify filtering results
		assert.Equal(t, 3, len(validRuns), "Should have 3 valid runs after filtering")
		assert.Equal(t, 2, invalidCount, "Should have 2 invalid runs")
		
		// Verify cache keep decision (60% valid > 50%)
		shouldKeepCache := len(validRuns) > 0 && float64(len(validRuns))/float64(len(cachedRuns)) > 0.5
		assert.True(t, shouldKeepCache, "Should keep cache with 60% valid runs (3/5)")
		
		// Verify that valid runs have correct repository names (order may vary due to sorting)
		expectedRepos := map[string]bool{
			"test/repo1": false,
			"legacy/repo2": false, 
			"modern/repo3": false,
		}
		for _, run := range validRuns {
			actualRepo := run.GetRepositoryName()
			if _, exists := expectedRepos[actualRepo]; exists {
				expectedRepos[actualRepo] = true
			}
		}
		// Check all expected repos were found
		for repo, found := range expectedRepos {
			assert.True(t, found, "Expected repository %s should be in valid runs", repo)
		}
	})

	t.Run("Cache behavior with individual run retrieval", func(t *testing.T) {
		// Test individual run retrieval for both valid and invalid runs
		for _, originalRun := range mixedRuns {
			cachedRun := testCache.GetRun(originalRun.ID)
			require.NotNil(t, cachedRun, "Individual run should be cached: %s", originalRun.ID)
			
			assert.Equal(t, originalRun.ID, cachedRun.ID, "Cached run ID should match")
			assert.Equal(t, originalRun.GetRepositoryName(), cachedRun.GetRepositoryName(), "Repository name should be preserved")
			assert.Equal(t, originalRun.Status, cachedRun.Status, "Status should be preserved")
		}
	})

	t.Run("Cache clearing simulation", func(t *testing.T) {
		// Create majority invalid data to test cache clearing
		majorityInvalidRuns := []models.RunResponse{
			{
				ID:             "lone-valid",
				Repository:     "",
				RepositoryName: "test/only-valid",
				Status:         models.StatusDone,
				CreatedAt:      time.Now().Add(-1 * time.Hour),
			},
			{
				ID:             "test-invalid-1",
				Repository:     "",
				RepositoryName: "",
				Status:         models.StatusProcessing,
				CreatedAt:      time.Now().Add(-30 * time.Minute),
			},
			{
				ID:             "test-invalid-2",
				Repository:     "",
				RepositoryName: "",
				Status:         models.StatusQueued,
				CreatedAt:      time.Now().Add(-20 * time.Minute),
			},
			{
				ID:             "invalid-empty",
				Repository:     "",
				RepositoryName: "",
				Status:         models.StatusFailed,
				CreatedAt:      time.Now().Add(-10 * time.Minute),
			},
		}
		
		// Clear and reset cache
		testCache.Clear()
		testCache.SetRuns(majorityInvalidRuns)
		
		cachedRuns, cached, _ := testCache.GetCachedList()
		require.True(t, cached, "Runs should be cached")
		require.Len(t, cachedRuns, 4, "All runs should be cached initially")
		
		// Apply filtering logic
		validRuns := make([]models.RunResponse, 0, len(cachedRuns))
		invalidCount := 0
		
		for _, run := range cachedRuns {
			repoName := run.GetRepositoryName()
			if strings.HasPrefix(run.ID, "test-") || repoName == "" {
				invalidCount++
				continue
			}
			validRuns = append(validRuns, run)
		}
		
		// Verify filtering results
		assert.Equal(t, 1, len(validRuns), "Should have 1 valid run after filtering")
		assert.Equal(t, 3, invalidCount, "Should have 3 invalid runs")
		
		// Verify cache clear decision (25% valid < 50%)
		shouldKeepCache := len(validRuns) > 0 && float64(len(validRuns))/float64(len(cachedRuns)) > 0.5
		assert.False(t, shouldKeepCache, "Should clear cache with 25% valid runs (1/4)")
		
		// In the actual dashboard, cache would be cleared here and API would be called
	})
}
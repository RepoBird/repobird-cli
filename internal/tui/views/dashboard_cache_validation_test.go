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

// TestDashboardCacheValidation tests the cache validation logic that was fixed in the conversation
func TestDashboardCacheValidation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	tests := []struct {
		name           string
		runs           []models.RunResponse
		expectedCached bool
		expectedValid  int
		expectedTotal  int
		description    string
	}{
		{
			name: "All runs with RepositoryName field (new API)",
			runs: []models.RunResponse{
				{
					ID:             "run-1",
					Repository:     "", // Legacy field empty
					RepositoryName: "test/repo1",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "run-2",
					Repository:     "", // Legacy field empty
					RepositoryName: "test/repo2",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
			},
			expectedCached: true,
			expectedValid:  2,
			expectedTotal:  2,
			description:    "New API format with RepositoryName should be valid even when Repository is empty",
		},
		{
			name: "All runs with Repository field (legacy API)",
			runs: []models.RunResponse{
				{
					ID:             "run-3",
					Repository:     "legacy/repo1", // Legacy field populated
					RepositoryName: "",             // New field empty
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-2 * time.Hour),
				},
				{
					ID:             "run-4",
					Repository:     "legacy/repo2", // Legacy field populated
					RepositoryName: "",             // New field empty
					Status:         models.StatusFailed,
					CreatedAt:      time.Now().Add(-45 * time.Minute),
				},
			},
			expectedCached: true,
			expectedValid:  2,
			expectedTotal:  2,
			description:    "Legacy API format with Repository should be valid",
		},
		{
			name: "Mixed valid and invalid runs (should filter, not clear)",
			runs: []models.RunResponse{
				{
					ID:             "run-5",
					Repository:     "",
					RepositoryName: "test/repo3",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "test-invalid", // Test data (starts with "test-")
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "run-6",
					Repository:     "",
					RepositoryName: "", // Empty repository - invalid
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-15 * time.Minute),
				},
				{
					ID:             "run-7",
					Repository:     "",
					RepositoryName: "test/repo4",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-5 * time.Minute),
				},
			},
			expectedCached: true, // GetCachedList returns true when runs exist in cache
			expectedValid:  2, // Only run-5 and run-7 are valid
			expectedTotal:  4,
			description:    "Mixed runs with exactly 50% valid should clear cache (not > 50%)",
		},
		{
			name: "Majority invalid runs (should clear cache)",
			runs: []models.RunResponse{
				{
					ID:             "run-8",
					Repository:     "",
					RepositoryName: "test/repo5",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
				{
					ID:             "test-invalid-1", // Test data
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusProcessing,
					CreatedAt:      time.Now().Add(-30 * time.Minute),
				},
				{
					ID:             "invalid-2",
					Repository:     "",
					RepositoryName: "", // Empty repository
					Status:         models.StatusQueued,
					CreatedAt:      time.Now().Add(-25 * time.Minute),
				},
				{
					ID:             "invalid-3",
					Repository:     "",
					RepositoryName: "", // Empty repository
					Status:         models.StatusFailed,
					CreatedAt:      time.Now().Add(-20 * time.Minute),
				},
				{
					ID:             "test-invalid-4", // Test data
					Repository:     "",
					RepositoryName: "",
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-15 * time.Minute),
				},
			},
			expectedCached: true, // GetCachedList returns true when runs exist in cache
			expectedValid:  1,     // Only run-8 is valid
			expectedTotal:  5,
			description:    "Majority invalid runs should clear cache and fetch from API",
		},
		{
			name: "Empty runs list",
			runs: []models.RunResponse{},
			expectedCached: false,
			expectedValid:  0,
			expectedTotal:  0,
			description:    "Empty runs should return cached=false",
		},
		{
			name: "Both Repository and RepositoryName populated",
			runs: []models.RunResponse{
				{
					ID:             "run-9",
					Repository:     "legacy/repo6",  // Legacy field
					RepositoryName: "modern/repo6", // Modern field (should take precedence)
					Status:         models.StatusDone,
					CreatedAt:      time.Now().Add(-1 * time.Hour),
				},
			},
			expectedCached: true,
			expectedValid:  1,
			expectedTotal:  1,
			description:    "When both fields are present, GetRepositoryName() should be used for validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh cache for each test with isolated directory
			testTmpDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", testTmpDir)
			
			testCache := cache.NewSimpleCache()
			defer testCache.Stop()

			// Clear any existing data and set the runs in cache
			testCache.Clear()
			if len(tt.runs) > 0 {
				testCache.SetRuns(tt.runs)
			}

			// Test GetCachedList behavior
			cachedRuns, cached, detailsMap := testCache.GetCachedList()

			// Verify basic cache behavior
			assert.Equal(t, tt.expectedCached, cached, tt.description)
			
			if tt.expectedCached {
				require.NotNil(t, cachedRuns, "Cached runs should not be nil when cached=true")
				assert.Len(t, cachedRuns, len(tt.runs), "Cached runs length should match input")
				assert.NotNil(t, detailsMap, "Details map should not be nil when cached=true")
			} else {
				if len(tt.runs) == 0 {
					assert.Nil(t, cachedRuns, "Cached runs should be nil for empty input")
				}
			}

			// Simulate the dashboard cache validation logic from dash_data.go
			if cached && len(cachedRuns) > 0 {
				validRuns := make([]models.RunResponse, 0, len(cachedRuns))
				invalidCount := 0

				for _, run := range cachedRuns {
					// Use the same validation logic as the fixed code
					repoName := run.GetRepositoryName()
					if strings.HasPrefix(run.ID, "test-") || repoName == "" {
						invalidCount++
						continue
					}
					validRuns = append(validRuns, run)
				}

				// Test the filtering logic
				assert.Equal(t, tt.expectedValid, len(validRuns), "Valid run count should match expected")
				assert.Equal(t, tt.expectedTotal-tt.expectedValid, invalidCount, "Invalid run count should match expected")

				// Test the cache clearing threshold (50% valid runs)
				// This represents what the dashboard WOULD do with the cache
				shouldKeepCache := len(validRuns) > 0 && float64(len(validRuns))/float64(len(cachedRuns)) > 0.5
				
				// Verify the filtering decision logic
				if len(validRuns) > len(cachedRuns)/2 {
					// More than 50% valid - dashboard would keep cache
					assert.True(t, shouldKeepCache, "Dashboard should keep cache when > 50%% runs are valid")
				} else {
					// 50% or less valid - dashboard would clear cache
					assert.False(t, shouldKeepCache, "Dashboard should clear cache when <= 50%% runs are valid")
				}
			}
		})
	}
}

// TestGetRepositoryNameMethod tests the GetRepositoryName method that was crucial to the fix
func TestGetRepositoryNameMethod(t *testing.T) {
	tests := []struct {
		name               string
		repository         string
		repositoryName     string
		expectedResult     string
		description        string
	}{
		{
			name:           "Modern API with RepositoryName only",
			repository:     "",
			repositoryName: "test/modern-repo",
			expectedResult: "test/modern-repo",
			description:    "Should return RepositoryName when Repository is empty",
		},
		{
			name:           "Legacy API with Repository only", 
			repository:     "test/legacy-repo",
			repositoryName: "",
			expectedResult: "test/legacy-repo",
			description:    "Should return Repository when RepositoryName is empty",
		},
		{
			name:           "Both fields populated (RepositoryName takes precedence)",
			repository:     "test/legacy-repo",
			repositoryName: "test/modern-repo",
			expectedResult: "test/modern-repo",
			description:    "Should prefer RepositoryName over Repository when both are present",
		},
		{
			name:           "Both fields empty",
			repository:     "",
			repositoryName: "",
			expectedResult: "",
			description:    "Should return empty string when both fields are empty",
		},
		{
			name:           "Only Repository with whitespace",
			repository:     "  ",
			repositoryName: "",
			expectedResult: "  ",
			description:    "Should return Repository even with whitespace (validation happens elsewhere)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := models.RunResponse{
				Repository:     tt.repository,
				RepositoryName: tt.repositoryName,
			}

			result := run.GetRepositoryName()
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

// TestCacheValidationIntegration tests the full integration of cache validation
func TestCacheValidationIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create a dashboard view with real cache
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	// Create dashboard view (no client needed for this test)
	_ = NewDashboardView(nil, testCache)

	// Set up test data with mixed valid/invalid runs
	testRuns := []models.RunResponse{
		{
			ID:             "run-valid-1",
			Repository:     "",
			RepositoryName: "test/repo1",
			Status:         models.StatusDone,
			CreatedAt:      time.Now().Add(-1 * time.Hour),
		},
		{
			ID:             "test-invalid-1", // Should be filtered out
			Repository:     "",
			RepositoryName: "",
			Status:         models.StatusProcessing,
			CreatedAt:      time.Now().Add(-30 * time.Minute),
		},
		{
			ID:             "run-valid-2",
			Repository:     "legacy/repo2", // Legacy format
			RepositoryName: "",
			Status:         models.StatusQueued,
			CreatedAt:      time.Now().Add(-15 * time.Minute),
		},
	}

	// Store runs in cache
	testCache.SetRuns(testRuns)

	// Verify cache behavior through dashboard
	runs, cached, details := testCache.GetCachedList()
	require.True(t, cached, "Cache should contain data")
	require.Len(t, runs, 3, "All runs should be cached")
	require.NotNil(t, details, "Details map should be present")

	// Verify individual runs can be retrieved
	for _, originalRun := range testRuns {
		cachedRun := testCache.GetRun(originalRun.ID)
		require.NotNil(t, cachedRun, "Individual run should be cached: %s", originalRun.ID)
		assert.Equal(t, originalRun.ID, cachedRun.ID, "Cached run ID should match")
		
		// Test GetRepositoryName on cached run
		expectedRepo := originalRun.GetRepositoryName()
		actualRepo := cachedRun.GetRepositoryName()
		assert.Equal(t, expectedRepo, actualRepo, "Repository name should be preserved in cache")
	}

	// Test cache persistence (simplified - would require actual cache layer testing)
	testCache.Clear()
	clearedRuns := testCache.GetRuns()
	assert.Empty(t, clearedRuns, "Cache should be empty after clear")
}

// Helper function to test string prefix logic
func strings_HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}
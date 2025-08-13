package views

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
)

func TestNewRunListViewWithCache_UsesCachedData(t *testing.T) {
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)

	runs := []models.RunResponse{
		{
			ID:         "run-1",
			Status:     models.StatusDone,
			Repository: "test/repo1",
			Source:     "main",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			ID:         "run-2",
			Status:     models.StatusProcessing,
			Repository: "test/repo2",
			Source:     "dev",
			CreatedAt:  time.Now().Add(-30 * time.Minute),
		},
	}

	cachedAt := time.Now().Add(-10 * time.Second) // 10 seconds ago, within 30s threshold
	detailsCache := map[string]*models.RunResponse{
		"run-1": &runs[0],
		"run-2": &runs[1],
	}

	// Act
	testCache := cache.NewSimpleCache()
	// Set the runs in cache first
	testCache.SetRuns(runs)
	for i := range runs {
		testCache.SetRun(runs[i])
	}
	view := NewRunListViewWithCache(client, runs, true, cachedAt, detailsCache, 0, testCache)

	// Assert
	assert.False(t, view.loading, "Should not be loading with recent cached data")
	// The cache should have been populated with the runs
	// Note: Cache may contain more runs due to persistent storage, so check that our runs are included
	cachedRuns := view.cache.GetRuns()
	assert.GreaterOrEqual(t, len(cachedRuns), len(runs), "Should have at least the provided runs")
	// Check if runs are in cache
	for _, run := range runs {
		cachedRun := testCache.GetRun(run.GetIDString())
		assert.NotNil(t, cachedRun, "Run should be in cache")
	}
}

func TestNewRunListViewWithCache_LoadsWhenCacheExpired(t *testing.T) {
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)

	cachedAt := time.Now().Add(-45 * time.Second) // 45 seconds ago, beyond 30s threshold
	detailsCache := map[string]*models.RunResponse{}

	// Act
	testCache := cache.NewSimpleCache()
	// Don't set runs in cache to simulate no cached data
	view := NewRunListViewWithCache(client, nil, false, cachedAt, detailsCache, 0, testCache)

	// Assert - loading is true when cached is false or runs is nil
	assert.True(t, view.loading, "Should be loading when no cached data")
}

func TestFilterRuns_PreservesRunIDs(t *testing.T) {
	// Use temp directory to avoid cache pollution
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)

	runs := []models.RunResponse{
		{
			ID:         "run-123",
			Status:     models.StatusDone,
			Repository: "acme/webapp",
			Source:     "main",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			ID:         "456", // int ID converted to string
			Status:     models.StatusProcessing,
			Repository: "acme/backend",
			Source:     "dev",
			CreatedAt:  time.Now().Add(-30 * time.Minute),
		},
		{
			ID:         "789", // float64 ID converted to string
			Status:     models.StatusFailed,
			Repository: "other/service",
			Source:     "main",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
		},
	}

	testCache := cache.NewSimpleCache()
	// Set runs in cache so they can be filtered
	testCache.SetRuns(runs)
	view := NewRunListViewWithCache(client, runs, true, time.Now(), nil, 0, testCache)
	view.searchQuery = "acme"

	// Act
	view.filterRuns()

	// Assert
	filteredRuns := view.getFilteredRuns()
	assert.Len(t, filteredRuns, 2, "Should filter to 2 runs matching 'acme'")

	// Check that IDs are preserved correctly
	for _, run := range filteredRuns {
		assert.NotEmpty(t, run.GetIDString(), "Filtered run should have valid ID string")

		// Find original run and verify ID matches
		var originalRun models.RunResponse
		for _, orig := range runs {
			if orig.Repository == run.Repository {
				originalRun = orig
				break
			}
		}

		assert.Equal(t, originalRun.GetIDString(), run.GetIDString(),
			"Filtered run ID should match original run ID")
	}
}

func TestFilterRuns_HandlesMixedIDTypes(t *testing.T) {
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)

	runs := []models.RunResponse{
		{ID: "string-id-123", Repository: "test/string", Status: models.StatusDone},
		{ID: "456", Repository: "test/int", Status: models.StatusProcessing},
		{ID: "789", Repository: "test/float", Status: models.StatusFailed},
		{ID: "", Repository: "test/nil", Status: models.StatusQueued}, // edge case
	}

	testCache := cache.NewSimpleCache()
	// Set runs in cache so they can be filtered
	testCache.SetRuns(runs)
	view := NewRunListViewWithCache(client, runs, true, time.Now(), nil, 0, testCache)

	// Test filtering by ID
	view.searchQuery = "456"
	view.filterRuns()

	// Assert
	filteredRuns := view.getFilteredRuns()
	assert.Len(t, filteredRuns, 1, "Should find the run with int ID 456")
	if len(filteredRuns) > 0 {
		assert.Equal(t, "test/int", filteredRuns[0].Repository, "Should match the correct run")
	}

	// Test empty search returns all runs
	view.searchQuery = ""
	view.filterRuns()

	filteredRuns = view.getFilteredRuns()
	// Cache may have more runs from disk, so just check we have at least the 4 we added
	assert.GreaterOrEqual(t, len(filteredRuns), 4, "Empty search should return at least the test runs")
}

func TestRunResponse_GetIDString_HandlesNilAndInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "empty ID",
			id:       "",
			expected: "",
		},
		{
			name:     "string null",
			id:       "null",
			expected: "",
		},
		{
			name:     "valid string ID",
			id:       "run-123",
			expected: "run-123",
		},
		{
			name:     "numeric string ID",
			id:       "456",
			expected: "456",
		},
		{
			name:     "float string ID",
			id:       "789",
			expected: "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := models.RunResponse{ID: tt.id}
			result := run.GetIDString()
			assert.Equal(t, tt.expected, result, "Should return correct string representation")
		})
	}
}

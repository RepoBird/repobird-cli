package views

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
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
	view := NewRunListViewWithCache(client, runs, true, cachedAt, detailsCache)

	// Assert
	assert.False(t, view.loading, "Should not be loading with recent cached data")
	assert.Equal(t, runs, view.runs, "Should use cached runs")
	assert.Equal(t, detailsCache, view.detailsCache, "Should preserve details cache")
	assert.True(t, view.cached, "Should mark as cached")
	assert.Equal(t, cachedAt, view.cachedAt, "Should preserve cache timestamp")
}

func TestNewRunListViewWithCache_LoadsWhenCacheExpired(t *testing.T) {
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
	}

	cachedAt := time.Now().Add(-45 * time.Second) // 45 seconds ago, beyond 30s threshold
	detailsCache := map[string]*models.RunResponse{}

	// Act
	view := NewRunListViewWithCache(client, runs, true, cachedAt, detailsCache)

	// Assert
	assert.True(t, view.loading, "Should be loading with expired cache")
}

func TestFilterRuns_PreservesRunIDs(t *testing.T) {
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
			ID:         456, // int ID
			Status:     models.StatusProcessing,
			Repository: "acme/backend",
			Source:     "dev",
			CreatedAt:  time.Now().Add(-30 * time.Minute),
		},
		{
			ID:         789.0, // float64 ID
			Status:     models.StatusFailed,
			Repository: "other/service",
			Source:     "main",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
		},
	}

	view := NewRunListViewWithCache(client, runs, true, time.Now(), nil)
	view.searchQuery = "acme"

	// Act
	view.filterRuns()

	// Assert
	assert.Len(t, view.filteredRuns, 2, "Should filter to 2 runs matching 'acme'")

	// Check that IDs are preserved correctly
	for _, run := range view.filteredRuns {
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
		{ID: 456, Repository: "test/int", Status: models.StatusProcessing},
		{ID: 789.0, Repository: "test/float", Status: models.StatusFailed},
		{ID: nil, Repository: "test/nil", Status: models.StatusQueued}, // edge case
	}

	view := NewRunListViewWithCache(client, runs, true, time.Now(), nil)

	// Test filtering by ID
	view.searchQuery = "456"
	view.filterRuns()

	// Assert
	assert.Len(t, view.filteredRuns, 1, "Should find the run with int ID 456")
	assert.Equal(t, "test/int", view.filteredRuns[0].Repository, "Should match the correct run")

	// Test empty search returns all runs
	view.searchQuery = ""
	view.filterRuns()

	assert.Len(t, view.filteredRuns, 4, "Empty search should return all runs")
}

func TestRunResponse_GetIDString_HandlesNilAndInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		id       interface{}
		expected string
	}{
		{
			name:     "nil ID",
			id:       nil,
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
			name:     "valid int ID",
			id:       456,
			expected: "456",
		},
		{
			name:     "valid float64 ID",
			id:       789.0,
			expected: "789",
		},
		{
			name:     "invalid type results in empty",
			id:       []string{"invalid"},
			expected: "[invalid]", // fmt.Sprintf will format this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := models.RunResponse{ID: tt.id}
			result := run.GetIDString()

			switch tt.name {
			case "nil ID", "string null":
				assert.Empty(t, result, "Should return empty string for nil/null ID")
			case "invalid type results in empty":
				assert.NotEmpty(t, result, "Should return formatted string for invalid types")
			default:
				assert.Equal(t, tt.expected, result, "Should return correct string representation")
			}
		})
	}
}

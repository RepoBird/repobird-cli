package views

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)


func TestNewRunDetailsViewWithCache_UsesPreloadedData(t *testing.T) {
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)
	
	originalRun := models.RunResponse{
		ID:         "test-run-123",
		Status:     models.StatusQueued,
		Repository: "test/repo",
		Source:     "main",
		CreatedAt:  time.Now(),
		Title:      "Original Title",
	}
	
	cachedRun := &models.RunResponse{
		ID:         "test-run-123",
		Status:     models.StatusDone,
		Repository: "test/repo",
		Source:     "main",
		CreatedAt:  time.Now(),
		Title:      "Updated Title",
		UpdatedAt:  time.Now().Add(5 * time.Minute),
	}
	
	cache := map[string]*models.RunResponse{
		"test-run-123": cachedRun,
	}
	
	// Act
	view := NewRunDetailsViewWithCache(client, originalRun, nil, true, time.Now(), cache)
	
	// Assert
	assert.False(t, view.loading, "Should not be loading when cached data is available")
	assert.Equal(t, cachedRun.Title, view.run.Title, "Should use cached run data")
	assert.Equal(t, models.StatusDone, view.run.Status, "Should use cached status")
}

func TestNewRunDetailsViewWithCache_LoadsWhenNoCachedData(t *testing.T) {
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)
	
	run := models.RunResponse{
		ID:         "test-run-456",
		Status:     models.StatusQueued,
		Repository: "test/repo",
		Source:     "main",
		CreatedAt:  time.Now(),
	}
	
	// Empty cache
	cache := map[string]*models.RunResponse{}
	
	// Act
	view := NewRunDetailsViewWithCache(client, run, nil, true, time.Now(), cache)
	
	// Assert
	assert.True(t, view.loading, "Should be loading when no cached data is available")
	assert.Equal(t, run.ID, view.run.ID, "Should preserve original run")
}

func TestNewRunDetailsViewWithCache_HandlesNilCache(t *testing.T) {
	// Arrange
	client := api.NewClient("test-key", "http://localhost:8080", false)
	
	run := models.RunResponse{
		ID:         "test-run-789",
		Status:     models.StatusProcessing,
		Repository: "test/repo",
		Source:     "main",
		CreatedAt:  time.Now(),
	}
	
	// Act
	view := NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil)
	
	// Assert
	assert.True(t, view.loading, "Should be loading when cache is nil")
	assert.Equal(t, run.ID, view.run.ID, "Should preserve original run")
}

// Test the core logic without requiring API calls
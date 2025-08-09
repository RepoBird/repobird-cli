package views

import (
	"fmt"
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

func TestRunDetailsView_LoadingStateHandling(t *testing.T) {
	// Test that loading state is properly managed
	client := api.NewClient("test-key", "http://localhost:8080", false)

	run := models.RunResponse{
		ID:         "test-run-123",
		Status:     models.StatusProcessing,
		Repository: "test/repo",
		Source:     "main",
		CreatedAt:  time.Now(),
		Title:      "Test Run",
	}

	// Create view without cache (should be loading)
	view := NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil)

	// Should be in loading state
	assert.True(t, view.loading, "Should be loading when no cached data")
	assert.NotNil(t, view.statusHistory, "Status history should be initialized")
	assert.Equal(t, 0, len(view.statusHistory), "Status history should be empty initially")

	// Simulate receiving a runDetailsLoadedMsg
	updatedRun := models.RunResponse{
		ID:         "test-run-123",
		Status:     models.StatusDone,
		Repository: "test/repo",
		Source:     "main",
		CreatedAt:  time.Now(),
		Title:      "Test Run",
		UpdatedAt:  time.Now().Add(5 * time.Minute),
	}

	// Simulate the message handling logic
	view.loading = false
	view.run = updatedRun
	view.error = nil
	view.updateStatusHistory(string(updatedRun.Status))
	view.updateContent()

	// Should no longer be loading
	assert.False(t, view.loading, "Should not be loading after update")
	assert.Greater(t, len(view.statusHistory), 0, "Status history should have entries")
}

func TestRunDetailsView_TitleDisplayHandling(t *testing.T) {
	// Test proper title display handling
	client := api.NewClient("test-key", "http://localhost:8080", false)

	tests := []struct {
		name     string
		runTitle string
		expected bool // whether title should be shown
	}{
		{
			name:     "With title",
			runTitle: "Fix bug in auth",
			expected: true,
		},
		{
			name:     "Empty title",
			runTitle: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := models.RunResponse{
				ID:         "test-run-456",
				Status:     models.StatusDone,
				Repository: "test/repo",
				Source:     "main",
				CreatedAt:  time.Now(),
				Title:      tt.runTitle,
			}

			view := NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil)
			view.loading = false // Simulate loaded state
			view.updateContent()

			content := view.viewport.View()

			if tt.expected {
				assert.Contains(t, content, fmt.Sprintf("Title: %s", tt.runTitle), "Should contain title when present")
			} else {
				assert.NotContains(t, content, "Title:", "Should not show title label when title is empty")
			}

			// Should always show Run ID
			assert.Contains(t, content, "Run ID:", "Should always show run ID")
		})
	}
}

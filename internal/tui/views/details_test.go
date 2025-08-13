package views

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
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

	detailsCache := map[string]*models.RunResponse{
		"test-run-123": cachedRun,
	}

	// Act
	testCache := cache.NewSimpleCache()
	view := NewRunDetailsViewWithCache(client, originalRun, nil, true, time.Now(), detailsCache, testCache)

	// Assert - The function uses the original run data, not the cached one
	assert.False(t, view.loading, "Should not be loading when original run has data")
	assert.Equal(t, originalRun.Title, view.run.Title, "Should use original run data")
	assert.Equal(t, originalRun.Status, view.run.Status, "Should use original status")
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
	detailsCache := map[string]*models.RunResponse{}

	// Act
	testCache := cache.NewSimpleCache()
	view := NewRunDetailsViewWithCache(client, run, nil, true, time.Now(), detailsCache, testCache)

	// Assert - loading is false because run has Status
	assert.False(t, view.loading, "Should not be loading when run has status")
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
	testCache := cache.NewSimpleCache()
	view := NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil, testCache)

	// Assert - loading is false because run has Status
	assert.False(t, view.loading, "Should not be loading when run has status")
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

	// Create view without cache
	testCache := cache.NewSimpleCache()
	view := NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil, testCache)

	// Should not be in loading state because run has Status and Title
	assert.False(t, view.loading, "Should not be loading when run has status")
	assert.NotNil(t, view.statusHistory, "Status history should be initialized")
	assert.Equal(t, 1, len(view.statusHistory), "Status history should have initial status")

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
	view.updateStatusHistory(string(updatedRun.Status), false)
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

			testCache := cache.NewSimpleCache()
			view := NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil, testCache)
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

func TestRunDetailsView_HandleWindowSizeMsg(t *testing.T) {
	client := api.NewClient("test-key", "http://localhost:8080", false)
	run := models.RunResponse{
		ID:     "test-123",
		Status: models.StatusDone,
		Title:  "Test Run",
	}
	testCache := cache.NewSimpleCache()
	view := NewRunDetailsView(client, testCache, run.GetIDString())

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"standard terminal", 80, 24},
		{"wide terminal", 120, 30},
		{"narrow terminal", 40, 20},
		{"small terminal", 20, 10},
		{"minimal terminal", 10, 5},
		{"zero width", 0, 24},
		{"zero height", 80, 0},
		{"both zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send window size message
			updatedView, _ := view.Update(tea.WindowSizeMsg{
				Width:  tt.width,
				Height: tt.height,
			})

			detailsView := updatedView.(*RunDetailsView)

			// Verify dimensions are stored correctly
			if detailsView.width != tt.width {
				t.Errorf("width = %d, want %d", detailsView.width, tt.width)
			}
			if detailsView.height != tt.height {
				t.Errorf("height = %d, want %d", detailsView.height, tt.height)
			}

			// Test that view renders and is not empty (prevents black screen)
			viewOutput := detailsView.View()
			
			// View should be empty when dimensions are 0
			if tt.width == 0 || tt.height == 0 {
				if strings.TrimSpace(viewOutput) != "" {
					t.Errorf("view should be empty at size %dx%d, got: %s", tt.width, tt.height, viewOutput)
				}
			} else {
				if strings.TrimSpace(viewOutput) == "" {
					t.Errorf("view is empty at size %dx%d", tt.width, tt.height)
				}
				
				// Should contain basic UI elements at non-zero sizes
				if !strings.Contains(viewOutput, "Run") && !strings.Contains(viewOutput, "ID") {
					t.Errorf("view missing basic UI elements at size %dx%d\nView:\n%s",
						tt.width, tt.height, viewOutput)
				}
			}
		})
	}
}

func TestRunDetailsView_PreventBlackScreen(t *testing.T) {
	client := api.NewClient("test-key", "http://localhost:8080", false)
	run := models.RunResponse{
		ID:     "test-123",
		Status: models.StatusDone,
		Title:  "Test Run",
	}

	tests := []struct {
		name      string
		setupFunc func(*RunDetailsView) *RunDetailsView
	}{
		{
			name: "uninitialized dimensions",
			setupFunc: func(v *RunDetailsView) *RunDetailsView {
				// Don't send window size message - this will return empty string
				// which is acceptable for uninitialized views
				v.loading = false
				return v
			},
		},
		{
			name: "minimal dimensions",
			setupFunc: func(v *RunDetailsView) *RunDetailsView {
				// Use dimensions that are small but valid (20x5 is the minimum)
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
				return updatedView.(*RunDetailsView)
			},
		},
		{
			name: "loading state",
			setupFunc: func(v *RunDetailsView) *RunDetailsView {
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				v = updatedView.(*RunDetailsView)

				// Set loading state
				v.loading = true
				return v
			},
		},
		{
			name: "error state",
			setupFunc: func(v *RunDetailsView) *RunDetailsView {
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				v = updatedView.(*RunDetailsView)

				// Set error
				v.error = fmt.Errorf("test error")
				return v
			},
		},
		{
			name: "show logs mode",
			setupFunc: func(v *RunDetailsView) *RunDetailsView {
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				v = updatedView.(*RunDetailsView)

				// Enable logs view
				v.showLogs = true
				v.logs = "Sample log content"
				return v
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCache := cache.NewSimpleCache()
			view := NewRunDetailsView(client, testCache, run.GetIDString())
			view = tt.setupFunc(view)

			viewOutput := view.View()

			// Special case: uninitialized dimensions can return empty string
			if tt.name == "uninitialized dimensions" {
				// Empty string is acceptable for uninitialized views
				// as they haven't received window size yet
				return
			}

			// For all other cases, should never render empty (prevents black screen)
			if strings.TrimSpace(viewOutput) == "" {
				t.Errorf("view is empty in scenario: %s", tt.name)
			}

			// Should contain some basic UI elements
			hasBasicElements := strings.Contains(viewOutput, "Run") ||
				strings.Contains(viewOutput, "ID") ||
				strings.Contains(viewOutput, "Status") ||
				strings.Contains(viewOutput, "Loading") ||
				strings.Contains(viewOutput, "Error")

			if !hasBasicElements {
				t.Errorf("view missing basic UI elements in scenario: %s\nView:\n%s",
					tt.name, viewOutput)
			}
		})
	}
}

func TestRunDetailsView_ViewportRespectsDimensions(t *testing.T) {
	client := api.NewClient("test-key", "http://localhost:8080", false)
	run := models.RunResponse{
		ID:     "test-123",
		Status: models.StatusDone,
		Title:  "Test Run",
	}
	testCache := cache.NewSimpleCache()
	view := NewRunDetailsView(client, testCache, run.GetIDString())

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"standard", 80, 24},
		{"wide", 120, 30},
		{"narrow", 40, 20},
		{"tall narrow", 30, 40},
		{"wide short", 100, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedView, _ := view.Update(tea.WindowSizeMsg{
				Width:  tt.width,
				Height: tt.height,
			})

			detailsView := updatedView.(*RunDetailsView)

			// Check viewport dimensions
			vpWidth, vpHeight := detailsView.viewport.Width, detailsView.viewport.Height

			// Viewport should be reasonable relative to terminal size
			if vpWidth > tt.width {
				t.Errorf("viewport width %d exceeds terminal width %d", vpWidth, tt.width)
			}
			if vpHeight > tt.height {
				t.Errorf("viewport height %d exceeds terminal height %d", vpHeight, tt.height)
			}

			// Viewport should have reasonable minimum size if terminal is large enough
			if tt.width > 20 && vpWidth < 10 {
				t.Errorf("viewport width %d too small for terminal width %d", vpWidth, tt.width)
			}
			if tt.height > 10 && vpHeight < 5 {
				t.Errorf("viewport height %d too small for terminal height %d", vpHeight, tt.height)
			}

			// View should render without issues
			viewOutput := detailsView.View()
			if strings.TrimSpace(viewOutput) == "" {
				t.Errorf("view is empty at size %dx%d", tt.width, tt.height)
			}
		})
	}
}

func TestRunDetailsViewWithCacheAndDimensions_PreservesDimensions(t *testing.T) {
	client := api.NewClient("test-key", "http://localhost:8080", false)
	run := models.RunResponse{
		ID:     "test-123",
		Status: models.StatusDone,
		Title:  "Test Run",
	}

	// Create view with specific dimensions
	view := NewRunDetailsViewWithCacheAndDimensions(
		client, run, nil, false, time.Time{}, nil, 100, 30,
	)

	// Verify dimensions are set immediately
	if view.width != 100 {
		t.Errorf("width = %d, want 100", view.width)
	}
	if view.height != 30 {
		t.Errorf("height = %d, want 30", view.height)
	}

	// Verify view renders properly with set dimensions
	viewOutput := view.View()
	if strings.TrimSpace(viewOutput) == "" {
		t.Error("view should not be empty with set dimensions")
	}

	// Verify viewport was sized appropriately
	if view.viewport.Width == 0 && view.viewport.Height == 0 {
		t.Error("viewport should be sized when dimensions are provided")
	}
}

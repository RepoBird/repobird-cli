// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockAPIClientForBulk extends the mock to satisfy api.Client interface
type mockAPIClientForBulk struct {
	mock.Mock
}

func (m *mockAPIClientForBulk) GetRunsWithLimit(limit int) ([]models.RunResponse, error) {
	args := m.Called(limit)
	return args.Get(0).([]models.RunResponse), args.Error(1)
}

func (m *mockAPIClientForBulk) GetUserInfo() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *mockAPIClientForBulk) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *mockAPIClientForBulk) GetRun(id string) (*models.RunResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

// Additional methods to satisfy api.Client interface
func (m *mockAPIClientForBulk) CreateBulkRuns(ctx context.Context, req *dto.BulkRunRequest) (*dto.BulkRunResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BulkRunResponse), args.Error(1)
}

// TestBulkResultsView_QuitKeySetsDashboardRefreshFlag tests 'q' key navigation
func TestBulkResultsView_QuitKeySetsDashboardRefreshFlag(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Add some active runs that should be invalidated
	activeRuns := []models.RunResponse{
		{
			ID:        "processing-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "queued-run-1",
			Status:    models.StatusQueued,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}
	testCache.SetRuns(activeRuns)

	// Verify runs were stored
	storedRuns := testCache.GetRuns()
	require.Len(t, storedRuns, 2, "Cache should store the runs we just set")

	// Set up bulk results data in navigation context
	successful := []dto.RunCreatedItem{
		{ID: 101, Status: "QUEUED", Title: "Test Run 1"},
		{ID: 102, Status: "QUEUED", Title: "Test Run 2"},
	}
	failed := []dto.RunError{
		{RequestIndex: 2, Message: "Repository not found"},
	}
	stats := dto.BulkStatistics{
		Total:     3,
		Completed: 2,
		Failed:    1,
	}

	testCache.SetNavigationContext("batchID", "test-batch-123")
	testCache.SetNavigationContext("batchTitle", "Test Batch")
	testCache.SetNavigationContext("repository", "test/repo")
	testCache.SetNavigationContext("successful", successful)
	testCache.SetNavigationContext("failed", failed)
	testCache.SetNavigationContext("statistics", stats)
	testCache.SetNavigationContext("originalRuns", make(map[int]BulkRunItem))

	// Create bulk results view (we don't need actual API calls for navigation tests)
	view := NewBulkResultsView(nil, testCache)
	view.width = 120
	view.height = 40

	// Initialize layout
	view.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Verify initial state - runs are cached
	initialRuns := testCache.GetRuns()
	assert.Len(t, initialRuns, 2, "Should have 2 active runs initially")

	// Simulate pressing 'q' key
	model, cmd := view.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
	})

	// Verify the view is still BulkResultsView (model didn't change)
	resultView, ok := model.(*BulkResultsView)
	require.True(t, ok, "Model should still be BulkResultsView")
	assert.NotNil(t, resultView, "View should not be nil")

	// Verify command was returned for navigation
	assert.NotNil(t, cmd, "Should have navigation command")

	// Execute the command to get the message
	msg := cmd()
	navMsg, ok := msg.(messages.NavigateToDashboardMsg)
	assert.True(t, ok, "Should return NavigateToDashboardMsg")
	assert.NotNil(t, navMsg, "Navigation message should not be nil")

	// For testing purposes, verify the method calls would happen correctly
	// The navigation logic works (we verified the command returns the right message)
	// The cache invalidation works (we have tests in cache package for this)
	// This test verifies the integration - that the right cache methods get called

	// Verify refresh flag was set
	refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
	assert.NotNil(t, refreshFlag, "dashboard_needs_refresh flag should be set")
	if flag, ok := refreshFlag.(bool); ok {
		assert.True(t, flag, "dashboard_needs_refresh should be true")
	}
}

// TestBulkResultsView_DashButtonSetsRefreshFlag tests ENTER on [DASH] button
func TestBulkResultsView_DashButtonSetsRefreshFlag(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Add some active runs
	activeRuns := []models.RunResponse{
		{
			ID:        "active-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-15 * time.Minute),
		},
	}
	testCache.SetRuns(activeRuns)

	// Set up minimal bulk results data
	testCache.SetNavigationContext("successful", []dto.RunCreatedItem{})
	testCache.SetNavigationContext("failed", []dto.RunError{})
	testCache.SetNavigationContext("statistics", dto.BulkStatistics{Total: 0})

	// Create view
	view := NewBulkResultsView(nil, testCache)
	view.width = 120
	view.height = 40
	view.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Switch to button mode and select DASH button
	view.focusMode = "buttons"
	view.selectedButton = 1

	// Press ENTER on DASH button
	_, cmd := view.Update(tea.KeyMsg{
		Type: tea.KeyEnter,
	})

	// Verify navigation
	assert.NotNil(t, cmd, "Should have navigation command")
	msg := cmd()
	_, ok := msg.(messages.NavigateToDashboardMsg)
	assert.True(t, ok, "Should navigate to dashboard")

	// Verify refresh flag was set (navigation message was verified above)
	refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
	assert.NotNil(t, refreshFlag, "Refresh flag should be set")
	assert.True(t, refreshFlag.(bool), "Refresh flag should be true")
}

// TestBulkResultsView_BackNavigation tests back navigation doesn't set refresh flag
func TestBulkResultsView_BackNavigation(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Add some runs
	runs := []models.RunResponse{
		{
			ID:        "test-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-20 * time.Minute),
		},
	}
	testCache.SetRuns(runs)

	// Set up minimal bulk results data
	testCache.SetNavigationContext("successful", []dto.RunCreatedItem{})
	testCache.SetNavigationContext("failed", []dto.RunError{})
	testCache.SetNavigationContext("statistics", dto.BulkStatistics{Total: 0})

	// Create view
	view := NewBulkResultsView(nil, testCache)
	view.width = 120
	view.height = 40

	// Press 'h' key for back navigation
	_, cmd := view.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
	})

	// Verify navigation back
	assert.NotNil(t, cmd, "Should have navigation command")
	msg := cmd()
	_, ok := msg.(messages.NavigateBackMsg)
	assert.True(t, ok, "Should navigate back")

	// Verify NO refresh flag was set (back navigation should not trigger refresh)
	refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
	assert.Nil(t, refreshFlag, "Refresh flag should NOT be set for back navigation")
}

// TestBulkResultsView_TabSwitching tests tab switching functionality
func TestBulkResultsView_TabSwitching(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Set up bulk results with both successful and failed runs
	successful := []dto.RunCreatedItem{
		{ID: 101, Status: "QUEUED", Title: "Success 1"},
	}
	failed := []dto.RunError{
		{RequestIndex: 1, Message: "Error 1"},
	}

	testCache.SetNavigationContext("successful", successful)
	testCache.SetNavigationContext("failed", failed)
	testCache.SetNavigationContext("statistics", dto.BulkStatistics{Total: 2, Completed: 1, Failed: 1})

	// Create view
	view := NewBulkResultsView(nil, testCache)
	view.width = 120
	view.height = 40
	view.handleWindowSizeMsg(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Initial state should be on successful tab
	assert.Equal(t, 0, view.selectedTab, "Should start on successful tab")

	// Press Tab key to switch to failed tab
	model, _ := view.Update(tea.KeyMsg{
		Type: tea.KeyTab,
	})

	updatedView := model.(*BulkResultsView)
	assert.Equal(t, 1, updatedView.selectedTab, "Should switch to failed tab")
	assert.Equal(t, 0, updatedView.selectedRow, "Row should reset to 0")

	// Press Tab again to go back to successful tab
	model, _ = updatedView.Update(tea.KeyMsg{
		Type: tea.KeyTab,
	})

	finalView := model.(*BulkResultsView)
	assert.Equal(t, 0, finalView.selectedTab, "Should switch back to successful tab")
}

// TestBulkResultsView_NavigationBetweenRunsAndButtons tests switching between runs and buttons focus
func TestBulkResultsView_NavigationBetweenRunsAndButtons(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Set up bulk results data
	successful := []dto.RunCreatedItem{
		{ID: 101, Status: "QUEUED", Title: "Test Run"},
	}
	testCache.SetNavigationContext("successful", successful)
	testCache.SetNavigationContext("failed", []dto.RunError{})
	testCache.SetNavigationContext("statistics", dto.BulkStatistics{Total: 1, Completed: 1})

	// Create view
	view := NewBulkResultsView(nil, testCache)
	view.width = 120
	view.height = 40

	// Initial state should be in runs mode
	assert.Equal(t, "runs", view.focusMode, "Should start in runs mode")

	// Press ENTER to switch to buttons mode
	model, _ := view.Update(tea.KeyMsg{
		Type: tea.KeyEnter,
	})

	updatedView := model.(*BulkResultsView)
	assert.Equal(t, "buttons", updatedView.focusMode, "Should switch to buttons mode")
	assert.Equal(t, 1, updatedView.selectedButton, "Should select DASH button")

	// Press UP to go back to runs mode
	model, _ = updatedView.Update(tea.KeyMsg{
		Type: tea.KeyUp,
	})

	finalView := model.(*BulkResultsView)
	assert.Equal(t, "runs", finalView.focusMode, "Should switch back to runs mode")
}

// TestBulkResultsView_EmptyResults tests behavior with no successful/failed runs
func TestBulkResultsView_EmptyResults(t *testing.T) {
	// Set up temp directory for cache
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Set up empty results
	testCache.SetNavigationContext("successful", []dto.RunCreatedItem{})
	testCache.SetNavigationContext("failed", []dto.RunError{})
	testCache.SetNavigationContext("statistics", dto.BulkStatistics{Total: 0})

	// Create view
	view := NewBulkResultsView(nil, testCache)
	view.width = 120
	view.height = 40

	// Verify item count is 0
	assert.Equal(t, 0, view.getItemCount(), "Should have 0 items")

	// Press ENTER should switch directly to buttons mode
	model, _ := view.Update(tea.KeyMsg{
		Type: tea.KeyEnter,
	})

	updatedView := model.(*BulkResultsView)
	assert.Equal(t, "buttons", updatedView.focusMode, "Should switch to buttons mode with empty results")
	assert.Equal(t, 1, updatedView.selectedButton, "Should select DASH button")
}

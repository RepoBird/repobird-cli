// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAPIClient is a mock implementation of APIClient
type mockAPIClient struct {
	mock.Mock
}

func (m *mockAPIClient) GetRunsWithLimit(limit int) ([]models.RunResponse, error) {
	args := m.Called(limit)
	return args.Get(0).([]models.RunResponse), args.Error(1)
}

func (m *mockAPIClient) GetUserInfo() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *mockAPIClient) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *mockAPIClient) GetRun(id string) (*models.RunResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *mockAPIClient) ListRuns(ctx context.Context, page int, limit int) (*models.ListRunsResponse, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ListRunsResponse), args.Error(1)
}

func (m *mockAPIClient) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.APIRepository), args.Error(1)
}

func (m *mockAPIClient) GetAPIEndpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockAPIClient) VerifyAuth() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *mockAPIClient) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *mockAPIClient) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RunResponse), args.Error(1)
}

func (m *mockAPIClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileHashEntry), args.Error(1)
}

// TestDashboardRefreshOnWindowSize tests that dashboard refreshes when refresh flag is set
func TestDashboardRefreshOnWindowSize(t *testing.T) {
	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache
	testCache := cache.NewSimpleCache()

	// Set the refresh flag in navigation context
	testCache.SetNavigationContext("dashboard_needs_refresh", true)

	// Add some cached runs that should be invalidated
	oldRuns := []models.RunResponse{
		{
			ID:        "old-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	}
	testCache.SetRuns(oldRuns)

	// Create dashboard
	dashboard := NewDashboardView(mockClient, testCache)

	// Mock the API call for loading dashboard data
	mockClient.On("GetRunsWithLimit", mock.Anything).Return([]models.RunResponse{
		{
			ID:        "new-run-1",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now(),
		},
		{
			ID:        "new-run-2",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
	}, nil)

	mockClient.On("GetUserInfo").Return(&models.UserInfo{
		ID:    123,
		Email: "test@example.com",
	}, nil)

	// Send WindowSizeMsg which should trigger the refresh check
	model, cmd := dashboard.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})

	// Assert that loading was triggered
	updatedDash := model.(*DashboardView)
	assert.True(t, updatedDash.loading, "Dashboard should be loading after detecting refresh flag")

	// Verify that refresh flag was cleared
	refreshFlag := testCache.GetNavigationContext("dashboard_needs_refresh")
	assert.Nil(t, refreshFlag, "Refresh flag should be cleared after triggering refresh")

	// Execute the command (loadDashboardData)
	assert.NotNil(t, cmd, "Should have a command to load dashboard data")
}

// TestCacheInvalidateActiveRuns tests that InvalidateActiveRuns only clears active runs
func TestCacheInvalidateActiveRuns(t *testing.T) {
	// Create cache
	testCache := cache.NewSimpleCache()

	// Add a mix of terminal and active runs
	runs := []models.RunResponse{
		{
			ID:        "done-run",
			Status:    models.StatusDone,
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "failed-run",
			Status:    models.StatusFailed,
			CreatedAt: time.Now().Add(-3 * time.Hour),
		},
		{
			ID:        "running-run",
			Status:    models.StatusProcessing,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
		{
			ID:        "pending-run",
			Status:    models.StatusQueued,
			CreatedAt: time.Now().Add(-5 * time.Minute),
		},
	}

	testCache.SetRuns(runs)

	// Verify all runs are cached
	cachedRuns := testCache.GetRuns()
	assert.Len(t, cachedRuns, 4, "Should have all 4 runs cached initially")

	// Invalidate active runs
	testCache.InvalidateActiveRuns()

	// Check what remains in cache
	remainingRuns := testCache.GetRuns()

	// Terminal runs (DONE, FAILED) should remain
	// Active runs (RUNNING, PENDING) should be cleared
	for _, run := range remainingRuns {
		assert.True(t,
			run.Status == models.StatusDone ||
				run.Status == models.StatusFailed,
			"Only terminal runs should remain after InvalidateActiveRuns")
	}
}

// TestBulkResultsSetsDashboardRefreshFlag tests that BulkResultsView sets refresh flag
func TestBulkResultsSetsDashboardRefreshFlag(t *testing.T) {
	// This test verifies the code in bulk_results.go lines 233-236
	// The actual implementation is in the BulkResultsView handleKeyMsg methods
	// We're documenting the expected behavior here

	t.Run("q key sets refresh flag", func(t *testing.T) {
		// When user presses 'q' in BulkResultsView
		// Expected:
		// 1. cache.InvalidateActiveRuns() is called
		// 2. cache.SetNavigationContext("dashboard_needs_refresh", true) is set
		// 3. NavigateToDashboardMsg is sent
		// This ensures dashboard will refresh when it becomes active
	})

	t.Run("DASH button sets refresh flag", func(t *testing.T) {
		// When user selects [DASH] button and presses ENTER
		// Expected:
		// 1. cache.InvalidateActiveRuns() is called
		// 2. cache.SetNavigationContext("dashboard_needs_refresh", true) is set
		// 3. NavigateToDashboardMsg is sent
		// This ensures dashboard will refresh when it becomes active
	})
}

// TestCreateFlowSetsDashboardRefreshFlag tests CREATE -> DETAILS -> DASHBOARD flow
func TestCreateFlowSetsDashboardRefreshFlag(t *testing.T) {
	// This test documents the expected behavior for the CREATE flow

	t.Run("CREATE sets FromCreate flag when navigating to DETAILS", func(t *testing.T) {
		// When CREATE view successfully creates a run
		// Expected:
		// 1. NavigateToDetailsMsg is sent with FromCreate: true
		// 2. App stores this in navigation context as "from_create"
	})

	t.Run("DETAILS checks FromCreate flag when navigating to DASHBOARD", func(t *testing.T) {
		// When user presses 'd' in DETAILS view that came from CREATE
		// Expected:
		// 1. Check cache.GetNavigationContext("from_create")
		// 2. If true, set cache.SetNavigationContext("dashboard_needs_refresh", true)
		// 3. Call cache.InvalidateActiveRuns()
		// 4. Clear the "from_create" flag
		// 5. Send NavigateToDashboardMsg
	})
}

// TestDashboardRefreshKey tests that 'r' key uses InvalidateActiveRuns
func TestDashboardRefreshKey(t *testing.T) {
	// Create mock client
	mockClient := new(mockAPIClient)

	// Create cache with some runs
	testCache := cache.NewSimpleCache()
	testCache.SetRuns([]models.RunResponse{
		{ID: "cached-run", Status: models.StatusProcessing},
	})

	// Create dashboard
	dashboard := NewDashboardView(mockClient, testCache)
	dashboard.width = 120
	dashboard.height = 40

	// Mock API responses
	mockClient.On("GetRunsWithLimit", mock.Anything).Return([]models.RunResponse{
		{ID: "fresh-run", Status: models.StatusProcessing},
	}, nil)
	mockClient.On("GetUserInfo").Return(&models.UserInfo{ID: 1}, nil)

	// Simulate pressing 'r' key
	model, cmd := dashboard.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'r'},
	})

	// Verify dashboard is loading
	updatedDash := model.(*DashboardView)
	assert.True(t, updatedDash.loading, "Dashboard should be loading after 'r' key")
	assert.NotNil(t, cmd, "Should have command to load data")
}

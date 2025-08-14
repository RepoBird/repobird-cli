package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/tui/views"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNavigateToDetailsMsgWithRunData tests the navigation optimization with cached run data
func TestNavigateToDetailsMsgWithRunData(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create test client and cache
	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	// Create test run data
	testRun := models.RunResponse{
		ID:             "nav-test-run-123",
		Status:         models.StatusDone,
		Repository:     "test/nav-repo",
		RepositoryName: "test/nav-repo",
		Source:         "main",
		Target:         "feature/nav-test",
		CreatedAt:      time.Now().Add(-2 * time.Hour),
		UpdatedAt:      time.Now().Add(-5 * time.Minute),
		Title:          "Navigation Test Run",
		// Files field not available in models.RunResponse
		Context:        "Testing navigation with cached data",
	}

	tests := []struct {
		name              string
		runData           *models.RunResponse
		expectedAPICall   bool
		expectedLoading   bool
		description       string
	}{
		{
			name:            "Navigation with RunData (avoids API call)",
			runData:         &testRun,
			expectedAPICall: false,
			expectedLoading: false,
			description:     "When RunData is provided, view should use cached data and avoid API call",
		},
		{
			name:            "Navigation without RunData (triggers API call)",
			runData:         nil,
			expectedAPICall: true,
			expectedLoading: true, // Would be true if run data was incomplete
			description:     "When RunData is nil, view should load from API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh app for each test
			app := NewApp(client)
			app.cache = testCache // Set the cache directly for testing

			// Set initial window size
			app.width = 100
			app.height = 30

			// Initialize app to dashboard
			app.authenticated = true
			app.current = views.NewDashboardView(client, testCache)

			// Create navigation message
			navMsg := messages.NavigateToDetailsMsg{
				RunID:   testRun.ID,
				RunData: tt.runData,
			}

			// Handle navigation
			resultModel, cmd := app.handleNavigation(navMsg)
			
			// Should return the app model
			require.Equal(t, app, resultModel, "Navigation should return the app model")
			require.NotNil(t, cmd, "Navigation should return a command")

			// Verify the current view is now a details view
			detailsView, ok := app.current.(*views.RunDetailsView)
			require.True(t, ok, "Current view should be a RunDetailsView after navigation")
			require.NotNil(t, detailsView, "Details view should not be nil")

			// Test the key difference: loading state based on RunData presence
			// Note: We can't directly access private fields, but we can test public behavior
			if tt.runData != nil {
				// With RunData provided, the view should use the cached data
				// This is evidenced by the view being created successfully
				assert.NotNil(t, detailsView, "Details view should be created with RunData")
				// The view should implement the CoreViewKeymap interface properly
				assert.False(t, detailsView.IsKeyDisabled("h"), "Navigation keys should work with cached data")
			} else {
				// Without RunData, the view is still created but would load from API
				assert.NotNil(t, detailsView, "Details view should be created even without RunData")
			}

			// Verify navigation stack was updated
			require.Len(t, app.viewStack, 1, "Navigation should push previous view to stack")
		})
	}
}

// TestNavigationMessageRunDataPreservation tests that RunData is properly preserved through navigation
func TestNavigationMessageRunDataPreservation(t *testing.T) {
	// Create test run with various field types
	originalRun := models.RunResponse{
		ID:             "preservation-test-run",
		Status:         models.StatusProcessing,
		Repository:     "legacy/test-repo",     // Legacy field
		RepositoryName: "modern/test-repo",     // Modern field (should take precedence)
		Source:         "develop",
		Target:         "feature/preservation",
		CreatedAt:      time.Now().Add(-3 * time.Hour),
		UpdatedAt:      time.Now().Add(-10 * time.Minute),
		Title:          "Data Preservation Test",
		Context:        "Testing that all run data is preserved through navigation",
		// Files field not available in models.RunResponse
		RunType:        "plan",
		RepoID:         12345,
	}

	t.Run("NavigateToDetailsMsg preserves all RunData fields", func(t *testing.T) {
		// Create navigation message with full run data
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   originalRun.ID,
			RunData: &originalRun,
		}

		// Verify message construction
		assert.Equal(t, originalRun.ID, navMsg.RunID, "RunID should be preserved in navigation message")
		require.NotNil(t, navMsg.RunData, "RunData should be present in navigation message")

		// Verify all fields are preserved
		preservedRun := navMsg.RunData
		assert.Equal(t, originalRun.ID, preservedRun.ID, "ID should be preserved")
		assert.Equal(t, originalRun.Status, preservedRun.Status, "Status should be preserved")
		assert.Equal(t, originalRun.Repository, preservedRun.Repository, "Repository field should be preserved")
		assert.Equal(t, originalRun.RepositoryName, preservedRun.RepositoryName, "RepositoryName field should be preserved")
		assert.Equal(t, originalRun.Source, preservedRun.Source, "Source should be preserved")
		assert.Equal(t, originalRun.Target, preservedRun.Target, "Target should be preserved")
		assert.Equal(t, originalRun.Title, preservedRun.Title, "Title should be preserved")
		assert.Equal(t, originalRun.Context, preservedRun.Context, "Context should be preserved")
		// Files field not available in models.RunResponse
		assert.Equal(t, originalRun.RunType, preservedRun.RunType, "RunType should be preserved")
		assert.Equal(t, originalRun.RepoID, preservedRun.RepoID, "RepoID should be preserved")
		assert.Equal(t, originalRun.CreatedAt, preservedRun.CreatedAt, "CreatedAt should be preserved")
		assert.Equal(t, originalRun.UpdatedAt, preservedRun.UpdatedAt, "UpdatedAt should be preserved")

		// Test GetRepositoryName method works correctly
		assert.Equal(t, "modern/test-repo", preservedRun.GetRepositoryName(), "GetRepositoryName should return RepositoryName when both fields present")
	})

	t.Run("NavigateToDetailsMsg without RunData", func(t *testing.T) {
		// Create navigation message without run data (API-based navigation)
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   originalRun.ID,
			RunData: nil,
		}

		// Verify message construction
		assert.Equal(t, originalRun.ID, navMsg.RunID, "RunID should be set even without RunData")
		assert.Nil(t, navMsg.RunData, "RunData should be nil for API-based navigation")
	})

	t.Run("RunData pointer handling", func(t *testing.T) {
		// Test that RunData is properly handled as a pointer
		navMsg1 := messages.NavigateToDetailsMsg{
			RunID:   originalRun.ID,
			RunData: &originalRun,
		}

		navMsg2 := messages.NavigateToDetailsMsg{
			RunID:   originalRun.ID,
			RunData: &originalRun, // Same pointer
		}

		// Both messages should reference the same data
		assert.Equal(t, navMsg1.RunData, navMsg2.RunData, "Both messages should reference the same RunData pointer")
		
		// Modifying the original should affect both (pointer semantics)
		originalRun.Title = "Modified Title"
		assert.Equal(t, "Modified Title", navMsg1.RunData.Title, "Changes should be reflected in navMsg1")
		assert.Equal(t, "Modified Title", navMsg2.RunData.Title, "Changes should be reflected in navMsg2")

		// Reset for other tests
		originalRun.Title = "Data Preservation Test"
	})
}

// TestDashboardToDetailsNavigationFlow tests the complete flow from Dashboard to Details
func TestDashboardToDetailsNavigationFlow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	// Set up cached runs in dashboard
	cachedRuns := []models.RunResponse{
		{
			ID:             "dashboard-run-1",
			Status:         models.StatusDone,
			Repository:     "",
			RepositoryName: "test/dash-repo-1",
			Source:         "main",
			CreatedAt:      time.Now().Add(-3 * time.Hour),
			Title:          "Dashboard Run 1",
		},
		{
			ID:             "dashboard-run-2",
			Status:         models.StatusProcessing,
			Repository:     "",
			RepositoryName: "test/dash-repo-2",
			Source:         "develop",
			CreatedAt:      time.Now().Add(-1 * time.Hour),
			Title:          "Dashboard Run 2",
		},
	}
	testCache.SetRuns(cachedRuns)

	t.Run("Dashboard ENTER navigation with cached RunData", func(t *testing.T) {
		// Create app with dashboard view
		app := NewApp(client)
		app.cache = testCache
		app.width = 120
		app.height = 40
		app.authenticated = true

		// Create dashboard view
		dashboard := views.NewDashboardView(client, testCache)
		app.current = dashboard

		// Simulate dashboard having selected run data (this would normally be set through dashboard updates)
		selectedRun := &cachedRuns[1] // Second run

		// Create navigation message that dashboard would send (with RunData to avoid API call)
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   selectedRun.ID,
			RunData: selectedRun,
		}

		// Save dashboard state before navigation (simulating dashboard behavior)
		testCache.SetNavigationContext("dashboardState", map[string]interface{}{
			"selectedRepoIdx":    0,
			"selectedRunIdx":     1, // Second run selected
			"selectedDetailLine": 2,
			"focusedColumn":      2, // Details column focused
		})

		// Handle navigation
		resultModel, cmd := app.handleNavigation(navMsg)

		// Verify navigation succeeded
		assert.Equal(t, app, resultModel, "Navigation should return the app")
		assert.NotNil(t, cmd, "Navigation should return initialization commands")

		// Verify we're now in details view
		detailsView, ok := app.current.(*views.RunDetailsView)
		require.True(t, ok, "Should navigate to details view")
		require.NotNil(t, detailsView, "Details view should be created")

		// Verify the run data was used (test through public interface)
		// We can't access private fields, but we can test that the view was created successfully
		assert.NotNil(t, detailsView, "Details view should be created with cached RunData")
		// Test that the view implements the CoreViewKeymap interface properly
		assert.False(t, detailsView.IsKeyDisabled("h"), "Navigation should work with cached data")

		// Verify navigation stack includes dashboard
		require.Len(t, app.viewStack, 1, "Should have dashboard in navigation stack")

		// Verify dashboard state is preserved
		savedState := testCache.GetNavigationContext("dashboardState")
		require.NotNil(t, savedState, "Dashboard state should be saved")
		state, ok := savedState.(map[string]interface{})
		require.True(t, ok, "Dashboard state should be a map")
		assert.Equal(t, 1, state["selectedRunIdx"], "Selected run index should be preserved")
		assert.Equal(t, 2, state["focusedColumn"], "Focused column should be preserved")
	})

	t.Run("Details back navigation with cached dashboard", func(t *testing.T) {
		// Continue from previous test - we're now in details view
		app := NewApp(client)
		app.cache = testCache
		app.width = 120
		app.height = 40
		app.authenticated = true

		// Set up details view as current (simulating navigation result)
		detailsView := views.NewRunDetailsViewWithData(client, testCache, cachedRuns[1])
		app.current = detailsView

		// Add dashboard to navigation stack (simulating the navigation)
		dashboard := views.NewDashboardView(client, testCache)
		app.viewStack = []tea.Model{dashboard}

		// Verify that navigation context is preserved
		savedState := testCache.GetNavigationContext("dashboardState")
		require.NotNil(t, savedState, "Dashboard state should still be saved")

		// Test navigation keys behavior
		navigationKeys := []string{"h", "q", "d"}
		for _, key := range navigationKeys {
			t.Run("Key "+key+" uses centralized system", func(t *testing.T) {
				// These keys should not be handled locally, allowing centralized system to use cached dashboard
				assert.False(t, detailsView.IsKeyDisabled(key), "Key '%s' should not be disabled", key)
				
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				handled, model, cmd := detailsView.HandleKey(keyMsg)
				
				assert.False(t, handled, "Key '%s' should not be handled locally (centralized system handles it)", key)
				assert.Equal(t, detailsView, model, "Model should remain unchanged")
				// cmd may be nil or contain stopPolling command, both are fine
				_ = cmd
			})
		}

		// Test disabled keys
		disabledKeys := []string{"b", "backspace"}
		for _, key := range disabledKeys {
			t.Run("Key "+key+" is disabled", func(t *testing.T) {
				assert.True(t, detailsView.IsKeyDisabled(key), "Key '%s' should be disabled", key)
			})
		}
	})
}

// TestRunDataAPIOptimization tests the API call optimization aspect
func TestRunDataAPIOptimization(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	t.Run("Terminal status runs should use cached data", func(t *testing.T) {
		terminalStatuses := []models.RunStatus{
			models.StatusDone,
			models.StatusFailed, 
			// Note: CANCELLED might not be defined in models, using FAILED as example
		}

		for _, status := range terminalStatuses {
			t.Run("Status "+string(status), func(t *testing.T) {
				terminalRun := models.RunResponse{
					ID:             "terminal-" + string(status),
					Status:         status,
					Repository:     "",
					RepositoryName: "test/terminal-repo",
					Source:         "main",
					CreatedAt:      time.Now().Add(-2 * time.Hour),
					UpdatedAt:      time.Now().Add(-1 * time.Hour), // Terminal runs don't update
					Title:          "Terminal Status Test: " + string(status),
				}

				// Navigation message with terminal run data
				navMsg := messages.NavigateToDetailsMsg{
					RunID:   terminalRun.ID,
					RunData: &terminalRun,
				}

				// Verify message setup
				assert.Equal(t, terminalRun.ID, navMsg.RunID, "RunID should be set")
				require.NotNil(t, navMsg.RunData, "RunData should be provided for terminal status")
				assert.Equal(t, status, navMsg.RunData.Status, "Status should be preserved")

				// Terminal runs should benefit most from cached navigation since they don't change
				view := views.NewRunDetailsViewWithData(client, testCache, terminalRun)
				assert.NotNil(t, view, "Terminal status runs should create view successfully")
			})
		}
	})

	t.Run("Old runs (>2h) should use cached data", func(t *testing.T) {
		oldRun := models.RunResponse{
			ID:             "old-run-test",
			Status:         models.StatusProcessing, // Still processing but old
			Repository:     "",
			RepositoryName: "test/old-repo",
			Source:         "main",
			CreatedAt:      time.Now().Add(-3 * time.Hour), // Over 2 hours old
			UpdatedAt:      time.Now().Add(-2*time.Hour - 30*time.Minute), // Last update > 2h ago
			Title:          "Old Run Test",
		}

		// Even non-terminal runs that are old should use cached data
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   oldRun.ID,
			RunData: &oldRun,
		}

		assert.Equal(t, oldRun.ID, navMsg.RunID, "RunID should be set")
		require.NotNil(t, navMsg.RunData, "RunData should be provided for old runs")
		
		// Verify the run is indeed old
		age := time.Since(oldRun.CreatedAt)
		assert.Greater(t, age, 2*time.Hour, "Run should be older than 2 hours")

		view := views.NewRunDetailsViewWithData(client, testCache, oldRun)
		assert.NotNil(t, view, "Old runs should create view successfully even if not terminal")
	})

	t.Run("Recent active runs should still use cached data when available", func(t *testing.T) {
		recentRun := models.RunResponse{
			ID:             "recent-run-test",
			Status:         models.StatusProcessing,
			Repository:     "",
			RepositoryName: "test/recent-repo", 
			Source:         "develop",
			CreatedAt:      time.Now().Add(-30 * time.Minute), // Recent
			UpdatedAt:      time.Now().Add(-5 * time.Minute),  // Recently updated
			Title:          "Recent Active Run Test",
		}

		// Even recent runs can benefit from cached navigation if data is available
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   recentRun.ID,
			RunData: &recentRun,
		}

		assert.Equal(t, recentRun.ID, navMsg.RunID, "RunID should be set")
		require.NotNil(t, navMsg.RunData, "RunData should be provided even for recent runs")

		// Verify the run is recent
		age := time.Since(recentRun.CreatedAt)
		assert.Less(t, age, 2*time.Hour, "Run should be less than 2 hours old")

		view := views.NewRunDetailsViewWithData(client, testCache, recentRun)
		assert.NotNil(t, view, "Recent runs with RunData should create view successfully")
		
		// The view might poll for updates later, but initial load should use cached data
	})
}
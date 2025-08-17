// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetailsViewNavigationKeyConsistency tests the navigation key consistency fixes
func TestDetailsViewNavigationKeyConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create test client and cache
	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	// Create test run data
	testRun := models.RunResponse{
		ID:             "test-run-123",
		Status:         models.StatusDone,
		Repository:     "test/repo",
		RepositoryName: "test/repo",
		Source:         "main",
		CreatedAt:      time.Now().Add(-1 * time.Hour),
		Title:          "Test Run",
	}

	tests := []struct {
		name        string
		key         string
		expectKey   string
		shouldBlock bool
		description string
	}{
		{
			name:        "h key should use centralized system",
			key:         "h",
			expectKey:   "h",
			shouldBlock: false,
			description: "'h' should not be disabled and should use centralized system (ActionNavigateToDashboard)",
		},
		{
			name:        "q key should use centralized system",
			key:         "q",
			expectKey:   "q",
			shouldBlock: false,
			description: "'q' should not be disabled and should use centralized system (ActionNavigateToDashboard)",
		},
		{
			name:        "d key should use centralized system",
			key:         "d",
			expectKey:   "d",
			shouldBlock: false,
			description: "'d' should not be disabled and should use centralized system (ActionNavigateToDashboard)",
		},
		{
			name:        "b key should be disabled",
			key:         "b",
			expectKey:   "",
			shouldBlock: true,
			description: "'b' should be disabled to prevent inconsistent back navigation",
		},
		{
			name:        "backspace key should be disabled",
			key:         "backspace",
			expectKey:   "",
			shouldBlock: true,
			description: "'backspace' should be disabled to prevent inconsistent back navigation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh details view for each test
			view := NewRunDetailsViewWithData(client, testCache, testRun)
			require.NotNil(t, view, "Details view should be created")

			// Test CoreViewKeymap interface implementation
			require.Implements(t, (*interface {
				IsKeyDisabled(string) bool
				HandleKey(tea.KeyMsg) (bool, tea.Model, tea.Cmd)
			})(nil), view, "Details view should implement CoreViewKeymap interface")

			// Test IsKeyDisabled method
			isDisabled := view.IsKeyDisabled(tt.key)
			if tt.shouldBlock {
				assert.True(t, isDisabled, "Key '%s' should be disabled: %s", tt.key, tt.description)
			} else {
				assert.False(t, isDisabled, "Key '%s' should not be disabled: %s", tt.key, tt.description)
			}

			// Test HandleKey method
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "backspace" {
				keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
			}

			handled, model, _ := view.HandleKey(keyMsg)

			if tt.shouldBlock {
				// Disabled keys should not be handled by HandleKey
				// (they're blocked at the IsKeyDisabled level)
				assert.False(t, handled, "Disabled key '%s' should not be handled by HandleKey", tt.key)
			} else {
				// For navigation keys that should use centralized system
				if tt.key == "h" || tt.key == "q" || tt.key == "d" {
					// These keys should return handled=false to let centralized system handle them
					assert.False(t, handled, "Navigation key '%s' should return handled=false for centralized processing", tt.key)
					assert.Equal(t, view, model, "Model should remain the same view")
					// cmd can be nil or stopPolling command
				}
			}
		})
	}
}

// TestDetailsViewNavigationBehaviorIntegration tests the full navigation flow
func TestDetailsViewNavigationBehaviorIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	testRun := models.RunResponse{
		ID:             "test-integration-run",
		Status:         models.StatusDone,
		Repository:     "test/integration-repo",
		RepositoryName: "test/integration-repo",
		Source:         "main",
		CreatedAt:      time.Now().Add(-30 * time.Minute),
		Title:          "Integration Test Run",
	}

	view := NewRunDetailsViewWithData(client, testCache, testRun)

	t.Run("Consistent navigation keys work identically", func(t *testing.T) {
		consistentKeys := []string{"h", "q", "d"}

		for _, key := range consistentKeys {
			t.Run("Key: "+key, func(t *testing.T) {
				// All these keys should behave identically for navigation consistency

				// Should not be disabled
				assert.False(t, view.IsKeyDisabled(key), "Navigation key '%s' should not be disabled", key)

				// Should let centralized system handle (return handled=false)
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				handled, model, _ := view.HandleKey(keyMsg)

				assert.False(t, handled, "Navigation key '%s' should not be handled locally", key)
				assert.Equal(t, view, model, "Model should remain unchanged for key '%s'", key)

				// For 'q' key, there might be a stopPolling command, but that's fine
				// The important thing is that handled=false so centralized system processes it
			})
		}
	})

	t.Run("Disabled keys are consistently blocked", func(t *testing.T) {
		disabledKeys := []struct {
			key     string
			keyType tea.KeyType
			runes   []rune
		}{
			{"b", tea.KeyRunes, []rune("b")},
			{"backspace", tea.KeyBackspace, nil},
		}

		for _, keyInfo := range disabledKeys {
			t.Run("Key: "+keyInfo.key, func(t *testing.T) {
				// Should be disabled
				assert.True(t, view.IsKeyDisabled(keyInfo.key), "Key '%s' should be disabled", keyInfo.key)

				// Create appropriate KeyMsg
				var keyMsg tea.KeyMsg
				if keyInfo.runes != nil {
					keyMsg = tea.KeyMsg{Type: keyInfo.keyType, Runes: keyInfo.runes}
				} else {
					keyMsg = tea.KeyMsg{Type: keyInfo.keyType}
				}

				// HandleKey should not handle disabled keys
				handled, _, _ := view.HandleKey(keyMsg)
				assert.False(t, handled, "Disabled key '%s' should not be handled", keyInfo.key)
			})
		}
	})

	t.Run("Navigation through Update method respects disabled keys", func(t *testing.T) {
		// Test that disabled keys don't trigger navigation when sent through Update
		originalView := view

		// Test 'b' key (should be ignored)
		bKeyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}
		model, cmd := view.Update(bKeyMsg)

		assert.Equal(t, originalView, model, "View should remain unchanged for disabled 'b' key")
		assert.Nil(t, cmd, "No command should be returned for disabled 'b' key")

		// Test backspace key (should be ignored)
		backspaceMsg := tea.KeyMsg{Type: tea.KeyBackspace}
		model, cmd = view.Update(backspaceMsg)

		assert.Equal(t, originalView, model, "View should remain unchanged for disabled backspace key")
		assert.Nil(t, cmd, "No command should be returned for disabled backspace key")
	})
}

// TestDetailsViewCachedNavigationBehavior tests that navigation uses cached dashboard
func TestDetailsViewCachedNavigationBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	// Set up cached dashboard state
	dashboardState := map[string]interface{}{
		"selectedRepoIdx":    1,
		"selectedRunIdx":     2,
		"selectedDetailLine": 3,
		"focusedColumn":      2,
	}
	testCache.SetNavigationContext("dashboardState", dashboardState)

	// Set up some cached runs to simulate a populated dashboard
	cachedRuns := []models.RunResponse{
		{
			ID:             "cached-run-1",
			Status:         models.StatusDone,
			RepositoryName: "test/cached-repo1",
			CreatedAt:      time.Now().Add(-2 * time.Hour),
		},
		{
			ID:             "cached-run-2",
			Status:         models.StatusProcessing,
			RepositoryName: "test/cached-repo2",
			CreatedAt:      time.Now().Add(-1 * time.Hour),
		},
	}
	testCache.SetRuns(cachedRuns)

	testRun := models.RunResponse{
		ID:             "current-run",
		Status:         models.StatusDone,
		Repository:     "test/current-repo",
		RepositoryName: "test/current-repo",
		Source:         "main",
		CreatedAt:      time.Now().Add(-15 * time.Minute),
		Title:          "Current Run",
	}

	view := NewRunDetailsViewWithData(client, testCache, testRun)

	t.Run("Navigation context preserved for dashboard restoration", func(t *testing.T) {
		// Verify that navigation context exists (simulates dashboard state saving)
		storedState := testCache.GetNavigationContext("dashboardState")
		require.NotNil(t, storedState, "Dashboard state should be stored in navigation context")

		state, ok := storedState.(map[string]interface{})
		require.True(t, ok, "Dashboard state should be a map")

		assert.Equal(t, 1, state["selectedRepoIdx"], "Selected repo index should be preserved")
		assert.Equal(t, 2, state["selectedRunIdx"], "Selected run index should be preserved")
		assert.Equal(t, 3, state["selectedDetailLine"], "Selected detail line should be preserved")
		assert.Equal(t, 2, state["focusedColumn"], "Focused column should be preserved")
	})

	t.Run("Cached runs available for dashboard restoration", func(t *testing.T) {
		// Verify that cached runs exist for dashboard to use
		runs, cached, details := testCache.GetCachedList()

		assert.True(t, cached, "Runs should be cached")
		require.Len(t, runs, 2, "Should have cached runs")
		assert.NotNil(t, details, "Details map should exist")

		// Cache sorts by CreatedAt descending (newest first)
		// cached-run-2 was created 1 hour ago, cached-run-1 was created 2 hours ago
		assert.Equal(t, "cached-run-2", runs[0].ID, "Newer run should be first (cache sorts by CreatedAt desc)")
		assert.Equal(t, "cached-run-1", runs[1].ID, "Older run should be second")

		// Verify that GetRepositoryName works on cached runs
		assert.Equal(t, "test/cached-repo2", runs[0].GetRepositoryName(), "Repository name should be preserved for newer run")
		assert.Equal(t, "test/cached-repo1", runs[1].GetRepositoryName(), "Repository name should be preserved for older run")
	})

	t.Run("Key handler behavior for cached navigation", func(t *testing.T) {
		// Test that navigation keys return handled=false to use centralized system
		navigationKeys := []string{"h", "q", "d"}

		for _, key := range navigationKeys {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			handled, model, _ := view.HandleKey(keyMsg)

			assert.False(t, handled, "Key '%s' should not be handled locally for cached navigation", key)
			assert.Equal(t, view, model, "Model should remain unchanged for key '%s'", key)

			// The centralized system will use the cached data and navigation context
			// to restore the dashboard without API calls
		}
	})
}

// TestDetailsViewConstructorWithData tests the constructor that enables cached navigation
func TestDetailsViewConstructorWithData(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	client := api.NewClient("test-key", "http://localhost:8080", false)
	testCache := cache.NewSimpleCache()
	defer testCache.Stop()

	testRun := models.RunResponse{
		ID:             "constructor-test-run",
		Status:         models.StatusDone,
		Repository:     "test/constructor-repo",
		RepositoryName: "test/constructor-repo",
		Source:         "main",
		Target:         "feature/test",
		CreatedAt:      time.Now().Add(-45 * time.Minute),
		UpdatedAt:      time.Now().Add(-5 * time.Minute),
		Title:          "Constructor Test Run",
	}

	t.Run("NewRunDetailsViewWithData avoids loading state", func(t *testing.T) {
		view := NewRunDetailsViewWithData(client, testCache, testRun)

		assert.False(t, view.loading, "View should not be in loading state when created with data")
		assert.Equal(t, testRun.ID, view.run.ID, "Run ID should be set from provided data")
		assert.Equal(t, testRun.Status, view.run.Status, "Run status should be set from provided data")
		assert.Equal(t, testRun.Title, view.run.Title, "Run title should be set from provided data")

		// Verify that the view implements CoreViewKeymap
		require.Implements(t, (*interface {
			IsKeyDisabled(string) bool
			HandleKey(tea.KeyMsg) (bool, tea.Model, tea.Cmd)
		})(nil), view, "View created with data should implement CoreViewKeymap")

		// Verify navigation keys are properly configured
		assert.False(t, view.IsKeyDisabled("h"), "'h' should not be disabled")
		assert.False(t, view.IsKeyDisabled("q"), "'q' should not be disabled")
		assert.True(t, view.IsKeyDisabled("b"), "'b' should be disabled")
		assert.True(t, view.IsKeyDisabled("backspace"), "'backspace' should be disabled")
	})

	t.Run("Regular constructor comparison", func(t *testing.T) {
		// Create view with regular constructor (would typically load from API)
		regularView := NewRunDetailsView(client, testCache, testRun.ID)

		// Create view with data constructor (avoids API call)
		dataView := NewRunDetailsViewWithData(client, testCache, testRun)

		// Both should have the same navigation key configuration
		keys := []string{"h", "q", "d", "b", "backspace"}
		for _, key := range keys {
			assert.Equal(t, regularView.IsKeyDisabled(key), dataView.IsKeyDisabled(key),
				"Key '%s' disabled state should be the same for both constructors", key)
		}
	})
}

// TestNavigationMessageWithRunData tests the NavigateToDetailsMsg with RunData
func TestNavigationMessageWithRunData(t *testing.T) {
	testRun := models.RunResponse{
		ID:             "nav-test-run",
		Status:         models.StatusProcessing,
		Repository:     "test/nav-repo",
		RepositoryName: "test/nav-repo",
		Source:         "develop",
		CreatedAt:      time.Now().Add(-20 * time.Minute),
		Title:          "Navigation Test Run",
	}

	t.Run("NavigateToDetailsMsg with RunData", func(t *testing.T) {
		// Create navigation message with cached run data
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   testRun.ID,
			RunData: &testRun, // This avoids API call in the target view
		}

		assert.Equal(t, testRun.ID, navMsg.RunID, "RunID should be set")
		require.NotNil(t, navMsg.RunData, "RunData should be provided")
		assert.Equal(t, testRun.ID, navMsg.RunData.ID, "RunData should contain the correct run")
		assert.Equal(t, testRun.Status, navMsg.RunData.Status, "RunData should preserve status")
		assert.Equal(t, testRun.GetRepositoryName(), navMsg.RunData.GetRepositoryName(), "RunData should preserve repository name")
	})

	t.Run("NavigateToDetailsMsg without RunData", func(t *testing.T) {
		// Create navigation message without cached run data (would trigger API call)
		navMsg := messages.NavigateToDetailsMsg{
			RunID:   testRun.ID,
			RunData: nil, // This would cause API call in the target view
		}

		assert.Equal(t, testRun.ID, navMsg.RunID, "RunID should be set")
		assert.Nil(t, navMsg.RunData, "RunData should be nil for API-based navigation")
	})
}

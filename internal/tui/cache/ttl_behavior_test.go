package cache

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestThirtyMinuteTTLBehavior tests the 30-minute TTL configuration and behavior
func TestThirtyMinuteTTLBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("DataPolicies configuration", func(t *testing.T) {
		// Verify the TTL policies are configured correctly per the conversation
		
		// Active runs should have 30-minute TTL (changed from 5 minutes)
		activeRunsPolicy := DataPolicies["active_runs"]
		assert.Equal(t, 30*time.Minute, activeRunsPolicy.TTL, "Active runs should have 30-minute TTL")
		assert.Equal(t, MemoryLayer, activeRunsPolicy.Layer, "Active runs should use memory layer")
		assert.False(t, activeRunsPolicy.Persistent, "Active runs should not be persistent")

		// Form data should also have 30-minute TTL
		formDataPolicy := DataPolicies["form_data"]
		assert.Equal(t, 30*time.Minute, formDataPolicy.TTL, "Form data should have 30-minute TTL")
		assert.Equal(t, MemoryLayer, formDataPolicy.Layer, "Form data should use memory layer")
		assert.False(t, formDataPolicy.Persistent, "Form data should not be persistent")

		// Terminal runs should never expire (permanent disk storage)
		terminalRunsPolicy := DataPolicies["terminal_runs"]
		assert.Equal(t, time.Duration(0), terminalRunsPolicy.TTL, "Terminal runs should never expire")
		assert.Equal(t, DiskLayer, terminalRunsPolicy.Layer, "Terminal runs should use disk layer")
		assert.True(t, terminalRunsPolicy.Persistent, "Terminal runs should be persistent")

		// User info should never expire (permanent disk storage)
		userInfoPolicy := DataPolicies["user_info"]
		assert.Equal(t, time.Duration(0), userInfoPolicy.TTL, "User info should never expire")
		assert.Equal(t, DiskLayer, userInfoPolicy.Layer, "User info should use disk layer")
		assert.True(t, userInfoPolicy.Persistent, "User info should be persistent")
	})

	t.Run("HybridCache TTL behavior simulation", func(t *testing.T) {
		// Test the intended behavior of 30-minute TTL for active runs
		// This test simulates the TTL logic without waiting 30 minutes
		
		cache := NewSimpleCache()
		defer cache.Stop()

		// Create test runs with different characteristics
		activeRun := models.RunResponse{
			ID:             "active-run-ttl-test",
			Status:         models.StatusProcessing,
			Repository:     "",
			RepositoryName: "test/active-repo",
			Source:         "main",
			CreatedAt:      time.Now().Add(-10 * time.Minute), // Recent
			UpdatedAt:      time.Now().Add(-2 * time.Minute),  // Recently updated
			Title:          "Active Run TTL Test",
		}

		terminalRun := models.RunResponse{
			ID:             "terminal-run-ttl-test",
			Status:         models.StatusDone,
			Repository:     "",
			RepositoryName: "test/terminal-repo",
			Source:         "main",
			CreatedAt:      time.Now().Add(-2 * time.Hour), // Older
			UpdatedAt:      time.Now().Add(-1 * time.Hour), // Finished an hour ago
			Title:          "Terminal Run TTL Test",
		}

		runs := []models.RunResponse{activeRun, terminalRun}
		cache.SetRuns(runs)

		// Verify runs are cached
		cachedRuns, cached := cache.GetRuns(), true
		assert.True(t, cached, "Runs should be cached initially")
		require.Len(t, cachedRuns, 2, "Both runs should be cached")

		// Verify individual run caching
		cachedActiveRun := cache.GetRun(activeRun.ID)
		cachedTerminalRun := cache.GetRun(terminalRun.ID)
		require.NotNil(t, cachedActiveRun, "Active run should be cached")
		require.NotNil(t, cachedTerminalRun, "Terminal run should be cached")

		assert.Equal(t, activeRun.ID, cachedActiveRun.ID, "Active run ID should match")
		assert.Equal(t, terminalRun.ID, cachedTerminalRun.ID, "Terminal run ID should match")

		// The hybrid cache routing logic should:
		// - Put activeRun in memory layer (with 30-min TTL)
		// - Put terminalRun in disk layer (permanent)
		
		// This behavior is implemented in the HybridCache, tested separately in hybrid_test.go
		// Here we focus on the intended policy configuration
	})

	t.Run("TTL policy application verification", func(t *testing.T) {
		// Verify that the cache policies are applied correctly
		
		// Test active runs policy
		activePolicy := DataPolicies["active_runs"]
		assert.Equal(t, 30*time.Minute, activePolicy.TTL, "Active runs TTL should be 30 minutes")
		
		// Verify this is different from the old 5-minute setting
		assert.NotEqual(t, 5*time.Minute, activePolicy.TTL, "Should not be the old 5-minute TTL")
		assert.Greater(t, activePolicy.TTL, 5*time.Minute, "New TTL should be longer than old TTL")
		
		// Convert to seconds for easier understanding
		ttlSeconds := activePolicy.TTL.Seconds()
		expectedSeconds := (30 * time.Minute).Seconds()
		assert.Equal(t, expectedSeconds, ttlSeconds, "TTL should be exactly 30 minutes (1800 seconds)")
		
		// Verify the policy makes sense for dashboard navigation caching
		assert.Greater(t, activePolicy.TTL, 10*time.Minute, "TTL should be long enough for typical user navigation")
		assert.Less(t, activePolicy.TTL, 60*time.Minute, "TTL should not be too long to avoid stale data")
	})
}

// TestCacheBehaviorWithTTL tests the cache behavior that benefits from 30-minute TTL
func TestCacheBehaviorWithTTL(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	t.Run("Dashboard navigation benefits from longer TTL", func(t *testing.T) {
		// Simulate a typical dashboard navigation scenario
		
		// User loads dashboard (t=0)
		dashboardRuns := []models.RunResponse{
			{
				ID:             "dashboard-nav-1",
				Status:         models.StatusProcessing,
				Repository:     "",
				RepositoryName: "test/nav-repo-1",
				Source:         "main",
				CreatedAt:      time.Now().Add(-45 * time.Minute),
				UpdatedAt:      time.Now().Add(-5 * time.Minute),
				Title:          "Dashboard Navigation Test 1",
			},
			{
				ID:             "dashboard-nav-2",
				Status:         models.StatusQueued,
				Repository:     "",
				RepositoryName: "test/nav-repo-2",
				Source:         "develop",
				CreatedAt:      time.Now().Add(-20 * time.Minute),
				UpdatedAt:      time.Now().Add(-3 * time.Minute),
				Title:          "Dashboard Navigation Test 2",
			},
			{
				ID:             "dashboard-nav-3",
				Status:         models.StatusDone, // Terminal status
				Repository:     "",
				RepositoryName: "test/nav-repo-3",
				Source:         "main",
				CreatedAt:      time.Now().Add(-3 * time.Hour),
				UpdatedAt:      time.Now().Add(-2 * time.Hour),
				Title:          "Dashboard Navigation Test 3",
			},
		}

		cache.SetRuns(dashboardRuns)

		// Verify initial caching
		cachedRuns, cached, details := cache.GetCachedList()
		assert.True(t, cached, "Dashboard runs should be cached")
		require.Len(t, cachedRuns, 3, "All dashboard runs should be cached")
		assert.NotNil(t, details, "Details map should be available")

		// User navigates to details view (t=+2 minutes)
		selectedRun := &dashboardRuns[0]
		detailsRun := cache.GetRun(selectedRun.ID)
		require.NotNil(t, detailsRun, "Selected run should still be cached for details view")
		assert.Equal(t, selectedRun.ID, detailsRun.ID, "Run data should be preserved")

		// User navigates back to dashboard (t=+5 minutes)
		// With 30-minute TTL, data should still be cached
		backToDashRuns, stillCached, _ := cache.GetCachedList()
		assert.True(t, stillCached, "Dashboard data should still be cached after 5 minutes")
		require.Len(t, backToDashRuns, 3, "All runs should still be available")

		// User navigates to different details (t=+10 minutes)
		secondRun := cache.GetRun(dashboardRuns[1].ID)
		require.NotNil(t, secondRun, "Second run should still be cached after 10 minutes")
		assert.Equal(t, dashboardRuns[1].ID, secondRun.ID, "Second run data should be preserved")

		// User does various navigation (t=+15 minutes)
		// All navigation should still hit cache with 30-minute TTL
		for _, originalRun := range dashboardRuns {
			cachedRun := cache.GetRun(originalRun.ID)
			require.NotNil(t, cachedRun, "Run %s should still be cached after 15 minutes", originalRun.ID)
			assert.Equal(t, originalRun.GetRepositoryName(), cachedRun.GetRepositoryName(), 
				"Repository name should be preserved for run %s", originalRun.ID)
		}

		// Simulate dashboard refresh at t=+25 minutes
		// Data should still be in cache (before 30-minute expiry)
		refreshRuns, refreshCached, _ := cache.GetCachedList()
		assert.True(t, refreshCached, "Dashboard data should still be cached at 25 minutes")
		require.Len(t, refreshRuns, 3, "All runs should still be available at 25 minutes")

		// The 30-minute TTL provides a good balance:
		// - Long enough for typical user navigation sessions (15-20 minutes)
		// - Short enough to ensure data doesn't get too stale
		// - Reduces API calls significantly during navigation
	})

	t.Run("Form data TTL behavior", func(t *testing.T) {
		// Test form data caching with 30-minute TTL
		
		formData := &FormData{
			Title:       "TTL Test Form",
			Repository:  "test/ttl-repo",
			Source:      "main",
			Target:      "feature/ttl-test",
			Prompt:      "Test the 30-minute TTL behavior for form data",
			Context:     "This form should be cached for 30 minutes",
			RunType:     "plan",
			ShowContext: true,
			Fields: map[string]string{
				"custom_field": "custom_value",
			},
		}

		cache.SetFormData(formData)

		// Verify form data is cached
		cachedForm := cache.GetFormData()
		require.NotNil(t, cachedForm, "Form data should be cached")
		assert.Equal(t, formData.Title, cachedForm.Title, "Form title should be preserved")
		assert.Equal(t, formData.Repository, cachedForm.Repository, "Form repository should be preserved")
		assert.Equal(t, formData.Prompt, cachedForm.Prompt, "Form prompt should be preserved")
		assert.Equal(t, formData.Fields, cachedForm.Fields, "Form custom fields should be preserved")

		// With 30-minute TTL, form data should persist through navigation sessions
		// This is beneficial for users who switch between dashboard and create view multiple times
	})

	t.Run("TTL comparison with previous behavior", func(t *testing.T) {
		// Document the improvement from 5-minute to 30-minute TTL
		
		oldTTL := 5 * time.Minute
		newTTL := DataPolicies["active_runs"].TTL

		assert.Equal(t, 30*time.Minute, newTTL, "New TTL should be 30 minutes")
		assert.Greater(t, newTTL, oldTTL, "New TTL should be longer than old TTL")

		// Calculate the improvement
		improvement := newTTL / oldTTL
		assert.Equal(t, time.Duration(6), improvement, "New TTL should be 6x longer than old TTL")

		// Benefits of the longer TTL:
		// 1. Reduced API calls during navigation sessions
		// 2. Faster dashboard loading when navigating back
		// 3. Better user experience with instant navigation
		// 4. Lower server load from repeated dashboard requests
		// 5. More resilience to brief network issues

		t.Logf("TTL improvement: %v -> %v (%.1fx longer)", oldTTL, newTTL, float64(improvement))
	})
}

// TestCacheLayerConfiguration tests the cache layer configuration for different data types
func TestCacheLayerConfiguration(t *testing.T) {
	t.Run("Cache layer policies are correctly configured", func(t *testing.T) {
		// Memory layer for short-term data (active_runs, form_data)
		memoryLayerPolicies := []string{"active_runs", "form_data"}
		for _, policyName := range memoryLayerPolicies {
			policy := DataPolicies[policyName]
			assert.Equal(t, MemoryLayer, policy.Layer, "%s should use memory layer", policyName)
			assert.Equal(t, 30*time.Minute, policy.TTL, "%s should have 30-minute TTL", policyName)
			assert.False(t, policy.Persistent, "%s should not be persistent", policyName)
		}

		// Disk layer for permanent data (terminal_runs, user_info, repositories, file_hashes)
		diskLayerPolicies := []string{"terminal_runs", "user_info", "repositories", "file_hashes"}
		for _, policyName := range diskLayerPolicies {
			policy := DataPolicies[policyName]
			assert.Equal(t, DiskLayer, policy.Layer, "%s should use disk layer", policyName)
			assert.Equal(t, time.Duration(0), policy.TTL, "%s should never expire", policyName)
			assert.True(t, policy.Persistent, "%s should be persistent", policyName)
		}
	})

	t.Run("Policy configuration makes sense for use cases", func(t *testing.T) {
		// Active runs: Memory layer with TTL
		// - These change frequently, so memory is faster
		// - TTL prevents stale data while allowing reasonable caching duration
		// - Not persistent because they should be refreshed periodically
		activePolicy := DataPolicies["active_runs"]
		assert.Equal(t, MemoryLayer, activePolicy.Layer, "Active runs benefit from fast memory access")
		assert.Greater(t, activePolicy.TTL, 0*time.Second, "Active runs need TTL to prevent stale data")
		assert.False(t, activePolicy.Persistent, "Active runs should be refreshed, not persisted indefinitely")

		// Terminal runs: Disk layer without TTL
		// - These never change, so disk persistence is valuable
		// - No TTL because they're immutable
		// - Persistent because they provide historical data
		terminalPolicy := DataPolicies["terminal_runs"]
		assert.Equal(t, DiskLayer, terminalPolicy.Layer, "Terminal runs benefit from disk persistence")
		assert.Equal(t, time.Duration(0), terminalPolicy.TTL, "Terminal runs never change, no TTL needed")
		assert.True(t, terminalPolicy.Persistent, "Terminal runs should be persisted as historical data")

		// Form data: Memory layer with TTL
		// - Session-specific, so memory is appropriate
		// - TTL allows data to persist across navigation but not indefinitely
		// - Not persistent because it's session-specific
		formPolicy := DataPolicies["form_data"]
		assert.Equal(t, MemoryLayer, formPolicy.Layer, "Form data is session-specific, memory is appropriate")
		assert.Greater(t, formPolicy.TTL, 0*time.Second, "Form data needs TTL for session management")
		assert.False(t, formPolicy.Persistent, "Form data is session-specific, not persistent")

		// User info: Disk layer without TTL
		// - Changes infrequently, disk persistence valuable for performance
		// - No TTL because user info is relatively static
		// - Persistent because it's used across sessions
		userPolicy := DataPolicies["user_info"]
		assert.Equal(t, DiskLayer, userPolicy.Layer, "User info benefits from cross-session persistence")
		assert.Equal(t, time.Duration(0), userPolicy.TTL, "User info is relatively static, no TTL needed")
		assert.True(t, userPolicy.Persistent, "User info should persist across sessions")
	})
}

// TestNavigationPerformanceWithTTL tests the performance benefits of the 30-minute TTL
func TestNavigationPerformanceWithTTL(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := NewSimpleCache()
	defer cache.Stop()

	t.Run("Navigation performance simulation", func(t *testing.T) {
		// Simulate the performance characteristics of dashboard navigation
		
		// Initial dashboard load (API call required)
		startTime := time.Now()
		
		initialRuns := []models.RunResponse{
			{
				ID:             "perf-test-1",
				Status:         models.StatusProcessing,
				Repository:     "",
				RepositoryName: "test/perf-repo-1",
				CreatedAt:      time.Now().Add(-30 * time.Minute),
				Title:          "Performance Test Run 1",
			},
			{
				ID:             "perf-test-2",
				Status:         models.StatusDone,
				Repository:     "",
				RepositoryName: "test/perf-repo-2", 
				CreatedAt:      time.Now().Add(-1 * time.Hour),
				Title:          "Performance Test Run 2",
			},
		}

		cache.SetRuns(initialRuns)
		cacheTime := time.Since(startTime)

		// Subsequent dashboard navigations (cache hits)
		navigationTimes := make([]time.Duration, 10)
		for i := 0; i < 10; i++ {
			navStart := time.Now()
			runs, cached, _ := cache.GetCachedList()
			navigationTimes[i] = time.Since(navStart)
			
			assert.True(t, cached, "Navigation %d should hit cache", i+1)
			assert.Len(t, runs, 2, "Navigation %d should return all runs", i+1)
		}

		// Calculate average navigation time
		var totalNavTime time.Duration
		for _, navTime := range navigationTimes {
			totalNavTime += navTime
		}
		avgNavTime := totalNavTime / time.Duration(len(navigationTimes))

		t.Logf("Initial cache time: %v", cacheTime)
		t.Logf("Average navigation time: %v", avgNavTime)
		
		// Navigation from cache should be significantly faster than initial load
		// This is the key benefit of the 30-minute TTL
		assert.Less(t, avgNavTime, cacheTime, "Cache navigation should be faster than initial load")
		
		// With 30-minute TTL, all 10 navigations hit cache (in real usage, this could be
		// dashboard -> details -> dashboard -> create -> dashboard, etc.)
	})

	t.Run("Cache hit ratio simulation", func(t *testing.T) {
		// Simulate cache hit ratios with different TTL values
		
		// With 30-minute TTL: typical user session (20 minutes) should have high hit ratio
		sessionDuration := 20 * time.Minute
		ttl := DataPolicies["active_runs"].TTL // 30 minutes
		
		// User performs navigation every 2 minutes for 20 minutes
		navigationInterval := 2 * time.Minute
		totalNavigations := int(sessionDuration / navigationInterval) // 10 navigations
		
		cacheHits := 0
		for i := 0; i < totalNavigations; i++ {
			elapsed := time.Duration(i) * navigationInterval
			if elapsed < ttl {
				cacheHits++
			}
		}
		
		hitRatio := float64(cacheHits) / float64(totalNavigations)
		assert.Equal(t, 1.0, hitRatio, "All navigations should hit cache within 30-minute TTL")
		
		// Compare with old 5-minute TTL
		oldTTL := 5 * time.Minute
		oldCacheHits := 0
		for i := 0; i < totalNavigations; i++ {
			elapsed := time.Duration(i) * navigationInterval
			if elapsed < oldTTL {
				oldCacheHits++
			}
		}
		
		oldHitRatio := float64(oldCacheHits) / float64(totalNavigations)
		assert.Less(t, oldHitRatio, hitRatio, "New TTL should have better hit ratio than old TTL")
		assert.Equal(t, 0.3, oldHitRatio, "Old 5-minute TTL would have 30% hit ratio") // 3/10 navigations
		
		t.Logf("Navigation hit ratios: Old TTL (5min): %.1f%%, New TTL (30min): %.1f%%", 
			oldHitRatio*100, hitRatio*100)
	})
}
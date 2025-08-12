package cache

import (
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSessionCache_OnlyStoresActiveRuns(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Should store active run
	activeRun := models.RunResponse{
		ID:        "test-1",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}
	err := cache.SetRun(activeRun)
	assert.NoError(t, err)
	
	cached, found := cache.GetRun("test-1")
	assert.True(t, found, "active run should be cached")
	assert.Equal(t, activeRun.ID, cached.ID)
	assert.Equal(t, activeRun.Status, cached.Status)
	
	// Should not store terminal run
	terminalRun := models.RunResponse{
		ID:        "test-2",
		Status:    models.StatusDone,
		CreatedAt: time.Now(),
	}
	err = cache.SetRun(terminalRun)
	assert.NoError(t, err)
	
	_, found = cache.GetRun("test-2")
	assert.False(t, found, "terminal run should not be cached in session")
}

func TestSessionCache_AutoRemovalOfTerminalRuns(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Store an active run
	run := models.RunResponse{
		ID:        "test-run",
		Status:    models.StatusProcessing,
		CreatedAt: time.Now(),
	}
	err := cache.SetRun(run)
	assert.NoError(t, err)
	
	// Verify it's cached
	cached, found := cache.GetRun("test-run")
	assert.True(t, found)
	assert.Equal(t, models.StatusProcessing, cached.Status)
	
	// Update to terminal status
	run.Status = models.StatusDone
	err = cache.SetRun(run)
	assert.NoError(t, err)
	
	// Should be removed from cache
	_, found = cache.GetRun("test-run")
	assert.False(t, found, "terminal run should be removed from session cache")
}

func TestSessionCache_BulkRunOperations(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	runs := []models.RunResponse{
		{ID: "run-1", Status: models.StatusQueued},
		{ID: "run-2", Status: models.StatusProcessing},
		{ID: "run-3", Status: models.StatusDone},      // Terminal
		{ID: "run-4", Status: models.StatusFailed},    // Terminal
		{ID: "run-5", Status: models.StatusInitializing},
	}
	
	err := cache.SetRuns(runs)
	assert.NoError(t, err)
	
	// GetRuns should only return active runs
	cachedRuns, found := cache.GetRuns()
	assert.True(t, found)
	assert.Len(t, cachedRuns, 3, "should only return active runs")
	
	// Verify only active runs are returned
	for _, run := range cachedRuns {
		assert.False(t, isTerminalState(run.Status), "should not return terminal runs")
	}
}

func TestSessionCache_InvalidateRun(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Add a run
	run := models.RunResponse{
		ID:     "test-run",
		Status: models.StatusProcessing,
	}
	_ = cache.SetRun(run)
	
	// Verify it exists
	_, found := cache.GetRun("test-run")
	assert.True(t, found)
	
	// Invalidate the run
	err := cache.InvalidateRun("test-run")
	assert.NoError(t, err)
	
	// Should no longer exist
	_, found = cache.GetRun("test-run")
	assert.False(t, found, "run should be invalidated")
}

func TestSessionCache_InvalidateActiveRuns(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Add multiple runs
	runs := []models.RunResponse{
		{ID: "run-1", Status: models.StatusQueued},
		{ID: "run-2", Status: models.StatusProcessing},
		{ID: "run-3", Status: models.StatusInitializing},
	}
	
	for _, run := range runs {
		_ = cache.SetRun(run)
	}
	
	// Verify they exist
	cachedRuns, found := cache.GetRuns()
	assert.True(t, found)
	assert.Len(t, cachedRuns, 3)
	
	// Invalidate all active runs
	err := cache.InvalidateActiveRuns()
	assert.NoError(t, err)
	
	// Should have no runs
	cachedRuns, found = cache.GetRuns()
	assert.False(t, found, "should have no runs after invalidation")
	assert.Empty(t, cachedRuns)
	
	// Individual runs should also be gone
	for _, run := range runs {
		_, found := cache.GetRun(run.ID)
		assert.False(t, found, "individual run should be invalidated")
	}
}

func TestSessionCache_FormData(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Store form data
	formData := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}
	err := cache.SetFormData("create-run-form", formData)
	assert.NoError(t, err)
	
	// Retrieve form data
	cached, found := cache.GetFormData("create-run-form")
	assert.True(t, found)
	
	cachedForm, ok := cached.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, formData, cachedForm)
	
	// Non-existent form data
	_, found = cache.GetFormData("non-existent")
	assert.False(t, found)
}

func TestSessionCache_DashboardData(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Create dashboard data
	dashData := &DashboardData{
		Runs: []models.RunResponse{
			{ID: "run-1", Status: models.StatusProcessing},
			{ID: "run-2", Status: models.StatusDone},
		},
		UserInfo: &models.UserInfo{
			ID:    123,
			Email: "test@example.com",
		},
		RepositoryList: []string{"repo1", "repo2"},
		LastUpdated:    time.Now(),
	}
	
	// Store dashboard data
	err := cache.SetDashboardData(dashData)
	assert.NoError(t, err)
	
	// Retrieve dashboard data
	cached, found := cache.GetDashboardData()
	assert.True(t, found)
	assert.NotNil(t, cached)
	assert.Len(t, cached.Runs, 2)
	assert.Equal(t, dashData.UserInfo.ID, cached.UserInfo.ID)
	assert.Equal(t, dashData.RepositoryList, cached.RepositoryList)
}

func TestSessionCache_Clear(t *testing.T) {
	cache := NewSessionCache()
	defer cache.Close()
	
	// Add various data
	run := models.RunResponse{
		ID:     "test-run",
		Status: models.StatusProcessing,
	}
	_ = cache.SetRun(run)
	_ = cache.SetFormData("form1", "data")
	
	dashData := &DashboardData{
		Runs:        []models.RunResponse{run},
		LastUpdated: time.Now(),
	}
	_ = cache.SetDashboardData(dashData)
	
	// Verify data exists
	_, found := cache.GetRun("test-run")
	assert.True(t, found)
	_, found = cache.GetFormData("form1")
	assert.True(t, found)
	_, found = cache.GetDashboardData()
	assert.True(t, found)
	
	// Clear cache
	err := cache.Clear()
	assert.NoError(t, err)
	
	// Verify all data is gone
	_, found = cache.GetRun("test-run")
	assert.False(t, found, "run should be cleared")
	_, found = cache.GetFormData("form1")
	assert.False(t, found, "form data should be cleared")
	_, found = cache.GetDashboardData()
	assert.False(t, found, "dashboard data should be cleared")
}

func TestSessionCache_TTLBehavior(t *testing.T) {
	// This is a conceptual test - in real usage, TTL would expire items
	// For unit testing, we're just verifying the cache accepts TTL operations
	cache := NewSessionCache()
	defer cache.Close()
	
	// Add run with TTL (5 minutes by default)
	run := models.RunResponse{
		ID:     "ttl-test",
		Status: models.StatusProcessing,
	}
	err := cache.SetRun(run)
	assert.NoError(t, err)
	
	// Should be immediately available
	cached, found := cache.GetRun("ttl-test")
	assert.True(t, found)
	assert.Equal(t, run.ID, cached.ID)
	
	// Form data has 30-minute TTL
	err = cache.SetFormData("ttl-form", "test-data")
	assert.NoError(t, err)
	
	data, found := cache.GetFormData("ttl-form")
	assert.True(t, found)
	assert.Equal(t, "test-data", data)
}
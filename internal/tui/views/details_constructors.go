package views

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// RunDetailsViewConfig holds configuration for creating a new RunDetailsView
type RunDetailsViewConfig struct {
	Client             APIClient
	Run                models.RunResponse
	ParentRuns         []models.RunResponse
	ParentCached       bool
	ParentCachedAt     time.Time
	ParentDetailsCache map[string]*models.RunResponse
	Cache              *cache.SimpleCache // Optional embedded cache
	// Dashboard state for restoration
	DashboardSelectedRepoIdx    int
	DashboardSelectedRunIdx     int
	DashboardSelectedDetailLine int
	DashboardFocusedColumn      int
}

// NewRunDetailsView creates a new RunDetailsView with default configuration
func NewRunDetailsView(client APIClient, run models.RunResponse) *RunDetailsView {
	// Create new cache instance
	cache := cache.NewSimpleCache()
	_ = cache.LoadFromDisk()

	// Get cached data
	runs, cached, detailsCache := cache.GetCachedList()
	var cachedAt time.Time
	if cached {
		cachedAt = time.Now()
	}
	return NewRunDetailsViewWithCache(client, run, runs, cached, cachedAt, detailsCache, cache)
}

// NewRunDetailsViewWithConfig creates a new RunDetailsView with the given configuration
func NewRunDetailsViewWithConfig(config RunDetailsViewConfig) *RunDetailsView {
	// Use provided cache or create new one
	embeddedCache := config.Cache
	if embeddedCache == nil {
		embeddedCache = cache.NewSimpleCache()
		_ = embeddedCache.LoadFromDisk()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	vp := viewport.New(80, 20)

	// Check if we have preloaded data for this run
	needsLoading := true
	run := config.Run
	runID := run.GetIDString()

	// Check cache for preloaded data
	if config.ParentDetailsCache != nil {
		if cachedRun, exists := config.ParentDetailsCache[runID]; exists && cachedRun != nil {
			debug.LogToFilef("DEBUG: Cache HIT for runID='%s'\n", runID)
			run = *cachedRun
			needsLoading = false
		} else {
			debug.LogToFilef("DEBUG: Cache MISS for runID='%s'\n", runID)
		}
	}

	v := &RunDetailsView{
		client:                      config.Client,
		run:                         run,
		keys:                        components.DefaultKeyMap,
		help:                        help.New(),
		viewport:                    vp,
		spinner:                     s,
		loading:                     needsLoading,
		showLogs:                    false,
		parentRuns:                  config.ParentRuns,
		parentCached:                config.ParentCached,
		parentCachedAt:              config.ParentCachedAt,
		parentDetailsCache:          config.ParentDetailsCache,
		statusHistory:               make([]string, 0),
		cacheRetryCount:             0,
		maxCacheRetries:             3,
		statusLine:                  components.NewStatusLine(),
		dashboardSelectedRepoIdx:    config.DashboardSelectedRepoIdx,
		dashboardSelectedRunIdx:     config.DashboardSelectedRunIdx,
		dashboardSelectedDetailLine: config.DashboardSelectedDetailLine,
		dashboardFocusedColumn:      config.DashboardFocusedColumn,
		cache:                       embeddedCache,
	}

	// Initialize status history with current status if we have cached data
	if !needsLoading {
		v.updateStatusHistory(string(run.Status), false)
		v.updateContent()
	}

	// Start in navigation mode
	v.navigationMode = true

	return v
}

// NewRunDetailsViewWithDashboardState creates a new details view with dashboard state for restoration
func NewRunDetailsViewWithDashboardState(
	client APIClient,
	run models.RunResponse,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt time.Time,
	parentDetailsCache map[string]*models.RunResponse,
	width int,
	height int,
	selectedRepoIdx int,
	selectedRunIdx int,
	selectedDetailLine int,
	focusedColumn int,
) *RunDetailsView {
	config := RunDetailsViewConfig{
		Client:                      client,
		Run:                         run,
		ParentRuns:                  parentRuns,
		ParentCached:                parentCached,
		ParentCachedAt:              parentCachedAt,
		ParentDetailsCache:          parentDetailsCache,
		DashboardSelectedRepoIdx:    selectedRepoIdx,
		DashboardSelectedRunIdx:     selectedRunIdx,
		DashboardSelectedDetailLine: selectedDetailLine,
		DashboardFocusedColumn:      focusedColumn,
	}

	v := NewRunDetailsViewWithConfig(config)

	// Set dimensions immediately if provided
	if width > 0 && height > 0 {
		v.width = width
		v.height = height
		// Apply dimensions to viewport immediately
		v.handleWindowSizeMsg(tea.WindowSizeMsg{Width: width, Height: height})
	}

	return v
}

// NewRunDetailsViewWithCacheAndDimensions creates a new details view with cache and dimensions
func NewRunDetailsViewWithCacheAndDimensions(
	client APIClient,
	run models.RunResponse,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt time.Time,
	parentDetailsCache map[string]*models.RunResponse,
	width int,
	height int,
) *RunDetailsView {
	// Create cache for this view
	cache := cache.NewSimpleCache()
	v := NewRunDetailsViewWithCache(client, run, parentRuns, parentCached, parentCachedAt, parentDetailsCache, cache)

	// Set dimensions immediately if provided
	if width > 0 && height > 0 {
		v.width = width
		v.height = height
		// Apply dimensions to viewport immediately
		v.handleWindowSizeMsg(tea.WindowSizeMsg{Width: width, Height: height})
	}

	return v
}

// NewRunDetailsViewWithCache maintains backward compatibility
func NewRunDetailsViewWithCache(
	client APIClient,
	run models.RunResponse,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt time.Time,
	parentDetailsCache map[string]*models.RunResponse,
	embeddedCache *cache.SimpleCache,
) *RunDetailsView {
	config := RunDetailsViewConfig{
		Client:             client,
		Run:                run,
		ParentRuns:         parentRuns,
		ParentCached:       parentCached,
		ParentCachedAt:     parentCachedAt,
		ParentDetailsCache: parentDetailsCache,
		Cache:              embeddedCache,
	}

	return NewRunDetailsViewWithConfig(config)
}
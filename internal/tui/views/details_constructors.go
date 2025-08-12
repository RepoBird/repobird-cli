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

// DetailsOption is a functional option for configuring RunDetailsView
type DetailsOption func(*RunDetailsView)

// WithCache sets a custom cache for the details view
func WithCache(c *cache.SimpleCache) DetailsOption {
	return func(v *RunDetailsView) {
		v.cache = c
	}
}

// WithDimensions sets the width and height for the details view
func WithDimensions(width, height int) DetailsOption {
	return func(v *RunDetailsView) {
		if width > 0 && height > 0 {
			v.width = width
			v.height = height
		}
	}
}

// WithDashboardState configures the view to return to dashboard with restored state
func WithDashboardState(selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn int) DetailsOption {
	return func(v *RunDetailsView) {
		v.dashboardSelectedRepoIdx = selectedRepoIdx
		v.dashboardSelectedRunIdx = selectedRunIdx
		v.dashboardSelectedDetailLine = selectedDetailLine
		v.dashboardFocusedColumn = focusedColumn
	}
}

// WithParentData sets parent runs and cache data
func WithParentData(parentRuns []models.RunResponse, parentCached bool, parentCachedAt time.Time, parentDetailsCache map[string]*models.RunResponse) DetailsOption {
	return func(v *RunDetailsView) {
		v.parentRuns = parentRuns
		v.parentCached = parentCached
		v.parentCachedAt = parentCachedAt
		v.parentDetailsCache = parentDetailsCache
	}
}

// WithConfig applies all settings from a RunDetailsViewConfig
func WithConfig(config RunDetailsViewConfig) DetailsOption {
	return func(v *RunDetailsView) {
		// Apply all config fields
		if config.Cache != nil {
			v.cache = config.Cache
		}
		v.parentRuns = config.ParentRuns
		v.parentCached = config.ParentCached
		v.parentCachedAt = config.ParentCachedAt
		v.parentDetailsCache = config.ParentDetailsCache
		v.dashboardSelectedRepoIdx = config.DashboardSelectedRepoIdx
		v.dashboardSelectedRunIdx = config.DashboardSelectedRunIdx
		v.dashboardSelectedDetailLine = config.DashboardSelectedDetailLine
		v.dashboardFocusedColumn = config.DashboardFocusedColumn
	}
}

// NewRunDetailsView creates a new RunDetailsView with functional options
func NewRunDetailsView(client APIClient, run models.RunResponse, opts ...DetailsOption) *RunDetailsView {
	// Create default cache instance
	defaultCache := cache.NewSimpleCache()
	_ = defaultCache.LoadFromDisk()

	// Get cached data
	runs, cached, detailsCache := defaultCache.GetCachedList()
	var cachedAt time.Time
	if cached {
		cachedAt = time.Now()
	}

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	// Create viewport
	vp := viewport.New(80, 20)

	// Check if we have preloaded data for this run
	needsLoading := true
	runID := run.GetIDString()

	// Check cache for preloaded data
	if detailsCache != nil {
		if cachedRun, exists := detailsCache[runID]; exists && cachedRun != nil {
			debug.LogToFilef("DEBUG: Cache HIT for runID='%s'\n", runID)
			run = *cachedRun
			needsLoading = false
		} else {
			debug.LogToFilef("DEBUG: Cache MISS for runID='%s'\n", runID)
		}
	}

	// Create the view with defaults
	v := &RunDetailsView{
		client:             client,
		run:                run,
		keys:               components.DefaultKeyMap,
		help:               help.New(),
		viewport:           vp,
		spinner:            s,
		loading:            needsLoading,
		showLogs:           false,
		parentRuns:         runs,
		parentCached:       cached,
		parentCachedAt:     cachedAt,
		parentDetailsCache: detailsCache,
		statusHistory:      make([]string, 0),
		cacheRetryCount:    0,
		maxCacheRetries:    3,
		statusLine:         components.NewStatusLine(),
		cache:              defaultCache,
		width:              80,   // default width
		height:             24,   // default height
		navigationMode:     true, // Start in navigation mode
	}

	// Apply all options
	for _, opt := range opts {
		opt(v)
	}

	// Initialize status history with current status if we have cached data
	if !needsLoading {
		v.updateStatusHistory(string(run.Status), false)
		v.updateContent()
	}

	// Apply dimensions to viewport after options
	if v.width > 0 && v.height > 0 {
		v.handleWindowSizeMsg(tea.WindowSizeMsg{Width: v.width, Height: v.height})
	}

	return v
}

// NewRunDetailsViewWithConfig creates a new RunDetailsView with the given configuration (backward compatibility)
func NewRunDetailsViewWithConfig(config RunDetailsViewConfig) *RunDetailsView {
	return NewRunDetailsView(config.Client, config.Run, WithConfig(config))
}

// NewRunDetailsViewWithDashboardState creates a new details view with dashboard state for restoration (backward compatibility)
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
	return NewRunDetailsView(client, run,
		WithParentData(parentRuns, parentCached, parentCachedAt, parentDetailsCache),
		WithDimensions(width, height),
		WithDashboardState(selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn),
	)
}

// NewRunDetailsViewWithCacheAndDimensions creates a new details view with cache and dimensions (backward compatibility)
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
	c := cache.NewSimpleCache()
	return NewRunDetailsView(client, run,
		WithCache(c),
		WithParentData(parentRuns, parentCached, parentCachedAt, parentDetailsCache),
		WithDimensions(width, height),
	)
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
	return NewRunDetailsView(client, run,
		WithCache(embeddedCache),
		WithParentData(parentRuns, parentCached, parentCachedAt, parentDetailsCache),
	)
}

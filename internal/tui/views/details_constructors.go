// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// NewRunDetailsView creates a new RunDetailsView with minimal parameters (new pattern)
func NewRunDetailsView(client APIClient, cache *cache.SimpleCache, runID string) *RunDetailsView {
	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	// Create viewport
	vp := viewport.New(80, 20)

	// Create the view with minimal state
	v := &RunDetailsView{
		client:           client,
		runID:            runID,
		run:              models.RunResponse{ID: runID}, // Minimal run object for loading
		keys:             components.DefaultKeyMap,
		help:             help.New(),
		viewport:         vp,
		spinner:          s,
		loading:          true, // Always start loading
		showLogs:         false,
		statusHistory:    make([]string, 0),
		cacheRetryCount:  0,
		maxCacheRetries:  3,
		statusLine:       components.NewStatusLine(),
		clipboardManager: components.NewClipboardManager(),
		cache:            cache, // Shared cache from app level
		width:            0,     // Don't set default width - wait for WindowSizeMsg
		height:           0,     // Don't set default height - wait for WindowSizeMsg
		navigationMode:   true,  // Start in navigation mode
		layout:           nil,   // Don't initialize layout until we have dimensions
	}

	return v
}

// NewRunDetailsViewWithData creates a new RunDetailsView with pre-loaded run data (avoids API calls)
func NewRunDetailsViewWithData(client APIClient, cache *cache.SimpleCache, run models.RunResponse) *RunDetailsView {
	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	// Create viewport
	vp := viewport.New(80, 20)

	// Create the view with the provided run data (no loading needed)
	v := &RunDetailsView{
		client:           client,
		runID:            run.GetIDString(),
		run:              run, // Use the provided run data
		keys:             components.DefaultKeyMap,
		help:             help.New(),
		viewport:         vp,
		spinner:          s,
		loading:          false, // Data is already loaded!
		showLogs:         false,
		statusHistory:    make([]string, 0),
		cacheRetryCount:  0,
		maxCacheRetries:  3,
		statusLine:       components.NewStatusLine(),
		clipboardManager: components.NewClipboardManager(),
		cache:            cache, // Shared cache from app level
		width:            0,     // Don't set default width - wait for WindowSizeMsg
		height:           0,     // Don't set default height - wait for WindowSizeMsg
		navigationMode:   true,  // Start in navigation mode
		layout:           nil,   // Don't initialize layout until we have dimensions
	}

	// Pre-populate the status history and content
	v.updateStatusHistory(string(run.Status), false)
	v.updateContentRefactored() // Initialize content with the run data

	return v
}

// Backward compatibility constructors - these delegate to the new minimal constructor

// RunDetailsViewConfig holds configuration for creating a new RunDetailsView
type RunDetailsViewConfig struct {
	Client APIClient
	RunID  string // Just the run ID, view will load its own data
}

// DetailsOption is a functional option for configuring RunDetailsView (deprecated)
type DetailsOption func(*RunDetailsView)

// WithCache sets a custom cache for the details view (deprecated)
func WithCache(c *cache.SimpleCache) DetailsOption {
	return func(v *RunDetailsView) {
		v.cache = c
	}
}

// WithDimensions sets the width and height for the details view (deprecated)
func WithDimensions(width, height int) DetailsOption {
	return func(v *RunDetailsView) {
		if width > 0 && height > 0 {
			v.width = width
			v.height = height
		}
	}
}

// WithDashboardState configures the view to return to dashboard with restored state (deprecated)
func WithDashboardState(selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn int) DetailsOption {
	return func(v *RunDetailsView) {
		// These fields no longer exist in the simplified view - ignore
		debug.LogToFilef("DEBUG: WithDashboardState called but dashboard state fields are deprecated\n")
	}
}

// WithParentData sets parent runs and cache data (deprecated)
func WithParentData(parentRuns []models.RunResponse, parentCached bool, parentCachedAt time.Time, parentDetailsCache map[string]*models.RunResponse) DetailsOption {
	return func(v *RunDetailsView) {
		// These fields no longer exist in the simplified view - ignore
		debug.LogToFilef("DEBUG: WithParentData called but parent data fields are deprecated\n")

		// If we have cache data, try to use it
		if v.cache != nil && parentDetailsCache != nil {
			// Store details cache in the shared cache if needed
			for _, runData := range parentDetailsCache {
				if runData != nil {
					// Store in cache for future use
					v.cache.SetRun(*runData)
				}
			}
		}
	}
}

// WithConfig applies all settings from a RunDetailsViewConfig (deprecated)
func WithConfig(config RunDetailsViewConfig) DetailsOption {
	return func(v *RunDetailsView) {
		debug.LogToFilef("DEBUG: WithConfig called - now using minimal config pattern\n")
		// Parent state passing is removed - view loads its own data
	}
}

// Backward compatibility - Old constructor pattern (deprecated)
// This maintains backward compatibility but should be migrated to NewRunDetailsView
func NewRunDetailsViewWithFunctionalOptions(client APIClient, run models.RunResponse, opts ...DetailsOption) *RunDetailsView {
	// Extract run ID
	runID := run.GetIDString()

	// Create with new minimal constructor
	defaultCache := cache.NewSimpleCache()
	_ = defaultCache.LoadFromDisk()

	v := NewRunDetailsView(client, defaultCache, runID)

	// If we already have run data, use it
	if run.Status != "" || run.Title != "" {
		v.run = run
		v.loading = false
		v.updateStatusHistory(string(run.Status), false)
		v.updateContentRefactored()
	}

	// Apply all options (mostly for backward compatibility)
	for _, opt := range opts {
		opt(v)
	}

	return v
}

// NewRunDetailsViewWithConfig creates a new RunDetailsView with the given configuration (backward compatibility)
func NewRunDetailsViewWithConfig(config RunDetailsViewConfig) *RunDetailsView {
	cacheInstance := cache.NewSimpleCache()
	_ = cacheInstance.LoadFromDisk()

	return NewRunDetailsView(config.Client, cacheInstance, config.RunID)
}

// NewRunDetailsViewWithDashboardState creates a new details view (backward compatibility)
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
	runID := run.GetIDString()
	defaultCache := cache.NewSimpleCache()
	_ = defaultCache.LoadFromDisk()

	// Store parent cache data
	for _, runData := range parentDetailsCache {
		if runData != nil {
			defaultCache.SetRun(*runData)
		}
	}

	view := NewRunDetailsView(client, defaultCache, runID)

	// Set dimensions
	if width > 0 && height > 0 {
		view.width = width
		view.height = height
	}

	// Use provided run data if available
	if run.Status != "" || run.Title != "" {
		view.run = run
		view.loading = false
		view.updateStatusHistory(string(run.Status), false)
		view.updateContentRefactored()
	}

	// Dashboard state fields are deprecated - just log
	debug.LogToFilef("DEBUG: Dashboard state parameters ignored in new pattern (repo=%d, run=%d, detail=%d, column=%d)\n",
		selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn)

	return view
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
	runID := run.GetIDString()
	viewCache := cache.NewSimpleCache()
	_ = viewCache.LoadFromDisk()

	// Store parent cache data
	for _, runData := range parentDetailsCache {
		if runData != nil {
			viewCache.SetRun(*runData)
		}
	}

	v := NewRunDetailsView(client, viewCache, runID)

	// Set dimensions
	if width > 0 && height > 0 {
		v.width = width
		v.height = height
	}

	// Use provided run data if available
	if run.Status != "" || run.Title != "" {
		v.run = run
		v.loading = false
		v.updateStatusHistory(string(run.Status), false)
		v.updateContentRefactored()
	}

	// Parent run data is deprecated - ignored

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
	runID := run.GetIDString()

	// Use provided cache or create new one
	viewCache := embeddedCache
	if viewCache == nil {
		viewCache = cache.NewSimpleCache()
		_ = viewCache.LoadFromDisk()
	}

	// Store parent cache data
	for _, runData := range parentDetailsCache {
		if runData != nil {
			viewCache.SetRun(*runData)
		}
	}

	v := NewRunDetailsView(client, viewCache, runID)

	// Use provided run data if available
	if run.Status != "" || run.Title != "" {
		v.run = run
		v.loading = false
		v.updateStatusHistory(string(run.Status), false)
		v.updateContentRefactored()
	}

	// Parent run data is deprecated - ignored

	return v
}

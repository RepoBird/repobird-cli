// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/utils"
)

// DashboardView is the main dashboard controller that manages different layout views
type DashboardView struct {
	client       APIClient
	keys         components.KeyMap
	help         help.Model
	disabledKeys map[string]bool // Keys that are disabled for this view

	// Dashboard state
	currentLayout      models.LayoutType
	selectedRepo       *models.Repository
	selectedRepoIdx    int
	selectedRunIdx     int
	focusedColumn      int            // 0: repositories, 1: runs, 2: details
	selectedDetailLine int            // Selected line in details column
	detailLines        []string       // Lines in details column for selection
	detailLineMemory   map[string]int // Remember selected detail line per run ID

	// All-runs layout using shared component
	allRunsList *components.ScrollableList

	// Dimensions
	width  int
	height int

	// Loading and error state
	loading      bool
	error        error
	initializing bool

	// Real data
	repositories    []models.Repository
	apiRepositories map[int]models.APIRepository // Map repo ID to API repository
	allRuns         []*models.RunResponse
	filteredRuns    []*models.RunResponse
	selectedRunData *models.RunResponse

	// Cache management
	lastDataRefresh time.Time
	refreshInterval time.Duration
	detailsCache    map[string]*models.RunResponse // Cached run details

	// User info
	userInfo *models.UserInfo
	userID   *int // User ID for cache isolation

	// Inline FZF mode for each column
	inlineFZF *components.InlineFZF
	fzfColumn int // Which column is in FZF mode (-1 = none)

	// Loading spinner
	spinner spinner.Model

	// Clipboard feedback
	copiedMessage     string
	copiedMessageTime time.Time
	clipboardManager  components.ClipboardManager

	// Unified status line component
	statusLine *components.StatusLine

	// Store original untruncated detail lines for copying
	detailLinesOriginal []string

	// URL selection for repositories
	showURLSelectionPrompt bool                  // Show URL selection prompt in status line
	pendingRepoForURL      *models.Repository    // Repository pending URL selection
	pendingAPIRepoForURL   *models.APIRepository // API repository data for URL generation

	// Vim keybinding state for 'gg' command
	lastGPressTime time.Time // Time when 'g' was last pressed
	waitingForG    bool      // Whether we're waiting for second 'g' in 'gg' command

	// New scrollable help view
	helpView *components.HelpView

	// Viewports for scrolling
	repoViewport    viewport.Model
	runsViewport    viewport.Model
	detailsViewport viewport.Model

	// Embedded cache (no globals!)
	cache *cache.SimpleCache
}

// Message types are defined in dashboard_messages.go

// NewDashboardViewWithState creates a new dashboard view with restored state
func NewDashboardViewWithState(client APIClient, cache *cache.SimpleCache, selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn int) *DashboardView {
	dashboard := NewDashboardView(client, cache)
	
	// Get the saved window size if available
	width, height := 80, 24 // defaults
	if stateData := cache.GetNavigationContext("dashboardState"); stateData != nil {
		if state, ok := stateData.(map[string]interface{}); ok {
			if w, ok := state["width"].(int); ok && w > 0 {
				width = w
			}
			if h, ok := state["height"].(int); ok && h > 0 {
				height = h
			}
		}
	}
	
	// Set the state that will be restored after data loads
	debug.LogToFilef("🔧 DASHBOARD: Creating dashboard with restored state - repo=%d, run=%d, detail=%d, column=%d, size=%dx%d 🔧\n",
		selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn, width, height)
	dashboard.selectedRepoIdx = selectedRepoIdx
	dashboard.selectedRunIdx = selectedRunIdx
	dashboard.selectedDetailLine = selectedDetailLine
	dashboard.focusedColumn = focusedColumn
	dashboard.width = width
	dashboard.height = height
	
	// Keep loading state true until data is loaded
	dashboard.loading = true
	dashboard.initializing = true
	
	// Update viewport sizes immediately to prevent panic
	dashboard.updateViewportSizes()
	
	// CRITICAL: Set initial content BEFORE any other operations to prevent panic
	// Must have actual newline-separated content to avoid capacity 1 issue
	safeContent := strings.Repeat("Loading...\n", 50) // Create 50 lines of safe content
	dashboard.repoViewport.SetContent(safeContent)
	dashboard.runsViewport.SetContent(safeContent)
	dashboard.detailsViewport.SetContent(safeContent)
	
	// NOW reset viewport scroll positions after content is set
	dashboard.repoViewport.GotoTop()
	dashboard.runsViewport.GotoTop()  
	dashboard.detailsViewport.GotoTop()
	
	debug.LogToFilef("✅ DASHBOARD: State applied with viewports reset to top (preventing panic) ✅\n")
	return dashboard
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(client APIClient, cache *cache.SimpleCache) *DashboardView {
	// Initialize spinner
	s := spinner.New()
	// Create a custom spinner with very obvious animation frames
	s.Spinner = spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    100 * time.Millisecond, // 10 FPS for smooth animation
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	dashboard := &DashboardView{
		client: client,
		cache:  cache, // Set the cache
		keys:   components.DefaultKeyMap,
		help:   help.New(),
		disabledKeys: map[string]bool{
			"esc": true, // Disable escape key on dashboard (used for overlays/modals)
			// 'h' is NOT disabled - used for column navigation (move left)
			// 'q' is not disabled - HandleKey handles it for quit
		},
		currentLayout:    models.LayoutTripleColumn,
		loading:          true,
		initializing:     true,
		refreshInterval:  30 * time.Second,
		apiRepositories:  make(map[int]models.APIRepository),
		detailLineMemory: make(map[string]int), // Initialize detail line memory
		fzfColumn:        -1,                   // No FZF mode initially
		spinner:          s,
		statusLine:       components.NewStatusLine(),
		helpView:         components.NewHelpView(),
		clipboardManager: components.NewClipboardManager(),
		repoViewport:     viewport.New(0, 0), // Will be sized in Update
		runsViewport:     viewport.New(0, 0),
		detailsViewport:  viewport.New(0, 0),
	}

	// Note: cache is already set from parameter, no need to create a new one

	// Initialize shared scrollable list component for all-runs layout
	dashboard.allRunsList = components.NewScrollableList(
		components.WithColumns(4), // ID, Repository, Status, Created
		components.WithValueNavigation(true),
		components.WithKeymaps(components.DefaultKeyMap),
	)

	// Initialize viewports with safe content to prevent panic
	safeContent := strings.Repeat("Loading...\n", 50)
	dashboard.repoViewport.SetContent(safeContent)
	dashboard.runsViewport.SetContent(safeContent)
	dashboard.detailsViewport.SetContent(safeContent)

	return dashboard
}

// IsKeyDisabled implements the CoreViewKeymap interface
func (d *DashboardView) IsKeyDisabled(keyString string) bool {
	disabled := d.disabledKeys[keyString]
	return disabled
}

// HandleKey implements the CoreViewKeymap interface
func (d *DashboardView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	switch keyMsg.String() {
	case "b":
		// Dashboard handles 'b' specially for bulk view navigation (overrides back navigation)
		// Save dashboard state before navigating
		debug.LogToFilef("💾 DASHBOARD: Saving state before BULK navigation - repo=%d, run=%d, detail=%d, column=%d 💾\n",
			d.selectedRepoIdx, d.selectedRunIdx, d.selectedDetailLine, d.focusedColumn)
		d.cache.SetNavigationContext("dashboardState", map[string]interface{}{
			"selectedRepoIdx":    d.selectedRepoIdx,
			"selectedRunIdx":     d.selectedRunIdx,
			"selectedDetailLine": d.selectedDetailLine,
			"focusedColumn":      d.focusedColumn,
		})
		// Navigate to bulk view
		return true, d, func() tea.Msg {
			return messages.NavigateToBulkMsg{}
		}
	case "h", "H":
		// Handle 'h' and 'H' for column navigation (move left)
		// Override centralized system behavior for dashboard-specific column navigation
		if d.focusedColumn > 0 {
			// Save detail line selection if leaving details column
			if d.focusedColumn == 2 && d.selectedRunData != nil {
				runID := d.selectedRunData.GetIDString()
				if runID != "" {
					d.detailLineMemory[runID] = d.selectedDetailLine
				}
			}
			// Move to the left column
			d.focusedColumn--
			return true, d, nil
		}
		// If already in leftmost column, don't trigger back navigation
		return true, d, nil
	case "q":
		// On dashboard, 'q' quits the app (with confirmation would be nice, but simple for now)
		return true, d, tea.Quit
	}
	// Let the centralized system handle everything else
	return false, d, nil
}

// Init implements the tea.Model interface
func (d *DashboardView) Init() tea.Cmd {
	// Initialize clipboard (will detect CGO availability)
	err := utils.InitClipboard()
	if err != nil {
		// Log error but don't fail - clipboard may not be available in some environments
		debug.LogToFilef("DEBUG: Failed to initialize clipboard: %v\n", err)
	}

	return tea.Batch(
		d.loadDashboardData(),
		d.loadUserInfo(),
		d.syncFileHashes(),
		// Initialize shared all-runs list component
		d.allRunsList.Init(),
		d.spinner.Tick,
	)
}

// syncFileHashesMsg is a message indicating file hash sync completed
// syncFileHashesMsg is defined in dashboard_messages.go

// syncFileHashes syncs file hashes from the API on startup

// View implements the tea.Model interface
func (d *DashboardView) View() string {
	if d.width <= 0 || d.height <= 0 {
		// Return a styled loading message instead of plain text
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render("⟳ Initializing dashboard...")
	}

	var content string

	// Always show title - left aligned
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		PaddingLeft(1)

	title := titleStyle.Render("Repobird.ai CLI")

	if d.error != nil {
		// Use global WindowLayout system for error state
		layout := components.NewWindowLayout(d.width, d.height)
		debug.LogToFilef("🔴 DASHBOARD ERROR: Created layout for %dx%d terminal 🔴\n", d.width, d.height)
		if !layout.IsValidDimensions() {
			return layout.GetMinimalView("Dashboard Error")
		}

		boxStyle := layout.CreateStandardBox()
		contentStyle := layout.CreateContentStyle()

		// Create error content
		errorContent := fmt.Sprintf("Error loading dashboard data: %s\n\nPress 'r' to retry, 'q' to quit", d.error.Error())

		// Get viewport dimensions for proper centering (no room reduction needed - status line is outside)
		viewportWidth, viewportHeight := layout.GetViewportDimensions()
		centeredContent := contentStyle.
			Width(viewportWidth).
			Height(viewportHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(errorContent)

		// Create status line separately (outside the box)
		statusLine := d.renderStatusLine("ERROR")

		// Follow StatusView pattern: render box content separately, then join with status line outside
		// No empty line - the box height already accounts for proper spacing
		boxedContent := boxStyle.Render(centeredContent)
		boxLines := strings.Count(boxedContent, "\n") + 1
		statusLines := strings.Count(statusLine, "\n") + 1
		result := lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
		totalLines := strings.Count(result, "\n") + 1
		debug.LogToFilef("🔴 DASHBOARD ERROR: Box=%d lines, Status=%d lines, Total=%d lines (should be %d) 🔴\n",
			boxLines, statusLines, totalLines, d.height)
		return result
	}

	// Show cached content while loading new data
	if d.loading && len(d.repositories) > 0 {
		// Show cached content with loading indicator
		switch d.currentLayout {
		case models.LayoutTripleColumn:
			content = d.renderTripleColumnLayout()
		case models.LayoutAllRuns:
			content = d.renderAllRunsLayout()
		case models.LayoutRepositoriesOnly:
			content = d.renderRepositoriesLayout()
		default:
			content = d.renderTripleColumnLayout()
		}
		// Layout functions already include status line - don't add another one
		return lipgloss.JoinVertical(lipgloss.Left, title, content)
	}

	if d.loading || d.initializing {
		// ASCII logo for RepoBird AI
		logo := `
██████╗ ███████╗██████╗  ██████╗ ██████╗ ██╗██████╗ ██████╗      █████╗ ██╗
██╔══██╗██╔════╝██╔══██╗██╔═══██╗██╔══██╗██║██╔══██╗██╔══██╗    ██╔══██╗██║
██████╔╝█████╗  ██████╔╝██║   ██║██████╔╝██║██████╔╝██║  ██║    ███████║██║
██╔══██╗██╔══╝  ██╔═══╝ ██║   ██║██╔══██╗██║██╔══██╗██║  ██║    ██╔══██║██║
██║  ██║███████╗██║     ╚██████╔╝██████╔╝██║██║  ██║██████╔╝    ██║  ██║██║
╚═╝  ╚═╝╚══════╝╚═╝      ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═╝╚═════╝     ╚═╝  ╚═╝╚═╝`

		// Use the animated spinner + loading text
		loadingText := d.spinner.View() + " Loading dashboard data..."

		// Calculate available height for content (total - title - status line)
		titleHeight := lipgloss.Height(title)
		statusLineHeight := 1 // Status line is always 1 line
		availableHeight := d.height - titleHeight - statusLineHeight

		// Style for the logo
		logoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")). // Blue color for logo
			Bold(true)

		// Style for loading text
		loadingTextStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")). // Bright cyan color
			Bold(true).
			MarginTop(2) // Add space between logo and loading text

		// Combine logo and loading text
		combinedContent := lipgloss.JoinVertical(
			lipgloss.Center,
			logoStyle.Render(logo),
			loadingTextStyle.Render(loadingText),
		)

		// Center everything vertically and horizontally in the available space
		centerStyle := lipgloss.NewStyle().
			Width(d.width).
			Height(availableHeight).
			Align(lipgloss.Center, lipgloss.Center)
		content = centerStyle.Render(combinedContent)

		// Always show status line even during loading
		statusline := d.renderStatusLine("DASH")
		return lipgloss.JoinVertical(lipgloss.Left, title, content, statusline)
	}

	// Render based on current layout
	switch d.currentLayout {
	case models.LayoutTripleColumn:
		content = d.renderTripleColumnLayout()
	case models.LayoutAllRuns:
		content = d.renderAllRunsLayout()
	case models.LayoutRepositoriesOnly:
		content = d.renderRepositoriesLayout()
	default:
		content = d.renderTripleColumnLayout()
	}

	finalView := lipgloss.JoinVertical(lipgloss.Left, title, content)

	// Note: Inline FZF is handled within column rendering, no overlay needed

	// Overlay help if requested

	// Status line is already included in the layout-specific rendering functions
	return finalView
}

// renderTripleColumnLayout renders the Miller Columns layout with real data

// copyToClipboard copies the given text to clipboard

// renderRepositoriesLayout renders the repositories-only layout

// renderRepositoriesColumn renders the left column with real repositories

// renderRunsColumn renders the center column with runs for selected repository

// renderDetailsColumn renders the right column with run details

// renderStatusInfo renders the status/user info overlay

// renderStatusLine renders the universal status line
// Update processes messages and updates the dashboard view
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle quit keys first
	if cmd := d.handleQuitKeys(msg); cmd != nil {
		return d, cmd
	}

	// Handle different message types
	switch msg := msg.(type) {
	case nil:
		return d.handleNilMessage()
	case spinner.TickMsg:
		cmds = append(cmds, d.handleSpinnerTick(msg))
	case tea.WindowSizeMsg:
		cmds = append(cmds, d.handleWindowSize(msg)...)
	case dashboardDataLoadedMsg:
		return d.handleDataLoaded(msg)
	case dashboardRepositorySelectedMsg:
		return d.handleRepositorySelected(msg)
	case dashboardUserInfoLoadedMsg:
		return d.handleUserInfoLoaded(msg)
	case syncFileHashesMsg:
		return d.handleSyncFileHashes(msg)
	case components.ClipboardBlinkMsg:
		return d.handleClipboardBlink(msg)
	case messageClearMsg:
		return d.handleMessageClear()
	case gKeyTimeoutMsg:
		return d.handleGKeyTimeout()
	case clearStatusMsg:
		return d.handleClearStatus()
	case components.FZFSelectedMsg:
		return d.handleFZFSelected(msg)
	case tea.KeyMsg:
		return d.handleKeyMessage(msg)
	default:
		// Pass unhandled messages through viewports if in respective layouts
		return d.handleDefaultMessage(msg)
	}

	return d, tea.Batch(cmds...)
}

// handleQuitKeys handles quit key combinations
func (d *DashboardView) handleQuitKeys(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	// Handle force quit regardless of state
	if keyMsg.String() == "Q" || keyMsg.Type == tea.KeyCtrlC {
		debug.LogToFilef("  FORCE QUIT requested\n")
		_ = d.cache.SaveToDisk()
		return tea.Quit
	}

	// Handle normal quit when not in special modes
	if keyMsg.String() == "q" && !d.showURLSelectionPrompt && d.inlineFZF == nil {
		debug.LogToFilef("  Normal quit requested\n")
		_ = d.cache.SaveToDisk()
		return tea.Quit
	}

	return nil
}

// handleNilMessage handles nil messages
func (d *DashboardView) handleNilMessage() (tea.Model, tea.Cmd) {
	debug.LogToFilef("  WARNING: Received nil message, ignoring\n")
	return d, nil
}

// handleSpinnerTick handles spinner tick messages
func (d *DashboardView) handleSpinnerTick(msg spinner.TickMsg) tea.Cmd {
	if d.loading || d.initializing {
		oldView := d.spinner.View()
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		newView := d.spinner.View()
		d.statusLine.UpdateSpinnerWithTick(msg)
		debug.LogToFilef("🔄 SPINNER: Tick processed - loading=%t initializing=%t, before='%s', after='%s', changed=%t 🔄\n",
			d.loading, d.initializing, oldView, newView, oldView != newView)
		return cmd
	}
	debug.LogToFilef("🔄 SPINNER: Ignoring tick - loading=%t initializing=%t 🔄\n", d.loading, d.initializing)
	return nil
}

// handleWindowSize handles window resize messages
func (d *DashboardView) handleWindowSize(msg tea.WindowSizeMsg) []tea.Cmd {
	var cmds []tea.Cmd
	
	d.width = msg.Width
	d.height = msg.Height

	// Check if dashboard needs refresh after navigation
	if cmd := d.checkAndTriggerRefresh(); cmd != nil {
		cmds = append(cmds, cmd, d.spinner.Tick)
	}

	// Update component sizes
	d.updateComponentSizes(msg)
	
	return cmds
}

// checkAndTriggerRefresh checks if refresh is needed and triggers it
func (d *DashboardView) checkAndTriggerRefresh() tea.Cmd {
	needsRefresh := d.cache.GetNavigationContext("dashboard_needs_refresh")
	if needsRefresh != nil {
		if refresh, ok := needsRefresh.(bool); ok && refresh {
			debug.LogToFilef("🔄 DASHBOARD: Detected refresh flag - triggering data reload 🔄\n")
			d.cache.SetNavigationContext("dashboard_needs_refresh", nil)
			d.loading = true
			return d.loadDashboardData()
		}
	}
	return nil
}

// updateComponentSizes updates sizes of various components
func (d *DashboardView) updateComponentSizes(msg tea.WindowSizeMsg) {
	if d.helpView != nil {
		d.helpView.SetSize(msg.Width, msg.Height)
	}
	if d.allRunsList != nil {
		d.allRunsList.Update(msg)
	}
	d.updateViewportSizes()
}

// handleDataLoaded handles dashboard data loaded messages
func (d *DashboardView) handleDataLoaded(msg dashboardDataLoadedMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("\n[DASHBOARD DATA LOADED MSG RECEIVED]\n")
	debug.LogToFilef("🔄 REFRESH: Data loaded - setting loading state to false 🔄\n")
	
	d.loading = false
	d.initializing = false
	d.statusLine.SetLoading(false)
	
	if msg.error != nil {
		return d.handleDataLoadError(msg)
	}
	
	return d.handleDataLoadSuccess(msg)
}

// handleDataLoadError handles errors when loading data
func (d *DashboardView) handleDataLoadError(msg dashboardDataLoadedMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("  ERROR: %v, retryExhausted: %v\n", msg.error, msg.retryExhausted)
	
	if msg.retryExhausted {
		debug.LogToFilef("  ❌ All retries exhausted, navigating to error view\n")
		return d, func() tea.Msg {
			return messages.NavigateToErrorMsg{
				Error:       msg.error,
				Message:     "Failed to load dashboard after 3 attempts",
				Recoverable: true,
			}
		}
	}
	
	d.error = msg.error
	return d, nil
}

// handleDataLoadSuccess handles successful data load
func (d *DashboardView) handleDataLoadSuccess(msg dashboardDataLoadedMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("  Repositories loaded: %d\n", len(msg.repositories))
	debug.LogToFilef("  Total runs loaded: %d\n", len(msg.allRuns))
	debug.LogToFilef("  Details cache loaded: %d\n", len(msg.detailsCache))

	// Debug: Show repository names
	debug.LogToFilef("  Repository list:\n")
	for i, repo := range msg.repositories {
		debug.LogToFilef("    [%d] '%s'\n", i, repo.Name)
	}

	d.repositories = msg.repositories
	d.allRuns = msg.allRuns
	d.detailsCache = msg.detailsCache
	d.lastDataRefresh = time.Now()

	d.updateViewportSizes()

	// Select first repository by default, or restore saved state
	if len(d.repositories) > 0 {
		if d.selectedRepoIdx >= 0 && d.selectedRepoIdx < len(d.repositories) {
			d.selectedRepo = &d.repositories[d.selectedRepoIdx]
		} else {
			d.selectedRepo = &d.repositories[0]
			d.selectedRepoIdx = 0
		}
		return d, d.selectRepository(d.selectedRepo)
	}
	
	return d, nil
}

// handleRepositorySelected handles repository selection messages
func (d *DashboardView) handleRepositorySelected(msg dashboardRepositorySelectedMsg) (tea.Model, tea.Cmd) {
	d.selectedRepo = msg.repository
	d.filteredRuns = msg.runs
	d.updateViewportContent()

	// Select first run by default, or restore saved state
	if len(d.filteredRuns) > 0 {
		if d.selectedRunIdx >= 0 && d.selectedRunIdx < len(d.filteredRuns) {
			d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
		} else {
			d.selectedRunData = d.filteredRuns[0]
			d.selectedRunIdx = 0
		}
		d.updateDetailLines()
		d.restoreOrInitDetailSelection()
	} else {
		d.selectedRunData = nil
		d.detailLines = nil
		d.detailLinesOriginal = nil
	}

	// Update the "All" count
	// d.updateAllRunsCount() // TODO: implement if needed
	
	return d, nil
}

// handleUserInfoLoaded handles user info loaded messages
func (d *DashboardView) handleUserInfoLoaded(msg dashboardUserInfoLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.error == nil && msg.userInfo != nil {
		d.userInfo = msg.userInfo
		// d.statusLine.SetEmail(msg.userInfo.Email) // TODO: check StatusLine API
	}
	return d, nil
}

// handleSyncFileHashes handles file hash sync messages
func (d *DashboardView) handleSyncFileHashes(msg syncFileHashesMsg) (tea.Model, tea.Cmd) {
	// d.fileHashes = msg.hashes // TODO: check field types
	return d, nil
}

// handleClipboardBlink handles clipboard blink messages
func (d *DashboardView) handleClipboardBlink(msg components.ClipboardBlinkMsg) (tea.Model, tea.Cmd) {
	// d.statusLine.Update(msg) // TODO: check StatusLine API
	return d, nil
}

// handleMessageClear handles message clear messages
func (d *DashboardView) handleMessageClear() (tea.Model, tea.Cmd) {
	d.copiedMessage = ""
	return d, nil
}

// handleGKeyTimeout handles g key timeout messages
func (d *DashboardView) handleGKeyTimeout() (tea.Model, tea.Cmd) {
	d.waitingForG = false
	return d, nil
}

// handleClearStatus handles clear status messages
func (d *DashboardView) handleClearStatus() (tea.Model, tea.Cmd) {
	// d.statusLine.ClearMessage() // TODO: check StatusLine API
	return d, nil
}

// handleFZFSelected handles FZF selection messages
func (d *DashboardView) handleFZFSelected(msg components.FZFSelectedMsg) (tea.Model, tea.Cmd) {
	if d.inlineFZF != nil {
		d.inlineFZF.Deactivate()
		d.inlineFZF = nil
	}
	d.fzfColumn = -1
	if !msg.Result.Canceled && msg.Result.Selected != "" {
		return d.processFZFSelection(msg)
	}
	return d, nil
}

// processFZFSelection processes an FZF selection based on the current column
func (d *DashboardView) processFZFSelection(msg components.FZFSelectedMsg) (tea.Model, tea.Cmd) {
	switch d.focusedColumn {
	case 0: // Repository column
		for i, repo := range d.repositories {
			if repo.Name == msg.Result.Selected {
				d.selectedRepoIdx = i
				d.selectedRepo = &d.repositories[i]
				return d, d.selectRepository(d.selectedRepo)
			}
		}
	case 1: // Runs column
		for i, run := range d.filteredRuns {
			if fmt.Sprintf("%s - %s", run.GetIDString(), run.Title) == msg.Result.Selected {
				d.selectedRunIdx = i
				d.selectedRunData = d.filteredRuns[i]
				d.updateDetailLines()
				break
			}
		}
	case 2: // Details column
		for i, line := range d.detailLines {
			if line == msg.Result.Selected {
				d.selectedDetailLine = i
				break
			}
		}
	}
	return d, nil
}

// handleKeyMessage handles key messages - main dispatcher
func (d *DashboardView) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Special mode handlers
	if d.showURLSelectionPrompt {
		return d.handleURLSelectionKeys(msg)
	}

	// Layout-specific key handling
	switch d.currentLayout {
	case models.LayoutTripleColumn:
		return d.handleTripleColumnKeys(msg)
	case models.LayoutAllRuns:
		return d.handleAllRunsKeys(msg)
	case models.LayoutRepositoriesOnly:
		return d.handleRepositoriesOnlyKeys(msg)
	default:
		return d, nil
	}
}

// handleURLSelectionKeys handles keys when URL selection prompt is active
func (d *DashboardView) handleURLSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		d.showURLSelectionPrompt = false
		d.pendingRepoForURL = nil
		d.pendingAPIRepoForURL = nil
		// d.statusLine.ClearMessage() // TODO: check StatusLine API
		return d, nil
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "o":
		return d.openRepoURL(false)
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "g":
		return d.openRepoURL(true)
		
	default:
		// Ignore other keys during URL selection
		return d, nil
	}
}

// openRepoURL opens a repository URL (GitHub or RepoBird)
func (d *DashboardView) openRepoURL(useGitHub bool) (tea.Model, tea.Cmd) {
	if d.pendingAPIRepoForURL == nil {
		return d, nil
	}

	var urlText string
	if useGitHub {
		// urlText = d.pendingAPIRepoForURL.GitHubURL // TODO: check field name
		urlText = "" // placeholder
	} else {
		urlText = fmt.Sprintf("https://repobird.ai/repo/%d", d.pendingAPIRepoForURL.ID)
	}

	d.showURLSelectionPrompt = false
	d.pendingRepoForURL = nil
	d.pendingAPIRepoForURL = nil

	if err := utils.OpenURLWithTimeout(urlText); err == nil {
		d.statusLine.SetTemporaryMessageWithType("🌐 Opened URL in browser", components.MessageSuccess, 1*time.Second)
	} else {
		d.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
	}
	
	return d, d.startMessageClearTimer(1 * time.Second)
}

// handleTripleColumnKeys handles keys for triple column layout
func (d *DashboardView) handleTripleColumnKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If inline FZF is active, handle input there first
	if d.inlineFZF != nil && d.inlineFZF.IsActive() {
		newFzf, cmd := d.inlineFZF.Update(msg)
		d.inlineFZF = newFzf
		
		// Check if FZF was deactivated (ESC pressed or Enter pressed)
		if !d.inlineFZF.IsActive() {
			// If Enter was pressed, handle selection
			if msg.String() == "enter" {
				// Use GetLastSelection since FZF has already been deactivated
				selected, originalIdx := d.inlineFZF.GetLastSelection()
				if selected != "" && originalIdx >= 0 {
					// Process the selection based on column using the original index
					switch d.fzfColumn {
					case 0: // Repository column
						// Use the original index directly
						if originalIdx < len(d.repositories) {
							d.selectedRepoIdx = originalIdx
							d.selectedRepo = &d.repositories[originalIdx]
							// Move focus to runs column BEFORE selecting repository
							d.focusedColumn = 1
							// Now select the repository which will load its runs
							cmd = d.selectRepository(d.selectedRepo)
						}
					case 1: // Runs column
						// For runs, we need to find in the filtered runs list
						// The FZF items were built from filteredRuns, so originalIdx maps to filteredRuns
						if originalIdx < len(d.filteredRuns) {
							d.selectedRunIdx = originalIdx
							d.selectedRunData = d.filteredRuns[originalIdx]
							d.updateDetailLines()
							d.restoreOrInitDetailSelection()
							// Move focus to details column
							d.focusedColumn = 2
						}
					case 2: // Details column
						// Details column uses the original index directly
						if originalIdx < len(d.detailLines) {
							d.selectedDetailLine = originalIdx
						}
						// Details column is already the last, no need to move focus
					}
				}
			}
			// Clean up FZF mode
			d.fzfColumn = -1
			d.inlineFZF = nil
			// Update viewports to restore normal view
			d.updateViewportContent()
			return d, cmd
		}
		// FZF is still active, update viewports to show filtered content
		d.updateViewportContent()
		return d, cmd
	}

	// Common keys first
	if cmd := d.handleCommonKeys(msg); cmd != nil {
		return d, cmd
	}

	// Navigation keys
	if cmd := d.handleMillerColumnsNavigation(msg); cmd != nil {
		return d, cmd
	}

	return d, nil
}

// handleAllRunsKeys handles keys for all runs layout
func (d *DashboardView) handleAllRunsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Common keys first
	if cmd := d.handleCommonKeys(msg); cmd != nil {
		return d, cmd
	}

	// Pass to list component
	if d.allRunsList != nil {
		d.allRunsList.Update(msg)
	}

	return d, nil
}

// handleRepositoriesOnlyKeys handles keys for repositories only layout
func (d *DashboardView) handleRepositoriesOnlyKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Common keys first
	if cmd := d.handleCommonKeys(msg); cmd != nil {
		return d, cmd
	}

	// Repository navigation
	return d.handleRepositoryNavigation(msg)
}

// handleCommonKeys handles keys common to all layouts
func (d *DashboardView) handleCommonKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, d.keys.LayoutSwitch):
		d.cycleLayout()
		return nil
		
	case key.Matches(msg, d.keys.LayoutTriple):
		d.currentLayout = models.LayoutTripleColumn
		return nil
		
	case key.Matches(msg, d.keys.LayoutAllRuns):
		d.currentLayout = models.LayoutAllRuns
		return nil
		
	case key.Matches(msg, d.keys.LayoutRepos):
		d.currentLayout = models.LayoutRepositoriesOnly
		return nil
		
	case key.Matches(msg, d.keys.Help):
		d.helpView = components.NewHelpView() // TODO: check parameters
		return nil
		
	case key.Matches(msg, d.keys.Quit):
		_ = d.cache.SaveToDisk()
		return tea.Quit
		
	case key.Matches(msg, d.keys.Refresh):
		return d.triggerRefresh()
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "f":
		return d.startFZFMode()
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
		return d.navigateToDetails()
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "s":
		return d.handleStatusCommand()
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "n":
		return d.navigateToCreateForm()
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "G":
		return d.jumpToBottom()
		
	case msg.Type == tea.KeyRunes && string(msg.Runes) == "g":
		return d.handleGKey()
	}
	
	return nil
}

// triggerRefresh triggers a data refresh
func (d *DashboardView) triggerRefresh() tea.Cmd {
	debug.LogToFilef("🔄 DASHBOARD: Manual refresh triggered 🔄\n")
	d.cache.Clear() // Clear entire cache for manual refresh
	d.loading = true
	return tea.Batch(d.loadDashboardData(), d.spinner.Tick)
}

// startFZFMode starts inline FZF mode for the current column
func (d *DashboardView) startFZFMode() tea.Cmd {
	var items []string
	var width int
	
	// Calculate column width based on focused column
	totalWidth := d.width - 6
	switch d.focusedColumn {
	case 0:
		width = totalWidth / 3
		for _, repo := range d.repositories {
			items = append(items, repo.Name)
		}
	case 1:
		width = totalWidth / 3
		for _, run := range d.filteredRuns {
			items = append(items, fmt.Sprintf("%s - %s", run.GetIDString(), run.Title))
		}
	case 2:
		width = totalWidth - (totalWidth/3)*2
		items = d.detailLines
	}
	
	if len(items) > 0 {
		d.fzfColumn = d.focusedColumn
		d.inlineFZF = components.NewInlineFZF(items, "Type to filter...", width-4)
		d.inlineFZF.Activate()
	}
	return nil
}

// navigateToDetails navigates to run details view
func (d *DashboardView) navigateToDetails() tea.Cmd {
	if d.selectedRunData != nil {
		// Save dashboard state before navigating
		debug.LogToFilef("💾 DASHBOARD: Saving state before navigation - repo=%d, run=%d, detail=%d, column=%d, size=%dx%d 💾\n",
			d.selectedRepoIdx, d.selectedRunIdx, d.selectedDetailLine, d.focusedColumn, d.width, d.height)
		d.cache.SetNavigationContext("dashboardState", map[string]interface{}{
			"selectedRepoIdx":    d.selectedRepoIdx,
			"selectedRunIdx":     d.selectedRunIdx,
			"selectedDetailLine": d.selectedDetailLine,
			"focusedColumn":      d.focusedColumn,
			"width":              d.width,
			"height":             d.height,
		})
		
		return func() tea.Msg {
			return messages.NavigateToDetailsMsg{RunData: d.selectedRunData}
		}
	}
	return nil
}

// handleStatusCommand handles the status command
func (d *DashboardView) handleStatusCommand() tea.Cmd {
	// Save dashboard state before navigating
	debug.LogToFilef("💾 DASHBOARD: Saving state before STATUS navigation - repo=%d, run=%d, detail=%d, column=%d 💾\n",
		d.selectedRepoIdx, d.selectedRunIdx, d.selectedDetailLine, d.focusedColumn)
	d.cache.SetNavigationContext("dashboardState", map[string]interface{}{
		"selectedRepoIdx":    d.selectedRepoIdx,
		"selectedRunIdx":     d.selectedRunIdx,
		"selectedDetailLine": d.selectedDetailLine,
		"focusedColumn":      d.focusedColumn,
	})
	
	return func() tea.Msg {
		return messages.NavigateToStatusMsg{}
	}
}

// navigateToCreateForm navigates to create form
func (d *DashboardView) navigateToCreateForm() tea.Cmd {
	if d.selectedRepo != nil {
		repositoryName := d.selectedRepo.Name
		
		// Save dashboard state before navigating
		debug.LogToFilef("💾 DASHBOARD: Saving state before CREATE navigation - repo=%d, run=%d, detail=%d, column=%d 💾\n",
			d.selectedRepoIdx, d.selectedRunIdx, d.selectedDetailLine, d.focusedColumn)
		d.cache.SetNavigationContext("dashboardState", map[string]interface{}{
			"selectedRepoIdx":    d.selectedRepoIdx,
			"selectedRunIdx":     d.selectedRunIdx,
			"selectedDetailLine": d.selectedDetailLine,
			"focusedColumn":      d.focusedColumn,
		})
		
		return func() tea.Msg {
			return messages.NavigateToCreateMsg{
				SelectedRepository: repositoryName,
			}
		}
	}
	return nil
}

// jumpToBottom jumps to the bottom of the current column
func (d *DashboardView) jumpToBottom() tea.Cmd {
	switch d.focusedColumn {
	case 0:
		if len(d.repositories) > 0 {
			d.selectedRepoIdx = len(d.repositories) - 1
			d.selectedRepo = &d.repositories[d.selectedRepoIdx]
			return d.selectRepository(d.selectedRepo)
		}
	case 1:
		if len(d.filteredRuns) > 0 {
			d.selectedRunIdx = len(d.filteredRuns) - 1
			d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
			d.updateDetailLines()
		}
	case 2:
		if len(d.detailLines) > 0 {
			d.selectedDetailLine = len(d.detailLines) - 1
		}
	}
	return nil
}

// handleGKey handles the 'g' key (for gg navigation)
func (d *DashboardView) handleGKey() tea.Cmd {
	if d.waitingForG {
		// Second 'g' - go to top
		d.waitingForG = false
		switch d.focusedColumn {
		case 0:
			if len(d.repositories) > 0 {
				d.selectedRepoIdx = 0
				d.selectedRepo = &d.repositories[0]
				return d.selectRepository(d.selectedRepo)
			}
		case 1:
			if len(d.filteredRuns) > 0 {
				d.selectedRunIdx = 0
				d.selectedRunData = d.filteredRuns[0]
				d.updateDetailLines()
			}
		case 2:
			d.selectedDetailLine = 0
		}
	} else {
		// First 'g' - wait for second
		d.waitingForG = true
		// return d.startGKeyTimer() // TODO: implement timer
		return nil
	}
	return nil
}

// handleRepositoryNavigation handles repository navigation in repos-only layout
func (d *DashboardView) handleRepositoryNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Implementation would go here - similar to repository column navigation
	// from handleMillerColumnsNavigation
	return d, nil
}

// handleDefaultMessage handles unrecognized messages
func (d *DashboardView) handleDefaultMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Pass through to viewports if needed
	switch d.currentLayout {
	case models.LayoutTripleColumn:
		// Could update viewports here if needed
	case models.LayoutAllRuns:
		if d.allRunsList != nil {
			d.allRunsList.Update(msg)
		}
	case models.LayoutRepositoriesOnly:
		// Could update repository viewport here
	}
	return d, nil
}

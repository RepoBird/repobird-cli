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
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

type RunDetailsView struct {
	client        APIClient
	run           models.RunResponse
	keys          components.KeyMap
	help          help.Model
	viewport      viewport.Model
	width         int
	height        int
	loading       bool
	error         error
	spinner       spinner.Model
	pollTicker    *time.Ticker
	pollStop      chan bool
	showLogs      bool
	logs          string
	statusHistory []string
	pollingStatus bool // Track if currently fetching status
	// Cache from parent list view
	parentRuns         []models.RunResponse
	parentCached       bool
	parentCachedAt     time.Time
	parentDetailsCache map[string]*models.RunResponse
	// Cache retry mechanism
	cacheRetryCount int
	maxCacheRetries int
	// Unified status line component
	statusLine *components.StatusLine
	// Clipboard feedback (still need blink timing)
	yankBlink     bool
	yankBlinkTime time.Time
	// Store full content for clipboard operations
	fullContent string
	// Row navigation
	selectedRow    int      // Currently selected row/field
	fieldLines     []string // Lines that can be selected (field values)
	fieldValues    []string // Actual field values for copying
	fieldIndices   []int    // Line indices of selectable fields in the viewport
	fieldRanges    [][2]int // Start and end line indices for each field (for multi-line fields)
	navigationMode bool     // Whether we're in navigation mode
	// Dashboard state for restoration
	dashboardSelectedRepoIdx    int
	dashboardSelectedRunIdx     int
	dashboardSelectedDetailLine int
	dashboardFocusedColumn      int
}

func NewRunDetailsView(client APIClient, run models.RunResponse) *RunDetailsView {
	// Get the current global cache
	runs, cached, cachedAt, detailsCache, _ := cache.GetCachedList()
	return NewRunDetailsViewWithCache(client, run, runs, cached, cachedAt, detailsCache)
}

// RunDetailsViewConfig holds configuration for creating a new RunDetailsView
type RunDetailsViewConfig struct {
	Client             APIClient
	Run                models.RunResponse
	ParentRuns         []models.RunResponse
	ParentCached       bool
	ParentCachedAt     time.Time
	ParentDetailsCache map[string]*models.RunResponse
	// Dashboard state for restoration
	DashboardSelectedRepoIdx    int
	DashboardSelectedRunIdx     int
	DashboardSelectedDetailLine int
	DashboardFocusedColumn      int
}

// NewRunDetailsViewWithConfig creates a new RunDetailsView with the given configuration
func NewRunDetailsViewWithConfig(config RunDetailsViewConfig) *RunDetailsView {
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
	v := NewRunDetailsViewWithCache(client, run, parentRuns, parentCached, parentCachedAt, parentDetailsCache)

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
) *RunDetailsView {
	config := RunDetailsViewConfig{
		Client:             client,
		Run:                run,
		ParentRuns:         parentRuns,
		ParentCached:       parentCached,
		ParentCachedAt:     parentCachedAt,
		ParentDetailsCache: parentDetailsCache,
	}

	return NewRunDetailsViewWithConfig(config)
}

func (v *RunDetailsView) Init() tea.Cmd {
	// Initialize clipboard (will detect CGO availability)
	err := utils.InitClipboard()
	if err != nil {
		// Log error but don't fail - clipboard may not be available in some environments
		debug.LogToFilef("DEBUG: Failed to initialize clipboard: %v\n", err)
	}

	var cmds []tea.Cmd

	// Send a window size message with stored dimensions if we have them
	if v.width > 0 && v.height > 0 {
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: v.width, Height: v.height}
		})
	}

	// Only load details if not already loaded from cache
	debug.LogToFilef("DEBUG: Init() - v.loading=%t, status='%s'\n", v.loading, v.run.Status)
	if v.loading {
		debug.LogToFilef("DEBUG: Need to load data for run '%s'\n", v.run.GetIDString())
		// Try cache one more time before making API call
		if v.parentDetailsCache != nil && v.cacheRetryCount < v.maxCacheRetries {
			runID := v.run.GetIDString()
			if cachedRun, exists := v.parentDetailsCache[runID]; exists && cachedRun != nil {
				// Cache hit on retry!
				debug.LogToFilef("DEBUG: Cache retry successful for runID='%s'\n", runID)

				v.run = *cachedRun
				v.loading = false
				v.updateStatusHistory(string(cachedRun.Status), false)
				v.updateContent()
			} else {
				v.cacheRetryCount++
				debug.LogToFilef("DEBUG: Cache miss on retry %d - making API call for runID='%s'\n", v.cacheRetryCount, runID)
				// Still no cache hit, load from API
				cmds = append(cmds, v.loadRunDetails())
				cmds = append(cmds, v.spinner.Tick)
			}
		} else {
			debug.LogToFilef("DEBUG: No cache available - making API call for runID='%s'\n", v.run.GetIDString())
			// Load from API
			cmds = append(cmds, v.loadRunDetails())
			cmds = append(cmds, v.spinner.Tick)
		}
	} else {
		debug.LogToFilef("DEBUG: Using cached data for run '%s' (status: %s)\n", v.run.GetIDString(), v.run.Status)
	}

	// Only start polling for active runs (startPolling checks status internally)
	cmds = append(cmds, v.startPolling())

	return tea.Batch(cmds...)
}

// handleWindowSizeMsg handles window resize events
func (v *RunDetailsView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height

	// Calculate actual height for viewport:
	// - Title: 2 lines (title + blank line)
	// - Header info: 2-3 lines (status, repo, etc.)
	// - Status bar: 1 line
	// - Help (when shown): estimate 3-4 lines
	nonViewportHeight := 6 // Base: title(2) + header(2) + separator(1) + status bar(1)

	viewportHeight := msg.Height - nonViewportHeight
	if viewportHeight < 3 {
		viewportHeight = 3 // Minimum usable height
	}

	v.viewport.Width = msg.Width
	v.viewport.Height = viewportHeight
	v.help.Width = msg.Width

	// Update content to reflow for new width
	v.updateContent()
}

// handleKeyInput handles all key input events
func (v *RunDetailsView) handleKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case msg.String() == "q", key.Matches(msg, v.keys.Back), msg.Type == tea.KeyEsc, msg.String() == "b", msg.Type == tea.KeyBackspace:
		v.stopPolling()
		// Return to dashboard view with restored state
		dashboard := NewDashboardViewWithState(
			v.client,
			v.dashboardSelectedRepoIdx,
			v.dashboardSelectedRunIdx,
			v.dashboardSelectedDetailLine,
			v.dashboardFocusedColumn,
		)
		// Set dimensions
		dashboard.width = v.width
		dashboard.height = v.height
		// Initialize and let it load data
		return dashboard, dashboard.Init()
	case msg.String() == "Q":
		// Capital Q to force quit from anywhere
		v.stopPolling()
		return v, tea.Quit
	case key.Matches(msg, v.keys.Help):
		// For now, just ignore help in details view
		// Could return to dashboard with docs shown if needed
		return v, nil
	case key.Matches(msg, v.keys.Refresh):
		v.loading = true
		v.error = nil
		cmds = append(cmds, v.loadRunDetails())
		cmds = append(cmds, v.spinner.Tick)
	// Removed logs functionality - not supported yet
	// case msg.String() == "l":
	//	v.showLogs = !v.showLogs
	//	v.updateContent()
	default:
		// Handle navigation in navigation mode
		if v.navigationMode {
			if cmd := v.handleRowNavigation(msg); cmd != nil {
				cmds = append(cmds, cmd)
			} else if cmd := v.handleClipboardOperations(msg.String()); cmd != nil {
				cmds = append(cmds, cmd)
			} else {
				// Handle viewport navigation as fallback
				v.handleViewportNavigation(msg)
			}
		} else {
			// Handle clipboard operations
			if cmd := v.handleClipboardOperations(msg.String()); cmd != nil {
				cmds = append(cmds, cmd)
			} else {
				// Handle viewport navigation
				v.handleViewportNavigation(msg)
			}
		}
	}

	return v, tea.Batch(cmds...)
}

// handleRowNavigation handles navigation between selectable rows/fields
func (v *RunDetailsView) handleRowNavigation(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if v.selectedRow < len(v.fieldValues)-1 {
			v.selectedRow++
			// Scroll viewport if needed to show selected field
			v.scrollToSelectedField()
		}
	case "k", "up":
		if v.selectedRow > 0 {
			v.selectedRow--
			// Scroll viewport if needed to show selected field
			v.scrollToSelectedField()
		}
	case "g":
		// Go to first field
		v.selectedRow = 0
		v.scrollToSelectedField()
	case "G":
		// Go to last field
		if len(v.fieldValues) > 0 {
			v.selectedRow = len(v.fieldValues) - 1
			v.scrollToSelectedField()
		}
	}
	return nil
}

// scrollToSelectedField ensures the selected field is visible in the viewport
func (v *RunDetailsView) scrollToSelectedField() {
	if v.selectedRow >= 0 && v.selectedRow < len(v.fieldRanges) {
		// Get the range of the selected field
		fieldRange := v.fieldRanges[v.selectedRow]
		startLine := fieldRange[0]
		endLine := fieldRange[1]

		viewportTop := v.viewport.YOffset
		viewportBottom := viewportTop + v.viewport.Height - 1

		// If the entire field is above the viewport, scroll to show the start
		if endLine < viewportTop {
			v.viewport.SetYOffset(startLine)
		} else if startLine > viewportBottom {
			// If the entire field is below the viewport, scroll to show as much as possible
			// Try to show the whole field if it fits
			fieldHeight := endLine - startLine + 1
			if fieldHeight <= v.viewport.Height {
				// Field fits in viewport, position it at the top
				v.viewport.SetYOffset(startLine)
			} else {
				// Field is larger than viewport, show the beginning
				v.viewport.SetYOffset(startLine)
			}
		}
		// If part of the field is visible, don't scroll
	}
}

// handleClipboardOperations handles clipboard-related key presses
func (v *RunDetailsView) handleClipboardOperations(key string) tea.Cmd {
	switch key {
	case "y":
		// Copy selected field value to clipboard
		var textToCopy string
		if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldValues) {
			textToCopy = v.fieldValues[v.selectedRow]
			if err := utils.WriteToClipboard(textToCopy); err == nil {
				// Show what's actually copied, truncated for display
				displayText := textToCopy
				maxLen := 30
				if len(displayText) > maxLen {
					displayText = displayText[:maxLen-3] + "..."
				}
				v.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("📋 Copied \"%s\"", displayText), components.MessageSuccess, 100*time.Millisecond)
			} else {
				v.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 100*time.Millisecond)
			}
		} else {
			// Copy current line to clipboard (old behavior)
			currentLine := v.getCurrentLine()
			if currentLine != "" {
				if err := utils.WriteToClipboard(currentLine); err == nil {
					// Show what's actually copied, truncated for display
					displayText := currentLine
					maxLen := 30
					if len(displayText) > maxLen {
						displayText = displayText[:maxLen-3] + "..."
					}
					v.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("📋 Copied \"%s\"", displayText), components.MessageSuccess, 100*time.Millisecond)
				} else {
					v.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 100*time.Millisecond)
				}
			} else {
				v.statusLine.SetTemporaryMessageWithType("✗ No line to copy", components.MessageError, 100*time.Millisecond)
			}
		}
		v.yankBlink = true
		v.yankBlinkTime = time.Now()
		return v.startYankBlinkAnimation()
	case "Y":
		// Copy all content to clipboard
		if err := v.copyAllContent(); err == nil {
			v.statusLine.SetTemporaryMessageWithType("📋 Copied all content", components.MessageSuccess, 100*time.Millisecond)
		} else {
			v.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 100*time.Millisecond)
		}
		v.yankBlink = true
		v.yankBlinkTime = time.Now()
		return v.startYankBlinkAnimation()
	case "o":
		// Open URL in browser if current selection contains a URL
		var urlText string
		if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldValues) {
			// Check if the selected field contains a URL
			fieldValue := v.fieldValues[v.selectedRow]
			if utils.IsURL(fieldValue) {
				urlText = utils.ExtractURL(fieldValue)
			}
		} else {
			// Fallback to current line (old behavior)
			currentLine := v.getCurrentLine()
			if utils.IsURL(currentLine) {
				urlText = utils.ExtractURL(currentLine)
			}
		}

		if urlText != "" {
			if err := utils.OpenURL(urlText); err == nil {
				v.statusLine.SetTemporaryMessageWithType("🌐 Opened URL in browser", components.MessageSuccess, 1*time.Second)
			} else {
				v.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
			}
			v.yankBlink = true
			v.yankBlinkTime = time.Now()
			return tea.Batch(v.startYankBlinkAnimation(), v.startMessageClearTimer(1*time.Second))
		}
	}
	return nil
}

// handleViewportNavigation handles viewport scrolling keys
func (v *RunDetailsView) handleViewportNavigation(msg tea.KeyMsg) {
	switch {
	case key.Matches(msg, v.keys.Up):
		v.viewport.ScrollUp(1)
	case key.Matches(msg, v.keys.Down):
		v.viewport.ScrollDown(1)
	case key.Matches(msg, v.keys.PageUp):
		v.viewport.HalfPageUp()
	case key.Matches(msg, v.keys.PageDown):
		v.viewport.HalfPageDown()
	case key.Matches(msg, v.keys.Home):
		v.viewport.GotoTop()
	case key.Matches(msg, v.keys.End):
		v.viewport.GotoBottom()
	}
}

// handleRunDetailsLoaded handles the runDetailsLoadedMsg message
func (v *RunDetailsView) handleRunDetailsLoaded(msg runDetailsLoadedMsg) {
	v.loading = false
	v.pollingStatus = false // Clear polling status
	v.run = msg.run
	v.error = msg.err
	if msg.err == nil {
		v.updateStatusHistory(string(msg.run.Status), false)
		// Cache the loaded details for future use
		cache.AddCachedDetail(msg.run.GetIDString(), &msg.run)
		debug.LogToFilef("DEBUG: Cached run details for ID '%s' (status: %s)\n", msg.run.GetIDString(), msg.run.Status)
	}
	v.updateContent()

	// Debug logging for successful load
	debug.LogToFilef("DEBUG: Successfully loaded run details for '%s'\n", msg.run.GetIDString())
}

// handlePolling handles the pollTickMsg message
func (v *RunDetailsView) handlePolling(msg pollTickMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if models.IsActiveStatus(string(v.run.Status)) {
		// Mark that we're fetching status
		v.pollingStatus = true
		v.updateStatusHistory("Fetching status...", true)
		v.updateContent()
		cmds = append(cmds, v.loadRunDetails())
		// Keep the polling going
		cmds = append(cmds, v.startPolling())
	} else {
		v.stopPolling()
	}
	return cmds
}

func (v *RunDetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return v.handleKeyInput(msg)

	case runDetailsLoadedMsg:
		v.handleRunDetailsLoaded(msg)

	case pollTickMsg:
		cmds = append(cmds, v.handlePolling(msg)...)

	case yankBlinkMsg:
		// Single blink: toggle off after being on
		if v.yankBlink {
			v.yankBlink = false // Turn off after being on - completes the single blink
		}
		// No more blinking after the single on-off cycle

	case messageClearMsg:
		// Trigger UI refresh when message expires (no action needed - just refresh)

	case spinner.TickMsg:
		if v.loading || v.pollingStatus {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var vpCmd tea.Cmd
	v.viewport, vpCmd = v.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return v, tea.Batch(cmds...)
}

func (v *RunDetailsView) View() string {
	if v.height == 0 || v.width == 0 {
		// Terminal dimensions not yet known
		return ""
	}

	// For very small terminals, render minimal content
	if v.height < 5 || v.width < 20 {
		return "Run ID: " + v.run.GetIDString()
	}

	// Calculate box dimensions - leave room for statusline at bottom and ensure top border shows
	boxWidth := v.width - 4   // Leave 2 chars margin on each side
	boxHeight := v.height - 3 // Leave room for statusline at bottom and top margin

	if boxWidth < 10 {
		boxWidth = 10
	}
	if boxHeight < 3 {
		boxHeight = 3
	}

	// Box style with rounded border
	boxStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	// Title bar (inside the box) - no background to avoid conflict with cursor
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")). // Use cyan color for text instead of background
		Width(boxWidth-2).                // Account for border
		Align(lipgloss.Center).
		Padding(0, 1)

	// Create title with status
	statusIcon := styles.GetStatusIcon(string(v.run.Status))
	idStr := v.run.GetIDString()
	if len(idStr) > 8 {
		idStr = idStr[:8]
	}
	titleText := fmt.Sprintf("Run #%s", idStr)
	if v.run.Title != "" {
		maxTitleLen := boxWidth - 20 // Leave room for status and padding
		if maxTitleLen > 0 && len(v.run.Title) > maxTitleLen {
			titleText += " - " + v.run.Title[:maxTitleLen] + "..."
		} else {
			titleText += " - " + v.run.Title
		}
	}
	titleText = fmt.Sprintf("%s %s %s", statusIcon, titleText, string(v.run.Status))

	// Add polling indicator if active
	if models.IsActiveStatus(string(v.run.Status)) {
		if v.pollingStatus {
			titleText += " [Fetching... " + v.spinner.View() + "]"
		} else {
			titleText += " [Monitoring ⟳]"
		}
	}

	title := titleStyle.Render(titleText)

	// Content area height (subtract title height from box interior)
	contentHeight := boxHeight - 3 // 1 for title, 2 for borders

	// Create viewport content
	var innerContent string
	if v.loading {
		// Center loading message
		loadingText := v.spinner.View() + " Loading run details..."
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Width(boxWidth-2).
			Height(contentHeight).
			Align(lipgloss.Center, lipgloss.Center)
		innerContent = lipgloss.JoinVertical(lipgloss.Left, title, loadingStyle.Render(loadingText))
	} else if v.error != nil {
		// Show error
		errorText := "Error: " + v.error.Error()
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Width(boxWidth-2).
			Height(contentHeight).
			Padding(1, 2)
		innerContent = lipgloss.JoinVertical(lipgloss.Left, title, errorStyle.Render(errorText))
	} else {
		// Render content with scrollable viewport
		v.viewport.Width = boxWidth - 4 // Account for border and padding
		v.viewport.Height = contentHeight

		// Get content with highlighting
		contentLines := v.renderContentWithCursor()
		content := strings.Join(contentLines, "\n")

		// Apply padding to content
		contentStyle := lipgloss.NewStyle().
			Width(boxWidth-2).
			Height(contentHeight).
			Padding(0, 1)

		innerContent = lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(content))
	}

	// Wrap in the box
	boxedContent := boxStyle.Render(innerContent)

	// Center the box on screen (leaving room for statusline and top margin)
	// Add 1 line of top margin to ensure top border is visible
	centeredBox := lipgloss.Place(v.width, v.height-1, lipgloss.Center, lipgloss.Center, boxedContent)

	// Create the statusline
	statusLine := v.renderStatusBar()

	// Join the centered box and statusline
	return lipgloss.JoinVertical(lipgloss.Left, centeredBox, statusLine)
}

// renderContentWithCursor renders the content with a visible row selector
func (v *RunDetailsView) renderContentWithCursor() []string {
	if v.showLogs {
		// For logs view, just return the viewport content as-is
		return strings.Split(v.viewport.View(), "\n")
	}

	// Get all content lines
	allLines := strings.Split(v.fullContent, "\n")
	if len(allLines) == 0 {
		return []string{}
	}

	// Calculate viewport bounds
	viewportHeight := v.viewport.Height
	if viewportHeight <= 0 {
		viewportHeight = v.height - 6 // Fallback calculation
	}

	// Get the current viewport offset
	viewportOffset := v.viewport.YOffset

	// Determine which lines are visible
	visibleLines := []string{}
	contentWidth := v.viewport.Width
	if contentWidth <= 0 {
		contentWidth = v.width - 6 // Account for box borders and padding
	}

	for i := viewportOffset; i < len(allLines) && i < viewportOffset+viewportHeight; i++ {
		line := allLines[i]

		// Truncate line if too long
		lineRunes := []rune(line)
		if len(lineRunes) > contentWidth {
			line = string(lineRunes[:contentWidth-3]) + "..."
		}

		// Check if this line should be highlighted
		shouldHighlight := false
		if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldRanges) {
			fieldRange := v.fieldRanges[v.selectedRow]
			if i >= fieldRange[0] && i <= fieldRange[1] {
				shouldHighlight = true
			}
		}

		if shouldHighlight {
			// Apply highlight style similar to dashboard
			var highlightedLine string
			if v.yankBlink && !v.yankBlinkTime.IsZero() && time.Since(v.yankBlinkTime) < 250*time.Millisecond {
				// Bright green flash for copy feedback
				highlightStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("82")). // Bright green
					Foreground(lipgloss.Color("0")).  // Black text
					Bold(true).
					Width(contentWidth).
					MaxWidth(contentWidth).
					Inline(true)
				highlightedLine = highlightStyle.Render(line)
			} else {
				// Normal focused highlight (matching dashboard style)
				highlightStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("255")).
					Width(contentWidth).
					MaxWidth(contentWidth).
					Inline(true)
				highlightedLine = highlightStyle.Render(line)
			}
			visibleLines = append(visibleLines, highlightedLine)
		} else {
			// Non-selected lines with width constraint
			styledLine := lipgloss.NewStyle().
				Width(contentWidth).
				MaxWidth(contentWidth).
				Inline(true).
				Render(line)
			visibleLines = append(visibleLines, styledLine)
		}
	}

	return visibleLines
}

func (v *RunDetailsView) renderHeader() string {
	statusIcon := styles.GetStatusIcon(string(v.run.Status))
	statusStyle := styles.GetStatusStyle(string(v.run.Status))
	status := statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, v.run.Status))

	idStr := v.run.GetIDString()
	if len(idStr) > 8 {
		idStr = idStr[:8]
	}
	title := fmt.Sprintf("Run #%s", idStr)
	if v.run.Title != "" {
		title += " - " + v.run.Title
	}

	// Truncate title if too long for terminal width
	if v.width > 25 && len(title) > v.width-20 {
		maxLen := v.width - 23
		if maxLen > 0 && maxLen < len(title) {
			title = title[:maxLen] + "..."
		}
	}

	header := styles.TitleStyle.MaxWidth(v.width).Render(title)

	if models.IsActiveStatus(string(v.run.Status)) {
		if v.pollingStatus {
			// Show active polling indicator
			pollingIndicator := styles.ProcessingStyle.Render(" [Fetching... " + v.spinner.View() + "]")
			header += pollingIndicator
		} else {
			// Show passive polling indicator
			pollingIndicator := styles.ProcessingStyle.Render(" [Monitoring ⟳]")
			header += pollingIndicator
		}
	}

	rightAlign := lipgloss.NewStyle().Align(lipgloss.Right).Width(v.width - lipgloss.Width(header))
	header += rightAlign.Render("Status: " + status)

	return header
}

// hasCurrentSelectionURL checks if the current selection contains a URL
func (v *RunDetailsView) hasCurrentSelectionURL() bool {
	if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldValues) {
		fieldValue := v.fieldValues[v.selectedRow]
		return utils.IsURL(fieldValue)
	}
	// Fallback to current line
	currentLine := v.getCurrentLine()
	return utils.IsURL(currentLine)
}

func (v *RunDetailsView) renderStatusBar() string {
	// Shorter options to fit better (removed logs functionality)
	options := "q:back j/k:nav y:copy Y:all r:refresh ?:help Q:quit"

	// Add URL opening hint if current selection has a URL
	if v.hasCurrentSelectionURL() {
		options = "o:url q:back j/k:nav y:copy Y:all r:refresh ?:help Q:quit"
	}

	// Use unified status line system
	return v.statusLine.
		SetWidth(v.width).
		SetLeft("[DETAILS]").
		SetRight("").
		SetHelp(options).
		Render()
}

func (v *RunDetailsView) updateContent() {
	var content strings.Builder
	var lines []string
	v.fieldLines = []string{}
	v.fieldValues = []string{}
	v.fieldIndices = []int{}
	v.fieldRanges = [][2]int{}
	lineCount := 0

	if v.showLogs {
		content.WriteString("═══ Logs ═══\n\n")
		if v.logs != "" {
			content.WriteString(v.logs)
		} else {
			content.WriteString("No logs available yet...\n")
		}
	} else {
		// Helper to add a single-line field and track its position
		addField := func(label, value string) {
			if value != "" {
				line := fmt.Sprintf("%s: %s", label, value)
				content.WriteString(line + "\n")
				lines = append(lines, line)
				v.fieldLines = append(v.fieldLines, line)
				v.fieldValues = append(v.fieldValues, value)
				v.fieldIndices = append(v.fieldIndices, lineCount)
				v.fieldRanges = append(v.fieldRanges, [2]int{lineCount, lineCount})
				lineCount++
			}
		}

		addSeparator := func(text string) {
			content.WriteString(text + "\n")
			lines = append(lines, text)
			lineCount++
		}

		// Display title only if it exists
		if v.run.Title != "" {
			addField("Title", v.run.Title)
		}
		// Display description if it exists (with truncation for display but full value for copying)
		if v.run.Description != "" {
			originalDescription := v.run.Description
			displayDescription := originalDescription
			// Truncate to single line with ellipsis if too long (for display only)
			if len(displayDescription) > 60 {
				displayDescription = displayDescription[:57] + "..."
			}
			// Remove newlines to keep it single line (for display only)
			displayDescription = strings.ReplaceAll(displayDescription, "\n", " ")

			// Add field with display text but store original value for copying
			line := fmt.Sprintf("Description: %s", displayDescription)
			content.WriteString(line + "\n")
			lines = append(lines, line)
			v.fieldLines = append(v.fieldLines, line)
			v.fieldValues = append(v.fieldValues, originalDescription) // Store original for copying
			v.fieldIndices = append(v.fieldIndices, lineCount)
			v.fieldRanges = append(v.fieldRanges, [2]int{lineCount, lineCount})
			lineCount++
		}
		addField("Run ID", v.run.GetIDString())
		addField("Repository", v.run.Repository)
		addField("Source Branch", v.run.Source)
		if v.run.Target != "" && v.run.Target != v.run.Source {
			addField("Target Branch", v.run.Target)
		}
		if v.run.RunType != "" {
			addField("Run Type", v.run.RunType)
		}
		if v.run.PrURL != nil && *v.run.PrURL != "" {
			addField("PR URL", *v.run.PrURL)
		}
		if v.run.TriggerSource != nil && *v.run.TriggerSource != "" {
			addField("Trigger Source", *v.run.TriggerSource)
		}
		addField("Created", v.run.CreatedAt.Format(time.RFC3339))

		if v.run.UpdatedAt.After(v.run.CreatedAt) && (v.run.Status == models.StatusDone || v.run.Status == models.StatusFailed) {
			duration := v.run.UpdatedAt.Sub(v.run.CreatedAt)
			addField("Duration", formatDuration(duration))
		}

		addSeparator("\n═══ Status History ═══")
		// Display status history in reverse order (most recent first)
		for i := len(v.statusHistory) - 1; i >= 0; i-- {
			content.WriteString(v.statusHistory[i] + "\n")
			lines = append(lines, v.statusHistory[i])
			lineCount++
		}

		// Helper to add multi-line field and track its range
		addMultilineField := func(label, value string) {
			if value != "" {
				v.fieldLines = append(v.fieldLines, label)
				v.fieldValues = append(v.fieldValues, value)
				v.fieldIndices = append(v.fieldIndices, lineCount)

				startLine := lineCount
				fieldLines := strings.Split(value, "\n")
				for _, fieldLine := range fieldLines {
					content.WriteString(fieldLine + "\n")
					lines = append(lines, fieldLine)
					lineCount++
				}
				endLine := lineCount - 1
				v.fieldRanges = append(v.fieldRanges, [2]int{startLine, endLine})
			}
		}

		if v.run.Prompt != "" {
			addSeparator("\n═══ Prompt ═══")
			addMultilineField("Prompt", v.run.Prompt)
		}

		// Show plan for plan-type runs that are completed (includes "plan", "pro-plan", etc.)
		if strings.Contains(strings.ToLower(v.run.RunType), "plan") && v.run.Status == models.StatusDone && v.run.Plan != "" {
			addSeparator("\n═══ Plan ═══")
			addMultilineField("Plan", v.run.Plan)
		}

		if v.run.Context != "" {
			addSeparator("\n═══ Context ═══")
			addMultilineField("Context", v.run.Context)
		}

		if v.run.Error != "" {
			addSeparator("\n═══ Error ═══")
			// Special handling for error to apply styling
			v.fieldLines = append(v.fieldLines, "Error")
			v.fieldValues = append(v.fieldValues, v.run.Error)
			v.fieldIndices = append(v.fieldIndices, lineCount)

			startLine := lineCount
			errorLines := strings.Split(v.run.Error, "\n")
			for _, errorLine := range errorLines {
				styledLine := styles.ErrorStyle.Render(errorLine)
				content.WriteString(styledLine + "\n")
				lines = append(lines, errorLine)
				lineCount++
			}
			endLine := lineCount - 1
			v.fieldRanges = append(v.fieldRanges, [2]int{startLine, endLine})
		}
	}

	// Store the full content for clipboard operations
	v.fullContent = content.String()

	// Set the content in the viewport (without highlighting, as we'll do that in rendering)
	v.viewport.SetContent(v.fullContent)

	// Ensure selected row is within bounds
	if v.selectedRow >= len(v.fieldValues) && len(v.fieldValues) > 0 {
		v.selectedRow = len(v.fieldValues) - 1
	} else if v.selectedRow < 0 && len(v.fieldValues) > 0 {
		v.selectedRow = 0
	}
}

// createHighlightedContent creates content with the selected field highlighted
func (v *RunDetailsView) createHighlightedContent(lines []string) string {
	if v.selectedRow < 0 || v.selectedRow >= len(v.fieldRanges) {
		return v.fullContent
	}

	// Get the range of lines for the selected field
	fieldRange := v.fieldRanges[v.selectedRow]
	startLine := fieldRange[0]
	endLine := fieldRange[1]

	var result strings.Builder
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("15"))

	for i, line := range lines {
		if i >= startLine && i <= endLine {
			// Highlight all lines in the field range
			result.WriteString(highlightStyle.Render(line))
		} else {
			result.WriteString(line)
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (v *RunDetailsView) updateStatusHistory(status string, isPolling bool) {
	timestamp := time.Now().Format("15:04:05")
	var entry string

	if isPolling {
		// Show polling indicator
		entry = fmt.Sprintf("[%s] 🔄 %s", timestamp, status)
	} else {
		// Regular status update - only add if different from last non-polling status
		if len(v.statusHistory) > 0 {
			// Find last non-polling status
			for i := len(v.statusHistory) - 1; i >= 0; i-- {
				if !strings.Contains(v.statusHistory[i], "🔄") && strings.Contains(v.statusHistory[i], status) {
					// Same status as before, don't add duplicate
					return
				}
			}
		}
		statusIcon := styles.GetStatusIcon(status)
		entry = fmt.Sprintf("[%s] %s %s", timestamp, statusIcon, status)
	}

	v.statusHistory = append(v.statusHistory, entry)

	// Keep history size reasonable
	if len(v.statusHistory) > 50 {
		v.statusHistory = v.statusHistory[len(v.statusHistory)-50:]
	}
}

func (v *RunDetailsView) loadRunDetails() tea.Cmd {
	// Capture the current run ID to ensure it doesn't get lost
	originalRunID := v.run.GetIDString()
	originalRun := v.run

	return func() tea.Msg {
		if originalRunID == "" {
			// Debug: Log empty run ID issue
			debug.LogToFile("DEBUG: LoadRunDetails called with empty runID - returning error\n")
			return runDetailsLoadedMsg{run: originalRun, err: fmt.Errorf("invalid run ID: empty string")}
		}

		// Debug: Log API call for run details
		debug.LogToFilef("DEBUG: LoadRunDetails calling GetRun for runID='%s'\n", originalRunID)

		runPtr, err := v.client.GetRun(originalRunID)
		if err != nil {
			debug.LogToFilef("DEBUG: GetRun failed for runID='%s', err=%v\n", originalRunID, err)
			return runDetailsLoadedMsg{run: originalRun, err: fmt.Errorf("API error for run %s: %w", originalRunID, err)}
		}

		if runPtr == nil {
			debug.LogToFilef("DEBUG: GetRun returned nil for runID='%s'\n", originalRunID)
			return runDetailsLoadedMsg{run: originalRun, err: fmt.Errorf("API returned nil for run %s", originalRunID)}
		}

		// Ensure the returned run has the correct ID
		updatedRun := *runPtr
		if updatedRun.GetIDString() == "" && originalRun.ID != "" {
			updatedRun.ID = originalRun.ID
		}

		debug.LogToFilef("DEBUG: LoadRunDetails successful for runID='%s', newID='%s'\n",
			originalRunID, updatedRun.GetIDString())

		return runDetailsLoadedMsg{run: updatedRun, err: nil}
	}
}

func (v *RunDetailsView) startPolling() tea.Cmd {
	if !models.IsActiveStatus(string(v.run.Status)) {
		debug.LogToFilef("DEBUG: Not polling - status '%s' is not active\n", v.run.Status)
		return nil
	}

	// Don't poll runs older than 3 hours
	if time.Since(v.run.CreatedAt) > 3*time.Hour {
		debug.LogToFilef("DEBUG: Not polling - run created %v ago (older than 3h)\n", time.Since(v.run.CreatedAt))
		return nil
	}

	debug.LogToFilef("DEBUG: Starting polling for active run '%s' (status: %s, age: %v)\n",
		v.run.GetIDString(), v.run.Status, time.Since(v.run.CreatedAt))

	v.pollTicker = time.NewTicker(10 * time.Second) // Poll every 10 seconds
	v.pollStop = make(chan bool)

	return func() tea.Msg {
		for {
			select {
			case <-v.pollTicker.C:
				return pollTickMsg{}
			case <-v.pollStop:
				return nil
			}
		}
	}
}

func (v *RunDetailsView) stopPolling() {
	if v.pollTicker != nil {
		v.pollTicker.Stop()
	}
	if v.pollStop != nil {
		select {
		case <-v.pollStop:
		default:
			close(v.pollStop)
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

type runDetailsLoadedMsg struct {
	run models.RunResponse
	err error
}

type clearStatusMsg struct{}

// getCurrentLine gets the current visible line
func (v *RunDetailsView) getCurrentLine() string {
	// Get the current line based on viewport position
	// The viewport tracks the top visible line
	visibleLines := strings.Split(v.viewport.View(), "\n")
	if len(visibleLines) > 0 {
		// Get first visible line (could be partial due to scrolling)
		currentLine := visibleLines[0]
		if currentLine != "" {
			return currentLine
		}
	}
	return ""
}

// copyCurrentLine copies the current visible line to clipboard
func (v *RunDetailsView) copyCurrentLine() error {
	// Get the current line based on viewport position
	// The viewport tracks the top visible line
	visibleLines := strings.Split(v.viewport.View(), "\n")
	if len(visibleLines) > 0 {
		// Get first visible line (could be partial due to scrolling)
		currentLine := visibleLines[0]
		if currentLine != "" {
			return utils.WriteToClipboard(currentLine)
		}
	}

	return fmt.Errorf("no line to copy")
}

// copyAllContent copies all content to clipboard
func (v *RunDetailsView) copyAllContent() error {
	if v.fullContent == "" {
		return fmt.Errorf("no content to copy")
	}
	return utils.WriteToClipboard(v.fullContent)
}

// startYankBlinkAnimation starts the single blink animation for clipboard feedback
func (v *RunDetailsView) startYankBlinkAnimation() tea.Cmd {
	return func() tea.Msg {
		// Single blink duration - quick flash (100ms)
		time.Sleep(100 * time.Millisecond)
		return yankBlinkMsg{}
	}
}

// startMessageClearTimer starts a timer to trigger UI refresh when message expires
func (v *RunDetailsView) startMessageClearTimer(duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return messageClearMsg{}
	}
}

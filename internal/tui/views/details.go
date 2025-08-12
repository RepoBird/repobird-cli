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
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

type RunDetailsView struct {
	client        APIClient
	runID         string // Store just the ID for loading
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
	// Shared cache from app level
	cache *cache.SimpleCache
}

// Constructors are defined in details_constructors.go

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

	// Load run details if needed
	debug.LogToFilef("DEBUG: Init() - v.loading=%t, runID='%s'\n", v.loading, v.runID)
	if v.loading {
		debug.LogToFilef("DEBUG: Need to load data for run '%s'\n", v.runID)
		
		// Check cache first
		if v.cache != nil {
			// Try to get from cache
			runs, _, detailsCache := v.cache.GetCachedList()
			if detailsCache != nil {
				if cachedRun, exists := detailsCache[v.runID]; exists && cachedRun != nil {
					// Cache hit!
					debug.LogToFilef("DEBUG: Cache hit for runID='%s'\n", v.runID)
					v.run = *cachedRun
					v.loading = false
					v.updateStatusHistory(string(cachedRun.Status), false)
					v.updateContent()
				} else {
					// Cache miss, load from API
					debug.LogToFilef("DEBUG: Cache miss - making API call for runID='%s'\n", v.runID)
					cmds = append(cmds, v.loadRunDetails())
					cmds = append(cmds, v.spinner.Tick)
				}
			} else {
				// No cache available, load from API
				debug.LogToFilef("DEBUG: No cache available - making API call for runID='%s'\n", v.runID)
				cmds = append(cmds, v.loadRunDetails())
				cmds = append(cmds, v.spinner.Tick)
			}
			
			// Also save runs to avoid unused variable error
			_ = runs
		} else {
			// No cache, load from API
			debug.LogToFilef("DEBUG: No cache configured - making API call for runID='%s'\n", v.runID)
			cmds = append(cmds, v.loadRunDetails())
			cmds = append(cmds, v.spinner.Tick)
		}
	} else {
		debug.LogToFilef("DEBUG: Already have data for run '%s' (status: %s)\n", v.runID, v.run.Status)
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
		// Navigate back or to dashboard
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}
	case msg.String() == "Q":
		// Capital Q to force quit from anywhere
		v.stopPolling()
		return v, tea.Quit
	case msg.String() == "d":
		// d key to go to dashboard
		v.stopPolling()
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
	case key.Matches(msg, v.keys.Help):
		// Navigate to dashboard with docs
		v.stopPolling()
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
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

// Clipboard operations are defined in details_clipboard.go

// Update handles incoming events
func (v *RunDetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
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
			// Also update the status line spinner
			v.statusLine.UpdateSpinner()
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
// Rendering methods are defined in details_rendering.go

// Rendering helper methods are defined in details_rendering.go

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
			addField("Duration", formatDurationDetails(duration))
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
// Status history and highlighting methods are defined in details_rendering.go

func (v *RunDetailsView) loadRunDetails() tea.Cmd {
	// Use the stored runID directly
	runID := v.runID

	return func() tea.Msg {
		if runID == "" {
			// Debug: Log empty run ID issue
			debug.LogToFile("DEBUG: LoadRunDetails called with empty runID - returning error\n")
			return runDetailsLoadedMsg{run: v.run, err: fmt.Errorf("invalid run ID: empty string")}
		}

		// Debug: Log API call for run details
		debug.LogToFilef("DEBUG: LoadRunDetails calling GetRun for runID='%s'\n", runID)

		runPtr, err := v.client.GetRun(runID)
		if err != nil {
			debug.LogToFilef("DEBUG: GetRun failed for runID='%s', err=%v\n", runID, err)
			return runDetailsLoadedMsg{run: v.run, err: fmt.Errorf("API error for run %s: %w", runID, err)}
		}

		if runPtr == nil {
			debug.LogToFilef("DEBUG: GetRun returned nil for runID='%s'\n", runID)
			return runDetailsLoadedMsg{run: v.run, err: fmt.Errorf("API returned nil for run %s", runID)}
		}

		// Ensure the returned run has the correct ID
		updatedRun := *runPtr
		if updatedRun.GetIDString() == "" && runID != "" {
			updatedRun.ID = runID
		}

		debug.LogToFilef("DEBUG: LoadRunDetails successful for runID='%s', newID='%s'\n",
			runID, updatedRun.GetIDString())

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

// handleRunDetailsLoaded handles the runDetailsLoadedMsg message
func (v *RunDetailsView) handleRunDetailsLoaded(msg runDetailsLoadedMsg) {
	v.loading = false
	v.pollingStatus = false // Clear polling status
	v.run = msg.run
	v.error = msg.err
	if msg.err == nil {
		v.updateStatusHistory(string(msg.run.Status), false)
		// Cache the loaded details for future use
		v.cache.SetRun(msg.run)
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

// Message types for details view
type runDetailsLoadedMsg struct {
	run models.RunResponse
	err error
}

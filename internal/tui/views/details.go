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
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

type RunDetailsView struct {
	client        *api.Client
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
	showHelp      bool
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
	// Clipboard feedback
	copiedMessage     string
	copiedMessageTime time.Time
	yankBlink         bool // Toggle for blinking effect
	// Store full content for clipboard operations
	fullContent string
}

func NewRunDetailsView(client *api.Client, run models.RunResponse) *RunDetailsView {
	// Get the current global cache
	runs, cached, cachedAt, detailsCache, _ := cache.GetCachedList()
	return NewRunDetailsViewWithCache(client, run, runs, cached, cachedAt, detailsCache)
}

// RunDetailsViewConfig holds configuration for creating a new RunDetailsView
type RunDetailsViewConfig struct {
	Client             *api.Client
	Run                models.RunResponse
	ParentRuns         []models.RunResponse
	ParentCached       bool
	ParentCachedAt     time.Time
	ParentDetailsCache map[string]*models.RunResponse
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
		client:             config.Client,
		run:                run,
		keys:               components.DefaultKeyMap,
		help:               help.New(),
		viewport:           vp,
		spinner:            s,
		loading:            needsLoading,
		showLogs:           false,
		parentRuns:         config.ParentRuns,
		parentCached:       config.ParentCached,
		parentCachedAt:     config.ParentCachedAt,
		parentDetailsCache: config.ParentDetailsCache,
		statusHistory:      make([]string, 0),
		cacheRetryCount:    0,
		maxCacheRetries:    3,
	}

	// Initialize status history with current status if we have cached data
	if !needsLoading {
		v.updateStatusHistory(string(run.Status), false)
		v.updateContent()
	}

	return v
}

// NewRunDetailsViewWithCacheAndDimensions creates a new details view with cache and dimensions
func NewRunDetailsViewWithCacheAndDimensions(
	client *api.Client,
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
	client *api.Client,
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
	if v.loading {
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
				// Still no cache hit, load from API
				cmds = append(cmds, v.loadRunDetails())
				cmds = append(cmds, v.spinner.Tick)
			}
		} else {
			// Load from API
			cmds = append(cmds, v.loadRunDetails())
			cmds = append(cmds, v.spinner.Tick)
		}
	}

	// Always start polling for active runs
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
	if v.showHelp {
		nonViewportHeight += 4
	}
	if v.copiedMessage != "" {
		nonViewportHeight++ // Feedback message takes a line
	}

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
	case key.Matches(msg, v.keys.Quit):
		v.stopPolling()
		return v, tea.Quit
	case key.Matches(msg, v.keys.Back):
		v.stopPolling()
		// Return to list view with cached data and current dimensions
		return NewRunListViewWithCacheAndDimensions(
			v.client, v.parentRuns, v.parentCached, v.parentCachedAt,
			v.parentDetailsCache, -1, v.width, v.height), nil
	case key.Matches(msg, v.keys.Help):
		v.showHelp = !v.showHelp
	case key.Matches(msg, v.keys.Refresh):
		v.loading = true
		v.error = nil
		cmds = append(cmds, v.loadRunDetails())
		cmds = append(cmds, v.spinner.Tick)
	case msg.String() == "l":
		v.showLogs = !v.showLogs
		v.updateContent()
	default:
		// Handle clipboard operations
		if cmd := v.handleClipboardOperations(msg.String()); cmd != nil {
			cmds = append(cmds, cmd)
		} else {
			// Handle viewport navigation
			v.handleViewportNavigation(msg)
		}
	}

	return v, tea.Batch(cmds...)
}

// handleClipboardOperations handles clipboard-related key presses
func (v *RunDetailsView) handleClipboardOperations(key string) tea.Cmd {
	switch key {
	case "y":
		// Copy current line to clipboard
		if err := v.copyCurrentLine(); err == nil {
			v.copiedMessage = "📋 Copied current line"
		} else {
			v.copiedMessage = "✗ Failed to copy"
		}
		v.copiedMessageTime = time.Now()
		v.yankBlink = true
		return v.startYankBlinkAnimation()
	case "Y":
		// Copy all content to clipboard
		if err := v.copyAllContent(); err == nil {
			v.copiedMessage = "📋 Copied all content"
		} else {
			v.copiedMessage = "✗ Failed to copy"
		}
		v.copiedMessageTime = time.Now()
		v.yankBlink = true
		return v.startYankBlinkAnimation()
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
		// Toggle blink state for animation effect
		v.yankBlink = !v.yankBlink
		// Continue blinking for the first second
		if time.Since(v.copiedMessageTime) < 1*time.Second {
			cmds = append(cmds, v.startYankBlinkAnimation())
		}

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
	if v.height < 3 || v.width < 10 {
		return "Run ID: " + v.run.GetIDString()
	}

	// Pre-allocate array for exactly terminal height lines
	lines := make([]string, v.height)
	lineIdx := 0

	// Header
	header := v.renderHeader()
	if lineIdx < len(lines) {
		lines[lineIdx] = header
		lineIdx++
	}

	// Separator line
	if lineIdx < len(lines) {
		separatorWidth := v.width
		if separatorWidth > 80 {
			separatorWidth = 80 // Reasonable max width
		}
		lines[lineIdx] = strings.Repeat("─", separatorWidth)
		lineIdx++
	}

	// Content area
	if v.loading {
		if lineIdx < len(lines) {
			lines[lineIdx] = v.spinner.View() + " Loading run details..."
			lineIdx++
		}
	} else if v.error != nil {
		if lineIdx < len(lines) {
			lines[lineIdx] = styles.ErrorStyle.Render("Error: " + v.error.Error())
			lineIdx++
		}
	} else {
		// Viewport content
		viewportContent := v.viewport.View()
		if viewportContent != "" {
			// Split viewport content into lines
			viewportLines := strings.Split(strings.TrimRight(viewportContent, "\n"), "\n")
			for _, line := range viewportLines {
				if lineIdx < len(lines)-1 { // Leave room for status bar
					lines[lineIdx] = line
					lineIdx++
				}
			}
		}
	}

	// Help content (if shown) - place just before status bar
	if v.showHelp {
		helpContent := v.help.View(v.keys)
		if helpContent != "" {
			helpLines := strings.Split(strings.TrimRight(helpContent, "\n"), "\n")
			// Place help starting from height-1-len(helpLines)
			helpStart := v.height - 1 - len(helpLines)
			if helpStart < lineIdx {
				helpStart = lineIdx
			}
			for i, line := range helpLines {
				helpIdx := helpStart + i
				if helpIdx < len(lines)-1 && helpIdx >= 0 {
					lines[helpIdx] = line
				}
			}
		}
	}

	// Status bar always goes in the last line
	if len(lines) > 0 {
		lines[len(lines)-1] = v.renderStatusBar()
	}

	// Join all lines with newlines
	// This creates exactly height-1 newlines, which is correct
	return strings.Join(lines, "\n")
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

func (v *RunDetailsView) renderStatusBar() string {
	options := "[b]ack [l]ogs [y]copy line [Y]copy all [r]efresh [?]help [q]uit"

	if v.showLogs {
		options = "[b]ack [l]details [y]copy line [Y]copy all [r]efresh [?]help [q]uit"
	}

	// Show copied message if recent with blinking effect
	if v.copiedMessage != "" && time.Since(v.copiedMessageTime) < 3*time.Second {
		copiedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

		// Apply blinking effect for the first second
		if time.Since(v.copiedMessageTime) < 1*time.Second && v.yankBlink {
			copiedStyle = copiedStyle.Blink(true)
		}

		options = copiedStyle.Render(v.copiedMessage) + " | " + options
	}

	return styles.StatusBarStyle.Width(v.width).Render(options)
}

func (v *RunDetailsView) updateContent() {
	var content strings.Builder

	if v.showLogs {
		content.WriteString("═══ Logs ═══\n\n")
		if v.logs != "" {
			content.WriteString(v.logs)
		} else {
			content.WriteString("No logs available yet...\n")
		}
	} else {
		// Display title only if it exists
		if v.run.Title != "" {
			content.WriteString(fmt.Sprintf("Title: %s\n", v.run.Title))
		}
		content.WriteString(fmt.Sprintf("Run ID: %s\n", v.run.GetIDString()))
		content.WriteString(fmt.Sprintf("Repository: %s\n", v.run.Repository))
		content.WriteString(fmt.Sprintf("Source Branch: %s\n", v.run.Source))
		if v.run.Target != "" && v.run.Target != v.run.Source {
			content.WriteString(fmt.Sprintf("Target Branch: %s\n", v.run.Target))
		}
		if v.run.RunType != "" {
			content.WriteString(fmt.Sprintf("Run Type: %s\n", v.run.RunType))
		}
		content.WriteString(fmt.Sprintf("Created: %s\n", v.run.CreatedAt.Format(time.RFC3339)))

		if v.run.UpdatedAt.After(v.run.CreatedAt) && (v.run.Status == models.StatusDone || v.run.Status == models.StatusFailed) {
			duration := v.run.UpdatedAt.Sub(v.run.CreatedAt)
			content.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(duration)))
		}

		content.WriteString("\n═══ Status History ═══\n")
		// Display status history in reverse order (most recent first)
		for i := len(v.statusHistory) - 1; i >= 0; i-- {
			content.WriteString(v.statusHistory[i] + "\n")
		}

		if v.run.Prompt != "" {
			content.WriteString("\n═══ Prompt ═══\n")
			content.WriteString(v.run.Prompt + "\n")
		}

		// Show plan for plan-type runs that are completed (includes "plan", "pro-plan", etc.)
		if strings.Contains(strings.ToLower(v.run.RunType), "plan") && v.run.Status == models.StatusDone && v.run.Plan != "" {
			content.WriteString("\n═══ Plan ═══\n")
			content.WriteString(v.run.Plan + "\n")
		}

		if v.run.Context != "" {
			content.WriteString("\n═══ Context ═══\n")
			content.WriteString(v.run.Context + "\n")
		}

		if v.run.Error != "" {
			content.WriteString("\n═══ Error ═══\n")
			content.WriteString(styles.ErrorStyle.Render(v.run.Error) + "\n")
		}
	}

	// Store the full content for clipboard operations
	v.fullContent = content.String()
	v.viewport.SetContent(v.fullContent)
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
		return nil
	}

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

type yankBlinkMsg struct{}

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

// startYankBlinkAnimation starts the blinking animation for clipboard feedback
func (v *RunDetailsView) startYankBlinkAnimation() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return yankBlinkMsg{}
	}
}

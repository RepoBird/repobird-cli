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
	"golang.design/x/clipboard"
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
		v.updateStatusHistory(string(run.Status))
		v.updateContent()
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
	// Initialize clipboard
	err := clipboard.Init()
	if err != nil {
		// Log error but don't fail - clipboard may not be available in some environments
		debug.LogToFilef("DEBUG: Failed to initialize clipboard: %v\n", err)
	}

	var cmds []tea.Cmd

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
				v.updateStatusHistory(string(cachedRun.Status))
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
		// Return to list view with cached data
		return NewRunListViewWithCache(v.client, v.parentRuns, v.parentCached, v.parentCachedAt, v.parentDetailsCache, -1), nil
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
			v.copiedMessage = "✓ Copied current line"
		} else {
			v.copiedMessage = "✗ Failed to copy"
		}
		v.copiedMessageTime = time.Now()
		return nil
	case "Y":
		// Copy all content to clipboard
		if err := v.copyAllContent(); err == nil {
			v.copiedMessage = "✓ Copied all content"
		} else {
			v.copiedMessage = "✗ Failed to copy"
		}
		v.copiedMessageTime = time.Now()
		return nil
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
	v.run = msg.run
	v.error = msg.err
	if msg.err == nil {
		v.updateStatusHistory(string(msg.run.Status))
	}
	v.updateContent()

	// Debug logging for successful load
	debug.LogToFilef("DEBUG: Successfully loaded run details for '%s'\n", msg.run.GetIDString())
}

// handlePolling handles the pollTickMsg message
func (v *RunDetailsView) handlePolling(msg pollTickMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if models.IsActiveStatus(string(v.run.Status)) {
		cmds = append(cmds, v.loadRunDetails())
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

	case spinner.TickMsg:
		if v.loading {
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
	var s strings.Builder

	header := v.renderHeader()
	s.WriteString(header)
	s.WriteString("\n")
	s.WriteString(strings.Repeat("─", v.width))
	s.WriteString("\n")

	if v.loading {
		s.WriteString(v.spinner.View() + " Loading run details...\n")
	} else if v.error != nil {
		s.WriteString(styles.ErrorStyle.Render("Error: "+v.error.Error()) + "\n")
	} else {
		s.WriteString(v.viewport.View())
		s.WriteString("\n")
	}

	statusBar := v.renderStatusBar()
	s.WriteString(statusBar)

	if v.showHelp {
		helpView := v.help.View(v.keys)
		s.WriteString("\n" + helpView)
	}

	return s.String()
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
	if v.width > 0 && len(title) > v.width-20 {
		title = title[:v.width-23] + "..."
	}

	header := styles.TitleStyle.MaxWidth(v.width).Render(title)

	if models.IsActiveStatus(string(v.run.Status)) {
		pollingIndicator := styles.ProcessingStyle.Render(" [Polling ⟳]")
		header += pollingIndicator
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

	// Show copied message if recent
	if v.copiedMessage != "" && time.Since(v.copiedMessageTime) < 2*time.Second {
		options = v.copiedMessage + " | " + options
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

func (v *RunDetailsView) updateStatusHistory(status string) {
	if len(v.statusHistory) == 0 || v.statusHistory[len(v.statusHistory)-1] != status {
		timestamp := time.Now().Format("15:04:05")
		statusIcon := styles.GetStatusIcon(status)
		entry := fmt.Sprintf("[%s] %s %s", timestamp, statusIcon, status)
		v.statusHistory = append(v.statusHistory, entry)
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

	v.pollTicker = time.NewTicker(5 * time.Second)
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

// copyCurrentLine copies the current visible line to clipboard
func (v *RunDetailsView) copyCurrentLine() error {
	// Get the current line based on viewport position
	// The viewport tracks the top visible line
	visibleLines := strings.Split(v.viewport.View(), "\n")
	if len(visibleLines) > 0 {
		// Get first visible line (could be partial due to scrolling)
		currentLine := visibleLines[0]
		if currentLine != "" {
			// Write returns a channel that signals when done
			done := clipboard.Write(clipboard.FmtText, []byte(currentLine))
			<-done // Wait for completion
			return nil
		}
	}

	return fmt.Errorf("no line to copy")
}

// copyAllContent copies all content to clipboard
func (v *RunDetailsView) copyAllContent() error {
	if v.fullContent == "" {
		return fmt.Errorf("no content to copy")
	}
	// Write returns a channel that signals when done
	done := clipboard.Write(clipboard.FmtText, []byte(v.fullContent))
	<-done // Wait for completion
	return nil
}

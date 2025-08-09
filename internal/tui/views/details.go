package views

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/styles"
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
}

func NewRunDetailsView(client *api.Client, run models.RunResponse) *RunDetailsView {
	return NewRunDetailsViewWithCache(client, run, nil, false, time.Time{}, nil)
}

func NewRunDetailsViewWithCache(client *api.Client, run models.RunResponse, parentRuns []models.RunResponse, parentCached bool, parentCachedAt time.Time, parentDetailsCache map[string]*models.RunResponse) *RunDetailsView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	vp := viewport.New(80, 20)

	// Check if we have preloaded data for this run
	needsLoading := true
	runID := run.GetIDString()

	// Debug logging to file (since we can't use stdout in TUI)
	debugInfo := fmt.Sprintf("DEBUG: NewRunDetailsViewWithCache - runID='%s', cacheSize=%d\n",
		runID, len(parentDetailsCache))

	if parentDetailsCache != nil {
		// List all cache keys for debugging
		cacheKeys := make([]string, 0, len(parentDetailsCache))
		for k := range parentDetailsCache {
			cacheKeys = append(cacheKeys, fmt.Sprintf("'%s'", k))
		}
		debugInfo += fmt.Sprintf("DEBUG: Cache keys: [%s]\n", strings.Join(cacheKeys, ", "))

		if cachedRun, exists := parentDetailsCache[runID]; exists && cachedRun != nil {
			debugInfo += fmt.Sprintf("DEBUG: Cache HIT for runID='%s'\n", runID)
			debugInfo += fmt.Sprintf("DEBUG: Cached run data - ID='%s', Title='%s', Repository='%s', Status='%s', Source='%s'\n", 
				cachedRun.GetIDString(), cachedRun.Title, cachedRun.Repository, cachedRun.Status, cachedRun.Source)
			run = *cachedRun
			needsLoading = false
		} else {
			debugInfo += fmt.Sprintf("DEBUG: Cache MISS for runID='%s' (exists=%v, cachedRun!=nil=%v)\n",
				runID, exists, cachedRun != nil)
		}
	} else {
		debugInfo += "DEBUG: parentDetailsCache is nil\n"
	}

	// Write debug info to a temporary file
	if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.WriteString(debugInfo)
		f.Close()
	}

	v := &RunDetailsView{
		client:             client,
		run:                run,
		keys:               components.DefaultKeyMap,
		help:               help.New(),
		viewport:           vp,
		spinner:            s,
		loading:            needsLoading,
		showLogs:           false,
		parentRuns:         parentRuns,
		parentCached:       parentCached,
		parentCachedAt:     parentCachedAt,
		parentDetailsCache: parentDetailsCache,
		statusHistory:      make([]string, 0),
		cacheRetryCount:    0,
		maxCacheRetries:    3,
	}

	debugInfo += fmt.Sprintf("DEBUG: Created view with loading=%v\n", needsLoading)
	if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.WriteString(debugInfo)
		f.Close()
	}

	// Initialize status history with current status if we have cached data
	if !needsLoading {
		v.updateStatusHistory(string(run.Status))
		v.updateContent()
	}

	return v
}

func (v *RunDetailsView) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Only load details if not already loaded from cache
	if v.loading {
		// Try cache one more time before making API call
		if v.parentDetailsCache != nil && v.cacheRetryCount < v.maxCacheRetries {
			runID := v.run.GetIDString()
			if cachedRun, exists := v.parentDetailsCache[runID]; exists && cachedRun != nil {
				// Cache hit on retry!
				debugInfo := fmt.Sprintf("DEBUG: Cache retry successful for runID='%s'\n", runID)
				if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					f.WriteString(debugInfo)
					f.Close()
				}

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

func (v *RunDetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.viewport.Width = msg.Width
		v.viewport.Height = msg.Height - 8
		v.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keys.Quit):
			v.stopPolling()
			return v, tea.Quit
		case key.Matches(msg, v.keys.Back):
			v.stopPolling()
			return NewRunListViewWithCache(v.client, v.parentRuns, v.parentCached, v.parentCachedAt, v.parentDetailsCache), nil
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
		case key.Matches(msg, v.keys.Up):
			v.viewport.LineUp(1)
		case key.Matches(msg, v.keys.Down):
			v.viewport.LineDown(1)
		case key.Matches(msg, v.keys.PageUp):
			v.viewport.HalfViewUp()
		case key.Matches(msg, v.keys.PageDown):
			v.viewport.HalfViewDown()
		case key.Matches(msg, v.keys.Home):
			v.viewport.GotoTop()
		case key.Matches(msg, v.keys.End):
			v.viewport.GotoBottom()
		}

	case runDetailsLoadedMsg:
		v.loading = false
		v.run = msg.run
		v.error = msg.err
		if msg.err == nil {
			v.updateStatusHistory(string(msg.run.Status))
		}
		v.updateContent()

		// Debug logging for successful load
		debugInfo := fmt.Sprintf("DEBUG: Successfully loaded run details for '%s'\n", msg.run.GetIDString())
		if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			f.WriteString(debugInfo)
			f.Close()
		}

	case pollTickMsg:
		if isActiveStatus(string(v.run.Status)) {
			cmds = append(cmds, v.loadRunDetails())
		} else {
			v.stopPolling()
		}

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

	if isActiveStatus(string(v.run.Status)) {
		pollingIndicator := styles.ProcessingStyle.Render(" [Polling ⟳]")
		header += pollingIndicator
	}

	rightAlign := lipgloss.NewStyle().Align(lipgloss.Right).Width(v.width - lipgloss.Width(header))
	header += rightAlign.Render("Status: " + status)

	return header
}

func (v *RunDetailsView) renderStatusBar() string {
	options := "[b]ack [l]ogs [r]efresh [?]help [q]uit"

	if v.showLogs {
		options = "[b]ack [l]details [r]efresh [?]help [q]uit"
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
		for _, status := range v.statusHistory {
			content.WriteString(status + "\n")
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

	v.viewport.SetContent(content.String())
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
			return runDetailsLoadedMsg{run: originalRun, err: fmt.Errorf("invalid run ID: empty string")}
		}

		runPtr, err := v.client.GetRun(originalRunID)
		if err != nil {
			return runDetailsLoadedMsg{run: originalRun, err: fmt.Errorf("API error for run %s: %w", originalRunID, err)}
		}

		if runPtr == nil {
			return runDetailsLoadedMsg{run: originalRun, err: fmt.Errorf("API returned nil for run %s", originalRunID)}
		}

		// Ensure the returned run has the correct ID
		updatedRun := *runPtr
		if updatedRun.GetIDString() == "" && originalRun.ID != nil {
			updatedRun.ID = originalRun.ID
		}

		return runDetailsLoadedMsg{run: updatedRun, err: nil}
	}
}

func (v *RunDetailsView) startPolling() tea.Cmd {
	if !isActiveStatus(string(v.run.Status)) {
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

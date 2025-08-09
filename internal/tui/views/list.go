package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
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

type RunListView struct {
	client       *api.Client
	runs         []models.RunResponse
	table        *components.Table
	keys         components.KeyMap
	help         help.Model
	width        int
	height       int
	loading      bool
	error        error
	spinner      spinner.Model
	pollTicker   *time.Ticker
	pollStop     chan bool
	showHelp     bool
	searchMode   bool
	searchQuery  string
	filteredRuns []models.RunResponse
	cached       bool
	cachedAt     time.Time
	// Preloaded run details cache
	detailsCache map[string]*models.RunResponse
	preloading   map[string]bool
	// User info for remaining runs counter
	userInfo *models.UserInfo
}

func NewRunListView(client *api.Client) *RunListView {
	// Try to get cached data from global cache
	runs, cached, cachedAt, detailsCache, selectedIndex := cache.GetCachedList()
	return NewRunListViewWithCache(client, runs, cached, cachedAt, detailsCache, selectedIndex)
}

func NewRunListViewWithCache(
	client *api.Client,
	runs []models.RunResponse,
	cached bool,
	cachedAt time.Time,
	detailsCache map[string]*models.RunResponse,
	selectedIndex int,
) *RunListView {
	// Enhanced debugging with more details
	debugInfo := fmt.Sprintf("DEBUG: Creating RunListViewWithCache - cached=%v, runs=%d, detailsCache=%d\n",
		cached, len(runs), len(detailsCache))
	if runs != nil && len(runs) > 0 {
		debugInfo += fmt.Sprintf("DEBUG: First run ID=%s, repo=%s\n", runs[0].GetIDString(), runs[0].Repository)
	}
	if detailsCache != nil && len(detailsCache) > 0 {
		var cacheKeys []string
		for k := range detailsCache {
			cacheKeys = append(cacheKeys, k)
			if len(cacheKeys) >= 3 { // Only show first 3 keys
				break
			}
		}
		debugInfo += fmt.Sprintf("DEBUG: Sample cache keys: %v\n", cacheKeys)
	}
	debug.LogToFile(debugInfo)
	columns := []components.Column{
		{Title: "ID", Width: 8},
		{Title: "Status", Width: 15},
		{Title: "Repository", Width: 25},
		{Title: "Time", Width: 12},
		{Title: "Branch", Width: 15},
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	// Only load if we have no cached data at all
	// User wants cache to always be used when available
	shouldLoad := !cached || runs == nil || len(runs) == 0

	// Use provided details cache or create new one
	if detailsCache == nil {
		detailsCache = make(map[string]*models.RunResponse)
	}

	v := &RunListView{
		client:       client,
		table:        components.NewTable(columns),
		keys:         components.DefaultKeyMap,
		help:         help.New(),
		spinner:      s,
		loading:      shouldLoad,
		runs:         runs,
		filteredRuns: runs,
		cached:       cached,
		cachedAt:     cachedAt,
		detailsCache: detailsCache,
		preloading:   make(map[string]bool),
	}

	// If we have cached data, update the table
	if !shouldLoad && runs != nil && len(runs) > 0 {
		v.updateTable()
		// Restore cursor position
		if selectedIndex >= 0 && selectedIndex < len(v.filteredRuns) {
			v.table.SetSelectedIndex(selectedIndex)
		}
	}

	return v
}

func (v *RunListView) Init() tea.Cmd {
	var cmds []tea.Cmd

	// If we have cached data, use it - don't auto-refresh
	if v.cached && v.runs != nil && len(v.runs) > 0 {
		// Don't show loading, data is already displayed
		v.loading = false
		cmds = append(cmds, v.startPolling())
	} else {
		// Need to load data
		v.loading = true
		cmds = append(cmds, v.loadRuns())
		cmds = append(cmds, v.spinner.Tick)
		cmds = append(cmds, v.startPolling())
	}

	// Always load user info
	cmds = append(cmds, v.loadUserInfo())

	return tea.Batch(cmds...)
}

func (v *RunListView) loadUserInfo() tea.Cmd {
	return func() tea.Msg {
		userInfo, err := v.client.VerifyAuth()
		return userInfoLoadedMsg{userInfo: userInfo, err: err}
	}
}

// handleWindowSizeMsg handles window resize events
func (v *RunListView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height
	v.table.SetDimensions(msg.Width, msg.Height-4)
	v.help.Width = msg.Width
}

// handleSearchMode handles search mode key input
func (v *RunListView) handleSearchMode(msg tea.KeyMsg) {
	switch msg.String() {
	case "enter":
		v.searchMode = false
		v.filterRuns()
	case "esc":
		v.searchMode = false
		v.searchQuery = ""
		v.filterRuns()
	case "backspace":
		if len(v.searchQuery) > 0 {
			v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
		}
	default:
		if len(msg.String()) == 1 {
			v.searchQuery += msg.String()
		}
	}
}

// handleKeyMsg handles normal mode key input
func (v *RunListView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if v.searchMode {
		v.handleSearchMode(msg)
		return v, nil
	}

	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, v.keys.Quit):
		v.stopPolling()
		return v, tea.Quit
	case key.Matches(msg, v.keys.Help):
		v.showHelp = !v.showHelp
	case key.Matches(msg, v.keys.Refresh):
		cmds = append(cmds, v.loadRuns())
	case key.Matches(msg, v.keys.Search):
		v.searchMode = true
		v.searchQuery = ""
	case key.Matches(msg, v.keys.Enter):
		return v.handleEnterKey()
	case key.Matches(msg, v.keys.New):
		return v.handleNewRunKey()
	case key.Matches(msg, v.keys.Up):
		v.table.MoveUp()
		cmds = append(cmds, v.preloadSelectedRun())
	case key.Matches(msg, v.keys.Down):
		v.table.MoveDown()
		cmds = append(cmds, v.preloadSelectedRun())
	case msg.String() == "shift+k" || msg.String() == "shift+up":
		v.table.PageUp()
	case msg.String() == "shift+j" || msg.String() == "shift+down":
		v.table.PageDown()
	case key.Matches(msg, v.keys.PageUp):
		v.table.PageUp()
	case key.Matches(msg, v.keys.PageDown):
		v.table.PageDown()
	case key.Matches(msg, v.keys.Home):
		v.table.GoToTop()
	case key.Matches(msg, v.keys.End):
		v.table.GoToBottom()
	}

	return v, tea.Batch(cmds...)
}

// handleEnterKey handles Enter key press to navigate to run details
func (v *RunListView) handleEnterKey() (tea.Model, tea.Cmd) {
	idx := v.table.GetSelectedIndex()
	if idx < 0 || idx >= len(v.filteredRuns) {
		return v, nil
	}

	run := v.filteredRuns[idx]
	runID := run.GetIDString()

	// Save cursor position to cache before navigating
	cache.SetSelectedIndex(idx)

	// Debug logging for Enter key press
	debugInfo := fmt.Sprintf("DEBUG: Enter pressed for run idx=%d, runID='%s', repo='%s'\n",
		idx, runID, run.Repository)
	debugInfo += fmt.Sprintf("DEBUG: Cache size=%d, runID in cache=%v, preloading=%v\n",
		len(v.detailsCache), v.detailsCache[runID] != nil, v.preloading[runID])
	debug.LogToFile(debugInfo)

	// Check if this run is currently being preloaded
	if v.preloading[runID] {
		debug.LogToFilef("DEBUG: Run %s is still preloading, adding small delay...\n", runID)
		return v, func() tea.Msg {
			time.Sleep(100 * time.Millisecond)
			return retryNavigationMsg{runIndex: idx}
		}
	}

	// Use preloaded details if available
	if detailed, ok := v.detailsCache[runID]; ok {
		debug.LogToFilef("DEBUG: Using cached data for runID='%s' - NAVIGATING TO DETAILS VIEW\n", runID)

		// Fix: Ensure the cached run has the correct ID
		cachedRun := *detailed
		if cachedRun.GetIDString() == "" && run.ID != "" {
			cachedRun.ID = run.ID
		}

		debug.LogToFilef("DEBUG: Fixed cached run ID from '%s' to '%s'\n", detailed.GetIDString(), cachedRun.GetIDString())
		return NewRunDetailsView(v.client, cachedRun), nil
	}

	debug.LogToFilef("DEBUG: No cached data for runID='%s', loading fresh - NAVIGATING TO DETAILS VIEW\n", runID)
	return NewRunDetailsView(v.client, run), nil
}

// handleNewRunKey handles the New key press to create a new run
func (v *RunListView) handleNewRunKey() (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: ListView creating NewCreateRunView - runs=%d, cached=%v, detailsCache=%d\n",
		len(v.runs), v.cached, len(v.detailsCache))
	return NewCreateRunViewWithCache(v.client, v.runs, v.cached, v.cachedAt, v.detailsCache), nil
}

// handleRunsLoaded handles the runsLoadedMsg message
func (v *RunListView) handleRunsLoaded(msg runsLoadedMsg) []tea.Cmd {
	var cmds []tea.Cmd

	debug.LogToFilef("DEBUG: ENTERED runsLoadedMsg case - %d runs loaded\n", len(msg.runs))

	v.loading = false
	v.runs = msg.runs
	v.error = msg.err
	v.cached = true
	v.cachedAt = time.Now()
	v.filterRuns()

	// Save to global cache
	if msg.err == nil && len(msg.runs) > 0 {
		cache.SetCachedList(msg.runs, v.detailsCache)
	}

	// Start preloading run details in background
	if msg.err == nil && len(msg.runs) > 0 {
		cmds = append(cmds, v.preloadRunDetails())
	}

	return cmds
}

// handleRunDetailsPreloaded handles the runDetailsPreloadedMsg message
func (v *RunListView) handleRunDetailsPreloaded(msg runDetailsPreloadedMsg) {
	debug.LogToFilef("DEBUG: ENTERED runDetailsPreloadedMsg case for runID='%s', err=%v, run!=nil=%v\n",
		msg.runID, msg.err, msg.run != nil)

	// Cache the loaded run details
	v.preloading[msg.runID] = false
	if msg.err == nil && msg.run != nil {
		v.detailsCache[msg.runID] = msg.run

		// Also save to global cache
		cache.AddCachedDetail(msg.runID, msg.run)

		// Debug logging
		debug.LogToFilef("DEBUG: Successfully cached run with key='%s', actualID='%s', title='%s', cacheSize=%d\n",
			msg.runID, msg.run.GetIDString(), msg.run.Title, len(v.detailsCache))
	} else {
		// Log errors too
		debug.LogToFilef("DEBUG: Failed to cache run with key='%s', err=%v\n", msg.runID, msg.err)
	}
}

// handleRetryNavigation handles the retryNavigationMsg message
func (v *RunListView) handleRetryNavigation(msg retryNavigationMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: ENTERED retryNavigationMsg case for runIndex=%d\n", msg.runIndex)

	// Retry navigation after a small delay
	idx := msg.runIndex
	if idx >= 0 && idx < len(v.filteredRuns) {
		run := v.filteredRuns[idx]
		runID := run.GetIDString()

		debug.LogToFilef("DEBUG: Retrying navigation for runID='%s', cache size=%d, in cache=%v\n",
			runID, len(v.detailsCache), v.detailsCache[runID] != nil)

		// Use cached data if available now
		if detailed, ok := v.detailsCache[runID]; ok {
			debug.LogToFilef("DEBUG: Retry successful - using cached data for runID='%s' - NAVIGATING TO DETAILS VIEW\n", runID)
			return NewRunDetailsViewWithCache(v.client, *detailed, v.runs, v.cached, v.cachedAt, v.detailsCache), nil
		}

		// Still not cached, load fresh
		debug.LogToFilef("DEBUG: Retry - still no cached data for runID='%s', loading fresh - NAVIGATING TO DETAILS VIEW\n", runID)
		return NewRunDetailsViewWithCache(v.client, run, v.runs, v.cached, v.cachedAt, v.detailsCache), nil
	}

	return v, nil
}

// handlePolling handles the pollTickMsg message
func (v *RunListView) handlePolling(msg pollTickMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if v.hasActiveRuns() {
		cmds = append(cmds, v.loadRuns())
	}
	return cmds
}

func (v *RunListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case runsLoadedMsg:
		cmds = append(cmds, v.handleRunsLoaded(msg)...)

	case runDetailsPreloadedMsg:
		v.handleRunDetailsPreloaded(msg)

	case userInfoLoadedMsg:
		if msg.err == nil && msg.userInfo != nil {
			v.userInfo = msg.userInfo
		}

	case retryNavigationMsg:
		return v.handleRetryNavigation(msg)

	case pollTickMsg:
		cmds = append(cmds, v.handlePolling(msg)...)

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *RunListView) View() string {
	var s strings.Builder

	// Truncate title if it's too wide for the terminal
	titleText := "RepoBird CLI - Runs"
	if v.width > 0 && v.width < len(titleText)+10 {
		titleText = "RepoBird"
	}
	title := styles.TitleStyle.MaxWidth(v.width).Render(titleText)
	s.WriteString(title)
	s.WriteString("\n\n")

	if v.loading {
		s.WriteString(v.spinner.View() + " Loading runs...\n")
	} else if v.error != nil {
		s.WriteString(styles.ErrorStyle.Render("Error: "+v.error.Error()) + "\n")
	} else {
		if v.searchMode {
			s.WriteString("Search: " + v.searchQuery + "_\n")
		} else if v.searchQuery != "" {
			s.WriteString("Filter: " + v.searchQuery + "\n")
		}

		s.WriteString(v.table.View())
		s.WriteString("\n")

		statusBar := v.renderStatusBar()
		s.WriteString(statusBar)
	}

	if v.showHelp {
		helpView := v.help.View(v.keys)
		s.WriteString("\n" + helpView)
	}

	return s.String()
}

func (v *RunListView) filterRuns() {
	if v.searchQuery == "" {
		v.filteredRuns = v.runs
	} else {
		v.filteredRuns = []models.RunResponse{}
		query := strings.ToLower(v.searchQuery)
		for _, run := range v.runs {
			if strings.Contains(strings.ToLower(run.GetIDString()), query) ||
				strings.Contains(strings.ToLower(run.Repository), query) ||
				strings.Contains(strings.ToLower(string(run.Status)), query) ||
				strings.Contains(strings.ToLower(run.Source), query) ||
				strings.Contains(strings.ToLower(run.Target), query) {
				v.filteredRuns = append(v.filteredRuns, run)
			}
		}
	}
	v.updateTable()
}

func (v *RunListView) updateTable() {
	rows := make([]components.Row, len(v.filteredRuns))
	for i, run := range v.filteredRuns {
		statusIcon := styles.GetStatusIcon(string(run.Status))
		statusText := fmt.Sprintf("%s %s", statusIcon, run.Status)
		timeAgo := formatTimeAgo(run.CreatedAt)
		branch := run.Source
		if run.Target != "" && run.Target != run.Source {
			branch = fmt.Sprintf("%s→%s", run.Source, run.Target)
		}

		idStr := run.GetIDString()
		if len(idStr) > 8 {
			idStr = idStr[:8]
		}
		rows[i] = components.Row{
			idStr,
			statusText,
			run.Repository,
			timeAgo,
			branch,
		}
	}
	v.table.SetRows(rows)
}

func (v *RunListView) renderStatusBar() string {
	left := fmt.Sprintf(" %d runs | %s", len(v.filteredRuns), v.table.StatusLine())

	activeCount := 0
	for _, run := range v.runs {
		if isActiveStatus(string(run.Status)) {
			activeCount++
		}
	}

	right := ""

	// Add remaining runs counter if user info is available
	if v.userInfo != nil {
		tier := v.userInfo.Tier
		if tier == "" {
			tier = "free"
		}
		right = fmt.Sprintf("%s: %d/%d runs ", tier, v.userInfo.RemainingRuns, v.userInfo.TotalRuns)
	}

	if activeCount > 0 {
		right += fmt.Sprintf("⟳ %d active ", activeCount)
	}

	right += "[n]ew [r]efresh [/]search [?]help [q]uit "

	padding := v.width - len(left) - len(right)
	if padding < 0 {
		padding = 0
	}

	return styles.StatusBarStyle.Width(v.width).Render(
		left + strings.Repeat(" ", padding) + right,
	)
}

func (v *RunListView) loadRuns() tea.Cmd {
	return func() tea.Msg {
		runPtrs, err := v.client.ListRuns(100, 0)
		if err != nil {
			return runsLoadedMsg{runs: nil, err: err}
		}

		runs := make([]models.RunResponse, len(runPtrs))
		for i, r := range runPtrs {
			runs[i] = *r
		}
		return runsLoadedMsg{runs: runs, err: nil}
	}
}

func (v *RunListView) startPolling() tea.Cmd {
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

func (v *RunListView) stopPolling() {
	if v.pollTicker != nil {
		v.pollTicker.Stop()
	}
	if v.pollStop != nil {
		close(v.pollStop)
	}
}

func (v *RunListView) hasActiveRuns() bool {
	for _, run := range v.runs {
		if isActiveStatus(string(run.Status)) {
			return true
		}
	}
	return false
}

func isActiveStatus(status string) bool {
	return models.IsActiveStatus(status)
}

func formatTimeAgo(t time.Time) string {
	return utils.FormatTimeAgo(t)
}

type runsLoadedMsg struct {
	runs []models.RunResponse
	err  error
}

type pollTickMsg struct{}

type runDetailsPreloadedMsg struct {
	runID string
	run   *models.RunResponse
	err   error
}

type userInfoLoadedMsg struct {
	userInfo *models.UserInfo
	err      error
}

type retryNavigationMsg struct {
	runIndex int
}

// selectRunsToPreload determines which runs should be preloaded
func (v *RunListView) selectRunsToPreload() []string {
	var toPreload []string

	// Start with the selected run
	if idx := v.table.GetSelectedIndex(); idx >= 0 && idx < len(v.filteredRuns) {
		run := v.filteredRuns[idx]
		runID := run.GetIDString()
		debug.LogToFilef("DEBUG: Selected run runID='%s', cached=%v, preloading=%v\n",
			runID, v.detailsCache[runID] != nil, v.preloading[runID])

		if _, cached := v.detailsCache[runID]; !cached && !v.preloading[runID] {
			toPreload = append(toPreload, runID)
		}
	}

	// Then add the first 10 runs
	maxPreload := 10
	for i := 0; i < len(v.runs) && len(toPreload) < maxPreload; i++ {
		runID := v.runs[i].GetIDString()
		if _, cached := v.detailsCache[runID]; !cached && !v.preloading[runID] {
			// Check if not already in toPreload
			found := false
			for _, id := range toPreload {
				if id == runID {
					found = true
					break
				}
			}
			if !found {
				toPreload = append(toPreload, runID)
			}
		}
	}

	return toPreload
}

// createSinglePreloadCmd creates a command to preload a single run
func (v *RunListView) createSinglePreloadCmd(runID string) tea.Cmd {
	v.preloading[runID] = true
	return func() tea.Msg {
		debug.LogToFilef("DEBUG: Starting API call for runID='%s'\n", runID)

		detailed, err := v.client.GetRun(runID)

		debug.LogToFilef("DEBUG: API call completed for runID='%s', err=%v, run!=nil=%v - SENDING runDetailsPreloadedMsg\n",
			runID, err, detailed != nil)

		msg := runDetailsPreloadedMsg{
			runID: runID,
			run:   detailed,
			err:   err,
		}

		debug.LogToFilef("DEBUG: About to return runDetailsPreloadedMsg for runID='%s'\n", runID)
		return msg
	}
}

// createPreloadCommands creates commands for preloading multiple runs
func (v *RunListView) createPreloadCommands(runIDs []string) []tea.Cmd {
	var cmds []tea.Cmd
	for _, runID := range runIDs {
		cmds = append(cmds, v.createSinglePreloadCmd(runID))
	}
	return cmds
}

func (v *RunListView) preloadRunDetails() tea.Cmd {
	debug.LogToFilef("DEBUG: preloadRunDetails called - runs=%d, filteredRuns=%d, cacheSize=%d\n",
		len(v.runs), len(v.filteredRuns), len(v.detailsCache))

	// Select runs to preload
	toPreload := v.selectRunsToPreload()

	debug.LogToFilef("DEBUG: Will preload %d runs: %v\n", len(toPreload), toPreload)

	// Create commands for preloading
	cmds := v.createPreloadCommands(toPreload)

	return tea.Batch(cmds...)
}

func (v *RunListView) preloadSelectedRun() tea.Cmd {
	if idx := v.table.GetSelectedIndex(); idx >= 0 && idx < len(v.filteredRuns) {
		run := v.filteredRuns[idx]
		runID := run.GetIDString()

		// Check if already cached or being loaded
		if _, cached := v.detailsCache[runID]; cached || v.preloading[runID] {
			return nil
		}

		v.preloading[runID] = true
		return func() tea.Msg {
			detailed, err := v.client.GetRun(runID)
			return runDetailsPreloadedMsg{
				runID: runID,
				run:   detailed,
				err:   err,
			}
		}
	}
	return nil
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

type RunListView struct {
	client      APIClient
	cache       *cache.SimpleCache
	table       *components.Table
	keys        components.KeyMap
	help        help.Model
	width       int
	height      int
	loading     bool
	error       error
	spinner     spinner.Model
	pollTicker  *time.Ticker
	pollStop    chan bool
	searchMode  bool
	searchQuery string
	// User info for remaining runs counter
	userInfo *models.UserInfo
	// Unified status line component
	statusLine *components.StatusLine
}

func NewRunListView(client APIClient) *RunListView {
	columns := []components.Column{
		{Title: "ID", Width: 8, MinWidth: 8, Flex: 0},           // Fixed width
		{Title: "Status", Width: 12, MinWidth: 12, Flex: 0},     // Fixed width
		{Title: "Repository", Width: 25, MinWidth: 20, Flex: 2}, // Flexible, gets 2x space
		{Title: "Time", Width: 10, MinWidth: 10, Flex: 0},       // Fixed width
		{Title: "Branch", Width: 15, MinWidth: 12, Flex: 1},     // Flexible, gets 1x space
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	return &RunListView{
		client:     client,
		cache:      cache.NewSimpleCache(),
		table:      components.NewTable(columns),
		keys:       components.DefaultKeyMap,
		help:       help.New(),
		spinner:    s,
		loading:    true,
		statusLine: components.NewStatusLine(),
	}
}

// NewRunListViewWithCacheAndDimensions creates a new list view with cache and dimensions
func NewRunListViewWithCacheAndDimensions(
	client APIClient,
	runs []models.RunResponse,
	cached bool,
	cachedAt time.Time,
	detailsCache map[string]*models.RunResponse,
	selectedIndex int,
	width int,
	height int,
) *RunListView {
	// Create new cache instance for this view
	cache := cache.NewSimpleCache()
	v := NewRunListViewWithCache(client, runs, cached, cachedAt, detailsCache, selectedIndex, cache)

	// Set dimensions immediately if provided
	if width > 0 && height > 0 {
		v.width = width
		v.height = height
		// Apply dimensions to table immediately
		v.handleWindowSizeMsg(tea.WindowSizeMsg{Width: width, Height: height})
	}

	return v
}

func NewRunListViewWithCache(
	client APIClient,
	runs []models.RunResponse,
	cached bool,
	cachedAt time.Time,
	detailsCache map[string]*models.RunResponse,
	selectedIndex int,
	embeddedCache *cache.SimpleCache,
) *RunListView {
	// Enhanced debugging with more details
	debugInfo := fmt.Sprintf("DEBUG: Creating RunListViewWithCache - cached=%v, runs=%d, detailsCache=%d\n",
		cached, len(runs), len(detailsCache))
	if len(runs) > 0 {
		debugInfo += fmt.Sprintf("DEBUG: First run ID=%s, repo=%s\n", runs[0].GetIDString(), runs[0].Repository)
	}
	if len(detailsCache) > 0 {
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
		{Title: "ID", Width: 8, MinWidth: 8, Flex: 0},           // Fixed width
		{Title: "Status", Width: 12, MinWidth: 12, Flex: 0},     // Fixed width
		{Title: "Repository", Width: 25, MinWidth: 20, Flex: 2}, // Flexible, gets 2x space
		{Title: "Time", Width: 10, MinWidth: 10, Flex: 0},       // Fixed width
		{Title: "Branch", Width: 15, MinWidth: 12, Flex: 1},     // Flexible, gets 1x space
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
		client:     client,
		table:      components.NewTable(columns),
		keys:       components.DefaultKeyMap,
		help:       help.New(),
		spinner:    s,
		loading:    shouldLoad,
		statusLine: components.NewStatusLine(),
		cache:      embeddedCache,
	}

	return v
}

func (v *RunListView) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Clear the screen to ensure proper rendering
	cmds = append(cmds, tea.ClearScreen)

	// Send a window size message with stored dimensions if we have them
	// This ensures the view knows its size immediately upon returning
	if v.width > 0 && v.height > 0 {
		debug.LogToFilef("DEBUG: ListView Init sending WindowSizeMsg with width=%d height=%d\n", v.width, v.height)
		// Immediately process the dimensions
		v.handleWindowSizeMsg(tea.WindowSizeMsg{Width: v.width, Height: v.height})
		// Also send the message for the update cycle
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: v.width, Height: v.height}
		})
	} else {
		debug.LogToFile("DEBUG: ListView Init has no dimensions stored\n")
	}

	// Check if we have cached data first
	cachedRuns := v.cache.GetRuns()
	if len(cachedRuns) > 0 {
		// Use cached data
		v.loading = false
		v.updateTableFromRuns(cachedRuns)
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
		if err == nil && userInfo != nil {
			// Set the current user for cache initialization
			services.SetCurrentUser(userInfo)
		}
		return userInfoLoadedMsg{userInfo: userInfo, err: err}
	}
}

// handleWindowSizeMsg handles window resize events
func (v *RunListView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height

	// Calculate height for the table
	// We need to account for:
	// - Title: 1 line
	// - Blank line after title: 1 line
	// - Search/filter (if active): 1 line
	// - Status bar: 1 line
	// - Help (if shown): ~4 lines
	nonTableHeight := 3 // Title (1) + blank after title (1) + status bar (1)

	if v.searchMode || v.searchQuery != "" {
		nonTableHeight++ // Search/filter line
	}

	// Help has been moved to docs view

	// Give the rest to the table
	tableHeight := msg.Height - nonTableHeight
	if tableHeight < 5 {
		tableHeight = 5 // Minimum for header + separator + a few rows
	}

	v.table.SetDimensions(msg.Width, tableHeight)
	v.help.Width = msg.Width

	// If this is the first time setting dimensions (width/height were 0),
	// reset scroll to ensure title and headers are visible
	if msg.Width > 0 && msg.Height > 0 {
		// Update the table again to ensure proper rendering
		v.filterRuns()
	}
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
	case msg.String() == "Q":
		// Capital Q to force quit from anywhere
		v.stopPolling()
		_ = v.cache.SaveToDisk()
		v.cache.Stop()
		return v, tea.Quit
	case key.Matches(msg, v.keys.Quit):
		// q goes back to dashboard
		v.stopPolling()
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
	case key.Matches(msg, v.keys.Help):
		// Return to dashboard and show docs
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
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
	case key.Matches(msg, v.keys.Down):
		v.table.MoveDown()
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
	filteredRuns := v.getFilteredRuns()
	if idx < 0 || idx >= len(filteredRuns) {
		return v, nil
	}

	run := filteredRuns[idx]
	runID := run.GetIDString()

	// Save cursor position to cache before navigating
	v.cache.SetSelectedIndex(idx)

	debug.LogToFilef("DEBUG: Enter pressed for run idx=%d, runID='%s', repo='%s' - NAVIGATING TO DETAILS VIEW\n",
		idx, runID, run.Repository)

	return v, func() tea.Msg {
		return messages.NavigateToDetailsMsg{
			RunID: run.GetIDString(),
		}
	}
}

// handleNewRunKey handles the New key press to create a new run
func (v *RunListView) handleNewRunKey() (tea.Model, tea.Cmd) {
	debug.LogToFile("DEBUG: ListView navigating to create view\n")

	// Return navigation message instead of creating view directly
	return v, func() tea.Msg {
		return messages.NavigateToCreateMsg{}
	}
}

// handleRunsLoaded handles the runsLoadedMsg message
func (v *RunListView) handleRunsLoaded(msg runsLoadedMsg) {
	debug.LogToFilef("DEBUG: ENTERED runsLoadedMsg case - %d runs loaded\n", len(msg.runs))

	v.loading = false
	v.error = msg.err

	// Save to cache
	if msg.err == nil && len(msg.runs) > 0 {
		v.cache.SetRuns(msg.runs)
		v.filterRuns()
	}
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
		v.handleRunsLoaded(msg)

	case userInfoLoadedMsg:
		if msg.err == nil && msg.userInfo != nil {
			v.userInfo = msg.userInfo
		}

	case pollTickMsg:
		cmds = append(cmds, v.handlePolling(msg)...)

	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			// Also update the status line spinner
			v.statusLine.UpdateSpinner()
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *RunListView) View() string {
	if v.height == 0 || v.width == 0 {
		// Terminal dimensions not yet known
		return ""
	}

	// Pre-allocate array for exactly terminal height lines
	lines := make([]string, v.height)
	lineIdx := 0

	// Title
	titleText := "RepoBird CLI - Runs"
	title := styles.TitleStyle.Width(v.width).Render(titleText)
	lines[lineIdx] = title
	lineIdx++

	// Blank line after title
	lines[lineIdx] = ""
	lineIdx++

	// Handle loading/error states or normal view
	if v.loading {
		lines[lineIdx] = v.spinner.View() + " Loading runs..."
		lineIdx++
	} else if v.error != nil {
		lines[lineIdx] = styles.ErrorStyle.Render("Error: " + v.error.Error())
		lineIdx++
	} else {
		// Search/filter line (if active)
		if v.searchMode {
			lines[lineIdx] = "Search: " + v.searchQuery + "_"
			lineIdx++
		} else if v.searchQuery != "" {
			lines[lineIdx] = "Filter: " + v.searchQuery
			lineIdx++
		}

		// Table view
		tableContent := v.table.View()
		if tableContent != "" {
			// Split table content into lines
			tableLines := strings.Split(strings.TrimRight(tableContent, "\n"), "\n")
			for _, line := range tableLines {
				if lineIdx < v.height-1 { // Leave room for status bar
					lines[lineIdx] = line
					lineIdx++
				}
			}
		}
	}

	// Help has been moved to docs view

	// Status bar always goes in the last line
	lines[v.height-1] = v.renderStatusBar()

	// Join all lines with newlines
	// This creates exactly height-1 newlines, which is correct
	return strings.Join(lines, "\n")
}

// Helper methods for cache access
func (v *RunListView) getRuns() []models.RunResponse {
	return v.cache.GetRuns()
}

func (v *RunListView) getFilteredRuns() []models.RunResponse {
	runs := v.getRuns()
	if v.searchQuery == "" {
		return runs
	}

	var filtered []models.RunResponse
	query := strings.ToLower(v.searchQuery)
	for _, run := range runs {
		if strings.Contains(strings.ToLower(run.GetIDString()), query) ||
			strings.Contains(strings.ToLower(run.Repository), query) ||
			strings.Contains(strings.ToLower(string(run.Status)), query) ||
			strings.Contains(strings.ToLower(run.Source), query) ||
			strings.Contains(strings.ToLower(run.Target), query) {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

func (v *RunListView) updateTableFromRuns(runs []models.RunResponse) {
	rows := make([]components.Row, len(runs))
	for i, run := range runs {
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

func (v *RunListView) filterRuns() {
	filteredRuns := v.getFilteredRuns()
	v.updateTableFromRuns(filteredRuns)
}

func (v *RunListView) renderStatusBar() string {
	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter("LIST", v.width)

	// Determine if we're loading
	isLoadingData := v.loading

	// Build data info string
	dataInfo := ""
	if !isLoadingData {
		filteredRuns := v.getFilteredRuns()
		dataInfo = fmt.Sprintf("%d runs | %s", len(filteredRuns), v.table.StatusLine())

		activeCount := 0
		for _, run := range v.getRuns() {
			if isActiveStatus(string(run.Status)) {
				activeCount++
			}
		}

		// Add remaining runs counter if user info is available
		if v.userInfo != nil {
			tier := v.userInfo.Tier
			if tier == "" {
				tier = "free"
			}

			// Show tier-specific runs with hardcoded totals
			if v.userInfo.TierDetails != nil {
				// Hardcoded tier totals
				// Check if tier contains "free" or "Free" (handles "Free Plan v1", etc.)
				var totalProRuns, totalPlanRuns int
				tierLower := strings.ToLower(tier)
				if strings.Contains(tierLower, "free") {
					// Free tier
					totalProRuns = 3
					totalPlanRuns = 5
				} else if strings.Contains(tierLower, "pro") {
					// Pro tier
					totalProRuns = 30
					totalPlanRuns = 35
				} else {
					// Default to pro tier totals for unknown tiers
					totalProRuns = 30
					totalPlanRuns = 35
				}

				// Handle admin credits that exceed defaults
				actualProTotal := totalProRuns
				actualPlanTotal := totalPlanRuns
				if v.userInfo.TierDetails.RemainingProRuns > totalProRuns {
					actualProTotal = v.userInfo.TierDetails.RemainingProRuns
				}
				if v.userInfo.TierDetails.RemainingPlanRuns > totalPlanRuns {
					actualPlanTotal = v.userInfo.TierDetails.RemainingPlanRuns
				}

				dataInfo += fmt.Sprintf(" | %s: %d/%d pro, %d/%d plan", tier,
					v.userInfo.TierDetails.RemainingProRuns, actualProTotal,
					v.userInfo.TierDetails.RemainingPlanRuns, actualPlanTotal)
			} else {
				// Fallback to legacy display
				dataInfo += fmt.Sprintf(" | %s: %d/%d runs", tier, v.userInfo.RemainingRuns, v.userInfo.TotalRuns)
			}
		}

		if activeCount > 0 {
			dataInfo += fmt.Sprintf(" | ⟳ %d active", activeCount)
		}
	}

	// Help text
	helpText := "[n]ew [r]efresh [/]search [?]help [h]back [q]dashboard [Q]uit"

	// Format left content consistently
	leftContent := formatter.FormatViewName()

	// Create status line using formatter
	statusLine := formatter.StandardStatusLine(leftContent, dataInfo, helpText)
	return statusLine.
		SetLoading(isLoadingData).
		Render()
}

func (v *RunListView) loadRuns() tea.Cmd {
	return func() tea.Msg {
		// Create context with 10-second timeout for list view
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Use the context-aware ListRuns method
		listResp, err := v.client.ListRuns(ctx, 1, 1000) // page 1, limit 1000
		if err != nil {
			return runsLoadedMsg{runs: nil, err: err}
		}

		// Convert pointer slice to value slice
		var runs []models.RunResponse
		if listResp != nil && listResp.Data != nil {
			runs = make([]models.RunResponse, len(listResp.Data))
			for i, r := range listResp.Data {
				if r != nil {
					runs[i] = *r
				}
			}
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
	for _, run := range v.getRuns() {
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

type userInfoLoadedMsg struct {
	userInfo *models.UserInfo
	err      error
}

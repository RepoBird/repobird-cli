package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
)

// DashboardView is the main dashboard controller that manages different layout views
type DashboardView struct {
	client *api.Client
	keys   components.KeyMap
	help   help.Model

	// Dashboard state
	currentLayout   models.LayoutType
	showHelp        bool
	selectedRepo    *models.Repository
	selectedRepoIdx int
	selectedRunIdx  int
	focusedColumn   int // 0: repositories, 1: runs, 2: details

	// Layout views (simplified for now)
	runListView *RunListView

	// Dimensions
	width  int
	height int

	// Loading and error state
	loading      bool
	error        error
	initializing bool

	// Real data
	repositories    []models.Repository
	allRuns         []*models.RunResponse
	filteredRuns    []*models.RunResponse
	selectedRunData *models.RunResponse

	// Cache management
	lastDataRefresh time.Time
	refreshInterval time.Duration
}

type dashboardDataLoadedMsg struct {
	repositories []models.Repository
	allRuns      []*models.RunResponse
	error        error
}

type dashboardRepositorySelectedMsg struct {
	repository *models.Repository
	runs       []*models.RunResponse
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(client *api.Client) *DashboardView {
	dashboard := &DashboardView{
		client:          client,
		keys:            components.DefaultKeyMap,
		help:            help.New(),
		currentLayout:   models.LayoutTripleColumn,
		loading:         true,
		initializing:    true,
		refreshInterval: 30 * time.Second,
	}

	// Initialize with existing list view
	dashboard.runListView = NewRunListView(client)

	return dashboard
}

// Init implements the tea.Model interface
func (d *DashboardView) Init() tea.Cmd {
	return tea.Batch(
		d.loadDashboardData(),
		d.runListView.Init(),
	)
}

// loadDashboardData loads data from cache or API
func (d *DashboardView) loadDashboardData() tea.Cmd {
	return func() tea.Msg {
		// Try to get cached repository overview first
		repositories, cached, err := cache.GetRepositoryOverview()
		if err == nil && cached && len(repositories) > 0 {
			// We have cached repo data, now get run data
			allRuns := make([]*models.RunResponse, 0)
			cachedRepoData, err := cache.GetAllCachedRepositoryData()
			if err == nil {
				for _, repoRuns := range cachedRepoData {
					allRuns = append(allRuns, repoRuns...)
				}
			}

			if len(allRuns) > 0 {
				return dashboardDataLoadedMsg{
					repositories: repositories,
					allRuns:      allRuns,
					error:        nil,
				}
			}
		}

		// No cache or cache is empty, fetch from API
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// First, try to get repositories from API
		apiRepositories, err := d.client.ListRepositories(ctx)
		if err != nil {
			// Fall back to building repos from runs if repository API fails
			return d.loadFromRunsOnly()
		}

		// Convert API repositories to dashboard models
		repositories = make([]models.Repository, len(apiRepositories))
		for i, apiRepo := range apiRepositories {
			// Construct full repository name
			repoName := apiRepo.Name
			if repoName == "" {
				repoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
			}

			repositories[i] = models.Repository{
				Name:        repoName,
				Description: "",                // API doesn't provide description
				RunCounts:   models.RunStats{}, // Will be populated below
			}
		}

		// Get runs to populate repository statistics
		runs, cached, _, detailsCache, _ := cache.GetCachedList()
		if !cached || len(runs) == 0 {
			// Fetch runs from API
			runsResp, err := d.client.ListRunsLegacy(100, 0)
			if err != nil {
				// Still return repos even if runs fail
				_ = cache.SetRepositoryOverview(repositories)
				return dashboardDataLoadedMsg{
					repositories: repositories,
					allRuns:      []*models.RunResponse{},
					error:        nil,
				}
			}

			// Convert to pointer slice
			allRuns := make([]*models.RunResponse, len(runsResp))
			copy(allRuns, runsResp)

			// Update repository statistics from runs
			repositories = d.updateRepositoryStats(repositories, allRuns)

			// Cache the data
			_ = cache.SetRepositoryOverview(repositories)

			// Cache runs by repository
			for _, repo := range repositories {
				repoRuns := cache.FilterRunsByRepository(allRuns, repo.Name)
				repoDetails := make(map[string]*models.RunResponse)

				// Add any cached details
				for _, run := range repoRuns {
					if detail, exists := detailsCache[run.GetIDString()]; exists {
						repoDetails[run.GetIDString()] = detail
					}
				}

				_ = cache.SetRepositoryData(repo.Name, repoRuns, repoDetails)
			}

			return dashboardDataLoadedMsg{
				repositories: repositories,
				allRuns:      allRuns,
				error:        nil,
			}
		}

		// Use cached run data
		allRuns := make([]*models.RunResponse, len(runs))
		for i, run := range runs {
			allRuns[i] = &run
		}

		// Update repository statistics from cached runs
		repositories = d.updateRepositoryStats(repositories, allRuns)
		_ = cache.SetRepositoryOverview(repositories)

		return dashboardDataLoadedMsg{
			repositories: repositories,
			allRuns:      allRuns,
			error:        nil,
		}
	}
}

// loadFromRunsOnly loads dashboard data using only runs (fallback method)
func (d *DashboardView) loadFromRunsOnly() tea.Msg {
	runs, cached, _, detailsCache, _ := cache.GetCachedList()
	if !cached || len(runs) == 0 {
		// Fetch from API
		runsResp, err := d.client.ListRunsLegacy(100, 0)
		if err != nil {
			return dashboardDataLoadedMsg{error: err}
		}

		// Convert to pointer slice
		allRuns := make([]*models.RunResponse, len(runsResp))
		copy(allRuns, runsResp)

		// Build repository overview from runs
		repositories := cache.BuildRepositoryOverviewFromRuns(allRuns)

		// Cache the data
		_ = cache.SetRepositoryOverview(repositories)

		// Cache runs by repository
		for _, repo := range repositories {
			repoRuns := cache.FilterRunsByRepository(allRuns, repo.Name)
			repoDetails := make(map[string]*models.RunResponse)

			// Add any cached details
			for _, run := range repoRuns {
				if detail, exists := detailsCache[run.GetIDString()]; exists {
					repoDetails[run.GetIDString()] = detail
				}
			}

			_ = cache.SetRepositoryData(repo.Name, repoRuns, repoDetails)
		}

		return dashboardDataLoadedMsg{
			repositories: repositories,
			allRuns:      allRuns,
			error:        nil,
		}
	}

	// Use cached run data
	allRuns := make([]*models.RunResponse, len(runs))
	for i, run := range runs {
		allRuns[i] = &run
	}

	// Build repository overview from cached runs
	repositories := cache.BuildRepositoryOverviewFromRuns(allRuns)
	_ = cache.SetRepositoryOverview(repositories)

	return dashboardDataLoadedMsg{
		repositories: repositories,
		allRuns:      allRuns,
		error:        nil,
	}
}

// updateRepositoryStats updates repository statistics from runs
func (d *DashboardView) updateRepositoryStats(repositories []models.Repository, allRuns []*models.RunResponse) []models.Repository {
	// Create a map for quick lookup
	repoMap := make(map[string]*models.Repository)
	for i := range repositories {
		repoMap[repositories[i].Name] = &repositories[i]
	}

	// Update statistics from runs
	for _, run := range allRuns {
		repoName := run.GetRepositoryName()
		if repoName == "" {
			continue
		}

		repo, exists := repoMap[repoName]
		if !exists {
			continue
		}

		// Update last activity if this run is more recent
		if run.UpdatedAt.After(repo.LastActivity) {
			repo.LastActivity = run.UpdatedAt
		}

		// Update run counts
		repo.RunCounts.Total++
		switch run.Status {
		case models.StatusQueued, models.StatusInitializing, models.StatusProcessing, models.StatusPostProcess:
			repo.RunCounts.Running++
		case models.StatusDone:
			repo.RunCounts.Completed++
		case models.StatusFailed:
			repo.RunCounts.Failed++
		}
	}

	return repositories
}

// selectRepository loads data for a specific repository
func (d *DashboardView) selectRepository(repo *models.Repository) tea.Cmd {
	if repo == nil {
		return nil
	}

	return func() tea.Msg {
		// Try cache first
		runs, _, cached, err := cache.GetRepositoryData(repo.Name)
		if err == nil && cached && len(runs) > 0 {
			return dashboardRepositorySelectedMsg{
				repository: repo,
				runs:       runs,
			}
		}

		// Filter from all runs
		runs = cache.FilterRunsByRepository(d.allRuns, repo.Name)

		// Cache the filtered data
		_ = cache.SetRepositoryData(repo.Name, runs, make(map[string]*models.RunResponse))

		return dashboardRepositorySelectedMsg{
			repository: repo,
			runs:       runs,
		}
	}
}

// Update implements the tea.Model interface
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

		// Update child view dimensions
		if d.runListView != nil {
			_, childCmd := d.runListView.Update(msg)
			if childCmd != nil {
				cmds = append(cmds, childCmd)
			}
		}

	case dashboardDataLoadedMsg:
		d.loading = false
		d.initializing = false
		if msg.error != nil {
			d.error = msg.error
		} else {
			d.repositories = msg.repositories
			d.allRuns = msg.allRuns
			d.lastDataRefresh = time.Now()

			// Select first repository by default
			if len(d.repositories) > 0 {
				d.selectedRepo = &d.repositories[0]
				d.selectedRepoIdx = 0
				cmds = append(cmds, d.selectRepository(d.selectedRepo))
			}
		}

	case dashboardRepositorySelectedMsg:
		d.selectedRepo = msg.repository
		d.filteredRuns = msg.runs

		// Select first run by default
		if len(d.filteredRuns) > 0 {
			d.selectedRunData = d.filteredRuns[0]
			d.selectedRunIdx = 0
		}

	case tea.KeyMsg:
		// Handle dashboard-specific keys first
		switch {
		case key.Matches(msg, d.keys.LayoutSwitch):
			d.cycleLayout()
			return d, nil
		case key.Matches(msg, d.keys.LayoutTriple):
			d.currentLayout = models.LayoutTripleColumn
			return d, nil
		case key.Matches(msg, d.keys.LayoutAllRuns):
			d.currentLayout = models.LayoutAllRuns
			return d, nil
		case key.Matches(msg, d.keys.LayoutRepos):
			d.currentLayout = models.LayoutRepositoriesOnly
			return d, nil
		case key.Matches(msg, d.keys.Help):
			d.showHelp = !d.showHelp
			return d, nil
		case key.Matches(msg, d.keys.Quit):
			return d, tea.Quit
		case key.Matches(msg, d.keys.Refresh):
			d.loading = true
			cmds = append(cmds, d.loadDashboardData())
			return d, tea.Batch(cmds...)
		default:
			// Handle navigation in Miller Columns layout
			switch d.currentLayout {
			case models.LayoutTripleColumn:
				cmd := d.handleMillerColumnsNavigation(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case models.LayoutAllRuns:
				// Delegate to run list view
				model, childCmd := d.runListView.Update(msg)
				d.runListView = model.(*RunListView)
				if childCmd != nil {
					cmds = append(cmds, childCmd)
				}
			}
		}
	default:
		// Delegate other messages to child views if needed
		if d.currentLayout == models.LayoutAllRuns && d.runListView != nil {
			model, childCmd := d.runListView.Update(msg)
			d.runListView = model.(*RunListView)
			if childCmd != nil {
				cmds = append(cmds, childCmd)
			}
		}
	}

	return d, tea.Batch(cmds...)
}

// handleMillerColumnsNavigation handles navigation in the Miller Columns layout
func (d *DashboardView) handleMillerColumnsNavigation(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, d.keys.Up) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "k"):
		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx > 0 {
				d.selectedRepoIdx--
				d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				return d.selectRepository(d.selectedRepo)
			}
		case 1: // Runs column
			if d.selectedRunIdx > 0 {
				d.selectedRunIdx--
				if len(d.filteredRuns) > d.selectedRunIdx {
					d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
				}
			}
		}

	case key.Matches(msg, d.keys.Down) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "j"):
		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx < len(d.repositories)-1 {
				d.selectedRepoIdx++
				d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				return d.selectRepository(d.selectedRepo)
			}
		case 1: // Runs column
			if d.selectedRunIdx < len(d.filteredRuns)-1 {
				d.selectedRunIdx++
				if len(d.filteredRuns) > d.selectedRunIdx {
					d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
				}
			}
		}

	case key.Matches(msg, d.keys.Right) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "l"):
		// Move focus to the right
		if d.focusedColumn < 2 {
			d.focusedColumn++
			// If moving to runs column and no run selected, select first
			if d.focusedColumn == 1 && len(d.filteredRuns) > 0 && d.selectedRunData == nil {
				d.selectedRunIdx = 0
				d.selectedRunData = d.filteredRuns[0]
			}
		}

	case key.Matches(msg, d.keys.Left) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "h"):
		// Move focus to the left
		if d.focusedColumn > 0 {
			d.focusedColumn--
		}
	}
	return nil
}

// cycleLayout cycles through available layouts
func (d *DashboardView) cycleLayout() {
	switch d.currentLayout {
	case models.LayoutTripleColumn:
		d.currentLayout = models.LayoutAllRuns
	case models.LayoutAllRuns:
		d.currentLayout = models.LayoutRepositoriesOnly
	case models.LayoutRepositoriesOnly:
		d.currentLayout = models.LayoutTripleColumn
	default:
		d.currentLayout = models.LayoutTripleColumn
	}
}

// View implements the tea.Model interface
func (d *DashboardView) View() string {
	if d.width <= 0 || d.height <= 0 {
		return "Initializing dashboard..."
	}

	var content string

	if d.error != nil {
		content = fmt.Sprintf("Error loading dashboard data: %s\n\nPress 'r' to retry, 'q' to quit", d.error.Error())
		return content
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
		return content
	}

	if d.loading || d.initializing {
		content = "Loading dashboard data..."
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(d.width).
			Align(lipgloss.Center).
			MarginTop(d.height / 2)
		content = loadingStyle.Render(content)
		return content
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

	return content
}

// renderTripleColumnLayout renders the Miller Columns layout with real data
func (d *DashboardView) renderTripleColumnLayout() string {
	// Calculate available height (minus statusline only)
	availableHeight := d.height - 1 // Account for statusline at bottom

	// Create three columns
	leftColumn := d.renderRepositoriesColumn()
	centerColumn := d.renderRunsColumn()
	rightColumn := d.renderDetailsColumn()

	// Column widths - full width, equally distributed
	leftWidth := d.width / 3
	centerWidth := d.width / 3
	rightWidth := d.width - leftWidth - centerWidth

	// Make columns full height with rounded borders
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth - 1).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	centerStyle := lipgloss.NewStyle().
		Width(centerWidth - 1).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("33"))

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth - 1).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftColumn),
		centerStyle.Render(centerColumn),
		rightStyle.Render(rightColumn),
	)

	// Create statusline
	statusline := d.renderStatusLine("Miller Columns")

	return lipgloss.JoinVertical(lipgloss.Left, columns, statusline)
}

// renderAllRunsLayout renders the timeline layout
func (d *DashboardView) renderAllRunsLayout() string {
	// Use the existing run list view
	runListContent := d.runListView.View()

	// Create statusline
	statusline := d.renderStatusLine("All Runs Timeline")

	return lipgloss.JoinVertical(lipgloss.Left, runListContent, statusline)
}

// renderRepositoriesLayout renders the repositories-only layout
func (d *DashboardView) renderRepositoriesLayout() string {
	// Render repositories table
	content := d.renderRepositoriesTable()

	// Create statusline
	statusline := d.renderStatusLine("Repositories Overview")

	return lipgloss.JoinVertical(lipgloss.Left, content, statusline)
}

// renderLayoutInfo renders the layout information bar
func (d *DashboardView) renderLayoutInfo(layoutName string) string {
	layoutStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Width(d.width).
		Align(lipgloss.Center)

	// Show data freshness
	dataInfo := ""
	if !d.lastDataRefresh.IsZero() {
		elapsed := time.Since(d.lastDataRefresh)
		if elapsed < time.Minute {
			dataInfo = " (data: fresh)"
		} else {
			dataInfo = fmt.Sprintf(" (data: %dm ago)", int(elapsed.Minutes()))
		}
	}

	return layoutStyle.Render(fmt.Sprintf("Layout: %s%s", layoutName, dataInfo))
}

// renderHeader renders the dashboard header
func (d *DashboardView) renderHeader(layoutName string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Width(d.width).
		Align(lipgloss.Center)

	layoutStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Width(d.width).
		Align(lipgloss.Center)

	title := titleStyle.Render("RepoBird Dashboard")

	// Show data freshness
	dataInfo := ""
	if !d.lastDataRefresh.IsZero() {
		elapsed := time.Since(d.lastDataRefresh)
		if elapsed < time.Minute {
			dataInfo = " (data: fresh)"
		} else {
			dataInfo = fmt.Sprintf(" (data: %dm ago)", int(elapsed.Minutes()))
		}
	}

	layout := layoutStyle.Render(fmt.Sprintf("Layout: %s%s", layoutName, dataInfo))

	return lipgloss.JoinVertical(lipgloss.Left, title, layout)
}

// renderFooter renders the dashboard footer with help
func (d *DashboardView) renderFooter() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(d.width).
		Align(lipgloss.Center)

	if d.showHelp {
		help := []string{
			"Navigation:",
			"  Shift+L: Switch Layout  |  Ctrl+1: Miller Columns  |  Ctrl+2: All Runs  |  Ctrl+3: Repositories",
			"  j/k: Move Up/Down  |  h/l: Move Left/Right  |  r: Refresh  |  ?: Toggle Help  |  q: Quit",
		}
		return helpStyle.Render(strings.Join(help, "\n"))
	}

	shortHelp := "? help  |  Shift+L switch  |  r refresh  |  q quit"
	return helpStyle.Render(shortHelp)
}

// renderRepositoriesColumn renders the left column with real repositories
func (d *DashboardView) renderRepositoriesColumn() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	if d.focusedColumn == 0 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("63"))
	}
	title := titleStyle.Render("Repositories")

	var items []string
	for i, repo := range d.repositories {
		statusIcon := d.getRepositoryStatusIcon(&repo)
		item := fmt.Sprintf("%s %s", statusIcon, repo.Name)

		// Highlight selected repository
		if i == d.selectedRepoIdx {
			if d.focusedColumn == 0 {
				item = lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("255")).Render(item)
			} else {
				item = lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255")).Render(item)
			}
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		items = []string{"No repositories"}
	}

	content := strings.Join(items, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// renderRunsColumn renders the center column with runs for selected repository
func (d *DashboardView) renderRunsColumn() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	if d.focusedColumn == 1 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("33"))
	}
	title := titleStyle.Render("Runs")

	if d.selectedRepo == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "Select a repository")
	}

	var items []string
	for i, run := range d.filteredRuns {
		statusIcon := d.getRunStatusIcon(run.Status)
		displayTitle := run.Title
		if displayTitle == "" {
			displayTitle = "Untitled Run"
		}
		if len(displayTitle) > 30 {
			displayTitle = displayTitle[:27] + "..."
		}

		item := fmt.Sprintf("%s %s", statusIcon, displayTitle)

		// Highlight selected run
		if i == d.selectedRunIdx {
			if d.focusedColumn == 1 {
				item = lipgloss.NewStyle().Background(lipgloss.Color("33")).Foreground(lipgloss.Color("255")).Render(item)
			} else {
				item = lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255")).Render(item)
			}
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		items = []string{fmt.Sprintf("No runs for %s", d.selectedRepo.Name)}
	}

	content := strings.Join(items, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// renderDetailsColumn renders the right column with run details
func (d *DashboardView) renderDetailsColumn() string {
	title := lipgloss.NewStyle().Bold(true).Render("Run Details")

	if d.selectedRunData == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "Select a run")
	}

	run := d.selectedRunData
	details := []string{
		fmt.Sprintf("ID: %s", run.GetIDString()),
		fmt.Sprintf("Status: %s", run.Status),
		fmt.Sprintf("Repository: %s", run.Repository),
	}

	if run.Source != "" && run.Target != "" {
		details = append(details, fmt.Sprintf("Branch: %s → %s", run.Source, run.Target))
	}

	details = append(details, fmt.Sprintf("Created: %s", run.CreatedAt.Format("Jan 2 15:04")))
	details = append(details, fmt.Sprintf("Updated: %s", run.UpdatedAt.Format("Jan 2 15:04")))

	if run.Title != "" {
		details = append(details, "", "Title:", run.Title)
	}

	if run.Prompt != "" {
		details = append(details, "", "Prompt:")
		// Wrap prompt text
		wrapped := d.wrapText(run.Prompt, 30)
		details = append(details, wrapped...)
	}

	if run.Error != "" {
		details = append(details, "", "Error:")
		wrapped := d.wrapText(run.Error, 30)
		details = append(details, wrapped...)
	}

	content := strings.Join(details, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// renderRepositoriesTable renders a table of repositories with real data
func (d *DashboardView) renderRepositoriesTable() string {
	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	header := fmt.Sprintf("%-25s %-8s %-8s %-10s %-8s %-15s",
		"Repository", "Total", "Running", "Completed", "Failed", "Last Activity")

	var rows []string
	rows = append(rows, headerStyle.Render(header))
	rows = append(rows, strings.Repeat("-", d.width-4))

	for _, repo := range d.repositories {
		statusIcon := d.getRepositoryStatusIcon(&repo)
		repoName := fmt.Sprintf("%s %s", statusIcon, repo.Name)
		lastActivity := d.formatTimeAgo(repo.LastActivity)

		row := fmt.Sprintf("%-25s %-8d %-8d %-10d %-8d %-15s",
			repoName,
			repo.RunCounts.Total,
			repo.RunCounts.Running,
			repo.RunCounts.Completed,
			repo.RunCounts.Failed,
			lastActivity)

		rows = append(rows, row)
	}

	if len(d.repositories) == 0 {
		rows = append(rows, "No repositories found")
	}

	return strings.Join(rows, "\n")
}

// getRepositoryStatusIcon returns an icon based on repository status
func (d *DashboardView) getRepositoryStatusIcon(repo *models.Repository) string {
	if repo.RunCounts.Running > 0 {
		return "🔄"
	} else if repo.RunCounts.Failed > 0 {
		return "❌"
	} else if repo.RunCounts.Completed > 0 {
		return "✅"
	}
	return "⚪"
}

// getRunStatusIcon returns an icon based on run status
func (d *DashboardView) getRunStatusIcon(status models.RunStatus) string {
	switch status {
	case models.StatusQueued:
		return "⏳"
	case models.StatusInitializing:
		return "🔄"
	case models.StatusProcessing:
		return "⚙️"
	case models.StatusPostProcess:
		return "📝"
	case models.StatusDone:
		return "✅"
	case models.StatusFailed:
		return "❌"
	default:
		return "❓"
	}
}

// formatTimeAgo formats time in a human-readable way
func (d *DashboardView) formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}

// wrapText wraps text to fit within specified width
func (d *DashboardView) wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		if len(currentLine) == 0 {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// renderStatusLine renders the universal status line
func (d *DashboardView) renderStatusLine(layoutName string) string {
	// Data freshness indicator
	dataInfo := "fresh"
	if !d.lastDataRefresh.IsZero() {
		elapsed := time.Since(d.lastDataRefresh)
		if elapsed >= time.Minute {
			dataInfo = fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
		}
	}
	if d.loading && len(d.repositories) > 0 {
		dataInfo = "⟳ refreshing..."
	}

	// Short help based on whether help is shown
	shortHelp := "? help | Shift+L layout | r refresh | q quit"
	if d.showHelp {
		shortHelp = "j/k up/down | h/l left/right | Shift+L layout | r refresh | ? help | q quit"
	}

	return components.DashboardStatusLine(d.width, layoutName, dataInfo, shortHelp)
}

package views

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
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
	currentLayout     models.LayoutType
	showHelp          bool
	selectedRepo      *models.Repository
	selectedRepoIdx   int
	selectedRunIdx    int
	focusedColumn     int // 0: repositories, 1: runs, 2: details
	selectedDetailLine int // Selected line in details column
	detailLines       []string // Lines in details column for selection

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
	apiRepositories map[int]models.APIRepository // Map repo ID to API repository
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
		apiRepositories: make(map[int]models.APIRepository),
	}

	// Initialize cache system
	_ = cache.InitializeDashboardCache()

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
		// First try to load from run cache which should always have data
		runs, cached, _, _, _ := cache.GetCachedList()
		if cached && len(runs) > 0 {
			// Convert to pointer slice
			allRuns := make([]*models.RunResponse, len(runs))
			for i, run := range runs {
				allRuns[i] = &run
			}

			// Try to get cached repository overview
			repositories, repoCached, _ := cache.GetRepositoryOverview()
			if !repoCached || len(repositories) == 0 {
				// Build repositories from runs if not cached
				repositories = cache.BuildRepositoryOverviewFromRuns(allRuns)
				_ = cache.SetRepositoryOverview(repositories)
			}

			return dashboardDataLoadedMsg{
				repositories: repositories,
				allRuns:      allRuns,
				error:        nil,
			}
		}

		// No cache, fetch from API
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Store API repositories for ID mapping
		d.apiRepositories = make(map[int]models.APIRepository)

		// First, try to get repositories from API
		apiRepositories, err := d.client.ListRepositories(ctx)
		if err != nil {
			// Fall back to building repos from runs if repository API fails
			return d.loadFromRunsOnly()
		}

		// Store API repositories by ID for quick lookup
		for _, apiRepo := range apiRepositories {
			d.apiRepositories[apiRepo.ID] = apiRepo
		}

		// Convert API repositories to dashboard models
		repositories := make([]models.Repository, 0, len(apiRepositories))
		for _, apiRepo := range apiRepositories {
			// Construct full repository name
			repoName := apiRepo.Name
			if repoName == "" {
				repoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
			}

			repositories = append(repositories, models.Repository{
				Name:        repoName,
				Description: "",                // API doesn't provide description
				RunCounts:   models.RunStats{}, // Will be populated below
			})
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
	// Create maps for quick lookup
	repoMap := make(map[string]*models.Repository)
	repoIDMap := make(map[int]*models.Repository) // Map by repo ID

	for i := range repositories {
		repoMap[repositories[i].Name] = &repositories[i]

		// Also map by ID if we have API repositories
		if d.apiRepositories != nil {
			for id, apiRepo := range d.apiRepositories {
				apiRepoName := apiRepo.Name
				if apiRepoName == "" {
					apiRepoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
				}
				if apiRepoName == repositories[i].Name {
					repoIDMap[id] = &repositories[i]
					break
				}
			}
		}
	}

	// Update statistics from runs
	for _, run := range allRuns {
		var repo *models.Repository

		// First try to match by repository name
		repoName := run.GetRepositoryName()
		if repoName != "" {
			repo = repoMap[repoName]
		}

		// If not found and we have a repo ID, try to match by ID
		if repo == nil && run.RepoID > 0 {
			repo = repoIDMap[run.RepoID]
		}

		if repo == nil {
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
		// Filter runs for this repository
		var filteredRuns []*models.RunResponse

		// First try to match by repository name
		for _, run := range d.allRuns {
			runRepoName := run.GetRepositoryName()
			if runRepoName == repo.Name {
				filteredRuns = append(filteredRuns, run)
				continue
			}

			// Also try to match by repo ID if we have API repositories
			if run.RepoID > 0 && d.apiRepositories != nil {
				if apiRepo, exists := d.apiRepositories[run.RepoID]; exists {
					apiRepoName := apiRepo.Name
					if apiRepoName == "" {
						apiRepoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
					}
					if apiRepoName == repo.Name {
						filteredRuns = append(filteredRuns, run)
					}
				}
			}
		}

		return dashboardRepositorySelectedMsg{
			repository: repo,
			runs:       filteredRuns,
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
			d.updateDetailLines()
		}

	case tea.KeyMsg:
		// Handle dashboard-specific keys first
		switch {
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "n":
			// Navigate to create new run view
			createView := NewCreateRunViewWithCache(d.client, nil, false, time.Time{}, nil)
			createView.width = d.width
			createView.height = d.height
			return createView, nil
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
					d.updateDetailLines()
				}
			}
		case 2: // Details column
			if d.selectedDetailLine > 0 {
				d.selectedDetailLine--
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
					d.updateDetailLines()
				}
			}
		case 2: // Details column
			if d.selectedDetailLine < len(d.detailLines)-1 {
				d.selectedDetailLine++
			}
		}

	case key.Matches(msg, d.keys.Enter):
		// Enter moves focus right and selects first item
		if d.focusedColumn < 2 {
			d.focusedColumn++
			if d.focusedColumn == 1 && len(d.filteredRuns) > 0 {
				// Moving to runs column, select first run if none selected
				if d.selectedRunData == nil && len(d.filteredRuns) > 0 {
					d.selectedRunIdx = 0
					d.selectedRunData = d.filteredRuns[0]
					d.updateDetailLines()
				}
			} else if d.focusedColumn == 2 {
				// Moving to details column, select first line
				d.selectedDetailLine = 0
			}
		}

	case msg.Type == tea.KeyBackspace:
		// Backspace moves focus left
		if d.focusedColumn > 0 {
			d.focusedColumn--
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "y":
		// Copy current row/line in any column
		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx < len(d.repositories) {
				repo := d.repositories[d.selectedRepoIdx]
				return d.copyToClipboard(repo.Name)
			}
		case 1: // Runs column
			if d.selectedRunIdx < len(d.filteredRuns) {
				run := d.filteredRuns[d.selectedRunIdx]
				text := fmt.Sprintf("%s - %s", run.GetIDString(), run.Title)
				return d.copyToClipboard(text)
			}
		case 2: // Details column
			if d.selectedDetailLine < len(d.detailLines) {
				return d.copyToClipboard(d.detailLines[d.selectedDetailLine])
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
				d.updateDetailLines()
			} else if d.focusedColumn == 2 {
				d.selectedDetailLine = 0
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

	// Always show title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Width(d.width).
		Align(lipgloss.Center)

	title := titleStyle.Render("Repobird.ai CLI")

	if d.error != nil {
		content = fmt.Sprintf("Error loading dashboard data: %s\n\nPress 'r' to retry, 'q' to quit", d.error.Error())
		return lipgloss.JoinVertical(lipgloss.Left, title, content)
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
		return lipgloss.JoinVertical(lipgloss.Left, title, content)
	}

	if d.loading || d.initializing {
		content = "Loading dashboard data..."
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(d.width).
			Align(lipgloss.Center).
			MarginTop((d.height - 2) / 2)
		content = loadingStyle.Render(content)
		return lipgloss.JoinVertical(lipgloss.Left, title, content)
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

	return lipgloss.JoinVertical(lipgloss.Left, title, content)
}

// renderTripleColumnLayout renders the Miller Columns layout with real data
func (d *DashboardView) renderTripleColumnLayout() string {
	// Calculate available height for columns
	// We receive (d.height - 1) from View() after title
	// We need to reserve 1 line for statusline
	availableHeight := d.height - 3 // Account for title (1) and statusline (1)

	// Column widths - full width, equally distributed
	leftWidth := d.width / 3
	centerWidth := d.width / 3
	rightWidth := d.width - leftWidth - centerWidth

	// Create column content with titles
	leftContent := d.renderRepositoriesColumn(leftWidth - 2, availableHeight - 2)
	centerContent := d.renderRunsColumn(centerWidth - 2, availableHeight - 2)
	rightContent := d.renderDetailsColumn(rightWidth - 2, availableHeight - 2)

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
		leftStyle.Render(leftContent),
		centerStyle.Render(centerContent),
		rightStyle.Render(rightContent),
	)

	// Create statusline
	statusline := d.renderStatusLine("Miller Columns")

	return lipgloss.JoinVertical(lipgloss.Left, columns, statusline)
}

// updateDetailLines updates the detail lines for the selected run
func (d *DashboardView) updateDetailLines() {
	d.detailLines = []string{}
	d.selectedDetailLine = 0
	
	if d.selectedRunData == nil {
		return
	}
	
	run := d.selectedRunData
	d.detailLines = []string{
		fmt.Sprintf("ID: %s", run.GetIDString()),
		fmt.Sprintf("Status: %s", run.Status),
		fmt.Sprintf("Repository: %s", run.GetRepositoryName()),
	}

	if run.Source != "" && run.Target != "" {
		d.detailLines = append(d.detailLines, fmt.Sprintf("Branch: %s → %s", run.Source, run.Target))
	}

	d.detailLines = append(d.detailLines, fmt.Sprintf("Created: %s", run.CreatedAt.Format("Jan 2 15:04")))
	d.detailLines = append(d.detailLines, fmt.Sprintf("Updated: %s", run.UpdatedAt.Format("Jan 2 15:04")))

	if run.Title != "" {
		d.detailLines = append(d.detailLines, "", "Title:", run.Title)
	}

	if run.Prompt != "" {
		d.detailLines = append(d.detailLines, "", "Prompt:")
		// Wrap prompt text - use a reasonable width
		wrapped := d.wrapText(run.Prompt, 50)
		d.detailLines = append(d.detailLines, wrapped...)
	}

	if run.Error != "" {
		d.detailLines = append(d.detailLines, "", "Error:")
		wrapped := d.wrapText(run.Error, 50)
		d.detailLines = append(d.detailLines, wrapped...)
	}
}

// copyToClipboard copies the given text to clipboard
func (d *DashboardView) copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		// Try to copy to clipboard using pbcopy on macOS, xclip on Linux
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("pbcopy")
		} else if runtime.GOOS == "linux" {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			return nil // Unsupported OS
		}
		
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return nil
	}
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

// renderRepositoriesColumn renders the left column with real repositories
func (d *DashboardView) renderRepositoriesColumn(width, height int) string {
	// Create title with underline
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("63"))
	
	if d.focusedColumn == 0 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("63"))
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}
	title := titleStyle.Render("Repositories")

	// Build items list
	var items []string
	for i, repo := range d.repositories {
		statusIcon := d.getRepositoryStatusIcon(&repo)
		item := fmt.Sprintf("%s %s", statusIcon, repo.Name)

		// Truncate if too long
		if len(item) > width-2 {
			item = item[:width-5] + "..."
		}

		// Highlight selected repository
		if i == d.selectedRepoIdx {
			if d.focusedColumn == 0 {
				item = lipgloss.NewStyle().
					Width(width).
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("255")).
					Render(item)
			} else {
				item = lipgloss.NewStyle().
					Width(width).
					Background(lipgloss.Color("240")).
					Foreground(lipgloss.Color("255")).
					Render(item)
			}
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		items = []string{"No repositories"}
	}

	// Calculate content height (subtract title height)
	contentHeight := height - 2
	content := strings.Join(items, "\n")
	
	// Pad content to fill height
	contentStyle := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(content))
}

// renderRunsColumn renders the center column with runs for selected repository
func (d *DashboardView) renderRunsColumn(width, height int) string {
	// Create title with underline
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("33"))
	
	if d.focusedColumn == 1 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("33"))
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}
	title := titleStyle.Render("Runs")

	var items []string
	if d.selectedRepo == nil {
		items = []string{"Select a repository"}
	} else {
		for i, run := range d.filteredRuns {
			statusIcon := d.getRunStatusIcon(run.Status)
			displayTitle := run.Title
			if displayTitle == "" {
				displayTitle = "Untitled Run"
			}
			
			// Truncate based on available width
			maxTitleLen := width - 5 // Account for icon and padding
			if len(displayTitle) > maxTitleLen {
				displayTitle = displayTitle[:maxTitleLen-3] + "..."
			}

			item := fmt.Sprintf("%s %s", statusIcon, displayTitle)

			// Highlight selected run
			if i == d.selectedRunIdx {
				if d.focusedColumn == 1 {
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("33")).
						Foreground(lipgloss.Color("255")).
						Render(item)
				} else {
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("240")).
						Foreground(lipgloss.Color("255")).
						Render(item)
				}
			}

			items = append(items, item)
		}

		if len(items) == 0 {
			items = []string{fmt.Sprintf("No runs for %s", d.selectedRepo.Name)}
		}
	}

	// Calculate content height (subtract title height)
	contentHeight := height - 2
	content := strings.Join(items, "\n")
	
	// Pad content to fill height
	contentStyle := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(content))
}

// renderDetailsColumn renders the right column with run details
func (d *DashboardView) renderDetailsColumn(width, height int) string {
	// Create title with underline
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240"))
	
	if d.focusedColumn == 2 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("63"))
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}
	title := titleStyle.Render("Run Details")

	var displayLines []string
	if d.selectedRunData == nil {
		displayLines = []string{"Select a run"}
	} else {
		// Build lines with selection highlighting
		for i, line := range d.detailLines {
			if d.focusedColumn == 2 && i == d.selectedDetailLine {
				// Highlight selected line
				line = lipgloss.NewStyle().
					Width(width-2).
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("255")).
					Render(line)
			}
			displayLines = append(displayLines, line)
		}
	}

	// Calculate content height (subtract title height)
	contentHeight := height - 2
	content := strings.Join(displayLines, "\n")
	
	// Pad content to fill height
	contentStyle := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(content))
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
	// Data freshness indicator - keep it short
	dataInfo := ""
	if d.loading && len(d.repositories) > 0 {
		dataInfo = "⟳"
	} else if !d.lastDataRefresh.IsZero() {
		elapsed := time.Since(d.lastDataRefresh)
		if elapsed < time.Minute {
			dataInfo = "✓"
		} else {
			dataInfo = fmt.Sprintf("%dm", int(elapsed.Minutes()))
		}
	}

	// Shorter help text to fit on one line
	shortHelp := "n new | y copy | ? help | r refresh | q quit"
	if d.showHelp {
		shortHelp = "n new | y copy | j/k ↑↓ | h/l ←→ | Enter → | Backspace ← | ? help | q quit"
	}

	return components.DashboardStatusLine(d.width, layoutName, dataInfo, shortHelp)
}

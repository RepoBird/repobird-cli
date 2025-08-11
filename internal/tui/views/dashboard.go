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
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
)

// DashboardView is the main dashboard controller that manages different layout views
type DashboardView struct {
	client *api.Client
	keys   components.KeyMap
	help   help.Model

	// Dashboard state
	currentLayout      models.LayoutType
	showHelp           bool
	showStatusInfo     bool // Show status/user info overlay
	selectedRepo       *models.Repository
	selectedRepoIdx    int
	selectedRunIdx     int
	focusedColumn      int      // 0: repositories, 1: runs, 2: details
	selectedDetailLine int      // Selected line in details column
	detailLines        []string // Lines in details column for selection

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

	// User info
	userInfo *models.UserInfo
	userID   *int // User ID for cache isolation

	// FZF mode for each column
	fzfMode   *components.FZFMode
	fzfColumn int // Which column is in FZF mode (-1 = none)

	// Loading spinner
	spinner spinner.Model

	// Clipboard feedback
	copiedMessage     string
	copiedMessageTime time.Time
	yankBlink         bool      // Toggle for blinking effect
	yankBlinkTime     time.Time // Time when blink started (separate from message timing)

	// Temporary status line messages (for URL opening, etc.)
	statusMessage     string
	statusMessageTime time.Time

	// Store original untruncated detail lines for copying
	detailLinesOriginal []string

	// Status info overlay navigation
	statusInfoSelectedRow int      // Currently selected row in status info
	statusInfoFields      []string // Field values that can be copied
	statusInfoFieldLines  []int    // Line numbers for each field

	// URL selection for repositories
	showURLSelectionPrompt bool                  // Show URL selection prompt in status line
	pendingRepoForURL      *models.Repository    // Repository pending URL selection
	pendingAPIRepoForURL   *models.APIRepository // API repository data for URL generation
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

type dashboardUserInfoLoadedMsg struct {
	userInfo *models.UserInfo
	error    error
}

type clearStatusMessageMsg struct{}

// NewDashboardView creates a new dashboard view
func NewDashboardView(client *api.Client) *DashboardView {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	dashboard := &DashboardView{
		client:          client,
		keys:            components.DefaultKeyMap,
		help:            help.New(),
		currentLayout:   models.LayoutTripleColumn,
		loading:         true,
		initializing:    true,
		refreshInterval: 30 * time.Second,
		apiRepositories: make(map[int]models.APIRepository),
		fzfColumn:       -1, // No FZF mode initially
		spinner:         s,
	}

	// Initialize cache system
	_ = cache.InitializeDashboardCache()

	// Initialize with existing list view
	dashboard.runListView = NewRunListView(client)

	return dashboard
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
		d.runListView.Init(),
		d.spinner.Tick,
	)
}

// loadUserInfo loads user information from the API
func (d *DashboardView) loadUserInfo() tea.Cmd {
	return func() tea.Msg {
		// First check if we have cached user info
		if cachedInfo := cache.GetCachedUserInfo(); cachedInfo != nil {
			return dashboardUserInfoLoadedMsg{
				userInfo: cachedInfo,
				error:    nil,
			}
		}

		// Fetch from API if not cached
		userInfo, err := d.client.GetUserInfo()
		if err == nil && userInfo != nil {
			// Cache the user info
			cache.SetCachedUserInfo(userInfo)
		}
		return dashboardUserInfoLoadedMsg{
			userInfo: userInfo,
			error:    err,
		}
	}
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
	case spinner.TickMsg:
		if d.loading || d.initializing {
			var cmd tea.Cmd
			d.spinner, cmd = d.spinner.Update(msg)
			return d, cmd
		}

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

	case dashboardUserInfoLoadedMsg:
		if msg.error == nil && msg.userInfo != nil {
			d.userInfo = msg.userInfo
			// Store user ID and reinitialize caches if needed
			if d.userID == nil || (d.userID != nil && *d.userID != msg.userInfo.ID) {
				d.userID = &msg.userInfo.ID
				// Reinitialize caches with user-specific paths
				cache.InitializeCacheForUser(d.userID)
				cache.InitializeDashboardForUser(d.userID)
			}
		}

	case yankBlinkMsg:
		// Single blink: toggle off after being on
		if d.yankBlink {
			d.yankBlink = false // Turn off after being on - completes the single blink
		}
		// No more blinking after the single on-off cycle

	case clearStatusMsg:
		// Clear the clipboard message after timeout
		d.copiedMessage = ""
		d.yankBlink = false

	case clearStatusMessageMsg:
		// Clear the temporary status message after timeout
		d.statusMessage = ""

	case components.FZFSelectedMsg:
		// Handle FZF selection result
		if !msg.Result.Canceled {
			switch d.fzfColumn {
			case 0: // Repository column
				if msg.Result.Index >= 0 && msg.Result.Index < len(d.repositories) {
					d.selectedRepoIdx = msg.Result.Index
					d.selectedRepo = &d.repositories[d.selectedRepoIdx]
					d.focusedColumn = 1 // Move to runs column
					cmds = append(cmds, d.selectRepository(d.selectedRepo))
				}
			case 1: // Runs column
				if msg.Result.Index >= 0 && msg.Result.Index < len(d.filteredRuns) {
					d.selectedRunIdx = msg.Result.Index
					d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
					d.updateDetailLines()
					d.focusedColumn = 2 // Move to details column
					d.selectedDetailLine = 0
				}
			case 2: // Details column
				if msg.Result.Index >= 0 && msg.Result.Index < len(d.detailLines) {
					d.selectedDetailLine = msg.Result.Index
				}
			}
		}
		// Deactivate FZF mode
		d.fzfColumn = -1
		d.fzfMode = nil
		return d, nil

	case tea.KeyMsg:
		// If FZF mode is active, handle input there first
		if d.fzfMode != nil && d.fzfMode.IsActive() {
			newFzf, cmd := d.fzfMode.Update(msg)
			d.fzfMode = newFzf
			return d, cmd
		}

		// Handle dashboard-specific keys
		switch {
		case msg.Type == tea.KeyEsc && d.showURLSelectionPrompt:
			// Close URL selection prompt with ESC
			d.showURLSelectionPrompt = false
			d.pendingRepoForURL = nil
			d.pendingAPIRepoForURL = nil
			return d, nil
		case d.showURLSelectionPrompt && (msg.Type == tea.KeyRunes && (string(msg.Runes) == "o" || string(msg.Runes) == "g")):
			// Handle URL selection
			var urlText string
			var message string

			if string(msg.Runes) == "o" && d.pendingAPIRepoForURL != nil {
				// Open RepoBird URL
				urlText = fmt.Sprintf("https://repobird.ai/repos/%d", d.pendingAPIRepoForURL.ID)
				message = "🌐 Opened RepoBird URL in browser"
			} else if string(msg.Runes) == "g" && d.pendingAPIRepoForURL != nil {
				// Open GitHub URL
				urlText = d.pendingAPIRepoForURL.RepoURL
				message = "🌐 Opened GitHub URL in browser"
			}

			// Clear the prompt
			d.showURLSelectionPrompt = false
			d.pendingRepoForURL = nil
			d.pendingAPIRepoForURL = nil

			if urlText != "" {
				if err := utils.OpenURL(urlText); err == nil {
					d.statusMessage = message
				} else {
					d.statusMessage = fmt.Sprintf("✗ Failed to open URL: %v", err)
				}
				d.statusMessageTime = time.Now()
				return d, d.startClearStatusMessageTimer()
			}
			return d, nil
		case msg.Type == tea.KeyEsc && d.showStatusInfo:
			// Close status info overlay with ESC
			d.showStatusInfo = false
			return d, nil
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "s" && !d.showStatusInfo:
			// Toggle status info view
			d.showStatusInfo = true
			// Initialize status info navigation
			d.initializeStatusInfoFields()
			// Refresh user info when showing
			cmds = append(cmds, d.loadUserInfo())
			return d, tea.Batch(cmds...)
		case d.showStatusInfo:
			// Handle navigation in status info overlay
			return d.handleStatusInfoNavigation(msg)
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "n":
			// Navigate to create new run view with selected repository
			config := CreateRunViewConfig{
				Client: d.client,
			}

			// Pass the selected repository if one is selected
			if d.selectedRepo != nil {
				config.SelectedRepository = d.selectedRepo.Name
			}

			createView := NewCreateRunViewWithConfig(config)
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
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "f":
			// Activate FZF mode for current column in dashboard
			if d.currentLayout == models.LayoutTripleColumn {
				d.activateFZFMode()
				return d, nil
			}
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

// isEmptyLine checks if a line in the details column is empty or just whitespace
func (d *DashboardView) isEmptyLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// findNextNonEmptyLine finds the next non-empty line starting from current index
func (d *DashboardView) findNextNonEmptyLine(startIdx int, direction int) int {
	if len(d.detailLines) == 0 {
		return startIdx
	}

	idx := startIdx
	for {
		idx += direction

		// Check bounds
		if idx < 0 {
			return startIdx // No non-empty line found upward
		}
		if idx >= len(d.detailLines) {
			return startIdx // No non-empty line found downward
		}

		// Check if line is non-empty
		if !d.isEmptyLine(d.detailLines[idx]) {
			return idx
		}

		// Prevent infinite loop (shouldn't happen but safety check)
		if idx == 0 && direction < 0 {
			return startIdx
		}
		if idx == len(d.detailLines)-1 && direction > 0 {
			return startIdx
		}
	}
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
				// Try to find previous non-empty line
				newIdx := d.findNextNonEmptyLine(d.selectedDetailLine, -1)
				if newIdx != d.selectedDetailLine {
					d.selectedDetailLine = newIdx
				} else {
					// If no non-empty line found, just move up one
					d.selectedDetailLine--
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
					d.updateDetailLines()
				}
			}
		case 2: // Details column
			if d.selectedDetailLine < len(d.detailLines)-1 {
				// Try to find next non-empty line
				newIdx := d.findNextNonEmptyLine(d.selectedDetailLine, 1)
				if newIdx != d.selectedDetailLine {
					d.selectedDetailLine = newIdx
				} else {
					// If no non-empty line found, just move down one
					d.selectedDetailLine++
				}
			}
		}

	case key.Matches(msg, d.keys.Tab):
		// Tab cycles through columns
		d.focusedColumn = (d.focusedColumn + 1) % 3
		if d.focusedColumn == 1 && len(d.filteredRuns) > 0 && d.selectedRunData == nil {
			// Moving to runs column, select first run if none selected
			d.selectedRunIdx = 0
			d.selectedRunData = d.filteredRuns[0]
			d.updateDetailLines()
		} else if d.focusedColumn == 2 {
			// Moving to details column, select first non-empty line
			d.selectedDetailLine = 0
			// Skip empty lines at the beginning
			if len(d.detailLines) > 0 && d.isEmptyLine(d.detailLines[0]) {
				newIdx := d.findNextNonEmptyLine(-1, 1) // Start from -1 to check from index 0
				if newIdx >= 0 && newIdx < len(d.detailLines) {
					d.selectedDetailLine = newIdx
				}
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
				// Moving to details column, select first non-empty line
				d.selectedDetailLine = 0
				// Skip empty lines at the beginning
				if len(d.detailLines) > 0 && d.isEmptyLine(d.detailLines[0]) {
					newIdx := d.findNextNonEmptyLine(-1, 1) // Start from -1 to check from index 0
					if newIdx >= 0 && newIdx < len(d.detailLines) {
						d.selectedDetailLine = newIdx
					}
				}
			}
		}

	case msg.Type == tea.KeyBackspace:
		// Backspace moves focus left
		if d.focusedColumn > 0 {
			d.focusedColumn--
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "y":
		// Copy current row/line in any column
		var textToCopy string

		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx < len(d.repositories) {
				repo := d.repositories[d.selectedRepoIdx]
				textToCopy = repo.Name
			}
		case 1: // Runs column
			if d.selectedRunIdx < len(d.filteredRuns) {
				run := d.filteredRuns[d.selectedRunIdx]
				textToCopy = fmt.Sprintf("%s - %s", run.GetIDString(), run.Title)
			}
		case 2: // Details column
			if d.selectedDetailLine < len(d.detailLinesOriginal) {
				// Use original untruncated text for copying
				textToCopy = d.detailLinesOriginal[d.selectedDetailLine]
			}
		}

		if textToCopy != "" {
			if err := d.copyToClipboard(textToCopy); err == nil {
				// Show what's actually on the clipboard, truncated for display if needed
				displayText := textToCopy
				maxLen := 30
				if len(displayText) > maxLen {
					displayText = displayText[:maxLen-3] + "..."
				}
				d.statusMessage = fmt.Sprintf("📋 Copied \"%s\"", displayText)
			} else {
				d.statusMessage = "✗ Failed to copy"
			}
			d.statusMessageTime = time.Now()
			d.yankBlink = true
			d.yankBlinkTime = time.Now()
			return tea.Batch(
				d.startYankBlinkAnimation(),
				d.startClearStatusMessageTimer(),
			)
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "o":
		// Open URL in browser if current selection contains a URL
		var urlText string

		switch d.focusedColumn {
		case 0: // Repository column - handle repository URLs
			if d.selectedRepoIdx < len(d.repositories) {
				repo := d.repositories[d.selectedRepoIdx]
				// Check if we can provide URL options
				apiRepo := d.getAPIRepositoryForRepo(&repo)
				if apiRepo != nil {
					// Show URL selection prompt in status line
					d.showURLSelectionPrompt = true
					d.pendingRepoForURL = &repo
					d.pendingAPIRepoForURL = apiRepo
					return nil
				}
			}
		case 1: // Runs column - could check for PR URLs in run data
			if d.selectedRunIdx < len(d.filteredRuns) {
				run := d.filteredRuns[d.selectedRunIdx]
				if run.PrURL != nil && *run.PrURL != "" {
					urlText = *run.PrURL
				}
			}
		case 2: // Details column - check if selected line contains a URL or is an ID field
			if d.selectedDetailLine < len(d.detailLinesOriginal) {
				lineText := d.detailLinesOriginal[d.selectedDetailLine]
				if utils.IsURL(lineText) {
					urlText = utils.ExtractURL(lineText)
				} else if d.selectedDetailLine == 0 && d.selectedRunData != nil {
					// First line is the ID field, generate RepoBird URL
					runID := d.selectedRunData.GetIDString()
					if utils.IsNonEmptyNumber(runID) {
						urlText = utils.GenerateRepoBirdURL(runID)
					}
				} else if d.selectedDetailLine == 2 && d.selectedRunData != nil {
					// Repository line - show URL selection prompt
					repoName := d.selectedRunData.GetRepositoryName()
					if repoName != "" {
						repo := d.getRepositoryByName(repoName)
						apiRepo := d.getAPIRepositoryForRepo(repo)
						if apiRepo != nil {
							// Show URL selection prompt in status line
							d.showURLSelectionPrompt = true
							d.pendingRepoForURL = repo
							d.pendingAPIRepoForURL = apiRepo
							return nil
						}
					}
				}
			}
		}

		if urlText != "" {
			if err := utils.OpenURL(urlText); err == nil {
				d.statusMessage = "🌐 Opened URL in browser"
			} else {
				d.statusMessage = fmt.Sprintf("✗ Failed to open URL: %v", err)
			}
			d.statusMessageTime = time.Now()
			return d.startClearStatusMessageTimer()
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
				// Select first non-empty line when moving to details
				d.selectedDetailLine = 0
				if len(d.detailLines) > 0 && d.isEmptyLine(d.detailLines[0]) {
					newIdx := d.findNextNonEmptyLine(-1, 1)
					if newIdx >= 0 && newIdx < len(d.detailLines) {
						d.selectedDetailLine = newIdx
					}
				}
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
	debug.LogToFilef("Title width: %d\n", lipgloss.Width(title))

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
		// Use the animated spinner + loading text
		loadingText := d.spinner.View() + " Loading dashboard data..."
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")). // Bright cyan color
			Bold(true).
			Width(d.width).
			Align(lipgloss.Center).
			MarginTop((d.height - 2) / 2)
		content = loadingStyle.Render(loadingText)
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

	finalView := lipgloss.JoinVertical(lipgloss.Left, title, content)

	// Overlay FZF selector if active
	if d.fzfMode != nil && d.fzfMode.IsActive() {
		return d.renderWithFZFOverlay(finalView)
	}

	// Overlay status info if requested
	if d.showStatusInfo {
		return d.renderStatusInfo()
	}

	debug.LogToFilef("Final view dimensions: width=%d, height=%d\n",
		lipgloss.Width(finalView), lipgloss.Height(finalView))
	return finalView
}

// renderTripleColumnLayout renders the Miller Columns layout with real data
func (d *DashboardView) renderTripleColumnLayout() string {
	// Debug logging
	debug.LogToFilef("Dashboard Layout: width=%d, height=%d\n", d.width, d.height)

	// Calculate available height for columns
	// We have d.height total, minus:
	// - 2 for title (1 line + spacing)
	// - 1 for statusline
	availableHeight := d.height - 3
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}

	// Column widths - calculate based on terminal width
	// Each box renders 2 pixels wider than its set width, so subtract 6 total (2 per column)
	// to ensure they fit within terminal width
	totalWidth := d.width - 6 // Subtract 6 to account for the 2-pixel expansion per box
	leftWidth := totalWidth / 3
	centerWidth := totalWidth / 3
	rightWidth := totalWidth - leftWidth - centerWidth // Use remaining width

	// Ensure minimum widths
	if leftWidth < 10 {
		leftWidth = 10
	}
	if centerWidth < 10 {
		centerWidth = 10
	}
	if rightWidth < 10 {
		rightWidth = 10
	}

	debug.LogToFilef("Column widths: left=%d, center=%d, right=%d, total=%d\n",
		leftWidth, centerWidth, rightWidth, leftWidth+centerWidth+rightWidth)

	// Make columns with rounded borders - use full available height
	// The Height() method in lipgloss includes borders in the total height
	columnHeight := availableHeight
	if columnHeight < 3 {
		columnHeight = 3
	}

	debug.LogToFilef("Column height: available=%d, column=%d\n", availableHeight, columnHeight)

	// Create column content with titles
	// Account for borders (2 chars for left/right, 2 for top/bottom)
	// Content width should be column width minus borders
	contentWidth1 := leftWidth - 2
	contentWidth2 := centerWidth - 2
	contentWidth3 := rightWidth - 2
	contentHeight := columnHeight - 2

	leftContent := d.renderRepositoriesColumn(contentWidth1, contentHeight)
	centerContent := d.renderRunsColumn(contentWidth2, contentHeight)
	rightContent := d.renderDetailsColumn(contentWidth3, contentHeight)

	debug.LogToFilef("Content dimensions: w1=%d, w2=%d, w3=%d, h=%d\n",
		contentWidth1, contentWidth2, contentWidth3, contentHeight)

	// Create styles for columns
	// Width() and Height() in lipgloss include the border in the total dimensions
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(columnHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	centerStyle := lipgloss.NewStyle().
		Width(centerWidth).
		Height(columnHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("33"))

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(columnHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	// Render each column
	leftBox := leftStyle.Render(leftContent)
	centerBox := centerStyle.Render(centerContent)
	rightBox := rightStyle.Render(rightContent)

	debug.LogToFilef("Box widths: left=%d, center=%d, right=%d\n",
		lipgloss.Width(leftBox), lipgloss.Width(centerBox), lipgloss.Width(rightBox))

	// Join columns horizontally - they should already fit the width exactly
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftBox,
		centerBox,
		rightBox,
	)

	finalWidth := lipgloss.Width(columns)
	debug.LogToFilef("Final columns width=%d (terminal width=%d)\n", finalWidth, d.width)

	// If columns still exceed terminal width (shouldn't happen with correct calculation)
	// Use PlaceHorizontal to constrain them
	if finalWidth > d.width {
		debug.LogToFilef("WARNING: Columns width %d exceeds terminal width %d, constraining...\n", finalWidth, d.width)
		columns = lipgloss.PlaceHorizontal(d.width, lipgloss.Left, columns)
	}

	// Create statusline
	statusline := d.renderStatusLine("DASH")
	debug.LogToFilef("Statusline width: %d\n", lipgloss.Width(statusline))

	// The statusline should be placed at the bottom with proper spacing
	// Place the columns and statusline in the available space
	columnsHeight := lipgloss.Height(columns)
	debug.LogToFilef("Heights: columns=%d, availableHeight=%d\n", columnsHeight, availableHeight)

	// Add notification line if there's a message to show
	var parts []string
	parts = append(parts, columns)

	// Add notification above status line if there's a message
	if notificationLine := d.renderNotificationLine(); notificationLine != "" {
		parts = append(parts, notificationLine)
	}

	parts = append(parts, statusline)

	// Use PlaceVertical to position the statusline at the bottom
	// The available height already accounts for title and statusline
	finalLayout := lipgloss.JoinVertical(lipgloss.Left, parts...)

	debug.LogToFilef("Triple column layout dimensions: width=%d, height=%d\n",
		lipgloss.Width(finalLayout), lipgloss.Height(finalLayout))
	return finalLayout
}

// updateDetailLines updates the detail lines for the selected run
func (d *DashboardView) updateDetailLines() {
	d.detailLines = []string{}
	d.detailLinesOriginal = []string{}
	d.selectedDetailLine = 0

	if d.selectedRunData == nil {
		return
	}

	run := d.selectedRunData
	// Calculate available width for text (accounting for padding)
	columnWidth := d.width / 3
	if columnWidth < 10 {
		columnWidth = 10
	}
	textWidth := columnWidth - 4 // Account for padding and borders
	if textWidth < 10 {
		textWidth = 10
	}

	// Helper function to truncate text to single line
	truncateLine := func(text string) string {
		if len(text) > textWidth {
			return text[:textWidth-3] + "..."
		}
		return text
	}

	// Helper to add both truncated and original lines
	addLine := func(text string) {
		d.detailLines = append(d.detailLines, truncateLine(text))
		d.detailLinesOriginal = append(d.detailLinesOriginal, text)
	}

	// Add single-line fields (truncated for display, original for copying)
	addLine(fmt.Sprintf("ID: %s", run.GetIDString()))
	addLine(fmt.Sprintf("Status: %s", run.Status))
	addLine(fmt.Sprintf("Repository: %s", run.GetRepositoryName()))

	// Show run type - normalize API values to display values
	if run.RunType != "" {
		displayType := "Run"
		runTypeLower := strings.ToLower(run.RunType)
		if strings.Contains(runTypeLower, "plan") {
			displayType = "Plan"
		}
		addLine(fmt.Sprintf("Type: %s", displayType))
	}

	if run.Source != "" && run.Target != "" {
		addLine(fmt.Sprintf("Branch: %s → %s", run.Source, run.Target))
	}

	addLine(fmt.Sprintf("Created: %s", run.CreatedAt.Format("Jan 2 15:04")))
	addLine(fmt.Sprintf("Updated: %s", run.UpdatedAt.Format("Jan 2 15:04")))

	// Show PR URL if available
	if run.PrURL != nil && *run.PrURL != "" {
		addLine(fmt.Sprintf("PR URL: %s", *run.PrURL))
	}

	// Show trigger source if available
	if run.TriggerSource != nil && *run.TriggerSource != "" {
		addLine(fmt.Sprintf("Trigger: %s", *run.TriggerSource))
	}

	// Title - single line truncated
	if run.Title != "" {
		addLine("")
		addLine("Title:")
		addLine(run.Title)
	}

	// Prompt - single line truncated
	if run.Prompt != "" {
		addLine("")
		addLine("Prompt:")
		addLine(run.Prompt)
	}

	// Error - single line truncated
	if run.Error != "" {
		addLine("")
		addLine("Error:")
		addLine(run.Error)
	}

	// Plan field - special handling (can be multi-line if space available)
	// This should be last so it can use remaining space
	if strings.Contains(strings.ToLower(run.RunType), "plan") && run.Status == models.StatusDone && run.Plan != "" {
		addLine("")
		addLine("Plan:")
		// For now, just show first line with ellipsis if there's more
		// The renderDetailsColumn will handle proper multi-line display
		lines := strings.Split(run.Plan, "\n")
		if len(lines) > 0 {
			// Store full plan in original, but truncate for display
			d.detailLinesOriginal[len(d.detailLinesOriginal)-1] = run.Plan // Replace last "Plan:" with full plan
			firstLine := truncateLine(lines[0])
			if len(lines) > 1 {
				firstLine = firstLine + " (...)"
			}
			d.detailLines[len(d.detailLines)-1] = firstLine // Update display version
		}
	}
}

// copyToClipboard copies the given text to clipboard
func (d *DashboardView) copyToClipboard(text string) error {
	return utils.WriteToClipboard(text)
}

// startYankBlinkAnimation starts the single blink animation for clipboard feedback
func (d *DashboardView) startYankBlinkAnimation() tea.Cmd {
	return func() tea.Msg {
		// Single blink duration - visible flash (150ms)
		time.Sleep(150 * time.Millisecond)
		return yankBlinkMsg{}
	}
}

// startClearStatusTimer starts a timer to clear the status message
func (d *DashboardView) startClearStatusTimer() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(250 * time.Millisecond)
		return clearStatusMsg{}
	}
}

// startClearStatusMessageTimer starts a timer to clear the temporary status message
func (d *DashboardView) startClearStatusMessageTimer() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)
		return clearStatusMessageMsg{}
	}
}

// renderAllRunsLayout renders the timeline layout
func (d *DashboardView) renderAllRunsLayout() string {
	// Use the existing run list view
	runListContent := d.runListView.View()

	// Create statusline
	statusline := d.renderStatusLine("RUNS")

	// Add notification above status line if there's a message
	var parts []string
	parts = append(parts, runListContent)
	if notificationLine := d.renderNotificationLine(); notificationLine != "" {
		parts = append(parts, notificationLine)
	}
	parts = append(parts, statusline)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRepositoriesLayout renders the repositories-only layout
func (d *DashboardView) renderRepositoriesLayout() string {
	// Render repositories table
	content := d.renderRepositoriesTable()

	// Create statusline
	statusline := d.renderStatusLine("REPOS")

	// Add notification above status line if there's a message
	var parts []string
	parts = append(parts, content)
	if notificationLine := d.renderNotificationLine(); notificationLine != "" {
		parts = append(parts, notificationLine)
	}
	parts = append(parts, statusline)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
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
				// Single blink: bright green briefly when yankBlink is true
				if d.yankBlink && time.Since(d.yankBlinkTime) < 250*time.Millisecond {
					// Bright green flash
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("82")). // Bright green
						Foreground(lipgloss.Color("0")).  // Black text
						Bold(true).
						Render(item)
				} else {
					// Normal focused highlight
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("255")).
						Render(item)
				}
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
					// Custom blinking: toggle between bright and normal colors
					if d.yankBlink && time.Since(d.yankBlinkTime) < 250*time.Millisecond {
						if d.yankBlink {
							// Bright green when visible
							item = lipgloss.NewStyle().
								Width(width).
								Background(lipgloss.Color("82")). // Bright green
								Foreground(lipgloss.Color("0")).  // Black text
								Bold(true).
								Render(item)
						} else {
							// Normal highlight when "off"
							item = lipgloss.NewStyle().
								Width(width).
								Background(lipgloss.Color("33")).
								Foreground(lipgloss.Color("255")).
								Render(item)
						}
					} else {
						// Normal focused highlight (no blinking)
						item = lipgloss.NewStyle().
							Width(width).
							Background(lipgloss.Color("33")).
							Foreground(lipgloss.Color("255")).
							Render(item)
					}
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
		// Calculate available content width
		contentWidth := width - 2 // Account for padding
		if contentWidth < 5 {
			contentWidth = 5
		}

		// Build lines with selection highlighting and proper width constraints
		for i, line := range d.detailLines {
			// Check if we should show RepoBird URL hint for ID line
			displayLine := line
			if d.focusedColumn == 2 && i == d.selectedDetailLine && i == 0 && d.selectedRunData != nil {
				// This is the ID line and it's selected, add URL hint if possible
				runID := d.selectedRunData.GetIDString()
				if utils.IsNonEmptyNumber(runID) {
					repobirdURL := utils.GenerateRepoBirdURL(runID)
					// Truncate URL to fit within available width, keeping the line readable
					maxURLLen := contentWidth - len(line) - 3 // 3 chars for " - "
					if maxURLLen > 10 {                       // Only show if we have reasonable space
						truncatedURL := repobirdURL
						if len(truncatedURL) > maxURLLen {
							truncatedURL = truncatedURL[:maxURLLen-3] + "..."
						}
						displayLine = line + " - " + truncatedURL
					}
				}
			}

			// Apply width constraint using lipgloss to prevent overflow
			styledLine := lipgloss.NewStyle().
				MaxWidth(contentWidth).
				Inline(true). // Force single line
				Render(displayLine)

			if d.focusedColumn == 2 && i == d.selectedDetailLine {
				// Custom blinking: toggle between bright and normal colors
				if d.copiedMessage != "" && time.Since(d.copiedMessageTime) < 250*time.Millisecond {
					if d.yankBlink {
						// Bright green when visible
						styledLine = lipgloss.NewStyle().
							MaxWidth(contentWidth).
							Inline(true).
							Background(lipgloss.Color("82")). // Bright green
							Foreground(lipgloss.Color("0")).  // Black text
							Bold(true).
							Render(displayLine)
					} else {
						// Normal highlight when "off"
						styledLine = lipgloss.NewStyle().
							MaxWidth(contentWidth).
							Inline(true).
							Background(lipgloss.Color("63")).
							Foreground(lipgloss.Color("255")).
							Render(displayLine)
					}
				} else {
					// Normal focused highlight (no blinking)
					styledLine = lipgloss.NewStyle().
						MaxWidth(contentWidth).
						Inline(true).
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("255")).
						Render(displayLine)
				}
			}
			displayLines = append(displayLines, styledLine)
		}

		// Special handling for plan field if it's the last item
		// Calculate remaining vertical space
		contentHeight := height - 2 // Subtract title height
		usedLines := len(displayLines)
		remainingLines := contentHeight - usedLines

		// If we have a plan field and remaining space, expand it
		if d.selectedRunData != nil &&
			strings.Contains(strings.ToLower(d.selectedRunData.RunType), "plan") &&
			d.selectedRunData.Status == models.StatusDone &&
			d.selectedRunData.Plan != "" &&
			remainingLines > 0 {
			// Find the plan line (should be last)
			for i := len(d.detailLines) - 1; i >= 0; i-- {
				if strings.HasPrefix(d.detailLines[i], "Plan:") || (i > 0 && d.detailLines[i-1] == "Plan:") {
					// Replace the truncated plan with wrapped version
					wrapped := d.wrapTextWithLimit(d.selectedRunData.Plan, contentWidth, remainingLines)
					if len(wrapped) > 0 {
						// Remove the truncated plan line
						if i < len(displayLines) {
							displayLines = displayLines[:i]
						}
						// Add wrapped lines
						for _, wLine := range wrapped {
							styledLine := lipgloss.NewStyle().
								MaxWidth(contentWidth).
								Render(wLine)
							displayLines = append(displayLines, styledLine)
						}
					}
					break
				}
			}
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

// initializeStatusInfoFields initializes the selectable fields for the status info overlay
func (d *DashboardView) initializeStatusInfoFields() {
	d.statusInfoFields = []string{}
	d.statusInfoFieldLines = []int{}
	d.statusInfoSelectedRow = 0

	lineNum := 0

	// User Info fields
	if d.userInfo != nil {
		lineNum++ // Section header

		if d.userInfo.Name != "" {
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.Name)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}

		if d.userInfo.Email != "" {
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.Email)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}

		if d.userInfo.GithubUsername != "" {
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.GithubUsername)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}

		// Tier
		tierDisplay := d.userInfo.Tier
		if tierDisplay == "" {
			tierDisplay = "Free"
		} else {
			tierDisplay = strings.ToUpper(tierDisplay[:1]) + tierDisplay[1:]
		}
		d.statusInfoFields = append(d.statusInfoFields, tierDisplay)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++

		// Runs remaining
		var runsRemaining string
		if d.userInfo.TierDetails != nil {
			runsRemaining = fmt.Sprintf("%d Run / %d Plan",
				d.userInfo.TierDetails.RemainingProRuns,
				d.userInfo.TierDetails.RemainingPlanRuns)
		} else {
			runsRemaining = fmt.Sprintf("%d / %d",
				d.userInfo.RemainingRuns,
				d.userInfo.TotalRuns)
		}
		d.statusInfoFields = append(d.statusInfoFields, runsRemaining)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++

		// Usage - skip the bar, just include percentage
		if d.userInfo.TotalRuns > 0 {
			usedRuns := d.userInfo.TotalRuns - d.userInfo.RemainingRuns
			percentage := float64(usedRuns) / float64(d.userInfo.TotalRuns) * 100
			d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%.1f%%", percentage))
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		} else {
			d.statusInfoFields = append(d.statusInfoFields, "Unlimited")
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}
	}

	// System Stats fields
	lineNum++ // Section header

	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%d", len(d.repositories)))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%d", len(d.allRuns)))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	// Run status counts
	var running, completed, failed int
	for _, run := range d.allRuns {
		switch run.Status {
		case "running", "pending":
			running++
		case "completed", "success":
			completed++
		case "failed", "error":
			failed++
		}
	}
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("🔄 %d  ✅ %d  ❌ %d", running, completed, failed))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	// Last refresh
	if !d.lastDataRefresh.IsZero() {
		timeSince := time.Since(d.lastDataRefresh)
		refreshText := fmt.Sprintf("%d seconds ago", int(timeSince.Seconds()))
		if timeSince.Minutes() > 1 {
			refreshText = fmt.Sprintf("%.1f minutes ago", timeSince.Minutes())
		}
		d.statusInfoFields = append(d.statusInfoFields, refreshText)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++
	}

	// Connection Info fields
	lineNum++ // Section header

	d.statusInfoFields = append(d.statusInfoFields, d.client.GetAPIEndpoint())
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	d.statusInfoFields = append(d.statusInfoFields, "Connected ✅")
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)

	// Select first field if available
	if len(d.statusInfoFields) > 0 {
		d.statusInfoSelectedRow = 0
	}
}

// handleStatusInfoNavigation handles navigation within the status info overlay
func (d *DashboardView) handleStatusInfoNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if d.statusInfoSelectedRow < len(d.statusInfoFields)-1 {
			d.statusInfoSelectedRow++
		}
		return d, nil
	case "k", "up":
		if d.statusInfoSelectedRow > 0 {
			d.statusInfoSelectedRow--
		}
		return d, nil
	case "g":
		d.statusInfoSelectedRow = 0
		return d, nil
	case "G":
		if len(d.statusInfoFields) > 0 {
			d.statusInfoSelectedRow = len(d.statusInfoFields) - 1
		}
		return d, nil
	case "y":
		// Copy selected field value
		if d.statusInfoSelectedRow >= 0 && d.statusInfoSelectedRow < len(d.statusInfoFields) {
			textToCopy := d.statusInfoFields[d.statusInfoSelectedRow]
			if err := d.copyToClipboard(textToCopy); err == nil {
				// Show what's actually copied, truncated for display
				displayText := textToCopy
				maxLen := 30
				if len(displayText) > maxLen {
					displayText = displayText[:maxLen-3] + "..."
				}
				d.copiedMessage = fmt.Sprintf("📋 Copied \"%s\"", displayText)
			} else {
				d.copiedMessage = "✗ Failed to copy"
			}
			d.copiedMessageTime = time.Now()
			d.yankBlink = true
			return d, tea.Batch(
				d.startYankBlinkAnimation(),
				d.startClearStatusTimer(),
			)
		}
		return d, nil
	case "s", "q", "b", "escape":
		// Close the overlay with q, s, b, or ESC
		d.showStatusInfo = false
		return d, nil
	case "Q":
		// Capital Q to force quit from anywhere
		return d, tea.Quit
	default:
		// Ignore other keys while in status info
		return d, nil
	}
}

// renderStatusInfo renders the status/user info overlay
func (d *DashboardView) renderStatusInfo() string {
	// Calculate box dimensions - leave room for statusline at bottom
	boxWidth := d.width - 4   // Leave 2 chars margin on each side
	boxHeight := d.height - 3 // Leave room for statusline at bottom

	// Box style with rounded border
	boxStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	// Title bar (inside the box)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("63")).
		Width(boxWidth-2). // Account for border
		Align(lipgloss.Center).
		Padding(0, 1)

	title := titleStyle.Render("System Status & User Information")

	// Content styles
	contentStyle := lipgloss.NewStyle().
		Width(boxWidth-2). // Account for border
		Padding(1, 2)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		MarginTop(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(25)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	var content []string
	lineNum := 0
	fieldIdx := 0

	// Helper to render a field line with highlight if selected
	renderField := func(label, value string, isField bool) string {
		renderedValue := value

		// Check if this field value should be highlighted
		if isField && fieldIdx < len(d.statusInfoFieldLines) && lineNum == d.statusInfoFieldLines[fieldIdx] {
			if fieldIdx == d.statusInfoSelectedRow {
				// Custom blinking: toggle between bright and normal colors
				if d.copiedMessage != "" && time.Since(d.copiedMessageTime) < 250*time.Millisecond {
					if d.yankBlink {
						// Bright green when visible
						highlightStyle := lipgloss.NewStyle().
							Background(lipgloss.Color("82")). // Bright green
							Foreground(lipgloss.Color("0")).  // Black text
							Bold(true)
						renderedValue = highlightStyle.Render(value)
					} else {
						// Normal highlight when "off"
						highlightStyle := lipgloss.NewStyle().
							Background(lipgloss.Color("63")).
							Foreground(lipgloss.Color("255"))
						renderedValue = highlightStyle.Render(value)
					}
				} else {
					// Normal focused highlight (no blinking)
					highlightStyle := lipgloss.NewStyle().
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("255"))
					renderedValue = highlightStyle.Render(value)
				}
			} else {
				// Apply normal value style
				renderedValue = valueStyle.Render(value)
			}
			fieldIdx++
		} else if isField {
			// Apply normal value style for non-selected fields
			renderedValue = valueStyle.Render(value)
		}

		lineNum++
		// Return the label with the (potentially highlighted) value
		return fmt.Sprintf("%s%s", labelStyle.Render(label), renderedValue)
	}

	// User Info Section
	if d.userInfo != nil {
		content = append(content, sectionStyle.Render("User Information"))
		lineNum++

		// Show name if available
		if d.userInfo.Name != "" {
			content = append(content, renderField("Name:", d.userInfo.Name, true))
		}

		// Show email
		if d.userInfo.Email != "" {
			content = append(content, renderField("Email:", d.userInfo.Email, true))
		}

		// Show GitHub username if available
		if d.userInfo.GithubUsername != "" {
			content = append(content, renderField("GitHub:", d.userInfo.GithubUsername, true))
		}

		// Show tier with better formatting
		tierDisplay := d.userInfo.Tier
		if tierDisplay == "" {
			tierDisplay = "Free"
		} else {
			// Capitalize first letter
			tierDisplay = strings.ToUpper(tierDisplay[:1]) + tierDisplay[1:]
		}
		content = append(content, renderField("Account Tier:", tierDisplay, true))

		// Show remaining runs with tier details if available
		var runsRemaining string
		var totalProRuns, totalPlanRuns int

		// Hardcoded tier totals
		// Check if tier contains "free" or "Free" (handles "Free Plan v1", etc.)
		tierLower := strings.ToLower(d.userInfo.Tier)
		if strings.Contains(tierLower, "free") || tierLower == "" {
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

		if d.userInfo.TierDetails != nil {
			// Handle cases where admin credits exceed defaults
			actualProTotal := totalProRuns
			actualPlanTotal := totalPlanRuns

			// If remaining runs exceed the default total, admin has credited extra
			if d.userInfo.TierDetails.RemainingProRuns > totalProRuns {
				actualProTotal = d.userInfo.TierDetails.RemainingProRuns
			}
			if d.userInfo.TierDetails.RemainingPlanRuns > totalPlanRuns {
				actualPlanTotal = d.userInfo.TierDetails.RemainingPlanRuns
			}

			runsRemaining = fmt.Sprintf("%d/%d Pro | %d/%d Plan",
				d.userInfo.TierDetails.RemainingProRuns,
				actualProTotal,
				d.userInfo.TierDetails.RemainingPlanRuns,
				actualPlanTotal)
		} else {
			runsRemaining = fmt.Sprintf("%d / %d",
				d.userInfo.RemainingRuns,
				d.userInfo.TotalRuns)
		}
		content = append(content, renderField("Runs Remaining:", runsRemaining, true))

		// Show usage percentage with visual bar
		if d.userInfo.TierDetails != nil {
			// Calculate usage for Pro runs
			proUsed := totalProRuns - d.userInfo.TierDetails.RemainingProRuns
			// Handle admin credits that exceed defaults
			if d.userInfo.TierDetails.RemainingProRuns > totalProRuns {
				proUsed = 0 // No usage if credited beyond default
			}
			proPercentage := 0.0
			if totalProRuns > 0 {
				proPercentage = float64(proUsed) / float64(totalProRuns) * 100
			}

			// Calculate usage for Plan runs
			planUsed := totalPlanRuns - d.userInfo.TierDetails.RemainingPlanRuns
			// Handle admin credits that exceed defaults
			if d.userInfo.TierDetails.RemainingPlanRuns > totalPlanRuns {
				planUsed = 0 // No usage if credited beyond default
			}
			planPercentage := 0.0
			if totalPlanRuns > 0 {
				planPercentage = float64(planUsed) / float64(totalPlanRuns) * 100
			}

			// Create visual bars
			barWidth := 10

			// Pro bar
			proFilledBars := int(proPercentage / 100 * float64(barWidth))
			if proFilledBars < 0 {
				proFilledBars = 0
			}
			if proFilledBars > barWidth {
				proFilledBars = barWidth
			}
			proEmptyBars := barWidth - proFilledBars
			if proEmptyBars < 0 {
				proEmptyBars = 0
			}
			proBar := strings.Repeat("█", proFilledBars) + strings.Repeat("░", proEmptyBars)

			// Plan bar
			planFilledBars := int(planPercentage / 100 * float64(barWidth))
			if planFilledBars < 0 {
				planFilledBars = 0
			}
			if planFilledBars > barWidth {
				planFilledBars = barWidth
			}
			planEmptyBars := barWidth - planFilledBars
			if planEmptyBars < 0 {
				planEmptyBars = 0
			}
			planBar := strings.Repeat("█", planFilledBars) + strings.Repeat("░", planEmptyBars)

			usageValue := fmt.Sprintf("Pro: %s %.0f%% | Plan: %s %.0f%%",
				proBar, proPercentage, planBar, planPercentage)
			content = append(content, renderField("Usage:", usageValue, true))
		} else if d.userInfo.TotalRuns > 0 {
			// Fallback to legacy display if no tier details
			usedRuns := d.userInfo.TotalRuns - d.userInfo.RemainingRuns
			percentage := float64(usedRuns) / float64(d.userInfo.TotalRuns) * 100
			barWidth := 20
			filledBars := int(percentage / 100 * float64(barWidth))
			if filledBars < 0 {
				filledBars = 0
			}
			if filledBars > barWidth {
				filledBars = barWidth
			}
			emptyBars := barWidth - filledBars
			if emptyBars < 0 {
				emptyBars = 0
			}
			bar := strings.Repeat("█", filledBars) + strings.Repeat("░", emptyBars)

			usageValue := fmt.Sprintf("%s %.1f%%", bar, percentage)
			content = append(content, renderField("Usage:", usageValue, true))
		} else {
			// Handle unlimited or zero total runs
			content = append(content, renderField("Usage:", "Unlimited", true))
		}
	} else {
		content = append(content, sectionStyle.Render("User Information"))
		lineNum++
		content = append(content, "Loading user info...")
		lineNum++
	}

	// System Stats Section
	content = append(content, sectionStyle.Render("Dashboard Statistics"))
	lineNum++

	content = append(content, renderField("Repositories:", fmt.Sprintf("%d", len(d.repositories)), true))
	content = append(content, renderField("Total Runs:", fmt.Sprintf("%d", len(d.allRuns)), true))

	// Count run statuses
	var running, completed, failed int
	for _, run := range d.allRuns {
		switch run.Status {
		case "running", "pending":
			running++
		case "completed", "success":
			completed++
		case "failed", "error":
			failed++
		}
	}

	runStatus := fmt.Sprintf("🔄 %d  ✅ %d  ❌ %d", running, completed, failed)
	content = append(content, renderField("Run Status:", runStatus, true))

	// Last refresh time
	if !d.lastDataRefresh.IsZero() {
		timeSince := time.Since(d.lastDataRefresh)
		refreshText := fmt.Sprintf("%d seconds ago", int(timeSince.Seconds()))
		if timeSince.Minutes() > 1 {
			refreshText = fmt.Sprintf("%.1f minutes ago", timeSince.Minutes())
		}
		content = append(content, renderField("Last Refresh:", refreshText, true))
	}

	// Connection Info Section
	content = append(content, sectionStyle.Render("Connection Info"))
	lineNum++

	content = append(content, renderField("API Endpoint:", d.client.GetAPIEndpoint(), true))
	content = append(content, renderField("Status:", "Connected ✅", true))

	// Build the main content
	mainContent := contentStyle.Render(strings.Join(content, "\n"))

	// Calculate remaining height for spacing inside the box
	innerHeight := boxHeight - 2                                           // Account for border
	contentHeight := lipgloss.Height(title) + lipgloss.Height(mainContent) // No statusline inside now
	remainingHeight := innerHeight - contentHeight
	spacing := ""
	if remainingHeight > 0 {
		spacing = strings.Repeat("\n", remainingHeight)
	}

	// Join everything together inside the box (without statusline)
	innerContent := lipgloss.JoinVertical(lipgloss.Left, title, mainContent, spacing)

	// Wrap in the box
	boxedContent := boxStyle.Render(innerContent)

	// Center the box on screen (leaving room for statusline)
	centeredBox := lipgloss.Place(d.width, d.height-1, lipgloss.Center, lipgloss.Center, boxedContent)

	// Create the statusline using the dashboard statusline component
	var statusLine string
	shortHelp := "[j/k]navigate [y]copy [s/q/b/ESC]back [Q]uit"

	// Show copied message prominently if recent
	if d.copiedMessage != "" && time.Since(d.copiedMessageTime) < 250*time.Millisecond {
		// Use the renderStatusLine method which handles copied messages
		statusLine = d.renderStatusLine("STATUS")
	} else {
		// Use components.DashboardStatusLine for consistent formatting
		statusLine = components.DashboardStatusLine(d.width, "STATUS", "", shortHelp)
	}

	// Join the centered box and statusline
	return lipgloss.JoinVertical(lipgloss.Left, centeredBox, statusLine)
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

// wrapTextWithLimit wraps text to fit within width and max lines
func (d *DashboardView) wrapTextWithLimit(text string, width int, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return []string{}
	}

	// First wrap normally
	lines := d.wrapText(text, width)

	// If it fits within maxLines, return as is
	if len(lines) <= maxLines {
		return lines
	}

	// Truncate to maxLines with ellipsis
	result := lines[:maxLines-1]
	lastLine := lines[maxLines-1]
	if len(lastLine) > width-5 {
		lastLine = lastLine[:width-5]
	}
	result = append(result, lastLine+" (...)")

	return result
}

// renderNotificationLine renders a notification line if there's a message to show
func (d *DashboardView) renderNotificationLine() string {
	// If we're showing a status message in the status line, don't show notification
	if d.statusMessage != "" && time.Since(d.statusMessageTime) < 1*time.Second {
		return ""
	}

	if d.copiedMessage == "" || time.Since(d.copiedMessageTime) >= 250*time.Millisecond {
		return ""
	}

	var notificationStyle lipgloss.Style
	if time.Since(d.copiedMessageTime) < 250*time.Millisecond {
		if d.yankBlink {
			// Bright and bold when visible
			notificationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Background(lipgloss.Color("235")).
				Bold(true).
				Width(d.width)
		} else {
			// Dimmer when "off" for blinking effect
			notificationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Background(lipgloss.Color("235")).
				Width(d.width)
		}
	} else {
		// After blinking period, show normally
		notificationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Background(lipgloss.Color("235")).
			Bold(true).
			Width(d.width)
	}

	return notificationStyle.Render(" " + d.copiedMessage)
}

// hasCurrentSelectionURL checks if the current selection contains a URL or can generate a RepoBird URL
func (d *DashboardView) hasCurrentSelectionURL() bool {
	switch d.focusedColumn {
	case 0: // Repository column - check if we have API repository data with URLs
		if d.selectedRepoIdx < len(d.repositories) {
			repo := d.repositories[d.selectedRepoIdx]
			apiRepo := d.getAPIRepositoryForRepo(&repo)
			return apiRepo != nil && apiRepo.RepoURL != ""
		}
		return false
	case 1: // Runs column - check for PR URL
		if d.selectedRunIdx < len(d.filteredRuns) {
			run := d.filteredRuns[d.selectedRunIdx]
			return run.PrURL != nil && *run.PrURL != ""
		}
	case 2: // Details column - check if selected line contains URL or can generate RepoBird URL
		if d.selectedDetailLine < len(d.detailLinesOriginal) {
			lineText := d.detailLinesOriginal[d.selectedDetailLine]
			if utils.IsURL(lineText) {
				return true
			}
			// Check if this is the ID field (first line) and we can generate a RepoBird URL
			if d.selectedDetailLine == 0 && d.selectedRunData != nil {
				runID := d.selectedRunData.GetIDString()
				return utils.IsNonEmptyNumber(runID)
			}
			// Check if this is the repository line (line 2) and we have repository data
			if d.selectedDetailLine == 2 && d.selectedRunData != nil {
				repoName := d.selectedRunData.GetRepositoryName()
				if repoName != "" {
					// Find the corresponding Repository object and check if it has URLs
					repo := d.getRepositoryByName(repoName)
					if repo != nil {
						apiRepo := d.getAPIRepositoryForRepo(repo)
						return apiRepo != nil && apiRepo.RepoURL != ""
					}
				}
			}
		}
	}
	return false
}

// renderStatusLine renders the universal status line
func (d *DashboardView) renderStatusLine(layoutName string) string {
	// Check for temporary status message (URL opening, copying, etc.) - highest priority
	if d.statusMessage != "" && time.Since(d.statusMessageTime) < 1*time.Second {
		// Create green status line for successful operations or red for errors
		var statusStyle lipgloss.Style
		if strings.Contains(d.statusMessage, "✗") {
			// Red for errors
			statusStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("196")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1)
		} else {
			// Green for success
			statusStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("82")).
				Foreground(lipgloss.Color("232")).
				Padding(0, 1)
		}

		statusLine := components.NewStatusLine().
			SetWidth(d.width).
			SetLeft(fmt.Sprintf("[%s]", layoutName)).
			SetRight("").
			SetHelp(d.statusMessage).
			SetStyle(statusStyle)

		return statusLine.Render()
	}

	// Always show normal status line now - notifications are shown above
	// Data freshness indicator - keep it very short
	dataInfo := ""
	if d.loading && len(d.repositories) > 0 {
		dataInfo = "loading"
	} else if !d.lastDataRefresh.IsZero() {
		elapsed := time.Since(d.lastDataRefresh)
		if elapsed < time.Minute {
			dataInfo = "fresh"
		} else {
			dataInfo = fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
		}
	}

	// Handle URL selection prompt
	if d.showURLSelectionPrompt {
		promptHelp := "Open URL: (o)RepoBird (g)GitHub [ESC]cancel"
		// Create yellow status line for URL selection prompt
		yellowStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("220")).
			Foreground(lipgloss.Color("232")).
			Padding(0, 1)

		statusLine := components.NewStatusLine().
			SetWidth(d.width).
			SetLeft(fmt.Sprintf("[%s]", layoutName)).
			SetRight(dataInfo).
			SetHelp(promptHelp).
			SetStyle(yellowStyle)

		return statusLine.Render()
	}

	// Compact help text
	shortHelp := "n:new f:fuzzy s:status y:copy ?:help r:refresh q:quit"
	if d.showHelp {
		shortHelp = "n:new f:fuzzy s:status y:copy j/k:↑↓ h/l:←→ Enter:→ BS:← ?:help q:quit"
	}

	// Add URL opening hint if current selection has a URL
	if d.hasCurrentSelectionURL() {
		shortHelp = "o:open-url " + shortHelp
	}

	return components.DashboardStatusLine(d.width, layoutName, dataInfo, shortHelp)
}

// activateFZFMode activates FZF mode for the current column
func (d *DashboardView) activateFZFMode() {
	var items []string

	switch d.focusedColumn {
	case 0: // Repository column
		items = make([]string, len(d.repositories))
		for i, repo := range d.repositories {
			statusIcon := d.getRepositoryStatusIcon(&repo)
			items[i] = fmt.Sprintf("%s %s", statusIcon, repo.Name)
		}
		d.fzfColumn = 0

	case 1: // Runs column
		if len(d.filteredRuns) > 0 {
			items = make([]string, len(d.filteredRuns))
			for i, run := range d.filteredRuns {
				statusIcon := d.getRunStatusIcon(run.Status)
				title := run.Title
				if title == "" {
					title = "Untitled"
				}
				items[i] = fmt.Sprintf("%s %s", statusIcon, title)
			}
			d.fzfColumn = 1
		}

	case 2: // Details column
		if len(d.detailLines) > 0 {
			items = d.detailLines
			d.fzfColumn = 2
		}
	}

	if len(items) > 0 {
		// Calculate appropriate width for FZF
		columnWidth := d.width / 3
		if d.focusedColumn == 2 {
			columnWidth = d.width - (2 * (d.width / 3))
		}

		d.fzfMode = components.NewFZFMode(items, columnWidth, 15)
		d.fzfMode.Activate()
	}
}

// renderWithFZFOverlay renders the dashboard with FZF dropdown overlay
func (d *DashboardView) renderWithFZFOverlay(baseView string) string {
	if d.fzfMode == nil || !d.fzfMode.IsActive() {
		return baseView
	}

	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")

	// Calculate position for FZF dropdown based on focused column and selected item
	columnWidth := d.width / 3
	var xOffset int
	var yOffset int

	switch d.fzfColumn {
	case 0: // Repository column
		xOffset = 2
		yOffset = 3 + d.selectedRepoIdx // Position at selected repository
	case 1: // Runs column
		xOffset = columnWidth + 2
		yOffset = 3 + d.selectedRunIdx // Position at selected run
	case 2: // Details column
		xOffset = (2 * columnWidth) + 2
		yOffset = 3 + d.selectedDetailLine // Position at selected detail line
	}

	// Ensure yOffset is within bounds
	if yOffset < 3 {
		yOffset = 3
	}
	if yOffset > len(baseLines)-15 {
		yOffset = len(baseLines) - 15
	}

	// Create FZF dropdown view
	fzfView := d.fzfMode.View()
	fzfLines := strings.Split(fzfView, "\n")

	// Create a new view with the FZF dropdown overlaid
	result := make([]string, len(baseLines))
	copy(result, baseLines)

	// Insert FZF dropdown at the calculated position
	for i, fzfLine := range fzfLines {
		lineIdx := yOffset + i
		if lineIdx >= 0 && lineIdx < len(result) {
			// Create the overlay line by combining base content and FZF dropdown
			if xOffset < len(result[lineIdx]) {
				// Preserve part of the base line before the dropdown
				basePart := ""
				if xOffset > 0 {
					minLen := xOffset
					if len(result[lineIdx]) < minLen {
						minLen = len(result[lineIdx])
					}
					basePart = result[lineIdx][:minLen]
				}
				// Add the FZF line
				result[lineIdx] = basePart + fzfLine
			} else {
				// Line is shorter than offset, pad and add FZF
				padding := strings.Repeat(" ", xOffset-len(result[lineIdx]))
				result[lineIdx] = result[lineIdx] + padding + fzfLine
			}
		}
	}

	return strings.Join(result, "\n")
}

// getAPIRepositoryForRepo finds the corresponding APIRepository for a Repository
func (d *DashboardView) getAPIRepositoryForRepo(repo *models.Repository) *models.APIRepository {
	if repo == nil || d.apiRepositories == nil {
		return nil
	}

	// Find matching API repository by name
	for _, apiRepo := range d.apiRepositories {
		apiRepoName := apiRepo.Name
		if apiRepoName == "" {
			apiRepoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
		}
		if apiRepoName == repo.Name {
			return &apiRepo
		}
	}

	return nil
}

// getRepositoryByName finds a Repository object by name
func (d *DashboardView) getRepositoryByName(name string) *models.Repository {
	if name == "" || len(d.repositories) == 0 {
		return nil
	}

	for i := range d.repositories {
		if d.repositories[i].Name == name {
			return &d.repositories[i]
		}
	}

	return nil
}

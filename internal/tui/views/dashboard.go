package views

import (
	"fmt"
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
	"github.com/repobird/repobird-cli/internal/utils"
)

// DashboardView is the main dashboard controller that manages different layout views
type DashboardView struct {
	client       APIClient
	keys         components.KeyMap
	help         help.Model
	disabledKeys map[string]bool // Keys that are disabled for this view

	// Dashboard state
	currentLayout      models.LayoutType
	showStatusInfo     bool // Show status/user info overlay
	showDocs           bool // Show documentation overlay
	selectedRepo       *models.Repository
	selectedRepoIdx    int
	selectedRunIdx     int
	focusedColumn      int      // 0: repositories, 1: runs, 2: details
	selectedDetailLine int      // Selected line in details column
	detailLines        []string // Lines in details column for selection

	// All-runs layout using shared component
	allRunsList *components.ScrollableList

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
	detailsCache    map[string]*models.RunResponse // Cached run details

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
	clipboardManager  components.ClipboardManager

	// Unified status line component
	statusLine *components.StatusLine

	// Store original untruncated detail lines for copying
	detailLinesOriginal []string

	// Status info overlay navigation
	statusInfoSelectedRow int      // Currently selected row in status info
	statusInfoFields      []string // Field values that can be copied
	statusInfoFieldLines  []int    // Line numbers for each field
	statusInfoKeyOffset   int      // Horizontal scroll offset for keys
	statusInfoValueOffset int      // Horizontal scroll offset for values
	statusInfoFocusColumn int      // 0 = key column, 1 = value column
	statusInfoKeys        []string // Full key text for each field

	// URL selection for repositories
	showURLSelectionPrompt bool                  // Show URL selection prompt in status line
	pendingRepoForURL      *models.Repository    // Repository pending URL selection
	pendingAPIRepoForURL   *models.APIRepository // API repository data for URL generation

	// Vim keybinding state for 'gg' command
	lastGPressTime time.Time // Time when 'g' was last pressed
	waitingForG    bool      // Whether we're waiting for second 'g' in 'gg' command

	// Documentation overlay state
	docsCurrentPage int
	docsSelectedRow int

	// New scrollable help view
	helpView *components.HelpView

	// Viewports for scrolling
	repoViewport    viewport.Model
	runsViewport    viewport.Model
	detailsViewport viewport.Model

	// Embedded cache (no globals!)
	cache *cache.SimpleCache
}

// Message types are defined in dashboard_messages.go

// NewDashboardViewWithState creates a new dashboard view with restored state
func NewDashboardViewWithState(client APIClient, selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn int) *DashboardView {
	dashboard := NewDashboardView(client)
	// Set the state that will be restored after data loads
	dashboard.selectedRepoIdx = selectedRepoIdx
	dashboard.selectedRunIdx = selectedRunIdx
	dashboard.selectedDetailLine = selectedDetailLine
	dashboard.focusedColumn = focusedColumn
	return dashboard
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(client APIClient) *DashboardView {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	dashboard := &DashboardView{
		client:           client,
		keys:             components.DefaultKeyMap,
		help:             help.New(),
		disabledKeys:     map[string]bool{"esc": true}, // Disable escape key on dashboard (b is for bulk navigation)
		currentLayout:    models.LayoutTripleColumn,
		loading:          true,
		initializing:     true,
		refreshInterval:  30 * time.Second,
		apiRepositories:  make(map[int]models.APIRepository),
		fzfColumn:        -1, // No FZF mode initially
		spinner:          s,
		statusLine:       components.NewStatusLine(),
		helpView:         components.NewHelpView(),
		clipboardManager: components.NewClipboardManager(),
		repoViewport:     viewport.New(0, 0), // Will be sized in Update
		runsViewport:     viewport.New(0, 0),
		detailsViewport:  viewport.New(0, 0),
		cache:            cache.NewSimpleCache(), // Embedded cache
	}

	// Load persisted cache data if available
	_ = dashboard.cache.LoadFromDisk()

	// Initialize shared scrollable list component for all-runs layout
	dashboard.allRunsList = components.NewScrollableList(
		components.WithColumns(4), // ID, Repository, Status, Created
		components.WithValueNavigation(true),
		components.WithKeymaps(components.DefaultKeyMap),
	)

	return dashboard
}

// IsKeyDisabled implements the CoreViewKeymap interface
func (d *DashboardView) IsKeyDisabled(keyString string) bool {
	disabled := d.disabledKeys[keyString]
	debug.LogToFilef("🔍 IsKeyDisabled('%s'): map=%v, result=%t 🔍\n", keyString, d.disabledKeys, disabled)
	return disabled
}

// HandleKey implements the CoreViewKeymap interface
func (d *DashboardView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	// Dashboard handles 'b' specially for bulk view navigation (overrides back navigation)
	if keyMsg.String() == "b" {
		// Navigate to bulk view
		return true, d, func() tea.Msg {
			return messages.NavigateToBulkMsg{}
		}
	}
	// Let the centralized system handle everything else
	return false, d, nil
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
		d.syncFileHashes(),
		// Initialize shared all-runs list component
		d.allRunsList.Init(),
		d.spinner.Tick,
	)
}

// syncFileHashesMsg is a message indicating file hash sync completed
// syncFileHashesMsg is defined in dashboard_messages.go

// syncFileHashes syncs file hashes from the API on startup
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Debug log all incoming messages
	debug.LogToFilef("\n[DASHBOARD UPDATE] Received message type: %T\n", msg)
	debug.LogToFilef("  Loading: %v, Initializing: %v\n", d.loading, d.initializing)

	// Always handle quit keys regardless of loading state
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		debug.LogToFilef("  Key pressed: %s (Type: %v)\n", keyMsg.String(), keyMsg.Type)
		// Handle force quit regardless of state
		if keyMsg.String() == "Q" || (keyMsg.Type == tea.KeyCtrlC) {
			debug.LogToFilef("  FORCE QUIT requested\n")
			d.cache.SaveToDisk()
			return d, tea.Quit
		}
		// Handle normal quit when not in special modes
		if keyMsg.String() == "q" && !d.showStatusInfo && !d.showDocs && !d.showURLSelectionPrompt && d.fzfMode == nil {
			debug.LogToFilef("  Normal quit requested\n")
			d.cache.SaveToDisk()
			return d, tea.Quit
		}
	}

	switch msg := msg.(type) {
	case nil:
		// Handle nil messages gracefully to prevent freezing
		debug.LogToFilef("  WARNING: Received nil message, ignoring\n")
		return d, nil

	case spinner.TickMsg:
		if d.loading || d.initializing {
			var cmd tea.Cmd
			d.spinner, cmd = d.spinner.Update(msg)
			// Also update the status line spinner
			d.statusLine.UpdateSpinner()
			// Don't return early - continue processing other messages
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

		// Update help view size
		if d.helpView != nil {
			d.helpView.SetSize(msg.Width, msg.Height)
		}

		// Update shared list component dimensions
		if d.allRunsList != nil {
			d.allRunsList.Update(msg)
		}

		// Update viewport sizes for Miller columns
		d.updateViewportSizes()

	case dashboardDataLoadedMsg:
		debug.LogToFilef("\n[DASHBOARD DATA LOADED MSG RECEIVED]\n")
		d.loading = false
		d.initializing = false
		if msg.error != nil {
			debug.LogToFilef("  ERROR: %v\n", msg.error)
			d.error = msg.error
		} else {
			debug.LogToFilef("  Repositories loaded: %d\n", len(msg.repositories))
			debug.LogToFilef("  Total runs loaded: %d\n", len(msg.allRuns))
			debug.LogToFilef("  Details cache loaded: %d\n", len(msg.detailsCache))

			// Debug: Show repository names
			debug.LogToFilef("  Repository list:\n")
			for i, repo := range msg.repositories {
				debug.LogToFilef("    [%d] '%s'\n", i, repo.Name)
			}

			d.repositories = msg.repositories
			d.allRuns = msg.allRuns
			d.detailsCache = msg.detailsCache
			d.lastDataRefresh = time.Now()

			// Update viewport sizes based on window
			d.updateViewportSizes()

			// Select first repository by default, or restore saved state
			if len(d.repositories) > 0 {
				// Check if we have saved state to restore
				if d.selectedRepoIdx >= 0 && d.selectedRepoIdx < len(d.repositories) {
					// Restore saved repository selection
					d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				} else {
					// Default to first repository
					d.selectedRepo = &d.repositories[0]
					d.selectedRepoIdx = 0
				}
				cmds = append(cmds, d.selectRepository(d.selectedRepo))
			}
		}

	case dashboardRepositorySelectedMsg:
		d.selectedRepo = msg.repository
		d.filteredRuns = msg.runs

		// Update viewport content when repository changes
		d.updateViewportContent()

		// Select first run by default, or restore saved state
		if len(d.filteredRuns) > 0 {
			// Check if we have saved run state to restore
			if d.selectedRunIdx >= 0 && d.selectedRunIdx < len(d.filteredRuns) {
				// Restore saved run selection
				d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
			} else {
				// Default to first run
				d.selectedRunData = d.filteredRuns[0]
				d.selectedRunIdx = 0
			}
			d.updateDetailLines()
			// Restore detail line selection if available after detail lines are updated
			if d.selectedDetailLine >= 0 && d.selectedDetailLine < len(d.detailLines) {
				// Keep the saved selection if it's within bounds
			} else if len(d.detailLines) > 0 {
				// Default to first non-empty line if saved selection is out of bounds
				d.selectedDetailLine = 0
				if d.isEmptyLine(d.detailLines[0]) {
					newIdx := d.findNextNonEmptyLine(-1, 1)
					if newIdx >= 0 && newIdx < len(d.detailLines) {
						d.selectedDetailLine = newIdx
					}
				}
			}
		}

	case dashboardUserInfoLoadedMsg:
		if msg.error == nil && msg.userInfo != nil {
			d.userInfo = msg.userInfo
			// Store user ID (no need to reinitialize embedded cache)
			if d.userID == nil || (d.userID != nil && *d.userID != msg.userInfo.ID) {
				d.userID = &msg.userInfo.ID
				// Each view has its own cache instance, no global initialization needed
			}
		}

	case syncFileHashesMsg:
		// File hash sync completed, no action needed
		debug.LogToFilef("  File hash sync completed\n")

	case components.ClipboardBlinkMsg:
		// Handle clipboard blink animation
		var clipCmd tea.Cmd
		d.clipboardManager, clipCmd = d.clipboardManager.Update(msg)
		return d, clipCmd

	case messageClearMsg:
		// Trigger UI refresh when message expires (no action needed - just refresh)

	case gKeyTimeoutMsg:
		// Cancel waiting for second 'g' after timeout
		d.waitingForG = false

	case clearStatusMsg:
		// Clear the clipboard message after timeout
		d.copiedMessage = ""
		d.clipboardManager.Reset()

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
		debug.LogToFilef("🔑 DASHBOARD KEYMSG: key='%s' 🔑\n", msg.String())
		debug.LogToFilef("🔒 IsKeyDisabled result: %t 🔒\n", d.IsKeyDisabled(msg.String()))

		// If FZF mode is active, handle input there first
		if d.fzfMode != nil && d.fzfMode.IsActive() {
			debug.LogToFilef("FZF mode is active, delegating to FZF\n")
			newFzf, cmd := d.fzfMode.Update(msg)
			d.fzfMode = newFzf
			return d, cmd
		}

		// Check if this key is disabled by the CoreViewKeymap interface
		if d.IsKeyDisabled(msg.String()) {
			debug.LogToFilef("🚫 DASHBOARD: Key '%s' is DISABLED - IGNORING 🚫\n", msg.String())
			// Key is disabled - ignore it completely
			return d, nil
		}

		debug.LogToFilef("✅ DASHBOARD: Key '%s' is NOT disabled, proceeding with local handling ✅\n", msg.String())

		// SPECIAL CASE: 'b' from dashboard should go to BULK view, not back
		if msg.String() == "b" {
			debug.LogToFilef("🎯 DASHBOARD: 'b' key detected - navigating to BULK view 🎯\n")
			cmds = append(cmds, func() tea.Msg {
				return messages.NavigateToBulkMsg{}
			})
			// Continue processing to ensure command gets executed
		}

		// Handle dashboard-specific keys
		switch {
		case msg.Type == tea.KeyEsc && d.showURLSelectionPrompt:
			// Close URL selection prompt with ESC
			d.showURLSelectionPrompt = false
			d.pendingRepoForURL = nil
			d.pendingAPIRepoForURL = nil
			return d, nil
		case d.showURLSelectionPrompt && msg.Type == tea.KeyRunes && string(msg.Runes) == "o":
			// Handle RepoBird URL selection
			if d.pendingAPIRepoForURL != nil {
				urlText := fmt.Sprintf("https://repobird.ai/repos/%d", d.pendingAPIRepoForURL.ID)
				message := "🌐 Opened RepoBird URL in browser"

				// Clear the prompt
				d.showURLSelectionPrompt = false
				d.pendingRepoForURL = nil
				d.pendingAPIRepoForURL = nil

				if err := utils.OpenURL(urlText); err == nil {
					d.statusLine.SetTemporaryMessageWithType(message, components.MessageSuccess, 1*time.Second)
				} else {
					d.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
				}
				return d, d.startMessageClearTimer(1 * time.Second)
			}
			return d, nil
		case d.showURLSelectionPrompt && msg.Type == tea.KeyRunes && string(msg.Runes) == "g":
			// Handle GitHub URL selection
			if d.pendingAPIRepoForURL != nil {
				urlText := d.pendingAPIRepoForURL.RepoURL
				message := "🌐 Opened GitHub URL in browser"

				// Clear the prompt
				d.showURLSelectionPrompt = false
				d.pendingRepoForURL = nil
				d.pendingAPIRepoForURL = nil

				if err := utils.OpenURL(urlText); err == nil {
					d.statusLine.SetTemporaryMessageWithType(message, components.MessageSuccess, 1*time.Second)
				} else {
					d.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
				}
				return d, d.startMessageClearTimer(1 * time.Second)
			}
			return d, nil
		case d.showURLSelectionPrompt:
			// Block all other keys when URL prompt is active
			// Enter key cancels the prompt
			if key.Matches(msg, d.keys.Enter) {
				d.showURLSelectionPrompt = false
				d.pendingRepoForURL = nil
				d.pendingAPIRepoForURL = nil
			}
			// Block all other keys by returning early
			return d, nil
		case msg.Type == tea.KeyEsc && d.showStatusInfo:
			// Close status info overlay with ESC
			d.showStatusInfo = false
			return d, nil
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "s" && !d.showStatusInfo:
			// Navigate to status view
			debug.LogToFilef("🏥 DASHBOARD: 's' key detected - navigating to STATUS view 🏥\n")
			cmds = append(cmds, func() tea.Msg {
				return messages.NavigateToStatusMsg{}
			})
			return d, tea.Batch(cmds...)
		case d.showDocs:
			// Handle navigation in help overlay
			return d.handleHelpNavigation(msg)
		case d.showStatusInfo:
			// Handle navigation in status info overlay
			return d.handleStatusInfoNavigation(msg)
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "n":
			// Navigate to create new run view
			var selectedRepository string
			if d.selectedRepo != nil {
				selectedRepository = d.selectedRepo.Name
			}

			// Return navigation message to create view
			return d, func() tea.Msg {
				return messages.NavigateToCreateMsg{
					SelectedRepository: selectedRepository,
				}
			}
		case key.Matches(msg, d.keys.Enter) && d.currentLayout == models.LayoutTripleColumn && d.focusedColumn == 2 && d.selectedRunData != nil:
			// If we're in the details column (column 2) in the triple column layout, open the full details view
			// Convert []*models.RunResponse to []models.RunResponse
			runs := make([]models.RunResponse, len(d.allRuns))
			for i, run := range d.allRuns {
				if run != nil {
					runs[i] = *run
				}
			}

			// Navigate to details view
			return d, func() tea.Msg {
				return messages.NavigateToDetailsMsg{
					RunID: d.selectedRunData.GetIDString(),
				}
			}
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
			// Toggle docs overlay
			d.showDocs = true
			d.docsCurrentPage = 0
			d.docsSelectedRow = 0
			return d, nil
		case key.Matches(msg, d.keys.Quit):
			// Save cache to disk before quitting
			_ = d.cache.SaveToDisk()
			d.cache.Stop()
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
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "v":
			// Navigate to file viewer
			return d, func() tea.Msg {
				return messages.NavigateToFileViewerMsg{}
			}
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "G":
			// Vim: Go to bottom of current column
			d.waitingForG = false // Cancel any pending 'gg' command
			switch d.focusedColumn {
			case 0: // Repository column
				if len(d.repositories) > 0 {
					d.selectedRepoIdx = len(d.repositories) - 1
					d.selectedRepo = &d.repositories[d.selectedRepoIdx]
					return d, d.selectRepository(d.selectedRepo)
				}
			case 1: // Runs column
				if len(d.filteredRuns) > 0 {
					d.selectedRunIdx = len(d.filteredRuns) - 1
					d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
					d.updateDetailLines()
				}
			case 2: // Details column
				if len(d.detailLines) > 0 {
					d.selectedDetailLine = len(d.detailLines) - 1
				}
			}
			return d, nil
		case msg.Type == tea.KeyRunes && string(msg.Runes) == "g":
			// Check for URL selection prompt first
			if d.showURLSelectionPrompt {
				// This 'g' is for GitHub URL selection, handled above
				return d, nil
			}

			if d.waitingForG {
				// This is the second 'g' in 'gg' - go to top
				d.waitingForG = false
				switch d.focusedColumn {
				case 0: // Repository column
					if len(d.repositories) > 0 {
						d.selectedRepoIdx = 0
						d.selectedRepo = &d.repositories[0]
						return d, d.selectRepository(d.selectedRepo)
					}
				case 1: // Runs column
					if len(d.filteredRuns) > 0 {
						d.selectedRunIdx = 0
						d.selectedRunData = d.filteredRuns[0]
						d.updateDetailLines()
					}
				case 2: // Details column
					if len(d.detailLines) > 0 {
						d.selectedDetailLine = 0
					}
				}
			} else {
				// First 'g' pressed - wait for second 'g'
				d.waitingForG = true
				d.lastGPressTime = time.Now()
				// Start a timer to cancel the 'gg' command after 1 second
				return d, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
					return gKeyTimeoutMsg{}
				})
			}
			return d, nil
		default:
			// Handle navigation in Miller Columns layout
			switch d.currentLayout {
			case models.LayoutTripleColumn:
				cmd := d.handleMillerColumnsNavigation(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case models.LayoutAllRuns:
				// Handle navigation with shared scrollable list
				d.allRunsList.Update(msg)
				// Handle selection actions for all-runs layout
				if key.Matches(msg, d.keys.Enter) {
					selected := d.allRunsList.GetSelected()
					if len(selected) > 0 && selected[0] != "" {
						// Navigate to details view with selected run ID
						return d, func() tea.Msg {
							return messages.NavigateToDetailsMsg{
								RunID: selected[0], // First column is run ID
							}
						}
					}
				}
			}
		}
	default:
		// Handle other messages for all-runs layout
		if d.currentLayout == models.LayoutAllRuns && d.allRunsList != nil {
			d.allRunsList.Update(msg)
		}
	}

	// CRITICAL: Check if this was a KeyMsg that wasn't handled by dashboard's local logic
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		debug.LogToFilef("🔍 DASHBOARD END: KeyMsg '%s' reached end of dashboard Update method 🔍\n", keyMsg.String())
		debug.LogToFilef("⚠️ DASHBOARD END: This key was NOT handled by dashboard locally! ⚠️\n")
		debug.LogToFilef("🚨 PROBLEM: Dashboard always returns here - key will NOT bubble up to App! 🚨\n")
	}

	return d, tea.Batch(cmds...)
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

	if d.error != nil {
		content = fmt.Sprintf("Error loading dashboard data: %s\n\nPress 'r' to retry, 'q' to quit", d.error.Error())
		statusline := d.renderStatusLine("DASH")
		return lipgloss.JoinVertical(lipgloss.Left, title, content, statusline)
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
		statusline := d.renderStatusLine("DASH")
		return lipgloss.JoinVertical(lipgloss.Left, title, content, statusline)
	}

	if d.loading || d.initializing {
		// ASCII logo for RepoBird AI
		logo := `
██████╗ ███████╗██████╗  ██████╗ ██████╗ ██╗██████╗ ██████╗      █████╗ ██╗
██╔══██╗██╔════╝██╔══██╗██╔═══██╗██╔══██╗██║██╔══██╗██╔══██╗    ██╔══██╗██║
██████╔╝█████╗  ██████╔╝██║   ██║██████╔╝██║██████╔╝██║  ██║    ███████║██║
██╔══██╗██╔══╝  ██╔═══╝ ██║   ██║██╔══██╗██║██╔══██╗██║  ██║    ██╔══██║██║
██║  ██║███████╗██║     ╚██████╔╝██████╔╝██║██║  ██║██████╔╝    ██║  ██║██║
╚═╝  ╚═╝╚══════╝╚═╝      ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═╝╚═════╝     ╚═╝  ╚═╝╚═╝`

		// Use the animated spinner + loading text
		loadingText := d.spinner.View() + " Loading dashboard data..."

		// Calculate available height for content (total - title - status line)
		titleHeight := lipgloss.Height(title)
		statusLineHeight := 1 // Status line is always 1 line
		availableHeight := d.height - titleHeight - statusLineHeight

		// Style for the logo
		logoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")). // Blue color for logo
			Bold(true)

		// Style for loading text
		loadingTextStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")). // Bright cyan color
			Bold(true).
			MarginTop(2) // Add space between logo and loading text

		// Combine logo and loading text
		combinedContent := lipgloss.JoinVertical(
			lipgloss.Center,
			logoStyle.Render(logo),
			loadingTextStyle.Render(loadingText),
		)

		// Center everything vertically and horizontally in the available space
		centerStyle := lipgloss.NewStyle().
			Width(d.width).
			Height(availableHeight).
			Align(lipgloss.Center, lipgloss.Center)
		content = centerStyle.Render(combinedContent)

		// Always show status line even during loading
		statusline := d.renderStatusLine("DASH")
		return lipgloss.JoinVertical(lipgloss.Left, title, content, statusline)
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

	// Overlay help if requested
	if d.showDocs {
		return d.renderHelp()
	}

	// Overlay status info if requested
	if d.showStatusInfo {
		return d.renderStatusInfo()
	}

	// Status line is already included in the layout-specific rendering functions
	return finalView
}

// renderTripleColumnLayout renders the Miller Columns layout with real data

// copyToClipboard copies the given text to clipboard

// renderRepositoriesLayout renders the repositories-only layout

// renderRepositoriesColumn renders the left column with real repositories

// renderRunsColumn renders the center column with runs for selected repository

// renderDetailsColumn renders the right column with run details

// renderStatusInfo renders the status/user info overlay

// renderStatusLine renders the universal status line

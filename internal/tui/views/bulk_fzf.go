package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// BulkViewMode represents different modes of the bulk view
type BulkViewMode int

const (
	BulkModeFileSelect BulkViewMode = iota
	BulkModeRunList
	BulkModeSubmitting
	BulkModeResults
)

// Message types for bulk FZF view
type bulkFZFConfigLoadedMsg struct {
	config *bulk.BulkConfig
	runs   []BulkRunItem
}

type bulkFZFSubmittedMsg struct {
	results []BulkRunResult
	err     error
}

type bulkFZFErrMsg struct {
	err error
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func countSelected(runs []BulkRunItem) int {
	count := 0
	for _, run := range runs {
		if run.Selected {
			count++
		}
	}
	return count
}

// BulkFZFView is the enhanced bulk view with proper FZF integration
type BulkFZFView struct {
	// API client
	client *api.Client

	// Dimensions
	width  int
	height int

	// Mode
	mode BulkViewMode

	// File selection
	fileSelector  *components.BulkFileSelector
	selectedFiles []string

	// Configuration
	bulkConfig *bulk.BulkConfig
	runs       []BulkRunItem

	// UI components
	spinner     spinner.Model
	statusLine  *components.StatusLine
	helpView    *components.HelpView
	showingHelp bool

	// Submission state
	submitting    bool
	error         error
	results       []BulkRunResult
	confirmSubmit bool // Confirmation state for bulk submission

	// Run list navigation
	selectedRunIdx int
	runListScroll  int

	// Keys
	keys bulkKeyMap
}

// NewBulkFZFView creates a new bulk view with FZF
func NewBulkFZFView(client *api.Client) *BulkFZFView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BulkFZFView{
		client:        client,
		mode:          BulkModeFileSelect,
		fileSelector:  components.NewBulkFileSelector(80, 20),
		statusLine:    components.NewStatusLine(),
		helpView:      components.NewHelpView(),
		spinner:       s,
		selectedFiles: []string{},
		runs:          []BulkRunItem{},
		keys:          defaultBulkKeyMap(),
	}
}

func (v *BulkFZFView) Init() tea.Cmd {
	// Don't activate file selector immediately - wait for user to press 'f'
	// Return both spinner tick and window size commands
	return tea.Batch(
		v.spinner.Tick,
		tea.WindowSize(), // Request window size
	)
}

func (v *BulkFZFView) tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return components.TickMsg(t)
	})
}

func (v *BulkFZFView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.fileSelector.SetDimensions(msg.Width, msg.Height)

	case components.TickMsg:
		// Forward tick to file selector if active
		if v.fileSelector != nil && v.fileSelector.IsActive() {
			newSelector, cmd := v.fileSelector.Update(msg)
			v.fileSelector = newSelector
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case tea.KeyMsg:
		// If help is showing, handle help view keys
		if v.showingHelp {
			switch msg.String() {
			case "?", "q", "esc":
				v.showingHelp = false
				return v, nil
			default:
				// Let help view handle navigation
				v.helpView.Update(msg)
				return v, nil
			}
		}

		// Handle confirmation mode first
		if v.confirmSubmit {
			switch msg.String() {
			case "y", "Y":
				// User confirmed - submit selected runs
				debug.LogToFileWithTimestampf("BULK_DEBUG: User confirmed bulk submission\n")
				v.confirmSubmit = false
				return v, v.submitRuns()
			case "n", "N", "esc":
				// User cancelled - exit confirmation mode
				debug.LogToFileWithTimestampf("BULK_DEBUG: User cancelled bulk submission\n")
				v.confirmSubmit = false
				return v, nil
			case "q", "Q":
				// Allow quit during confirmation
				return v, tea.Quit
			default:
				// Ignore all other input during confirmation
				return v, nil
			}
		}

		// Handle mode-specific keys
		switch v.mode {
		case BulkModeFileSelect:
			return v.handleFileSelectKeys(msg)
		case BulkModeRunList:
			return v.handleRunListKeys(msg)
		case BulkModeResults:
			return v.handleResultsKeys(msg)
		}

	case spinner.TickMsg:
		if v.mode == BulkModeSubmitting {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case components.BulkFileSelectedMsg:
		if !msg.Canceled && len(msg.Files) > 0 {
			v.selectedFiles = msg.Files
			return v, v.loadSelectedFiles()
		}
		// If canceled, stay in file select mode

	case bulkFZFErrMsg:
		// Handle error loading config files
		debug.LogToFileWithTimestampf("BULK_DEBUG: Error loading config: %v\n", msg.err)
		v.error = msg.err

		// Format error message for display
		errorMsg := msg.err.Error()
		// Simplify common error messages
		if strings.Contains(errorMsg, "failed to parse") {
			if strings.Contains(errorMsg, "validation failed") {
				// Extract the validation error details
				parts := strings.Split(errorMsg, "validation failed: ")
				if len(parts) > 1 {
					errorMsg = "Validation failed: " + parts[1]
				}
			} else if strings.Contains(errorMsg, "not valid JSON or YAML") {
				errorMsg = "Invalid file format - must be JSON, YAML, JSONL, or Markdown"
			} else {
				errorMsg = "Failed to parse config file - check file format and syntax"
			}
		} else if strings.Contains(errorMsg, "required") {
			errorMsg = "Missing required fields in config"
		}

		// Show error in status line
		v.statusLine.SetTemporaryMessageWithType(
			fmt.Sprintf("❌ %s", errorMsg),
			components.MessageError,
			7*time.Second,
		)
		// Stay in file select mode so user can try again
		v.mode = BulkModeFileSelect
		return v, nil

	case bulkFZFConfigLoadedMsg:
		v.bulkConfig = msg.config
		v.runs = msg.runs
		v.mode = BulkModeRunList
		return v, nil

	case bulkFZFSubmittedMsg:
		debug.LogToFileWithTimestampf("BULK_DEBUG: Received submission result - error: %v, results count: %d\n", msg.err, len(msg.results))
		v.submitting = false
		if msg.err != nil {
			debug.LogToFileWithTimestampf("BULK_DEBUG: Setting error: %v\n", msg.err)
			v.error = msg.err
			v.statusLine.SetTemporaryMessageWithType(
				fmt.Sprintf("Error: %v", msg.err),
				components.MessageError,
				5*time.Second,
			)
		} else {
			debug.LogToFileWithTimestampf("BULK_DEBUG: Setting results: %+v\n", msg.results)
			v.results = msg.results
		}
		v.mode = BulkModeResults
		debug.LogToFileWithTimestampf("BULK_DEBUG: Switched to BulkModeResults, results len: %d\n", len(v.results))
		return v, nil
	}

	// Update file selector for non-key messages (to avoid double processing keys)
	if v.fileSelector != nil && v.fileSelector.IsActive() {
		// Only process non-key messages here to avoid duplicate key handling
		if _, isKeyMsg := msg.(tea.KeyMsg); !isKeyMsg {
			newSelector, cmd := v.fileSelector.Update(msg)
			v.fileSelector = newSelector
			cmds = append(cmds, cmd)
		}
	}

	return v, tea.Batch(cmds...)
}

func (v *BulkFZFView) View() string {
	// Set default dimensions if not set
	if v.width <= 0 {
		v.width = 80
	}
	if v.height <= 0 {
		v.height = 24
	}

	// If help is showing, display help view
	if v.showingHelp {
		v.statusLine.SetWidth(v.width).
			SetLeft("[HELP]").
			SetRight("q/Esc: close help")
		return lipgloss.JoinVertical(
			lipgloss.Left,
			v.helpView.View(),
			v.statusLine.Render(),
		)
	}

	// If file selector is active, show it
	if v.fileSelector != nil && v.fileSelector.IsActive() {
		return v.fileSelector.View(v.statusLine)
	}

	// Regular view modes
	debug.LogToFileWithTimestampf("BULK_DEBUG: View() called with mode: %d\n", v.mode)
	switch v.mode {
	case BulkModeFileSelect:
		return v.renderFileSelectView()
	case BulkModeRunList:
		return v.renderRunListView()
	case BulkModeSubmitting:
		return v.renderSubmittingView()
	case BulkModeResults:
		debug.LogToFileWithTimestampf("BULK_DEBUG: About to render results view\n")
		return v.renderResultsView()
	default:
		return "Unknown mode"
	}
}

func (v *BulkFZFView) renderFileSelectView() string {
	// Calculate available height following create view pattern
	availableHeight := v.height - 3 // Status bar + margins
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate panel dimensions with proper border accounting
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}
	panelHeight := availableHeight
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Build panel content
	var content strings.Builder

	// Title inside the panel
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	content.WriteString(titleStyle.Render("Bulk Run Configuration"))
	content.WriteString("\n\n")

	// Instructions
	instructionsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	if len(v.selectedFiles) == 0 {
		content.WriteString(instructionsStyle.Render(strings.Join([]string{
			"Select configuration files for bulk run creation:",
			"",
			"• Press 'f' to open file selector with FZF filtering",
			"• Select one or more files (single files with multiple runs supported)",
			"• Supports JSON, YAML, JSONL, and Markdown formats",
			"• Maximum 10 runs per batch",
			"",
			"Press 'f' to begin file selection or 'b'/'q' to go back",
		}, "\n")))
	} else {
		// Show selected files
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)
		content.WriteString(selectedStyle.Render(fmt.Sprintf("Selected Files (%d):", len(v.selectedFiles))))
		content.WriteString("\n\n")

		for i, file := range v.selectedFiles {
			content.WriteString(fmt.Sprintf("  %d. %s\n", i+1, file))
		}
		content.WriteString("\n")

		// Actions
		content.WriteString(instructionsStyle.Render(strings.Join([]string{
			"Actions:",
			"  Enter - Load selected files and proceed",
			"  f     - Add more files",
			"  c     - Clear selection",
			"  b/q   - Go back to dashboard",
		}, "\n")))
	}

	// Create bordered panel following create.go pattern
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Render panel with top margin to prevent border cutoff
	panel := panelStyle.Render(content.String())
	panelWithMargin := lipgloss.NewStyle().MarginTop(2).Render(panel)

	// Setup statusline without redundant file count (shown in file selector itself)
	statusText := "[BULK]"

	// Use SetHelp to put commands right after the label instead of far right
	statusLine := v.statusLine.SetWidth(v.width).
		SetLeft(statusText).
		SetRight("").
		SetHelp("f:files | Enter:load | ?:help | b/q:back").
		Render()

	// Join content and status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		panelWithMargin,
		statusLine,
	)
}

func (v *BulkFZFView) renderRunListView() string {
	if v.bulkConfig == nil {
		return "No configuration loaded"
	}

	// Calculate available height following create view pattern
	availableHeight := v.height - 3 // Status bar + margins
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate panel dimensions with proper border accounting
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}
	panelHeight := availableHeight
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Build panel content
	var content strings.Builder

	// Title inside the panel
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	content.WriteString(titleStyle.Render(fmt.Sprintf("Bulk Runs - %d tasks", len(v.runs))))
	content.WriteString("\n\n")

	// Repository info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	info := fmt.Sprintf("Repository: %s | Source: %s | Type: %s",
		v.bulkConfig.Repository,
		v.bulkConfig.Source,
		v.bulkConfig.RunType,
	)
	content.WriteString(infoStyle.Render(info))
	content.WriteString("\n\n")

	// Build run list
	contentHeight := panelHeight - 6 // Account for title, info, padding, borders
	visibleRuns := contentHeight - 2
	startIdx := 0
	if v.selectedRunIdx >= visibleRuns {
		startIdx = v.selectedRunIdx - visibleRuns + 1
	}
	endIdx := min(len(v.runs), startIdx+visibleRuns)

	for i := startIdx; i < endIdx; i++ {
		run := v.runs[i]
		prefix := "  "
		if i == v.selectedRunIdx {
			prefix = "> "
		}

		checkbox := "[ ]"
		if run.Selected {
			checkbox = "[✓]"
		}

		line := fmt.Sprintf("%s%s %s", prefix, checkbox, run.Title)
		if run.Title == "" {
			line = fmt.Sprintf("%s%s Run %d: %s", prefix, checkbox, i+1,
				truncateString(run.Prompt, 50))
		}

		if i == v.selectedRunIdx {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Render(line)
		}

		content.WriteString(line)
		if i < endIdx-1 {
			content.WriteString("\n")
		}
	}

	// Create bordered panel following create.go pattern
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Render panel with top margin to prevent border cutoff
	panel := panelStyle.Render(content.String())
	panelWithMargin := lipgloss.NewStyle().MarginTop(2).Render(panel)

	// Setup statusline - handle confirmation mode with yellow background
	var statusLine string
	if v.confirmSubmit {
		selectedCount := countSelected(v.runs)
		confirmStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("0")).
			Width(v.width).
			Align(lipgloss.Center)

		statusContent := fmt.Sprintf("[CONFIRM] ⚠️  Submit %d selected runs? [y] yes [n] no", selectedCount)
		statusLine = confirmStyle.Render(statusContent)
	} else {
		// Regular status line without redundant selection count (shown in FZF header)
		statusText := "[RUNS]"

		// Use SetHelp to put commands right after the label instead of far right
		statusLine = v.statusLine.SetWidth(v.width).
			SetLeft(statusText).
			SetRight("").
			SetHelp("Space:toggle | Ctrl+A:all | Ctrl+D:none | Enter:confirm | ?:help | b/q:back").
			Render()
	}

	// Join content and status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		panelWithMargin,
		statusLine,
	)
}

func (v *BulkFZFView) renderSubmittingView() string {
	// Calculate available height following create view pattern
	availableHeight := v.height - 3 // Status bar + margins
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate panel dimensions with proper border accounting
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}
	panelHeight := availableHeight
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Build panel content
	var content strings.Builder

	// Center the spinner and text in the panel
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Width(panelWidth - 4).Align(lipgloss.Center).Render(v.spinner.View()))
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Width(panelWidth - 4).Align(lipgloss.Center).Render("Submitting bulk runs..."))
	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Width(panelWidth - 4).Align(lipgloss.Center).Render(fmt.Sprintf("Processing %d tasks", len(v.runs))))

	// Create bordered panel following create.go pattern
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Render panel with top margin to prevent border cutoff
	panel := panelStyle.Render(content.String())
	panelWithMargin := lipgloss.NewStyle().MarginTop(2).Render(panel)

	// Setup statusline
	statusLine := v.statusLine.SetWidth(v.width).
		SetLeft("[BULK]").
		SetRight("Submitting...").
		Render()

	// Join content and status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		panelWithMargin,
		statusLine,
	)
}

func (v *BulkFZFView) renderResultsView() string {
	debug.LogToFileWithTimestampf("BULK_DEBUG: Rendering results view - results count: %d, error: %v\n", len(v.results), v.error)

	// Calculate available height following create view pattern
	availableHeight := v.height - 3 // Status bar + margins
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Calculate panel dimensions with proper border accounting
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}
	panelHeight := availableHeight
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Build panel content
	var content strings.Builder

	// Title inside the panel
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	content.WriteString(titleStyle.Render("Bulk Run Results"))
	content.WriteString("\n\n")

	if v.error != nil {
		debug.LogToFileWithTimestampf("BULK_DEBUG: Rendering error: %v\n", v.error)
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", v.error)))
		content.WriteString("\n\n")
	}

	if len(v.results) > 0 {
		debug.LogToFileWithTimestampf("BULK_DEBUG: Rendering %d results\n", len(v.results))
		successCount := 0
		failCount := 0

		for _, result := range v.results {
			if result.Status == "failed" {
				failCount++
			} else {
				successCount++
			}
		}

		// Summary
		summaryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
		summary := fmt.Sprintf("✓ Success: %d  ✗ Failed: %d  Total: %d",
			successCount, failCount, len(v.results))
		content.WriteString(summaryStyle.Render(summary))
		content.WriteString("\n\n")

		// Details
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Details:"))
		content.WriteString("\n")

		for _, result := range v.results {
			icon := "✓"
			color := lipgloss.Color("10")
			if result.Status == "failed" {
				icon = "✗"
				color = lipgloss.Color("9")
			}

			resultLine := lipgloss.NewStyle().
				Foreground(color).
				Render(fmt.Sprintf("%s %s (ID: %d)", icon, result.Title, result.ID))

			content.WriteString(resultLine)
			content.WriteString("\n")

			if result.Error != "" {
				content.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
			}
		}
	} else {
		debug.LogToFileWithTimestampf("BULK_DEBUG: No results to display - results empty\n")
		// Display message when no results
		messageStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		content.WriteString(messageStyle.Render("No results to display"))
		content.WriteString("\n\n")
		content.WriteString("This could indicate:")
		content.WriteString("\n• The submission is still processing")
		content.WriteString("\n• An error occurred during submission")
		content.WriteString("\n• The API response was empty")
	}

	// Create bordered panel following create.go pattern
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Render panel with top margin to prevent border cutoff
	panel := panelStyle.Render(content.String())
	panelWithMargin := lipgloss.NewStyle().MarginTop(2).Render(panel)

	// Use SetHelp to put commands right after the label instead of far right
	statusLine := v.statusLine.SetWidth(v.width).
		SetLeft("[RESULTS]").
		SetRight("").
		SetHelp("Enter:new | ?:help | b:back to runs | q:dashboard").
		Render()

	// Join content and status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		panelWithMargin,
		statusLine,
	)
}

// Key handlers
func (v *BulkFZFView) handleFileSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If file selector is active, let it handle keys
	if v.fileSelector.IsActive() {
		newSelector, cmd := v.fileSelector.Update(msg)
		v.fileSelector = newSelector
		return v, cmd
	}

	// Main file select mode keys
	switch msg.String() {
	case "?":
		// Toggle help view
		v.showingHelp = !v.showingHelp
		if v.showingHelp {
			v.helpView.SetSize(v.width, v.height-1) // -1 for status line
		}
		return v, nil

	case "q":
		// Go back to dashboard
		dashboard := NewDashboardView(v.client)
		dashboard.width = v.width
		dashboard.height = v.height
		return dashboard, dashboard.Init()

	case "b", "backspace":
		// Also go back to dashboard
		dashboard := NewDashboardView(v.client)
		dashboard.width = v.width
		dashboard.height = v.height
		debug.LogToFileWithTimestampf("BULK_DEBUG: Going back to dashboard from file selection\n")
		return dashboard, dashboard.Init()

	case "Q":
		// Quit entire program (capital Q)
		return v, tea.Quit

	case "f":
		// Activate file selector
		cmd := v.fileSelector.Activate()
		// Just return the activation command, it handles its own ticking
		return v, cmd

	case "c":
		// Clear selection
		v.selectedFiles = []string{}
		return v, nil

	case "enter":
		// Load files if any selected
		if len(v.selectedFiles) > 0 {
			return v, v.loadSelectedFiles()
		}
	}

	return v, nil
}

func (v *BulkFZFView) handleRunListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		// Go back to file selection
		v.mode = BulkModeFileSelect
		return v, nil

	case "b", "backspace":
		// Also go back to file selection
		v.mode = BulkModeFileSelect
		debug.LogToFileWithTimestampf("BULK_DEBUG: Going back to file selection from run list\n")
		return v, nil

	case "Q":
		// Quit entire program
		return v, tea.Quit

	case "up", "k":
		if v.selectedRunIdx > 0 {
			v.selectedRunIdx--
		} else if len(v.runs) > 0 {
			// Wraparound to bottom
			v.selectedRunIdx = len(v.runs) - 1
		}

	case "down", "j":
		if v.selectedRunIdx < len(v.runs)-1 {
			v.selectedRunIdx++
		} else if len(v.runs) > 0 {
			// Wraparound to top
			v.selectedRunIdx = 0
		}

	case " ", "space":
		// Toggle selection
		if v.selectedRunIdx < len(v.runs) {
			v.runs[v.selectedRunIdx].Selected = !v.runs[v.selectedRunIdx].Selected
		}

	case "ctrl+a":
		// Select all
		for i := range v.runs {
			v.runs[i].Selected = true
		}

	case "ctrl+d":
		// Select none (ctrl+d for deselect)
		for i := range v.runs {
			v.runs[i].Selected = false
		}

	case "enter":
		// Enter confirmation mode before submitting
		v.confirmSubmit = true
		debug.LogToFileWithTimestampf("BULK_DEBUG: Entering submission confirmation mode\n")
		return v, nil
	}

	return v, nil
}

func (v *BulkFZFView) handleResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		// Go back to dashboard
		dashboard := NewDashboardView(v.client)
		dashboard.width = v.width
		dashboard.height = v.height
		return dashboard, dashboard.Init()

	case "b", "backspace":
		// Go back to run list view
		v.mode = BulkModeRunList
		debug.LogToFileWithTimestampf("BULK_DEBUG: Going back to run list from results\n")
		return v, nil

	case "Q":
		// Quit entire program
		return v, tea.Quit

	case "enter":
		// Start new batch
		v.mode = BulkModeFileSelect
		v.selectedFiles = []string{}
		v.runs = []BulkRunItem{}
		v.results = []BulkRunResult{}
		v.error = nil
		// Reset file selector
		v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		return v, nil
	}

	return v, nil
}

// Commands
func (v *BulkFZFView) loadSelectedFiles() tea.Cmd {
	return func() tea.Msg {
		// Load bulk configuration from files
		config, err := bulk.LoadBulkConfig(v.selectedFiles)
		if err != nil {
			return bulkFZFErrMsg{err}
		}

		// Convert to BulkRunItems
		var runs []BulkRunItem
		for _, run := range config.Runs {
			runs = append(runs, BulkRunItem{
				Prompt:   run.Prompt,
				Title:    run.Title,
				Target:   run.Target,
				Context:  run.Context,
				Selected: true,
			})
		}

		return bulkFZFConfigLoadedMsg{
			config: config,
			runs:   runs,
		}
	}
}

func (v *BulkFZFView) submitRuns() tea.Cmd {
	v.mode = BulkModeSubmitting
	v.submitting = true

	return func() tea.Msg {
		debug.LogToFileWithTimestampf("BULK_DEBUG: Starting bulk submission, total runs: %d\n", len(v.runs))

		// Filter selected runs
		var selectedRuns []BulkRunItem
		for _, run := range v.runs {
			if run.Selected {
				selectedRuns = append(selectedRuns, run)
			}
		}

		debug.LogToFileWithTimestampf("BULK_DEBUG: Selected runs count: %d\n", len(selectedRuns))

		if len(selectedRuns) == 0 {
			debug.LogToFileWithTimestampf("BULK_DEBUG: No runs selected, returning error\n")
			return bulkFZFSubmittedMsg{err: fmt.Errorf("no runs selected")}
		}

		// Create API request
		var runItems []dto.RunItem
		for i, run := range selectedRuns {
			// Generate file hash
			hashContent := fmt.Sprintf("%s-%s-%s-%s-%d",
				v.bulkConfig.Repository,
				run.Prompt,
				run.Target,
				run.Context,
				i,
			)
			hash := cache.CalculateStringHash(hashContent)

			runItems = append(runItems, dto.RunItem{
				Prompt:   run.Prompt,
				Title:    run.Title,
				Target:   run.Target,
				Context:  run.Context,
				FileHash: hash,
			})
		}

		req := &dto.BulkRunRequest{
			RepositoryName: v.bulkConfig.Repository,
			RepoID:         v.bulkConfig.RepoID,
			RunType:        v.bulkConfig.RunType,
			SourceBranch:   v.bulkConfig.Source,
			BatchTitle:     v.bulkConfig.BatchTitle,
			Force:          v.bulkConfig.Force,
			Runs:           runItems,
			Options: dto.BulkOptions{
				Parallel: 5,
			},
		}

		// Submit to API
		debug.LogToFileWithTimestampf("BULK_DEBUG: Submitting API request: %+v\n", req)
		ctx := context.Background()
		resp, err := v.client.CreateBulkRuns(ctx, req)
		if err != nil {
			debug.LogToFileWithTimestampf("BULK_DEBUG: API call failed: %v\n", err)
			return bulkFZFSubmittedMsg{err: err}
		}

		debug.LogToFileWithTimestampf("BULK_DEBUG: API response: successful runs: %d, failed: %d\n", len(resp.Data.Successful), len(resp.Data.Failed))

		// Convert response to results
		var results []BulkRunResult
		for _, run := range resp.Data.Successful {
			debug.LogToFileWithTimestampf("BULK_DEBUG: Adding successful run: ID=%d, Title=%s, Status=%s\n", run.ID, run.Title, run.Status)
			results = append(results, BulkRunResult{
				ID:     run.ID,
				Title:  run.Title,
				Status: run.Status,
				URL:    "", // URL not provided in spec for successful runs
			})
		}

		for _, runErr := range resp.Data.Failed {
			debug.LogToFileWithTimestampf("BULK_DEBUG: Adding failed run: Prompt=%s, Error=%s\n", runErr.Prompt, runErr.Error)
			results = append(results, BulkRunResult{
				Title:  runErr.Prompt, // Use prompt as title since title might be empty
				Status: "failed",
				Error:  runErr.Message,
			})
		}

		debug.LogToFileWithTimestampf("BULK_DEBUG: Returning results, count: %d\n", len(results))
		return bulkFZFSubmittedMsg{
			results: results,
			err:     nil,
		}
	}
}

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
	submitting bool
	error      error
	results    []BulkRunResult

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

	case bulkFZFConfigLoadedMsg:
		v.bulkConfig = msg.config
		v.runs = msg.runs
		v.mode = BulkModeRunList
		return v, nil

	case bulkFZFSubmittedMsg:
		v.submitting = false
		if msg.err != nil {
			v.error = msg.err
			v.statusLine.SetTemporaryMessageWithType(
				fmt.Sprintf("Error: %v", msg.err),
				components.MessageError,
				5*time.Second,
			)
		} else {
			v.results = msg.results
		}
		v.mode = BulkModeResults
		return v, nil
	}

	// Update file selector if active
	if v.fileSelector != nil && v.fileSelector.IsActive() {
		newSelector, cmd := v.fileSelector.Update(msg)
		v.fileSelector = newSelector
		cmds = append(cmds, cmd)
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
	switch v.mode {
	case BulkModeFileSelect:
		return v.renderFileSelectView()
	case BulkModeRunList:
		return v.renderRunListView()
	case BulkModeSubmitting:
		return v.renderSubmittingView()
	case BulkModeResults:
		return v.renderResultsView()
	default:
		return "Unknown mode"
	}
}

func (v *BulkFZFView) renderFileSelectView() string {
	// This is shown when file selector is not active
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	title := titleStyle.Render("Bulk Run Configuration")

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Render(strings.Join([]string{
			"Select configuration files for bulk run creation:",
			"",
			"• Press 'f' to open file selector with FZF filtering",
			"• Select multiple files to combine into a batch",
			"• Supports JSON, YAML, JSONL, and Markdown formats",
			"• Maximum 10 runs per batch",
			"",
			"Press 'f' to begin file selection or 'q' to quit",
		}, "\n"))

	// Show previously selected files if any
	var selectedList string
	if len(v.selectedFiles) > 0 {
		fileListStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			MarginTop(1).
			MarginBottom(1)

		var files []string
		for i, file := range v.selectedFiles {
			files = append(files, fmt.Sprintf("%d. %s", i+1, file))
		}

		selectedList = fileListStyle.Render(
			lipgloss.NewStyle().Bold(true).Render("Selected Files:\n\n") +
				strings.Join(files, "\n"),
		)

		// Update instructions
		instructions = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1).
			Render(strings.Join([]string{
				"Actions:",
				"  Enter - Load selected files and proceed",
				"  f     - Add more files",
				"  c     - Clear selection",
				"  q     - Cancel and quit",
			}, "\n"))
	}

	// Calculate available height
	availableHeight := v.height - 3 // Reserve for statusline

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		instructions,
		selectedList,
	)

	// Center the content
	centeredContent := lipgloss.Place(
		v.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Setup statusline
	statusText := "[BULK]"
	if len(v.selectedFiles) > 0 {
		statusText = fmt.Sprintf("[BULK] %d file(s) selected", len(v.selectedFiles))
	}

	v.statusLine.SetWidth(v.width).
		SetLeft(statusText).
		SetRight("f: select files | Enter: load | ?: help | q: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredContent,
		v.statusLine.Render(),
	)
}

func (v *BulkFZFView) renderRunListView() string {
	if v.bulkConfig == nil {
		return "No configuration loaded"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	title := titleStyle.Render(fmt.Sprintf("Bulk Runs - %d tasks", len(v.runs)))

	// Repository info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(1)

	info := fmt.Sprintf("Repository: %s | Source: %s | Type: %s",
		v.bulkConfig.Repository,
		v.bulkConfig.Source,
		v.bulkConfig.RunType,
	)

	// Build run list
	listStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(v.width - 4).
		Height(v.height - 10)

	var runLines []string
	for i, run := range v.runs {
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

		runLines = append(runLines, line)
	}

	runList := listStyle.Render(strings.Join(runLines, "\n"))

	// Status line
	v.statusLine.SetWidth(v.width).
		SetLeft(fmt.Sprintf("[RUNS] %d selected", countSelected(v.runs))).
		SetRight("Space: toggle | a: all | n: none | Enter: submit | ?: help | q: back")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		infoStyle.Render(info),
		runList,
		v.statusLine.Render(),
	)
}

func (v *BulkFZFView) renderSubmittingView() string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		v.spinner.View(),
		"",
		"Submitting bulk runs...",
		fmt.Sprintf("Processing %d tasks", len(v.runs)),
	)

	centeredContent := lipgloss.Place(
		v.width,
		v.height-1,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	v.statusLine.SetWidth(v.width).
		SetLeft("[BULK]").
		SetRight("Submitting...")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredContent,
		v.statusLine.Render(),
	)
}

func (v *BulkFZFView) renderResultsView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	title := titleStyle.Render("Bulk Run Results")

	var content strings.Builder

	if v.error != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			MarginBottom(1)
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", v.error)))
	}

	if len(v.results) > 0 {
		successCount := 0
		failCount := 0

		for _, result := range v.results {
			if result.Status == "failed" {
				failCount++
			} else {
				successCount++
			}
		}

		summaryStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1).
			MarginBottom(1)

		summary := fmt.Sprintf("✓ Success: %d\n✗ Failed: %d\nTotal: %d",
			successCount, failCount, len(v.results))

		content.WriteString(summaryStyle.Render(summary))
		content.WriteString("\n\nDetails:\n")

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

			content.WriteString(resultLine + "\n")

			if result.Error != "" {
				content.WriteString(fmt.Sprintf("  Error: %s\n", result.Error))
			}
		}
	}

	v.statusLine.SetWidth(v.width).
		SetLeft("[RESULTS]").
		SetRight("Enter: new batch | ?: help | q: back")

	resultBox := lipgloss.NewStyle().
		Width(v.width - 4).
		Height(v.height - 6).
		Padding(1).
		Render(content.String())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		resultBox,
		v.statusLine.Render(),
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
		return NewDashboardView(v.client), nil

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

	case "Q":
		// Quit entire program
		return v, tea.Quit

	case "up", "k":
		if v.selectedRunIdx > 0 {
			v.selectedRunIdx--
		}

	case "down", "j":
		if v.selectedRunIdx < len(v.runs)-1 {
			v.selectedRunIdx++
		}

	case " ", "space":
		// Toggle selection
		if v.selectedRunIdx < len(v.runs) {
			v.runs[v.selectedRunIdx].Selected = !v.runs[v.selectedRunIdx].Selected
		}

	case "a":
		// Select all
		for i := range v.runs {
			v.runs[i].Selected = true
		}

	case "n":
		// Select none
		for i := range v.runs {
			v.runs[i].Selected = false
		}

	case "enter":
		// Submit selected runs
		return v, v.submitRuns()
	}

	return v, nil
}

func (v *BulkFZFView) handleResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		// Go back to dashboard
		return NewDashboardView(v.client), nil

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
		// Filter selected runs
		var selectedRuns []BulkRunItem
		for _, run := range v.runs {
			if run.Selected {
				selectedRuns = append(selectedRuns, run)
			}
		}

		if len(selectedRuns) == 0 {
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
		ctx := context.Background()
		resp, err := v.client.CreateBulkRuns(ctx, req)
		if err != nil {
			return bulkFZFSubmittedMsg{err: err}
		}

		// Convert response to results
		var results []BulkRunResult
		for _, run := range resp.Runs {
			results = append(results, BulkRunResult{
				ID:     run.ID,
				Title:  run.Title,
				Status: run.Status,
				URL:    run.RunURL,
			})
		}

		for _, runErr := range resp.Errors {
			results = append(results, BulkRunResult{
				Title:  runErr.Title,
				Status: "failed",
				Error:  runErr.Error,
			})
		}

		return bulkFZFSubmittedMsg{
			results: results,
			err:     nil,
		}
	}
}

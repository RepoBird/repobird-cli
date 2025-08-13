package views

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// BulkMode represents the current mode of the bulk view
type BulkMode int

const (
	ModeInstructions BulkMode = iota // Initial screen with instructions
	ModeFileBrowser               // FZF file browser (full screen)
	ModeRunList                   // Run validation list
	ModeRunEdit                   // Individual run editing
	ModeProgress                  // Submission progress
	ModeResults                   // Final results
)

// BulkRunItem represents a single run in the bulk collection
type BulkRunItem struct {
	Prompt   string
	Title    string
	Target   string
	Context  string
	Selected bool
	Status   RunStatus
	Error    string
	FileHash string
}

// RunStatus represents the status of a run
type RunStatus int

const (
	StatusPending RunStatus = iota
	StatusQueued
	StatusProcessing
	StatusCompleted
	StatusFailed
	StatusCancelled
)


// BulkView represents the bulk runs TUI view
type BulkView struct {
	// API client
	client *api.Client

	// Configuration
	repository   string
	repoID       int
	sourceBranch string
	runType      string
	batchTitle   string
	force        bool

	// Runs collection
	runs        []BulkRunItem
	selectedRun int

	// UI state
	mode         BulkMode
	fileSelector *components.BulkFileSelector // Only used in ModeFileBrowser
	help         help.Model
	keys         bulkKeyMap
	width        int
	height       int

	// Submission state
	submitting bool
	batchID    string
	results    []BulkRunResult
	error      error

	// Components
	spinner    spinner.Model
	statusLine *components.StatusLine
	layout     *components.WindowLayout
}

// bulkKeyMap defines key bindings for the bulk view
type bulkKeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Actions
	Select    key.Binding
	Submit    key.Binding
	Cancel    key.Binding
	Edit      key.Binding
	Add       key.Binding
	Delete    key.Binding
	Duplicate key.Binding
	ToggleAll key.Binding
	FZF       key.Binding

	// Mode switches
	FileMode key.Binding
	ListMode key.Binding

	// Global
	Help key.Binding
	Quit key.Binding
}

// NewBulkView creates a new bulk view
func NewBulkView(client *api.Client) *BulkView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BulkView{
		client:     client,
		mode:       ModeInstructions, // Start with instructions, not file selector
		help:       help.New(),
		keys:       defaultBulkKeyMap(),
		spinner:    s,
		runType:    "run",
		runs:       []BulkRunItem{},
		statusLine: components.NewStatusLine(),
	}
}

func defaultBulkKeyMap() bulkKeyMap {
	return bulkKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup/ctrl+u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn/ctrl+d", "page down"),
		),
		Select: key.NewBinding(
			key.WithKeys(" ", "enter"),
			key.WithHelp("space/enter", "select"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("ctrl+c", "esc"),
			key.WithHelp("esc", "cancel"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Duplicate: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "duplicate"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "toggle all"),
		),
		FZF: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fuzzy find"),
		),
		FileMode: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "file mode"),
		),
		ListMode: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "list mode"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+q"),
			key.WithHelp("q", "quit"),
		),
	}
}

// Init initializes the bulk view
func (v *BulkView) Init() tea.Cmd {
	debug.LogToFile("DEBUG: BulkView.Init() called\n")
	return v.spinner.Tick // Just start spinner, no file selector yet
}

// Update handles messages for the bulk view
func (v *BulkView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: BulkView.Update() received message: %T\n", msg)
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debug.LogToFilef("DEBUG: BulkView - handling WindowSizeMsg: %dx%d\n", msg.Width, msg.Height)
		v.width = msg.Width
		v.height = msg.Height
		v.help.Width = msg.Width
		
		// Create layout for proper sizing
		v.layout = components.NewWindowLayout(msg.Width, msg.Height)

		// Update file selector dimensions
		if v.fileSelector != nil {
			v.fileSelector.SetDimensions(msg.Width, msg.Height)
		}

	case tea.KeyMsg:
		debug.LogToFilef("DEBUG: BulkView - handling KeyMsg: '%s', mode=%d\n", msg.String(), v.mode)
		
		// Handle global quit keys regardless of mode
		if msg.String() == "Q" || msg.Type == tea.KeyCtrlC {
			return v, tea.Quit
		}
		
		// FIRST: Handle components that need raw key input (like FZF)
		switch v.mode {
		case ModeFileBrowser:
			if v.fileSelector != nil {
				// Check if this is a navigation key that we should handle
				if msg.Type == tea.KeyEsc || msg.String() == "q" {
					// Handle navigation keys
					debug.LogToFile("DEBUG: BulkView - handling navigation key in FileBrowser\n")
					return v.handleFileBrowserKeys(msg)
				} else {
					// Pass all other keys directly to file selector for typing/search
					debug.LogToFilef("DEBUG: BulkView - passing key '%s' to file selector\n", msg.String())
					newFileSelector, cmd := v.fileSelector.Update(msg)
					v.fileSelector = newFileSelector
					cmds = append(cmds, cmd)
					return v, tea.Batch(cmds...)
				}
			}
		}
		
		// SECOND: Handle view-specific navigation keys
		switch v.mode {
		case ModeInstructions:
			debug.LogToFile("DEBUG: BulkView - delegating to handleInstructionsKeys\n")
			return v.handleInstructionsKeys(msg)
		case ModeFileBrowser:
			debug.LogToFile("DEBUG: BulkView - delegating to handleFileBrowserKeys\n")
			return v.handleFileBrowserKeys(msg)
		case ModeRunList:
			debug.LogToFile("DEBUG: BulkView - delegating to handleRunListKeys\n")
			return v.handleRunListKeys(msg)
		}

	case components.BulkFileSelectedMsg:
		// File(s) selected, load configurations from actual files
		debug.LogToFilef("DEBUG: BulkView - files selected: %v\n", msg.Files)
		if !msg.Canceled && len(msg.Files) > 0 {
			return v.loadFiles(msg.Files)
		}
		return v, nil

	case bulkRunsLoadedMsg:
		// Runs loaded from files
		debug.LogToFilef("DEBUG: BulkView - runs loaded: %d\n", len(msg.runs))
		v.runs = msg.runs
		v.repository = msg.repository
		v.repoID = msg.repoID
		v.sourceBranch = msg.source
		v.runType = msg.runType
		v.batchTitle = msg.batchTitle
		v.mode = ModeRunList
		v.selectedRun = 0
		return v, nil

	case bulkSubmittedMsg:
		// Bulk submission completed
		v.submitting = false
		v.batchID = msg.batchID
		v.results = msg.results
		if msg.err != nil {
			v.error = msg.err
			v.mode = ModeResults
		} else {
			v.mode = ModeProgress
			// Start polling for progress
			return v, v.pollProgress()
		}
		return v, nil

	case bulkProgressMsg:
		// Progress update received - would need to update progress view
		if msg.completed {
			v.mode = ModeResults
		}
		return v, nil

	case bulkCancelledMsg:
		// Bulk operation cancelled
		v.mode = ModeResults
		return v, nil

	case errMsg:
		// Error occurred
		v.error = msg.err
		return v, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Component updates are now handled directly in key processing above

	debug.LogToFilef("DEBUG: BulkView.Update() - returning with %d commands\n", len(cmds))
	return v, tea.Batch(cmds...)
}

// Event handlers for different modes
// handleInstructionsKeys handles keys in the instructions mode
func (v *BulkView) handleInstructionsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: BulkView.handleInstructionsKeys() - key='%s'\n", msg.String())
	switch {
	case key.Matches(msg, v.keys.Quit):
		// Navigate back to dashboard instead of quitting directly
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}
	case key.Matches(msg, v.keys.FileMode) || msg.String() == "F":
		// Switch to file browser mode and initialize file selector
		v.mode = ModeFileBrowser
		if v.fileSelector == nil {
			v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		}
		return v, v.fileSelector.Activate()
	case key.Matches(msg, v.keys.ListMode):
		if len(v.runs) > 0 {
			v.mode = ModeRunList
		}
		return v, nil
	default:
		// Ignore other keys in instructions mode
		return v, nil
	}
}

// handleFileBrowserKeys handles navigation keys in the file browser mode
func (v *BulkView) handleFileBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: BulkView.handleFileBrowserKeys() - navigation key='%s'\n", msg.String())
	switch {
	case key.Matches(msg, v.keys.Quit) || msg.Type == tea.KeyEsc:
		// Go back to instructions mode
		v.mode = ModeInstructions
		v.fileSelector = nil // Clear file selector
		return v, nil
	default:
		// This should only be called for navigation keys now
		debug.LogToFilef("DEBUG: BulkView.handleFileBrowserKeys() - unhandled navigation key: '%s'\n", msg.String())
		return v, nil
	}
}

func (v *BulkView) handleRunListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		// Navigate back to dashboard instead of quitting directly
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}
	case key.Matches(msg, v.keys.Up):
		if v.selectedRun > 0 {
			v.selectedRun--
		}
	case key.Matches(msg, v.keys.Down):
		if v.selectedRun < len(v.runs)-1 {
			v.selectedRun++
		}
	case key.Matches(msg, v.keys.Select):
		if v.selectedRun < len(v.runs) {
			v.runs[v.selectedRun].Selected = !v.runs[v.selectedRun].Selected
		}
	case key.Matches(msg, v.keys.FileMode):
		// Switch back to file browser mode
		v.mode = ModeFileBrowser
		if v.fileSelector == nil {
			v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		}
	case key.Matches(msg, v.keys.Submit):
		// Submit selected bulk runs
		return v, v.submitBulkRuns()
	}
	return v, nil
}

// View renders the bulk view
func (v *BulkView) View() string {
	debug.LogToFilef("DEBUG: BulkView.View() called - mode=%d, width=%d, height=%d\n", v.mode, v.width, v.height)
	
	if v.width <= 0 || v.height <= 0 {
		debug.LogToFile("DEBUG: BulkView - no dimensions, returning initializing message\n")
		return "⟳ Initializing Bulk View..."
	}
	
	switch v.mode {
	case ModeInstructions:
		debug.LogToFile("DEBUG: BulkView - rendering instructions\n")
		return v.renderInstructions()
	case ModeFileBrowser:
		debug.LogToFile("DEBUG: BulkView - rendering file browser\n")
		return v.renderFileBrowser()
	case ModeRunList:
		debug.LogToFile("DEBUG: BulkView - rendering run list\n")
		return v.renderRunList()
	case ModeRunEdit:
		debug.LogToFile("DEBUG: BulkView - rendering run edit\n")
		return v.renderRunEdit()
	case ModeProgress:
		debug.LogToFile("DEBUG: BulkView - rendering progress\n")
		return v.renderProgress()
	case ModeResults:
		debug.LogToFile("DEBUG: BulkView - rendering results\n")
		return v.renderResults()
	default:
		debug.LogToFilef("DEBUG: BulkView - unknown mode: %d\n", v.mode)
		return "Unknown mode"
	}
}

// renderInstructions renders the initial instructions screen
func (v *BulkView) renderInstructions() string {
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Instructions content
	instructions := []string{
		"Welcome to the Bulk Operations interface.",
		"",
		"This tool allows you to:",
		"• Process multiple configuration files at once",
		"• Submit batch runs with different parameters",
		"• Track progress across multiple operations",
		"",
		"Press F to browse and select configuration files to get started.",
		"",
		"Supported file formats:",
		"• JSON (.json) - Task configuration files",
		"• YAML (.yaml, .yml) - Configuration files",
		"• JSONL (.jsonl) - Line-delimited JSON",
		"• Markdown (.md) - Documentation with task blocks",
	}

	content := strings.Join(instructions, "\n")

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	title := titleStyle.Render("Bulk Operations")
	styledContent := contentStyle.Render(content)
	
	// Create the main container with proper dimensions
	boxWidth, boxHeight := v.layout.GetBoxDimensions()
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line
	statusLine := v.renderStatusLine("BULK")

	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// renderFileBrowser renders the dedicated file browser page
func (v *BulkView) renderFileBrowser() string {
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Get file selector content
	var content string
	if v.fileSelector != nil {
		// Render file selector WITHOUT the original status line
		content = v.fileSelector.View(nil) // Pass nil to avoid double status line
	} else {
		content = "File selector not initialized"
	}

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	title := titleStyle.Render("Select Configuration Files")
	styledContent := contentStyle.Render(content)
	
	// Create the main container with proper dimensions
	boxWidth, boxHeight := v.layout.GetBoxDimensions()
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line - this is the ONLY status line for this mode
	statusLine := v.renderStatusLine("FZF-BULK")

	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// renderRunList renders the run list view
func (v *BulkView) renderRunList() string {
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render(fmt.Sprintf("Bulk Runs (%d)", len(v.runs)))

	// Repository info
	repoInfo := fmt.Sprintf("Repository: %s | Source: %s | Type: %s",
		v.repository, v.sourceBranch, v.runType)
	repoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Runs list
	var runsList strings.Builder
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	for i, run := range v.runs {
		prefix := "  "
		if i == v.selectedRun {
			prefix = "> "
		}

		title := run.Title
		if title == "" {
			title = fmt.Sprintf("Run %d", i+1)
		}

		statusIcon := ""
		if run.Selected {
			statusIcon = "[✓] "
		} else {
			statusIcon = "[ ] "
		}

		line := fmt.Sprintf("%s%s%s", prefix, statusIcon, title)
		if i == v.selectedRun {
			line = selectedStyle.Render(line)
		}
		runsList.WriteString(line + "\n")
	}

	// Render status line
	statusLine := v.renderStatusLine("BULK")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		repoStyle.Render(repoInfo),
		"",
		runsList.String(),
		statusLine,
	)
}

// renderStatusLine renders the status line
func (v *BulkView) renderStatusLine(layoutName string) string {
	// Simple help text based on current mode
	var helpText string
	switch v.mode {
	case ModeInstructions:
		helpText = "F:browse files q:quit ?:help"
	case ModeFileBrowser:
		helpText = "↑↓:navigate space:select enter:confirm esc:back q:quit"
	case ModeRunList:
		helpText = "↑↓:navigate space:toggle F:files ctrl+s:submit q:quit"
	default:
		helpText = "q:quit ?:help"
	}

	return v.statusLine.
		SetWidth(v.width).
		SetLeft(fmt.Sprintf("[%s]", layoutName)).
		SetRight("").
		SetHelp(helpText).
		ResetStyle().
		SetLoading(false).
		Render()
}

// renderRunEdit renders the run editing view (placeholder)
func (v *BulkView) renderRunEdit() string {
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	content := "Run Edit Mode - Implementation in bulk_run_editor.go"
	statusLine := v.renderStatusLine("BULK")
	
	return lipgloss.JoinVertical(lipgloss.Left, content, statusLine)
}

// renderProgress renders the progress view (placeholder)
func (v *BulkView) renderProgress() string {
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	var content string
	if v.submitting {
		content = fmt.Sprintf("%s Submitting bulk runs...", v.spinner.View())
	} else {
		content = "Progress Mode - Implementation in bulk_progress_view.go"
	}
	
	statusLine := v.renderStatusLine("BULK")
	
	return lipgloss.JoinVertical(lipgloss.Left, content, statusLine)
}

// renderResults renders the results view (placeholder)
func (v *BulkView) renderResults() string {
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Bulk Run Results")

	var content strings.Builder

	if v.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", v.error)))
	}

	if len(v.results) > 0 {
		content.WriteString("\nCreated Runs:\n")
		for _, result := range v.results {
			statusIcon := "✓"
			statusColor := lipgloss.Color("10")

			if result.Status == "failed" {
				statusIcon = "✗"
				statusColor = lipgloss.Color("9")
			}

			style := lipgloss.NewStyle().Foreground(statusColor)
			content.WriteString(fmt.Sprintf("  %s %s (ID: %d)\n",
				style.Render(statusIcon),
				result.Title,
				result.ID,
			))

			if result.Error != "" {
				content.WriteString(fmt.Sprintf("    Error: %s\n", result.Error))
			}
			if result.URL != "" {
				content.WriteString(fmt.Sprintf("    URL: %s\n", result.URL))
			}
		}
	}

	if v.batchID != "" {
		content.WriteString(fmt.Sprintf("\nBatch ID: %s\n", v.batchID))
	}

	statusLine := v.renderStatusLine("BULK")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content.String(),
		statusLine,
	)
}
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

	// Layout systems
	doubleColumnLayout *components.DoubleColumnLayout // For FZF file browser

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

	case components.FilesLoadedMsg:
		// Forward file loading message to file selector if it exists
		debug.LogToFilef("DEBUG: BulkView - received FilesLoadedMsg, forwarding to file selector\n")
		if v.fileSelector != nil {
			newFileSelector, cmd := v.fileSelector.Update(msg)
			v.fileSelector = newFileSelector
			cmds = append(cmds, cmd)
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
				// The CoreViewKeymap.HandleKey will handle INPUT mode keys like 'q', 'b', 'backspace'
				// So here we only need to handle the remaining keys
				
				// In NAV mode, check if this is a navigation key we should handle
				if !v.fileSelector.GetInputMode() {
					if msg.Type == tea.KeyEsc || msg.String() == "q" {
						// Handle navigation keys in NAV mode
						debug.LogToFile("DEBUG: BulkView.Update - handling navigation key in FileBrowser (NAV mode)\n")
						return v.handleFileBrowserKeys(msg)
					}
				}
				
				// For all other keys, pass to file selector
				// Note: Keys handled by HandleKey() won't reach here
				debug.LogToFilef("DEBUG: BulkView.Update - passing key '%s' to file selector\n", msg.String())
				newFileSelector, cmd := v.fileSelector.Update(msg)
				v.fileSelector = newFileSelector
				cmds = append(cmds, cmd)
				return v, tea.Batch(cmds...)
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
		debug.LogToFilef("DEBUG: BulkView - files selected: %v, canceled: %v\n", msg.Files, msg.Canceled)
		if msg.Canceled {
			// User canceled file selection, go back to instructions
			debug.LogToFile("DEBUG: BulkView - file selection canceled, returning to instructions\n")
			v.mode = ModeInstructions
			v.fileSelector = nil // Clear file selector
			return v, nil
		}
		if len(msg.Files) > 0 {
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
		// Error occurred - show it in instructions mode
		debug.LogToFilef("DEBUG: BulkView - error occurred: %v\n", msg.err)
		v.error = msg.err
		// Go back to instructions mode to show the error
		v.mode = ModeInstructions
		v.fileSelector = nil // Clear file selector
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
	case key.Matches(msg, v.keys.FZF) || msg.String() == "f":
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
	case key.Matches(msg, v.keys.FZF):
		// Switch back to file browser mode
		v.mode = ModeFileBrowser
		if v.fileSelector == nil {
			v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		}
	case key.Matches(msg, v.keys.FileMode):
		// Switch to file browser mode (uppercase F key)
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
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}
	
	// Instructions content
	var instructionLines []string
	
	// Show error if there is one
	if v.error != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		
		instructionLines = append(instructionLines, 
			errorStyle.Render("❌ Error loading configuration files:"),
			"",
			fmt.Sprintf("%v", v.error),
			"",
			"Please check that your selected files contain valid configuration.",
			"",
			"---",
			"",
		)
	}
	
	instructionLines = append(instructionLines,
		"Welcome to the Bulk Operations interface.",
		"",
		"This tool allows you to:",
		"• Process multiple configuration files at once",
		"• Submit batch runs with different parameters",
		"• Track progress across multiple operations",
		"",
		"Press f to browse and select configuration files to get started.",
		"",
		"Supported file formats:",
		"• JSON (.json) - Task configuration files",
		"• YAML (.yaml, .yml) - Configuration files",
		"• JSONL (.jsonl) - Line-delimited JSON",
		"• Markdown (.md) - Documentation with task blocks",
	)

	content := strings.Join(instructionLines, "\n")

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	title := titleStyle.Render("Bulk Operations")
	styledContent := contentStyle.Render(content)
	
	// Get proper dimensions from layout
	boxWidth, boxHeight := v.layout.GetBoxDimensions()
	
	// Create the main container with proper dimensions
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line
	statusLine := v.renderStatusLine("BULK")

	// Join with status line
	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// renderFileBrowser renders the dedicated file browser page with proper double-column layout
func (v *BulkView) renderFileBrowser() string {
	// Initialize double column layout for FZF + preview
	if v.doubleColumnLayout == nil {
		v.doubleColumnLayout = components.NewDoubleColumnLayout(v.width, v.height, &components.DoubleColumnConfig{
			LeftRatio:  0.6, // 60% for file list
			RightRatio: 0.4, // 40% for preview
			Gap:        1,
		})
	} else {
		v.doubleColumnLayout.Update(v.width, v.height)
	}

	// Get content from file selector using the new double-column methods
	var leftContent, rightContent string
	if v.fileSelector != nil {
		// Get content dimensions for each column
		leftWidth, leftHeight, rightWidth, rightHeight := v.doubleColumnLayout.GetContentDimensions()
		
		// Get file list and preview content from selector
		leftContent = v.fileSelector.GetFileListContent(leftWidth, leftHeight)
		rightContent = v.fileSelector.GetPreviewContent(rightWidth, rightHeight)
	} else {
		leftContent = "File selector not initialized"
		rightContent = "No preview available"
	}

	// Status line with mode indicator
	statusLine := v.renderStatusLine("FZF-BULK")

	// Use double column layout to render everything
	return v.doubleColumnLayout.RenderWithTitle(
		"Select Configuration Files",
		leftContent,
		rightContent,
		statusLine,
	)
}

// renderRunList renders the run list view
func (v *BulkView) renderRunList() string {
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	// Title with count
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

		runTitle := run.Title
		if runTitle == "" {
			runTitle = fmt.Sprintf("Run %d", i+1)
		}

		statusIcon := ""
		if run.Selected {
			statusIcon = "[✓] "
		} else {
			statusIcon = "[ ] "
		}

		line := fmt.Sprintf("%s%s%s", prefix, statusIcon, runTitle)
		if i == v.selectedRun {
			line = selectedStyle.Render(line)
		}
		runsList.WriteString(line + "\n")
	}

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		repoStyle.Render(repoInfo),
		"",
		runsList.String(),
	)

	// Style and size the content
	styledContent := contentStyle.Render(content)

	// Get proper dimensions from layout
	boxWidth, boxHeight := v.layout.GetBoxDimensions()

	// Create the main container with proper dimensions
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line
	statusLine := v.renderStatusLine("BULK")

	// Join with status line
	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// renderStatusLine renders the status line
func (v *BulkView) renderStatusLine(layoutName string) string {
	// Simple help text based on current mode
	var helpText string
	var modeIndicator string
	
	switch v.mode {
	case ModeInstructions:
		helpText = "f:browse files q:quit ?:help"
	case ModeFileBrowser:
		if v.fileSelector != nil && v.fileSelector.GetInputMode() {
			modeIndicator = " [INPUT]"
			helpText = "↑↓:nav space:select type:filter esc:nav mode enter:confirm ctrl+a:all"
		} else {
			modeIndicator = " [NAV]"
			helpText = "↑↓/j/k:nav space:select i:input mode b/esc:back enter:confirm"
		}
	case ModeRunList:
		helpText = "↑↓:navigate space:toggle f:files ctrl+s:submit q:quit"
	default:
		helpText = "q:quit ?:help"
	}

	// Compose the left side with layout name and mode indicator
	leftText := fmt.Sprintf("[%s]%s", layoutName, modeIndicator)

	return v.statusLine.
		SetWidth(v.width).
		SetLeft(leftText).
		SetRight("").
		SetHelp(helpText).
		ResetStyle().
		SetLoading(false).
		Render()
}

// renderRunEdit renders the run editing view
func (v *BulkView) renderRunEdit() string {
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	title := titleStyle.Render("Edit Run")

	var contentLines []string
	if v.selectedRun < len(v.runs) {
		run := v.runs[v.selectedRun]
		contentLines = append(contentLines,
			fmt.Sprintf("Title: %s", run.Title),
			fmt.Sprintf("Target: %s", run.Target),
			fmt.Sprintf("Prompt: %s", run.Prompt),
			"",
			"Edit mode not yet implemented.",
			"Press ESC to return to list.",
		)
	} else {
		contentLines = append(contentLines, "No run selected for editing.")
	}

	content := strings.Join(contentLines, "\n")
	styledContent := contentStyle.Render(content)

	// Get proper dimensions from layout
	boxWidth, boxHeight := v.layout.GetBoxDimensions()

	// Create the main container with proper dimensions
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line
	statusLine := v.renderStatusLine("BULK")

	// Join with status line
	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// renderProgress renders the progress view
func (v *BulkView) renderProgress() string {
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	title := titleStyle.Render("Bulk Run Progress")

	var contentLines []string
	if v.submitting {
		contentLines = append(contentLines,
			fmt.Sprintf("%s Submitting bulk runs...", v.spinner.View()),
			"",
			fmt.Sprintf("Batch: %s", v.batchTitle),
		)
	} else if v.batchID != "" {
		contentLines = append(contentLines,
			fmt.Sprintf("Batch ID: %s", v.batchID),
			"",
			"Monitoring progress...",
		)
	} else {
		contentLines = append(contentLines,
			"Initializing batch submission...",
		)
	}

	content := strings.Join(contentLines, "\n")
	styledContent := contentStyle.Render(content)

	// Get proper dimensions from layout
	boxWidth, boxHeight := v.layout.GetBoxDimensions()

	// Create the main container with proper dimensions
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line
	statusLine := v.renderStatusLine("BULK")

	// Join with status line
	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// renderResults renders the results view
func (v *BulkView) renderResults() string {
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Use WindowLayout system for consistent styling
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

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
	} else if v.error == nil {
		content.WriteString("\nNo results to display.\n")
	}

	if v.batchID != "" {
		content.WriteString(fmt.Sprintf("\nBatch ID: %s\n", v.batchID))
	}

	// Style the content
	styledContent := contentStyle.Render(content.String())

	// Get proper dimensions from layout
	boxWidth, boxHeight := v.layout.GetBoxDimensions()

	// Create the main container with proper dimensions
	mainContainer := boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, "", styledContent))

	// Status line
	statusLine := v.renderStatusLine("BULK")

	// Join with status line
	return lipgloss.JoinVertical(lipgloss.Left, mainContainer, statusLine)
}

// Implement CoreViewKeymap interface to control key behavior

// IsKeyDisabled returns true if the given key should be ignored for this view
func (v *BulkView) IsKeyDisabled(keyString string) bool {
	// We don't disable any keys - we handle them in HandleKey instead
	return false
}

// HandleKey allows views to provide custom handling for specific keys
func (v *BulkView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	keyString := keyMsg.String()
	debug.LogToFilef("DEBUG: BulkView.HandleKey - received key '%s', mode=%d\n", keyString, v.mode)
	
	// Handle 'q' key for all modes - should go back, not quit app
	if keyString == "q" {
		debug.LogToFilef("DEBUG: BulkView.HandleKey - handling 'q' key for navigation\n")
		
		// In ModeInstructions, 'q' goes back to dashboard
		if v.mode == ModeInstructions {
			debug.LogToFilef("DEBUG: BulkView.HandleKey - 'q' in instructions mode, going back to dashboard\n")
			return true, v, func() tea.Msg {
				return messages.NavigateBackMsg{}
			}
		}
		
		// In ModeFileBrowser with INPUT mode, 'q' is text input
		if v.mode == ModeFileBrowser && v.fileSelector != nil && v.fileSelector.GetInputMode() {
			debug.LogToFilef("DEBUG: BulkView.HandleKey - passing 'q' to file selector as text input\n")
			newFileSelector, cmd := v.fileSelector.Update(keyMsg)
			v.fileSelector = newFileSelector
			return true, v, cmd
		}
		
		// In other modes, delegate to mode-specific handlers
	}
	
	// When in ModeFileBrowser with FZF INPUT mode, handle keys specially
	if v.mode == ModeFileBrowser && v.fileSelector != nil && v.fileSelector.GetInputMode() {
		debug.LogToFilef("DEBUG: BulkView.HandleKey - in FZF INPUT mode, handling key '%s'\n", keyString)
		
		// In INPUT mode, we need to intercept navigation keys and pass them to file selector
		switch keyString {
		case "backspace":
			debug.LogToFilef("DEBUG: BulkView.HandleKey - passing backspace to file selector for deletion\n")
			// Pass backspace to file selector for text deletion
			newFileSelector, cmd := v.fileSelector.Update(keyMsg)
			v.fileSelector = newFileSelector
			return true, v, cmd
			
		case "b":
			debug.LogToFilef("DEBUG: BulkView.HandleKey - passing 'b' to file selector as text input\n")
			// In INPUT mode, 'b' is just a character to type, not back!
			newFileSelector, cmd := v.fileSelector.Update(keyMsg)
			v.fileSelector = newFileSelector
			return true, v, cmd
		}
	}
	
	debug.LogToFilef("DEBUG: BulkView.HandleKey - not handling key '%s', returning false\n", keyString)
	// Let the default Update method handle everything else
	return false, v, nil
}
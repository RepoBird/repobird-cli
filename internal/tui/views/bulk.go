package views

import (
	"strings"
	
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// BulkMode represents the current mode of the bulk view
type BulkMode int

const (
	ModeFileSelect BulkMode = iota
	ModeRunList
	ModeRunEdit
	ModeProgress
	ModeResults
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
	fileSelector *components.BulkFileSelector
	runEditor    *RunEditor
	progressView *BulkProgressView
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
	spinner spinner.Model
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

// NewBulkFZFView creates a new bulk FZF view (compatibility function)
func NewBulkFZFView(client *api.Client) *BulkView {
	return NewBulkView(client)
}

// NewBulkView creates a new bulk view
func NewBulkView(client *api.Client) *BulkView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	fileSelector := components.NewBulkFileSelector(80, 24)
	// Don't activate immediately - show instructions first
	
	return &BulkView{
		client:       client,
		mode:         ModeFileSelect,
		fileSelector: fileSelector,
		runEditor:    NewRunEditor(),
		progressView: NewBulkProgressView(),
		help:         help.New(),
		keys:         defaultBulkKeyMap(),
		spinner:      s,
		runType:      "run",
		runs:         []BulkRunItem{},
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
	return v.spinner.Tick
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

	case tea.KeyMsg:
		debug.LogToFilef("DEBUG: BulkView - handling KeyMsg: '%s', mode=%d\n", msg.String(), v.mode)
		switch v.mode {
		case ModeFileSelect:
			debug.LogToFile("DEBUG: BulkView - delegating to handleFileSelectKeys\n")
			return v.handleFileSelectKeys(msg)
		case ModeRunList:
			debug.LogToFile("DEBUG: BulkView - delegating to handleRunListKeys\n")
			return v.handleRunListKeys(msg)
		case ModeRunEdit:
			debug.LogToFile("DEBUG: BulkView - delegating to handleRunEditKeys\n")
			return v.handleRunEditKeys(msg)
		case ModeProgress:
			debug.LogToFile("DEBUG: BulkView - delegating to handleProgressKeys\n")
			return v.handleProgressKeys(msg)
		case ModeResults:
			debug.LogToFile("DEBUG: BulkView - delegating to handleResultsKeys\n")
			return v.handleResultsKeys(msg)
		}

	case fileSelectedMsg:
		// File(s) selected, load configurations
		return v.loadFiles(msg.files)

	case bulkRunsLoadedMsg:
		// Runs loaded from files
		v.runs = msg.runs
		v.repository = msg.repository
		v.repoID = msg.repoID
		v.sourceBranch = msg.source
		v.runType = msg.runType
		v.batchTitle = msg.batchTitle
		v.mode = ModeRunList
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

	case bulkProgressMsg:
		// Progress update received
		v.progressView.UpdateProgress(msg)
		if msg.completed {
			v.mode = ModeResults
		}
		return v, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update sub-components based on mode
	switch v.mode {
	case ModeRunEdit:
		cmd := v.runEditor.UpdateRunEditor(msg)
		cmds = append(cmds, cmd)
	case ModeProgress:
		cmd := v.progressView.UpdateProgressView(msg)
		cmds = append(cmds, cmd)
	}

	debug.LogToFilef("DEBUG: BulkView.Update() - returning with %d commands\n", len(cmds))
	return v, tea.Batch(cmds...)
}

// Event handlers for different modes
func (v *BulkView) handleFileSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: BulkView.handleFileSelectKeys() - key='%s'\n", msg.String())
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.ListMode):
		if len(v.runs) > 0 {
			v.mode = ModeRunList
		}
		return v, nil
	case key.Matches(msg, v.keys.FileMode):
		// 'F' key pressed - activate file selector and start loading files
		debug.LogToFilef("DEBUG: BulkView.handleFileSelectKeys() - 'F' key pressed, activating file selector\n")
		cmd := v.fileSelector.Activate()
		return v, cmd
	default:
		// File selector doesn't need standard Bubble Tea updates
		debug.LogToFilef("DEBUG: BulkView.handleFileSelectKeys() - unhandled key: '%s'\n", msg.String())
		return v, nil
	}
}

func (v *BulkView) handleRunListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
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
	case key.Matches(msg, v.keys.Submit):
		return v, v.submitBulkRuns()
	case key.Matches(msg, v.keys.FileMode):
		v.mode = ModeFileSelect
	}
	return v, nil
}

func (v *BulkView) handleRunEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, v.keys.Cancel) {
		v.mode = ModeRunList
		return v, nil
	}
	cmd := v.runEditor.UpdateRunEditor(msg)
	return v, cmd
}

func (v *BulkView) handleProgressKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.Cancel):
		return v, v.cancelBatch()
	}
	return v, nil
}

func (v *BulkView) handleResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.FileMode):
		v.mode = ModeFileSelect
		v.runs = []BulkRunItem{}
		v.results = []BulkRunResult{}
		v.error = nil
		v.batchID = ""
	}
	return v, nil
}

// Rendering methods
func (v *BulkView) renderFileSelect() string {
	debug.LogToFile("DEBUG: BulkView.renderFileSelect() called\n")
	if v.fileSelector == nil {
		debug.LogToFile("DEBUG: BulkView.renderFileSelect() - fileSelector is nil!\n")
		return "File selector not initialized"
	}
	
	// Check if file selector is active
	if !v.fileSelector.IsActive() {
		debug.LogToFile("DEBUG: BulkView.renderFileSelect() - showing instructions (selector not active)\n")
		return v.renderBulkInstructions()
	}
	
	debug.LogToFile("DEBUG: BulkView.renderFileSelect() - calling fileSelector.View()\n")
	result := v.fileSelector.View(nil) // Pass nil for StatusLine - will need to be fixed
	debug.LogToFilef("DEBUG: BulkView.renderFileSelect() - result length: %d\n", len(result))
	return result
}

func (v *BulkView) renderRunList() string {
	return "Run List Mode - Implementation in separate files"
}

func (v *BulkView) renderRunEdit() string {
	return v.runEditor.View()
}

func (v *BulkView) renderProgress() string {
	return v.progressView.View()
}

func (v *BulkView) renderResults() string {
	return "Results Mode - Implementation in separate files"
}

// renderBulkInstructions renders the initial instruction screen for bulk operations
func (v *BulkView) renderBulkInstructions() string {
	debug.LogToFile("DEBUG: BulkView.renderBulkInstructions() called\n")
	
	if v.width <= 0 || v.height <= 0 {
		return "⟳ Initializing Bulk Instructions..."
	}

	// Calculate available space
	availableHeight := v.height - 3 // Reserve space for title and status
	availableWidth := v.width - 4   // Account for padding

	// Create instruction text
	instructions := []string{
		"📦 Bulk Run Operations",
		"",
		"This view allows you to run multiple AI tasks from configuration files.",
		"",
		"Instructions:",
		"• Press 'F' to browse and select configuration files",
		"• Supported formats: JSON, YAML, JSONL, Markdown",
		"• You can select multiple files to process together",
		"• Each file can contain one or more run configurations",
		"",
		"Key Bindings:",
		"• F     - Browse files (Fuzzy finder)",
		"• L     - View run list (if runs are loaded)",
		"• q     - Quit to dashboard",
		"• ?     - Show help",
		"",
		"Ready to get started? Press 'F' to browse files!",
	}

	// Style for the instructions
	instructionStyle := lipgloss.NewStyle().
		Width(availableWidth).
		Height(availableHeight).
		Padding(2).
		Margin(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Align(lipgloss.Center, lipgloss.Center)

	// Join instructions and apply styling
	content := strings.Join(instructions, "\n")
	styledContent := instructionStyle.Render(content)

	return styledContent
}

// View renders the bulk view
func (v *BulkView) View() string {
	debug.LogToFilef("DEBUG: BulkView.View() called - mode=%d, width=%d, height=%d\n", v.mode, v.width, v.height)
	
	if v.width <= 0 || v.height <= 0 {
		debug.LogToFile("DEBUG: BulkView - no dimensions, returning initializing message\n")
		return "⟳ Initializing Bulk View..."
	}
	
	switch v.mode {
	case ModeFileSelect:
		debug.LogToFile("DEBUG: BulkView - rendering file select\n")
		return v.renderFileSelect()
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

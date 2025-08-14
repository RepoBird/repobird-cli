package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
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
	ModeFileBrowser                  // FZF file browser (full screen)
	ModeRunList                      // Run validation list
	ModeRunEdit                      // Individual run editing
	ModeProgress                     // Submission progress
	ModeResults                      // Final results
	ModeExamples                     // Examples view with yank functionality
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
	mode           BulkMode
	fileSelector   *components.BulkFileSelector // Only used in ModeFileBrowser
	help           help.Model
	keys           bulkKeyMap
	width          int
	height         int
	viewport       viewport.Model // For scrollable content in RunList mode
	selectedButton int            // For button navigation
	focusMode      string         // "runs" or "buttons" for run list navigation
	selectedExample int           // Selected example in examples view

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

	vp := viewport.New(80, 20) // Default size, will be updated
	vp.YPosition = 0

	return &BulkView{
		client:         client,
		mode:           ModeInstructions, // Start with instructions, not file selector
		help:           help.New(),
		keys:           defaultBulkKeyMap(),
		spinner:        s,
		runType:        "run",
		runs:           []BulkRunItem{},
		statusLine:     components.NewStatusLine(),
		viewport:       vp,
		selectedButton: 1, // Start with first button selected
		focusMode:      "runs", // Start focused on runs
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

		// Update viewport dimensions from layout
		// The viewport needs the content area dimensions
		viewportWidth, viewportHeight := v.layout.GetContentDimensions()
		debug.LogToFilef("🎯 BULK WindowSize: terminal=%dx%d, content=%dx%d\n",
			msg.Width, msg.Height, viewportWidth, viewportHeight)

		// The viewport needs to fit inside the box's content area
		// Box has border (2 chars) and horizontal padding (2 chars)
		v.viewport.Width = viewportWidth - 2 // Account for horizontal padding
		// Reduce height to prevent content overflow
		v.viewport.Height = viewportHeight - 2 // Account for border expansion

		debug.LogToFilef("🎯 BULK WindowSize: viewport set to %dx%d\n",
			v.viewport.Width, v.viewport.Height)

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
					if msg.Type == tea.KeyEsc || msg.String() == "q" || msg.String() == "L" {
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
			debug.LogToFilef("DEBUG: BulkView - delegating to handleRunListKeys, key='%s'\n", msg.String())
			return v.handleRunListKeys(msg)
		case ModeExamples:
			debug.LogToFile("DEBUG: BulkView - delegating to handleExamplesKeys\n")
			return v.handleExamplesKeys(msg)
		}

	case components.BulkFileSelectedMsg:
		// File(s) selected, load configurations from actual files
		debug.LogToFilef("DEBUG: BulkView - files selected: %v, canceled: %v\n", msg.Files, msg.Canceled)
		if msg.Canceled {
			// User canceled file selection
			debug.LogToFile("DEBUG: BulkView - file selection canceled, returning to parent BULK mode\n")
			v.fileSelector = nil // Clear file selector

			// Always go back to the parent view (where we came from)
			// If we have runs already, we came from run list, otherwise from instructions
			if len(v.runs) > 0 {
				debug.LogToFile("DEBUG: BulkView - returning to ModeRunList (has runs)\n")
				v.mode = ModeRunList
				v.focusMode = "runs"
				v.updateRunListViewport()
			} else {
				debug.LogToFile("DEBUG: BulkView - returning to ModeInstructions (no runs)\n")
				v.mode = ModeInstructions
			}
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
		v.focusMode = "runs" // Start with runs focused
		// Update viewport content for run list
		v.updateRunListViewport()
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
		debug.LogToFilef("DEBUG: BulkView - error occurred: %v\n", msg.err)
		v.error = msg.err
		v.fileSelector = nil // Clear file selector

		// If we have runs already loaded, stay in run list mode
		// Otherwise go to instructions to show the error
		if len(v.runs) > 0 {
			v.mode = ModeRunList
			v.focusMode = "runs"
			v.updateRunListViewport()
		} else {
			v.mode = ModeInstructions
		}
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
	case key.Matches(msg, v.keys.Up):
		// Navigate buttons up
		if v.selectedButton > 1 {
			v.selectedButton--
		}
		return v, nil
	case key.Matches(msg, v.keys.Down):
		// Navigate buttons down
		maxButton := 3 // Files, Examples, Dashboard
		if len(v.runs) > 0 {
			maxButton = 4 // Files, Examples, Runs, Dashboard
		}
		if v.selectedButton < maxButton {
			v.selectedButton++
		}
		return v, nil
	case key.Matches(msg, v.keys.PageUp):
		v.viewport.HalfViewUp()
		return v, nil
	case key.Matches(msg, v.keys.PageDown):
		v.viewport.HalfViewDown()
		return v, nil
	case msg.String() == "enter" || msg.String() == " ":
		// Handle button activation
		switch v.selectedButton {
		case 1: // Select Files
			v.mode = ModeFileBrowser
			if v.fileSelector == nil {
				v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
			}
			return v, v.fileSelector.Activate()
		case 2: // Examples
			v.mode = ModeExamples
			v.selectedExample = 0
			return v, nil
		case 3: // View Runs or Dashboard
			if len(v.runs) > 0 {
				// View Runs button when runs exist
				v.mode = ModeRunList
				v.focusMode = "runs"
				v.updateRunListViewport()
			} else {
				// Dashboard button when no runs
				return v, func() tea.Msg {
					return messages.NavigateToDashboardMsg{}
				}
			}
		case 4: // Dashboard (when runs exist)
			return v, func() tea.Msg {
				return messages.NavigateToDashboardMsg{}
			}
		}
		return v, nil
	case key.Matches(msg, v.keys.FZF) || msg.String() == "f":
		// Switch to file browser mode and initialize file selector
		v.mode = ModeFileBrowser
		if v.fileSelector == nil {
			v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		}
		return v, v.fileSelector.Activate()
	case key.Matches(msg, v.keys.ListMode), msg.String() == "L":
		if len(v.runs) > 0 {
			v.mode = ModeRunList
			v.focusMode = "runs"
			v.updateRunListViewport()
		}
		return v, nil
	case msg.String() == "d":
		// Quick dashboard navigation
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
	default:
		// Pass other keys to viewport for scrolling
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
}

// handleFileBrowserKeys handles navigation keys in the file browser mode
func (v *BulkView) handleFileBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: BulkView.handleFileBrowserKeys() - navigation key='%s'\n", msg.String())
	switch {
	case key.Matches(msg, v.keys.Quit) || msg.Type == tea.KeyEsc:
		// Clear file selector
		v.fileSelector = nil

		// If we have runs, go to run list, otherwise instructions
		if len(v.runs) > 0 {
			v.mode = ModeRunList
			v.focusMode = "runs"
			v.updateRunListViewport()
		} else {
			v.mode = ModeInstructions
		}
		return v, nil
	case key.Matches(msg, v.keys.ListMode):
		// Switch to run list mode if runs exist
		if len(v.runs) > 0 {
			v.mode = ModeRunList
			v.focusMode = "runs"
			v.updateRunListViewport()
		}
		return v, nil
	default:
		// This should only be called for navigation keys now
		debug.LogToFilef("DEBUG: BulkView.handleFileBrowserKeys() - unhandled navigation key: '%s'\n", msg.String())
		return v, nil
	}
}

func (v *BulkView) handleRunListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	debug.LogToFilef("DEBUG: handleRunListKeys called with key='%s', focusMode='%s', selectedButton=%d\n", 
		msg.String(), v.focusMode, v.selectedButton)

	switch {
	case key.Matches(msg, v.keys.Quit):
		// Navigate back to dashboard instead of quitting directly
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

	case key.Matches(msg, v.keys.Up), msg.String() == "k":
		// Navigate up in runs or to buttons
		if v.focusMode == "runs" {
			if v.selectedRun > 0 {
				v.selectedRun--
				v.updateRunListViewport()
				v.ensureSelectedVisible()
			} else {
				// At top of runs, switch to buttons
				v.focusMode = "buttons"
				v.selectedButton = 2 // Select [DASH] button
				v.ensureButtonsVisible()
			}
		} else {
			// In buttons mode, navigate up
			if v.selectedButton > 1 {
				v.selectedButton--
			} else {
				// At top button, wrap to runs
				v.focusMode = "runs"
				v.selectedRun = len(v.runs) - 1
				v.ensureSelectedVisible()
			}
		}

	case key.Matches(msg, v.keys.Down), msg.String() == "j":
		// Navigate down in runs or to buttons
		if v.focusMode == "runs" {
			if v.selectedRun < len(v.runs)-1 {
				v.selectedRun++
				v.updateRunListViewport()
				v.ensureSelectedVisible()
			} else {
				// At bottom of runs, switch to buttons
				v.focusMode = "buttons"
				v.selectedButton = 1 // Select [FZF-FILES] button
				v.ensureButtonsVisible()
			}
		} else {
			// In buttons mode, navigate down
			if v.selectedButton < 2 {
				v.selectedButton++
			} else {
				// At bottom button, wrap to runs
				v.focusMode = "runs"
				v.selectedRun = 0
				v.ensureSelectedVisible()
			}
		}

	case msg.String() == "tab":
		// Toggle between runs and buttons
		if v.focusMode == "runs" {
			v.focusMode = "buttons"
			v.selectedButton = 1
			v.ensureButtonsVisible()
		} else {
			v.focusMode = "runs"
			if v.selectedRun >= len(v.runs) {
				v.selectedRun = 0
			}
			v.ensureSelectedVisible()
		}

	case key.Matches(msg, v.keys.PageUp):
		v.viewport.HalfViewUp()

	case key.Matches(msg, v.keys.PageDown):
		v.viewport.HalfViewDown()

	case key.Matches(msg, v.keys.Select), msg.String() == " ":
		// Toggle selection for current run (only in runs mode)
		if v.focusMode == "runs" && v.selectedRun < len(v.runs) {
			v.runs[v.selectedRun].Selected = !v.runs[v.selectedRun].Selected
			v.updateRunListViewport()
		}

	case msg.String() == "enter":
		debug.LogToFile("DEBUG: Enter key pressed in handleRunListKeys\n")
		debug.LogToFilef("DEBUG: focusMode=%s, selectedButton=%d\n", v.focusMode, v.selectedButton)
		
		if v.focusMode == "buttons" {
			// Handle button selection
			debug.LogToFile("DEBUG: In buttons mode\n")
			
			if v.selectedButton == 1 {
				// [FZF-FILES] button
				debug.LogToFile("DEBUG: Button 1 selected - switching to file browser\n")
				v.mode = ModeFileBrowser
				if v.fileSelector == nil {
					v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
				}
				return v, v.fileSelector.Activate()
			}
			
			if v.selectedButton == 2 {
				// [DASH] button
				debug.LogToFile("DEBUG: Button 2 selected - navigating to dashboard\n")
				return v, func() tea.Msg {
					return messages.NavigateToDashboardMsg{}
				}
			}
			
			// Shouldn't reach here, but return to be safe
			debug.LogToFile("DEBUG: Unknown button selected\n")
			return v, nil
		}
		
		// In runs mode - submit selected bulk runs
		debug.LogToFile("DEBUG: In runs mode, submitting bulk runs\n")
		return v, v.submitBulkRuns()

	case key.Matches(msg, v.keys.FZF), msg.String() == "f":
		// Switch back to file browser mode
		v.mode = ModeFileBrowser
		if v.fileSelector == nil {
			v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		}
		return v, v.fileSelector.Activate()

	case msg.String() == "d":
		// Quick dashboard navigation
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}

	case key.Matches(msg, v.keys.Submit):
		// Submit selected bulk runs
		return v, v.submitBulkRuns()
	default:
		// Pass other keys to viewport
		var vpCmd tea.Cmd
		v.viewport, vpCmd = v.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return v, tea.Batch(cmds...)
}

// updateRunListViewport is now called from renderRunList to update content
func (v *BulkView) updateRunListViewport() {
	// Content is now updated directly in renderRunList
	// This method is kept for compatibility with existing calls
}

// ensureSelectedVisible ensures the selected item is visible in the viewport
func (v *BulkView) ensureSelectedVisible() {
	if v.focusMode == "buttons" {
		// Ensure buttons are visible
		v.ensureButtonsVisible()
	} else {
		// Calculate the line position of the selected item
		lineHeight := 1 // Each item takes 1 line
		selectedLine := v.selectedRun * lineHeight

		// Get current viewport position
		viewportTop := v.viewport.YOffset
		viewportBottom := viewportTop + v.viewport.Height - 1

		// Adjust viewport if selected item is not visible
		if selectedLine < viewportTop {
			// Selected item is above viewport
			v.viewport.SetYOffset(selectedLine)
		} else if selectedLine > viewportBottom {
			// Selected item is below viewport
			newOffset := selectedLine - v.viewport.Height + 1
			if newOffset < 0 {
				newOffset = 0
			}
			v.viewport.SetYOffset(newOffset)
		}
	}
}

// ensureButtonsVisible scrolls the viewport to show the buttons
func (v *BulkView) ensureButtonsVisible() {
	// Calculate where buttons are in the content
	// Format: Title (1 line) + separator (1 line) + blank (1 line) + 
	// instructions (2 lines) + blank (1 line) + header (1 line) + blank (1 line) +
	// all runs + blank (2 lines) + buttons (2 lines)
	buttonStartLine := 8 + len(v.runs) + 2 // Header lines + runs + spacing
	
	// Get current viewport position
	viewportTop := v.viewport.YOffset
	viewportBottom := viewportTop + v.viewport.Height - 1
	
	// Check if buttons are visible
	if buttonStartLine > viewportBottom {
		// Buttons are below viewport, scroll down to show them
		// Position so buttons are at the bottom of the viewport with some margin
		newOffset := buttonStartLine - v.viewport.Height + 4 // Show buttons with some context
		if newOffset < 0 {
			newOffset = 0
		}
		v.viewport.SetYOffset(newOffset)
		debug.LogToFilef("DEBUG: Scrolling to show buttons at line %d, new offset=%d\n", buttonStartLine, newOffset)
	} else if buttonStartLine < viewportTop {
		// Buttons are above viewport (rare case when scrolled too far down)
		v.viewport.SetYOffset(buttonStartLine - 2) // Show with some context above
		debug.LogToFilef("DEBUG: Scrolling up to show buttons at line %d\n", buttonStartLine)
	}
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
		debug.LogToFilef("🎯🎯🎯 ENTERING renderInstructions mode=0\n")
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
	case ModeExamples:
		debug.LogToFile("DEBUG: BulkView - rendering examples\n")
		return v.renderExamples()
	default:
		debug.LogToFilef("DEBUG: BulkView - unknown mode: %d\n", v.mode)
		return "Unknown mode"
	}
}

// renderInstructions renders the initial instructions screen with scrollable content
func (v *BulkView) renderInstructions() string {
	debug.LogToFilef("🎯 BULK renderInstructions: width=%d, height=%d\n", v.width, v.height)

	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Build complete content with clear title
	var fullContent strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	fullContent.WriteString(titleStyle.Render("📋 Bulk Operations") + "\n\n")

	if v.error != nil {
		// Show error message
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

		fullContent.WriteString(errorStyle.Render("❌ Error loading files:") + "\n")
		fullContent.WriteString(fmt.Sprintf("%v", v.error) + "\n\n")
		fullContent.WriteString("Press f to try again.\n\n")
	} else if len(v.runs) > 0 {
		// This shouldn't happen - if runs are loaded we should be in run list mode
		// But just in case, provide a way to get there
		fullContent.WriteString(fmt.Sprintf("✓ %d runs loaded\n\n", len(v.runs)))
		fullContent.WriteString("Press L to view runs\n\n")
	} else {
		// Concise format info and field reference
		fullContent.WriteString("Formats: JSON, YAML, or Markdown (with frontmatter). Multiple runs per file supported.\n")
		fullContent.WriteString("Fields: repository* prompt* | title target context source runType batchTitle force\n")
	}

	// Add buttons section
	fullContent.WriteString("\n")
	fullContent.WriteString(lipgloss.NewStyle().Bold(true).Render("Actions:") + "\n\n")

	// Simple button styles - no borders, just text
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Show buttons vertically (one per row)
	// Button 1: Files
	if v.selectedButton == 1 {
		fullContent.WriteString(selectedStyle.Render("▸ 📁 Files") + "\n")
	} else {
		fullContent.WriteString(normalStyle.Render("  📁 Files") + "\n")
	}

	// Button 2: Examples (always shown)
	buttonNum := 2
	if v.selectedButton == buttonNum {
		fullContent.WriteString(selectedStyle.Render("▸ 📚 Examples") + "\n")
	} else {
		fullContent.WriteString(normalStyle.Render("  📚 Examples") + "\n")
	}
	
	// Button 3: View Runs (only if runs loaded)
	if len(v.runs) > 0 {
		buttonNum++
		if v.selectedButton == buttonNum {
			fullContent.WriteString(selectedStyle.Render("▸ 📋 Runs") + "\n")
		} else {
			fullContent.WriteString(normalStyle.Render("  📋 Runs") + "\n")
		}
	}

	// Button 4 (or 3): Dashboard
	buttonNum++
	if v.selectedButton == buttonNum {
		fullContent.WriteString(selectedStyle.Render("▸ [DASH]") + "\n")
	} else {
		fullContent.WriteString(normalStyle.Render("  [DASH]") + "\n")
	}

	debug.LogToFilef("🎯 BULK: buttons shown vertically, selected=%d\n", v.selectedButton)

	// Add navigation hint
	fullContent.WriteString("\n")
	fullContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true).Render("↑↓ nav • Enter select") + "\n")

	debug.LogToFilef("🎯 BULK total content built: %d lines\n", strings.Count(fullContent.String(), "\n"))

	// Set viewport content
	contentStr := fullContent.String()
	debug.LogToFilef("🎯 BULK instructions content length: %d chars, %d lines\n", len(contentStr), strings.Count(contentStr, "\n"))
	v.viewport.SetContent(contentStr)

	debug.LogToFilef("🎯 BULK viewport dimensions: width=%d, height=%d\n", v.viewport.Width, v.viewport.Height)
	debug.LogToFilef("🎯 BULK terminal dimensions: width=%d, height=%d\n", v.width, v.height)

	// Create box for the viewport - leave room for borders and status line
	// Need extra space to prevent border cutoff
	boxWidth := v.width - 2
	boxHeight := v.height - 4 // Extra space to prevent top border cutoff

	debug.LogToFilef("🎯 BULK box dimensions: width=%d, height=%d\n", boxWidth, boxHeight)
	debug.LogToFilef("🎯 BULK content lines: %d, viewport can show: %d lines\n", strings.Count(contentStr, "\n"), v.viewport.Height)

	// Check if content overflows viewport
	contentLines := strings.Count(contentStr, "\n")
	if contentLines > v.viewport.Height {
		debug.LogToFilef("⚠️ BULK OVERFLOW: Content has %d lines but viewport only shows %d lines\n", contentLines, v.viewport.Height)
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1). // Horizontal padding only
		Width(boxWidth).
		Height(boxHeight)

	// Render viewport in box
	viewportView := v.viewport.View()
	debug.LogToFilef("🎯 BULK viewport view lines: %d\n", strings.Count(viewportView, "\n"))

	// Check if content overflows and needs scrolling
	var scrollIndicator string
	if !v.viewport.AtTop() || !v.viewport.AtBottom() {
		scrollPercent := int(float64(v.viewport.YOffset) / float64(v.viewport.TotalLineCount()-v.viewport.Height) * 100)
		if scrollPercent < 0 {
			scrollPercent = 0
		}
		if scrollPercent > 100 {
			scrollPercent = 100
		}
		scrollIndicator = fmt.Sprintf("[%d%%]", scrollPercent)
		debug.LogToFilef("🎯 BULK scroll: offset=%d, total=%d, height=%d, percent=%d%%\n",
			v.viewport.YOffset, v.viewport.TotalLineCount(), v.viewport.Height, scrollPercent)
	}

	boxedContent := boxStyle.Render(viewportView)
	debug.LogToFilef("🎯 BULK boxed content lines: %d\n", strings.Count(boxedContent, "\n"))

	// Status line with scroll indicator on the right
	statusLine := v.renderStatusLineWithScroll("BULK", scrollIndicator)

	// Join with status line
	final := lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
	debug.LogToFilef("🎯 BULK final output lines: %d\n", strings.Count(final, "\n"))
	debug.LogToFilef("🎯 BULK final height should be: %d (terminal height)\n", v.height)

	return final
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

// renderRunList renders the run list view with scrollable viewport
func (v *BulkView) renderRunList() string {
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Count selected runs
	selectedCount := 0
	for _, run := range v.runs {
		if run.Selected {
			selectedCount++
		}
	}

	// Build the complete content for viewport
	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	content.WriteString(titleStyle.Render(fmt.Sprintf("📋 Bulk Runs (%d total, %d selected)", len(v.runs), selectedCount)))
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", 50) + "\n\n")

	// Instructions
	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	content.WriteString(instructionStyle.Render("↑↓/j/k navigate, space select, enter submit"))
	content.WriteString("\n\n")

	// Add header for runs
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	content.WriteString(headerStyle.Render("Agentic Runs Extracted from Files:") + "\n\n")

	// Run items
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	normalStyle := lipgloss.NewStyle()

	if len(v.runs) == 0 {
		content.WriteString(lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("243")).Render("No runs loaded yet. Press f to add files."))
		content.WriteString("\n")
	} else {
		for i, run := range v.runs {
			statusIcon := "[ ]"
			if run.Selected {
				statusIcon = "[✓]"
			}

			runTitle := run.Title
			if runTitle == "" {
				runTitle = fmt.Sprintf("Run %d", i+1)
			}

			// Truncate title if too long
			maxTitleLen := v.viewport.Width - 10
			if maxTitleLen > 0 && len(runTitle) > maxTitleLen {
				runTitle = runTitle[:maxTitleLen-3] + "..."
			}

			line := fmt.Sprintf("%s %s", statusIcon, runTitle)

			// Only show selection marker when focus is on runs
			if v.focusMode == "runs" && i == v.selectedRun {
				content.WriteString(selectedStyle.Render("▸ " + line))
			} else {
				content.WriteString(normalStyle.Render("  " + line))
			}
			content.WriteString("\n")
		}
	}

	// Add spacing
	content.WriteString("\n\n")

	// Simple button styles - no borders, just text
	normalBtnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	selectedBtnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Navigation buttons with selection highlight
	if v.focusMode == "buttons" && v.selectedButton == 1 {
		content.WriteString(selectedBtnStyle.Render("▸ [FZF-FILES]") + "\n")
	} else {
		content.WriteString(normalBtnStyle.Render("  [FZF-FILES]") + "\n")
	}
	
	if v.focusMode == "buttons" && v.selectedButton == 2 {
		content.WriteString(selectedBtnStyle.Render("▸ [DASH]") + "\n")
	} else {
		content.WriteString(normalBtnStyle.Render("  [DASH]") + "\n")
	}

	// Set viewport content
	v.viewport.SetContent(content.String())

	// Create box for the viewport - adjust for border cutoff
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(v.width - 2).
		Height(v.height - 3) // Account for status line and prevent border cutoff

	// Check if content overflows and needs scrolling
	var scrollIndicator string
	if !v.viewport.AtTop() || !v.viewport.AtBottom() {
		percentScrolled := v.viewport.ScrollPercent()
		position := "TOP"
		if v.viewport.AtBottom() {
			position = "BOTTOM"
		} else if percentScrolled > 0 {
			position = fmt.Sprintf("%d%%", int(percentScrolled*100))
		}
		scrollIndicator = fmt.Sprintf("[%s]", position)
	}

	// Render viewport in box
	viewportView := v.viewport.View()
	boxedContent := boxStyle.Render(viewportView)

	// Status line with scroll indicator on the right
	statusLine := v.renderStatusLineWithScroll("BULK", scrollIndicator)

	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
}

// renderStatusLine renders the status line
func (v *BulkView) renderStatusLine(layoutName string) string {
	return v.renderStatusLineWithScroll(layoutName, "")
}

// renderStatusLineWithScroll renders the status line with optional scroll indicator
func (v *BulkView) renderStatusLineWithScroll(layoutName string, scrollIndicator string) string {
	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter(layoutName, v.width)

	// Simple help text based on current mode
	var helpText string
	var mode string

	switch v.mode {
	case ModeInstructions:
		helpText = "↑↓:nav enter:select f:files [h]back [q]dashboard"
	case ModeFileBrowser:
		if v.fileSelector != nil && v.fileSelector.GetInputMode() {
			mode = "INPUT"
			helpText = "↑↓:nav space:select type:filter esc:nav mode enter:confirm ctrl+a:all"
		} else {
			mode = "NAV"
			helpText = "↑↓/j/k:nav space:select i:input mode esc:back enter:confirm"
		}
	case ModeRunList:
		if v.focusMode == "buttons" {
			helpText = "↑↓:nav enter:select tab:switch-to-runs [q]dashboard"
		} else {
			helpText = "↑↓/j/k:nav space:toggle enter:submit tab:buttons f:files [q]dashboard"
		}
	default:
		helpText = "[h]back [q]dashboard ?:help"
	}

	// Format left content consistently
	leftContent := formatter.FormatViewNameWithMode(mode)

	// Create status line using formatter
	statusLine := formatter.StandardStatusLine(leftContent, scrollIndicator, helpText)
	return statusLine.Render()
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

	// Handle ESC key specially in ModeFileBrowser
	if keyString == "esc" && v.mode == ModeFileBrowser {
		debug.LogToFilef("DEBUG: BulkView.HandleKey - handling 'esc' in FileBrowser mode\n")

		// Pass ESC to the file selector to handle the two-stage exit
		// First ESC: exit INPUT mode to NAV mode
		// Second ESC: cancel file selection and return to parent BULK mode
		if v.fileSelector != nil {
			newFileSelector, cmd := v.fileSelector.Update(keyMsg)
			v.fileSelector = newFileSelector

			// The file selector will send BulkFileSelectedMsg{Canceled: true}
			// when it wants to exit, which we handle in Update()
			return true, v, cmd
		}
	}

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

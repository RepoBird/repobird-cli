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
	mode             BulkMode
	fileSelector     *components.BulkFileSelector // Only used in ModeFileBrowser
	help             help.Model
	keys             bulkKeyMap
	width            int
	height           int
	viewport         viewport.Model // For scrollable content in RunList mode
	selectedButton   int            // 0=none/runs, 1=add files, 2=submit, 3=dashboard
	navigationFocus  string         // "runs" or "buttons"

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
		client:          client,
		mode:            ModeInstructions, // Start with instructions, not file selector
		help:            help.New(),
		keys:            defaultBulkKeyMap(),
		spinner:         s,
		runType:         "run",
		runs:            []BulkRunItem{},
		statusLine:      components.NewStatusLine(),
		viewport:        vp,
		selectedButton:  1, // Start with first button selected
		navigationFocus: "runs",
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
		
		// Subtract space for border, padding, and title
		v.viewport.Width = viewportWidth - 2  // Account for padding
		v.viewport.Height = viewportHeight - 3 // Account for title, padding, and margin
		
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
			debug.LogToFile("DEBUG: BulkView - delegating to handleRunListKeys\n")
			return v.handleRunListKeys(msg)
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
			v.updateRunListViewport()
		}
		return v, nil
	case msg.String() == "d":
		// Quick dashboard navigation
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
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
		// Clear file selector
		v.fileSelector = nil
		
		// If we have runs, go to run list, otherwise instructions
		if len(v.runs) > 0 {
			v.mode = ModeRunList
			v.updateRunListViewport()
		} else {
			v.mode = ModeInstructions
		}
		return v, nil
	case key.Matches(msg, v.keys.ListMode):
		// Switch to run list mode if runs exist
		if len(v.runs) > 0 {
			v.mode = ModeRunList
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
	
	switch {
	case key.Matches(msg, v.keys.Quit):
		// Navigate back to dashboard instead of quitting directly
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}
	case msg.String() == "tab":
		// Switch focus between runs and buttons
		if v.navigationFocus == "runs" {
			v.navigationFocus = "buttons"
			v.selectedButton = 1 // Start with first button
		} else {
			v.navigationFocus = "runs"
			v.selectedButton = 0
		}
		v.updateRunListViewport()
		
	case key.Matches(msg, v.keys.Up):
		if v.navigationFocus == "runs" {
			if v.selectedRun > 0 {
				v.selectedRun--
			} else {
				// Wrap to buttons at bottom
				v.navigationFocus = "buttons"
				v.selectedButton = 3 // Dashboard button
			}
			v.updateRunListViewport()
			v.ensureSelectedVisible()
		} else {
			// In button navigation, up goes back to runs
			v.navigationFocus = "runs"
			v.selectedButton = 0
			v.selectedRun = len(v.runs) - 1 // Select last run
			v.updateRunListViewport()
			v.ensureSelectedVisible()
		}
		
	case key.Matches(msg, v.keys.Down):
		if v.navigationFocus == "runs" {
			if v.selectedRun < len(v.runs)-1 {
				v.selectedRun++
			} else {
				// Wrap to buttons at top
				v.navigationFocus = "buttons"
				v.selectedButton = 1 // Add files button
			}
			v.updateRunListViewport()
			v.ensureSelectedVisible()
		} else {
			// In button navigation, down goes to first run
			v.navigationFocus = "runs"
			v.selectedButton = 0
			v.selectedRun = 0
			v.updateRunListViewport()
			v.ensureSelectedVisible()
		}
		
	case key.Matches(msg, v.keys.Left):
		if v.navigationFocus == "buttons" && v.selectedButton > 1 {
			v.selectedButton--
		}
		
	case key.Matches(msg, v.keys.Right):
		if v.navigationFocus == "buttons" && v.selectedButton < 3 {
			v.selectedButton++
		}
		
	case key.Matches(msg, v.keys.PageUp):
		v.viewport.HalfViewUp()
	case key.Matches(msg, v.keys.PageDown):
		v.viewport.HalfViewDown()
		
	case key.Matches(msg, v.keys.Select), msg.String() == " ":
		if v.navigationFocus == "runs" {
			if v.selectedRun < len(v.runs) {
				v.runs[v.selectedRun].Selected = !v.runs[v.selectedRun].Selected
				v.updateRunListViewport()
			}
		} else {
			// Handle button selection with space
			return v.handleButtonPress()
		}
		
	case msg.String() == "enter":
		if v.navigationFocus == "buttons" {
			// Handle button selection with enter
			return v.handleButtonPress()
		} else {
			// Submit selected bulk runs
			return v, v.submitBulkRuns()
		}
		
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

// handleButtonPress handles button activation
func (v *BulkView) handleButtonPress() (tea.Model, tea.Cmd) {
	switch v.selectedButton {
	case 1: // Add Files
		v.mode = ModeFileBrowser
		if v.fileSelector == nil {
			v.fileSelector = components.NewBulkFileSelector(v.width, v.height)
		}
		return v, v.fileSelector.Activate()
	case 2: // Submit
		return v, v.submitBulkRuns()
	case 3: // Dashboard
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
	default:
		return v, nil
	}
}

// updateRunListViewport is now called from renderRunList to update content
func (v *BulkView) updateRunListViewport() {
	// Content is now updated directly in renderRunList
	// This method is kept for compatibility with existing calls
}

// ensureSelectedVisible ensures the selected item is visible in the viewport
func (v *BulkView) ensureSelectedVisible() {
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

	// Build complete content (title will be in box border)
	var fullContent strings.Builder
	fullContent.WriteString("📋 Bulk Operations\n")
	fullContent.WriteString(strings.Repeat("─", 50) + "\n\n")
	
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
		// Initial state - detailed instructions
		fullContent.WriteString("Process multiple runs simultaneously\n\n")
		
		fullContent.WriteString(lipgloss.NewStyle().Bold(true).Render("Required fields for each run:") + "\n")
		fullContent.WriteString("  • prompt - The task instructions\n")
		fullContent.WriteString("  • repository - Target repository (org/repo format)\n")
		fullContent.WriteString("  • source - Base branch to work from\n")
		fullContent.WriteString("  • target - New branch for changes\n")
		fullContent.WriteString("  • runType - 'run' or 'approval'\n\n")
		
		fullContent.WriteString(lipgloss.NewStyle().Bold(true).Render("Optional fields:") + "\n")
		fullContent.WriteString("  • title - Display name for the run\n")
		fullContent.WriteString("  • context - Additional context\n")
		fullContent.WriteString("  • files - Specific files to focus on\n\n")
		
		fullContent.WriteString(lipgloss.NewStyle().Bold(true).Render("Supported formats:") + "\n")
		fullContent.WriteString("  • JSON (.json) - Standard configuration\n")
		fullContent.WriteString("  • YAML (.yaml, .yml) - Alternative format\n")
		fullContent.WriteString("  • JSONL (.jsonl) - Line-delimited JSON\n")
		fullContent.WriteString("  • Markdown (.md) - With embedded configs\n\n")
		
		fullContent.WriteString("Press f to select configuration files\n")
	}
	
	// Add buttons
	fullContent.WriteString("\n")
	
	// Create button indicators
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 1).
		MarginRight(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))
	
	selectButton := buttonStyle.Render("[f] Select Files")
	dashButton := buttonStyle.Render("[d] Dashboard")
	
	var buttons string
	if len(v.runs) > 0 {
		reviewButton := buttonStyle.Render("[L] View Runs")
		buttons = lipgloss.JoinHorizontal(lipgloss.Left, selectButton, reviewButton, dashButton)
		debug.LogToFilef("🎯 BULK buttons (with runs): %s\n", buttons)
	} else {
		buttons = lipgloss.JoinHorizontal(lipgloss.Left, selectButton, dashButton)
		debug.LogToFilef("🎯 BULK buttons (no runs): %s\n", buttons)
	}
	
	fullContent.WriteString(buttons)
	debug.LogToFilef("🎯 BULK total content built: %d lines\n", strings.Count(fullContent.String(), "\n"))
	
	// Set viewport content
	contentStr := fullContent.String()
	debug.LogToFilef("🎯 BULK instructions content length: %d chars, %d lines\n", len(contentStr), strings.Count(contentStr, "\n"))
	v.viewport.SetContent(contentStr)
	
	debug.LogToFilef("🎯 BULK viewport dimensions: width=%d, height=%d\n", v.viewport.Width, v.viewport.Height)
	debug.LogToFilef("🎯 BULK terminal dimensions: width=%d, height=%d\n", v.width, v.height)
	
	// Create box for the viewport - leave room for borders and status line
	boxWidth := v.width - 2
	boxHeight := v.height - 2 // Account for status line
	
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
		Padding(0, 1).
		Width(boxWidth).
		Height(boxHeight)
	
	// Render viewport in box
	viewportView := v.viewport.View()
	debug.LogToFilef("🎯 BULK viewport view lines: %d\n", strings.Count(viewportView, "\n"))
	
	// Add scroll indicator if content overflows
	scrollIndicator := ""
	if !v.viewport.AtTop() || !v.viewport.AtBottom() {
		scrollPercent := int(float64(v.viewport.YOffset) / float64(v.viewport.TotalLineCount()-v.viewport.Height) * 100)
		if scrollPercent < 0 {
			scrollPercent = 0
		}
		if scrollPercent > 100 {
			scrollPercent = 100
		}
		scrollIndicator = fmt.Sprintf(" [%d%%]", scrollPercent)
		debug.LogToFilef("🎯 BULK scroll: offset=%d, total=%d, height=%d, percent=%d%%\n", 
			v.viewport.YOffset, v.viewport.TotalLineCount(), v.viewport.Height, scrollPercent)
	}
	
	boxedContent := boxStyle.Render(viewportView)
	debug.LogToFilef("🎯 BULK boxed content lines: %d\n", strings.Count(boxedContent, "\n"))
	
	// Status line with scroll indicator
	statusLine := v.renderStatusLine("BULK" + scrollIndicator)
	
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
	content.WriteString(instructionStyle.Render("Navigate with ↑↓, select with space, Tab to switch to buttons"))
	content.WriteString("\n\n")
	
	// Required params info
	paramStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	content.WriteString(paramStyle.Render("Required: repository, source branch, run type, prompts") + "\n")
	if v.repository != "" {
		content.WriteString(paramStyle.Render(fmt.Sprintf("Current: %s | %s | %s", v.repository, v.sourceBranch, v.runType)) + "\n")
	}
	content.WriteString("\n")
	
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
			
			if v.navigationFocus == "runs" && i == v.selectedRun {
				content.WriteString(selectedStyle.Render("▸ " + line))
			} else {
				content.WriteString(normalStyle.Render("  " + line))
			}
			content.WriteString("\n")
		}
	}
	
	// Add spacing before buttons
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", 50) + "\n")
	content.WriteString("Actions:\n\n")
	
	// Create action buttons
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 1).
		MarginRight(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))
	
	selectedButtonStyle := buttonStyle.Copy().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("0")).
		Bold(true)
	
	focusedButtonStyle := buttonStyle.Copy().
		BorderForeground(lipgloss.Color("205")).
		Foreground(lipgloss.Color("205"))
	
	// Build button row
	var buttons []string
	
	// Add Files button
	if v.navigationFocus == "buttons" && v.selectedButton == 1 {
		buttons = append(buttons, focusedButtonStyle.Render("▸ [f] Add Files"))
	} else {
		buttons = append(buttons, buttonStyle.Render("[f] Add Files"))
	}
	
	// Submit button
	submitText := "[Enter] Submit"
	if selectedCount > 0 {
		submitText = fmt.Sprintf("[Enter] Submit %d", selectedCount)
	}
	
	if v.navigationFocus == "buttons" && v.selectedButton == 2 {
		if selectedCount > 0 {
			buttons = append(buttons, selectedButtonStyle.Copy().
				BorderForeground(lipgloss.Color("205")).
				Render("▸ " + submitText))
		} else {
			buttons = append(buttons, focusedButtonStyle.Render("▸ " + submitText))
		}
	} else if selectedCount > 0 {
		buttons = append(buttons, selectedButtonStyle.Render(submitText))
	} else {
		buttons = append(buttons, buttonStyle.Render(submitText))
	}
	
	// Dashboard button
	if v.navigationFocus == "buttons" && v.selectedButton == 3 {
		buttons = append(buttons, focusedButtonStyle.Render("▸ [d] Dashboard"))
	} else {
		buttons = append(buttons, buttonStyle.Render("[d] Dashboard"))
	}
	
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, buttons...))
	
	// Set viewport content
	v.viewport.SetContent(content.String())
	
	// Create box for the viewport
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(v.width - 2).
		Height(v.height - 2) // Account for status line
	
	// Get scroll indicator
	scrollIndicator := ""
	if !v.viewport.AtTop() || !v.viewport.AtBottom() {
		percentScrolled := v.viewport.ScrollPercent()
		position := "TOP"
		if v.viewport.AtBottom() {
			position = "BOTTOM"
		} else if percentScrolled > 0 {
			position = fmt.Sprintf("%d%%", int(percentScrolled*100))
		}
		scrollIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf(" [%s]", position))
	}
	
	// Render viewport in box
	viewportView := v.viewport.View()
	boxedContent := boxStyle.Render(viewportView)
	
	// Add scroll indicator to title if needed
	if scrollIndicator != "" {
		lines := strings.Split(boxedContent, "\n")
		if len(lines) > 0 {
			// Add scroll indicator to the top-right corner
			lines[0] = lines[0][:len(lines[0])-len(scrollIndicator)-1] + scrollIndicator + lines[0][len(lines[0])-1:]
			boxedContent = strings.Join(lines, "\n")
		}
	}
	
	// Status line
	statusLine := v.renderStatusLine("BULK")
	
	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
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
		if v.navigationFocus == "buttons" {
			helpText = "←→:select-button enter/space:activate tab:switch-to-runs q:back"
		} else {
			helpText = "↑↓:navigate space:toggle tab:switch-to-buttons enter:submit q:back"
		}
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

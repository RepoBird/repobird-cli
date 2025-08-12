package views

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
)

type CreateRunView struct {
	client        APIClient
	keys          components.KeyMap
	help          help.Model
	width         int
	height        int
	focusIndex    int
	fields        []textinput.Model
	promptArea    textarea.Model
	contextArea   textarea.Model
	submitting    bool
	error         error
	success       bool
	createdRun    *models.RunResponse
	useFileInput  bool
	filePathInput textinput.Model
	// Input mode tracking
	inputMode     components.InputMode
	exitRequested bool
	// Back button
	backButtonFocused bool
	// Submit button
	submitButtonFocused bool
	// Error handling
	errorButtonFocused bool
	errorRowFocused    bool // For selecting the error message row
	// State preservation when returning from error
	prevFocusIndex          int
	prevBackButtonFocused   bool
	prevSubmitButtonFocused bool
	// Unified status line component
	statusLine *components.StatusLine
	// Clipboard feedback (still need blink timing)
	yankBlink     bool
	yankBlinkTime time.Time
	// Repository selector
	repoSelector *components.RepositorySelector
	// FZF mode for repository selection
	fzfMode   *components.FZFMode
	fzfActive bool
	// Prompt collapsed state
	promptCollapsed bool
	showContext     bool // Whether to show context field
	// Run type toggle
	runType models.RunType
	// Config file loading
	configLoader             *config.ConfigLoader
	fileSelector             *components.FileSelector
	configFileSelector       *components.ConfigFileSelector
	lastLoadedFile           string
	configFileSelectorActive bool
	fileSelectorLoading      bool
	// Reset confirmation state
	resetConfirmMode bool
	// Vim keybinding state for 'gg' command
	lastGPressTime time.Time // Time when 'g' was last pressed
	waitingForG    bool      // Whether we're waiting for second 'g' in 'gg' command
	// File hash tracking for duplicate detection
	currentFileHash string
	isDuplicateRun  bool
	// Submission state tracking
	isSubmitting    bool
	submitStartTime time.Time
	// Duplicate run confirmation state
	isDuplicateConfirm bool
	duplicateRunID     string
	pendingTask        models.RunRequest
	// Embedded cache
	cache *cache.SimpleCache
}

func NewCreateRunView(client APIClient) *CreateRunView {
	debug.LogToFile("DEBUG: Creating CreateView with clean constructor\n")

	// Create own cache instance
	embeddedCache := cache.NewSimpleCache()
	_ = embeddedCache.LoadFromDisk()

	v := &CreateRunView{
		client:             client,
		keys:               components.DefaultKeyMap,
		help:               help.New(),
		cache:              embeddedCache,
		runType:            models.RunTypeRun,
		inputMode:          components.NormalMode,
		statusLine:         components.NewStatusLine(),
		configLoader:       config.NewConfigLoader(),
		fileSelector:       components.NewFileSelector(80, 10),       // Default dimensions
		configFileSelector: components.NewConfigFileSelector(80, 20), // Enhanced selector with preview
	}

	v.repoSelector = components.NewRepositorySelector()

	// Initialize fields BEFORE loading form data
	debug.LogToFile("DEBUG: Calling initializeInputFields\n")
	v.initializeInputFields()

	// Load saved form data first - this should populate the fields
	debug.LogToFile("DEBUG: Calling loadFormData\n")
	v.loadFormData()

	// Debug: Check what values are in fields after loading
	if len(v.fields) >= 5 {
		debug.LogToFilef("DEBUG: After loadFormData in NewCreateRunViewWithConfig - fields[0]=%s, fields[1]=%s, fields[2]=%s, fields[3]=%s, promptArea=%d chars\n",
			v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value(), len(v.promptArea.Value()))
	}

	// Only override repository with dashboard selection if appropriate
	if cfg.SelectedRepository != "" {
		// Dashboard passed a selected repository
		// Only use it if the current repository field is empty
		if len(v.fields) >= 1 && v.fields[0].Value() == "" {
			v.fields[0].SetValue(cfg.SelectedRepository)
			debug.LogToFilef("DEBUG: Set repository from dashboard: %s\n", cfg.SelectedRepository)
		} else {
			debug.LogToFilef("DEBUG: Keeping existing repository: %s (dashboard had: %s)\n", v.fields[0].Value(), cfg.SelectedRepository)
		}
		// Otherwise keep the loaded form data
	} else {
		// No dashboard selection - autofill if repository is still empty
		if len(v.fields) >= 1 && v.fields[0].Value() == "" {
			v.autofillRepository()
		}
	}

	return v
}

// initializeInputFields sets up all the input fields
func (v *CreateRunView) initializeInputFields() {
	// Repository field (first)
	repoInput := textinput.New()
	repoInput.Placeholder = "org/repo (required, leave empty to auto-detect)"
	repoInput.CharLimit = 100
	repoInput.Width = 50
	// Don't focus by default - we start in normal mode

	// Prompt area (second)
	promptArea := textarea.New()
	promptArea.Placeholder = "Describe what you want the AI to do..."
	promptArea.SetWidth(60)
	promptArea.SetHeight(5) // Default expanded height
	promptArea.CharLimit = 5000

	// Optional fields at the end
	sourceInput := textinput.New()
	sourceInput.Placeholder = "main (leave empty to auto-detect)"
	sourceInput.CharLimit = 50
	sourceInput.Width = 30

	targetInput := textinput.New()
	targetInput.Placeholder = "feature/branch-name (auto-generated if empty)"
	targetInput.CharLimit = 50
	targetInput.Width = 30

	titleInput := textinput.New()
	titleInput.Placeholder = "Brief title (optional)"
	titleInput.CharLimit = 100
	titleInput.Width = 50

	issueInput := textinput.New()
	issueInput.Placeholder = "#123 (optional)"
	issueInput.CharLimit = 20
	issueInput.Width = 20

	contextArea := textarea.New()
	contextArea.Placeholder = "Additional context (optional, press 'c' to show/hide)..."
	contextArea.SetWidth(60)
	contextArea.SetHeight(2) // Minimum 2 lines to prevent layout shifts
	contextArea.CharLimit = 2000

	filePathInput := textinput.New()
	filePathInput.Placeholder = "Path to task JSON file"
	filePathInput.CharLimit = 200
	filePathInput.Width = 50

	// DON'T call autoDetectGit here - it overwrites saved form data!
	// We'll only auto-detect when fields are actually empty after loading

	// Reorder: repository, then prompt area is handled separately, then other fields
	v.fields = []textinput.Model{
		repoInput,   // 0: Repository
		sourceInput, // 1: Source branch
		targetInput, // 2: Target branch
		titleInput,  // 3: Title (now optional)
		issueInput,  // 4: Issue
	}
	v.promptArea = promptArea
	v.contextArea = contextArea
	v.filePathInput = filePathInput
	v.focusIndex = 0
	v.showContext = false         // Hide context by default
	v.runType = models.RunTypeRun // Default to regular run
}

// loadFormData loads saved form data from cache
func (v *CreateRunView) loadFormData() {
	savedData := v.cache.GetFormData()
	if savedData != nil && len(v.fields) >= 5 {
		debug.LogToFilef("DEBUG: loadFormData START - Repository: %s, Prompt: %d chars, Source: %s, Target: %s, Title: %s\n",
			savedData.Repository, len(savedData.Prompt), savedData.Source, savedData.Target, savedData.Title)

		v.fields[0].SetValue(savedData.Repository)
		v.fields[1].SetValue(savedData.Source)
		v.fields[2].SetValue(savedData.Target)
		v.fields[3].SetValue(savedData.Title)
		v.fields[4].SetValue(savedData.Issue)
		v.promptArea.SetValue(savedData.Prompt)
		v.contextArea.SetValue(savedData.Context)
		if savedData.Context != "" {
			v.showContext = true
		}
		// Load saved run type if available
		if savedData.RunType != "" {
			v.runType = models.RunType(savedData.RunType)
		}

		// Debug: Verify fields were actually set
		debug.LogToFilef("DEBUG: loadFormData AFTER SET - fields[0]=%s, fields[1]=%s, fields[2]=%s, fields[3]=%s, promptArea=%d chars\n",
			v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value(), len(v.promptArea.Value()))
	} else {
		if savedData == nil {
			debug.LogToFile("DEBUG: loadFormData - savedData is nil!\n")
		} else {
			debug.LogToFilef("DEBUG: loadFormData - fields not initialized, len(v.fields)=%d\n", len(v.fields))
		}
	}
}

// autofillRepository sets the repository field with the most appropriate default
func (v *CreateRunView) autofillRepository() {
	// Only autofill if the repository field is empty (now at index 0)
	if len(v.fields) >= 2 {
		// Auto-detect from git if fields are empty
		if repo, branch, err := pkgutils.GetGitInfo(); err == nil {
			if v.fields[0].Value() == "" && repo != "" {
				v.fields[0].SetValue(repo)
				debug.LogToFilef("DEBUG: Auto-filled repository from git: %s\n", repo)
			}
			if v.fields[1].Value() == "" && branch != "" {
				v.fields[1].SetValue(branch)
				debug.LogToFilef("DEBUG: Auto-filled source branch from git: %s\n", branch)
			}
		} else if v.fields[0].Value() == "" {
			// Fallback to repository selector if git detection fails
			defaultRepo := v.repoSelector.GetDefaultRepository()
			if defaultRepo != "" {
				v.fields[0].SetValue(defaultRepo)
				debug.LogToFilef("DEBUG: Auto-filled repository from selector: %s\n", defaultRepo)
			}
		}
	}
}

// NewCreateRunViewWithCache maintains backward compatibility
func NewCreateRunViewWithCache(
	client APIClient,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt time.Time,
	parentDetailsCache map[string]*models.RunResponse,
	embeddedCache *cache.SimpleCache,
) *CreateRunView {
	debug.LogToFilef("DEBUG: Creating CreateView - parentRuns=%d, parentCached=%v, detailsCache=%d\n",
		len(parentRuns), parentCached, len(parentDetailsCache))

	config := CreateRunViewConfig{
		Client:             client,
		ParentRuns:         parentRuns,
		ParentCached:       parentCached,
		ParentCachedAt:     parentCachedAt,
		ParentDetailsCache: parentDetailsCache,
		Cache:              embeddedCache,
	}

	return NewCreateRunViewWithConfig(config)
}

func (v *CreateRunView) Init() tea.Cmd {
	debug.LogToFilef("DEBUG: CreateRunView.Init() called - width=%d, height=%d\n", v.width, v.height)
	var cmds []tea.Cmd

	// Send a window size message with stored dimensions if we have them
	if v.width > 0 && v.height > 0 {
		debug.LogToFilef("DEBUG: CreateRunView.Init() - sending stored window size: %dx%d\n", v.width, v.height)
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: v.width, Height: v.height}
		})
	} else {
		debug.LogToFile("DEBUG: CreateRunView.Init() - no stored dimensions, waiting for WindowSizeMsg\n")
	}

	// Load file hash cache in the background
	cmds = append(cmds, v.loadFileHashCache())

	cmds = append(cmds, textinput.Blink)
	debug.LogToFilef("DEBUG: CreateRunView.Init() - returning %d commands\n", len(cmds))
	return tea.Batch(cmds...)
}

// handleWindowSizeMsg handles window resize events
func (v *CreateRunView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height
	v.help.Width = msg.Width

	// Update config file selector dimensions if it exists
	if v.configFileSelector != nil {
		v.configFileSelector.SetDimensions(msg.Width, msg.Height)
	}

	// Make text areas use most of the available width with some padding
	textAreaWidth := msg.Width - 20
	if textAreaWidth < 40 {
		textAreaWidth = 40 // Minimum usable width
	}

	// Update widths for all input fields to be responsive
	for i := range v.fields {
		v.fields[i].Width = min(textAreaWidth, 80) // Cap at 80 for readability
	}

	v.promptArea.SetWidth(min(textAreaWidth, 100))
	v.contextArea.SetWidth(min(textAreaWidth, 100))

	// Set appropriate heights - prompt can be 2 lines when collapsed, 5 when expanded
	if !v.promptCollapsed {
		v.promptArea.SetHeight(5)
	} else {
		v.promptArea.SetHeight(2) // Minimum 2 lines even when collapsed
	}
	v.contextArea.SetHeight(2) // Minimum 2 lines to prevent layout shifts
}

// handleInsertMode handles keyboard input in insert mode
func (v *CreateRunView) handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// In insert mode, handle ESC to enter normal mode
	if msg.String() == "esc" {
		v.inputMode = components.NormalMode
		v.exitRequested = false
		v.blurAllFields()
		return v, nil
	}

	// In insert mode, handle text input and field navigation
	switch {
	case msg.String() == "ctrl+f":
		// Activate FZF mode for repository field if focused on it (now index 1)
		if !v.useFileInput && v.focusIndex == 1 && !v.fzfActive {
			v.activateFZFMode()
			return v, nil
		} else {
			// Original file input toggle behavior
			v.useFileInput = !v.useFileInput
			if v.useFileInput {
				v.filePathInput.Focus()
			} else {
				v.fields[0].Focus()
				v.focusIndex = 1 // Repository is now at index 1
			}
		}
	case msg.String() == "ctrl+r":
		// Trigger repository selector when repository field is focused
		if !v.useFileInput && v.focusIndex == 1 {
			return v, v.selectRepository()
		}
	case msg.String() == "ctrl+s" || msg.String() == "ctrl+enter":
		if !v.submitting && !v.isSubmitting {
			debug.LogToFile("DEBUG: Ctrl+S pressed in INSERT MODE - submitting run\n")
			return v, v.submitRun()
		}
	case msg.String() == "enter":
		// For prompt area (index 3) or context area, allow Enter for newlines
		if v.focusIndex == 3 || (v.showContext && v.focusIndex == len(v.fields)+3) {
			// Handle text input for prompt/context areas - Enter creates newlines
			cmds = append(cmds, v.updateFields(msg)...)
		} else {
			// For other fields, Exit insert mode when Enter is pressed
			v.inputMode = components.NormalMode
			v.exitRequested = false
			v.blurAllFields()
			return v, nil
		}
	case key.Matches(msg, v.keys.Tab):
		v.nextField()
	case key.Matches(msg, v.keys.ShiftTab):
		v.prevField()
	default:
		// Handle text input
		if v.useFileInput {
			var cmd tea.Cmd
			v.filePathInput, cmd = v.filePathInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			cmds = append(cmds, v.updateFields(msg)...)
		}
	}

	return v, tea.Batch(cmds...)
}

// handleNormalMode handles keyboard input in normal mode
func (v *CreateRunView) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error state first
	if v.error != nil && !v.submitting {
		return v.handleErrorMode(msg)
	}

	// Handle reset confirmation mode
	if v.resetConfirmMode {
		switch msg.String() {
		case "y":
			// Confirm reset - clear all fields and cache
			v.clearAllFields()
			v.resetConfirmMode = false
			v.lastLoadedFile = ""
			v.runType = models.RunTypeRun
			v.showContext = false
			v.promptCollapsed = false
			debug.LogToFile("DEBUG: User confirmed reset - all fields cleared\n")
			return v, nil
		case "n", "esc":
			// Cancel reset
			v.resetConfirmMode = false
			debug.LogToFile("DEBUG: User cancelled reset\n")
			return v, nil
		default:
			// Ignore other keys in reset confirm mode
			return v, nil
		}
	}

	switch {
	case msg.String() == "Q":
		// Capital Q to force quit from anywhere
		return v, tea.Quit
	case key.Matches(msg, v.keys.Quit), key.Matches(msg, v.keys.Back), msg.String() == "esc":
		// q, b, or ESC all go back to dashboard
		if !v.submitting {
			v.saveFormData()
			debug.LogToFile("DEBUG: CreateView q/b/ESC - returning to dashboard\n")
			// Return to dashboard view
			dashboard := NewDashboardView(v.client)
			dashboard.width = v.width
			dashboard.height = v.height
			return dashboard, dashboard.Init()
		}
	case key.Matches(msg, v.keys.Help):
		// Return to dashboard and show docs
		dashboard := NewDashboardView(v.client)
		dashboard.width = v.width
		dashboard.height = v.height
		dashboard.showDocs = true
		dashboard.docsCurrentPage = 4 // Show Create Run Form page
		return dashboard, dashboard.Init()
	case msg.String() == "i":
		// 'i' enters insert mode
		v.inputMode = components.InsertMode
		v.exitRequested = false
		v.updateFocus()
	case msg.String() == "enter":
		if v.backButtonFocused {
			// Enter on back button returns to dashboard
			v.saveFormData()
			debug.LogToFile("DEBUG: Back button pressed - returning to dashboard\n")
			dashboard := NewDashboardView(v.client)
			dashboard.width = v.width
			dashboard.height = v.height
			return dashboard, dashboard.Init()
		} else if v.submitButtonFocused {
			// Enter on submit button submits the run
			if !v.submitting && !v.isSubmitting {
				debug.LogToFile("DEBUG: Submit button pressed - submitting run\n")
				return v, v.submitRun()
			}
		} else if v.focusIndex == 0 {
			// Enter on load config field (index 0) activates file selector
			// Don't process if already loading or active
			if !v.fileSelectorLoading && !v.configFileSelectorActive {
				v.fileSelectorLoading = true
				return v, v.activateConfigFileSelector()
			}
		} else if v.focusIndex == 1 {
			// Enter on run type field (index 1) toggles it
			if v.runType == models.RunTypeRun {
				v.runType = models.RunTypePlan
			} else {
				v.runType = models.RunTypeRun
			}
		} else if v.focusIndex == 2 {
			// Repository field - enter insert mode
			v.inputMode = components.InsertMode
			v.exitRequested = false
			v.updateFocus()
		} else {
			// Enter on other fields enters insert mode
			v.inputMode = components.InsertMode
			v.exitRequested = false
			v.updateFocus()
		}
	case key.Matches(msg, v.keys.Up) || msg.String() == "k":
		v.prevField()
	case key.Matches(msg, v.keys.Down) || msg.String() == "j":
		v.nextField()
	case msg.String() == "ctrl+s":
		if !v.submitting && !v.isSubmitting {
			debug.LogToFile("DEBUG: Ctrl+S pressed in NORMAL MODE - submitting run\n")
			return v, v.submitRun()
		}
	case msg.String() == "ctrl+l":
		v.clearAllFields()
	case msg.String() == "ctrl+x":
		v.clearCurrentField()
	case msg.String() == "f":
		// In normal mode, 'f' activates FZF
		if v.focusIndex == 0 {
			// Load config field - activate file selector
			// Don't process if already loading or active
			if !v.fileSelectorLoading && !v.configFileSelectorActive {
				v.fileSelectorLoading = true
				return v, v.activateConfigFileSelector()
			}
		} else if v.focusIndex == 2 && !v.fzfActive {
			// Repository field - activate FZF for repository selection
			v.activateFZFMode()
			return v, nil
		}
	case msg.String() == "c":
		// Toggle context field visibility
		v.showContext = !v.showContext
	case msg.String() == "t":
		// Toggle between run types
		if v.runType == models.RunTypeRun {
			v.runType = models.RunTypePlan
		} else {
			v.runType = models.RunTypeRun
		}
	case msg.String() == "d":
		// Delete current field value for string input fields only (not load config or run type)
		if v.focusIndex != 0 && v.focusIndex != 1 { // Skip load config field (index 0) and run type field (index 1)
			v.clearCurrentField()
		}
	case msg.String() == "r":
		// Enter reset confirmation mode
		v.resetConfirmMode = true
		debug.LogToFile("DEBUG: Entering reset confirmation mode\n")
		return v, nil
	case msg.String() == "G":
		// Vim: Go to bottom (last field or submit button)
		v.waitingForG = false // Cancel any pending 'gg' command
		// Calculate total fields: 1 (load config) + 1 (run type) + 1 (repo) + 1 (prompt) + 4 (other fields) + context (if shown)
		totalFields := len(v.fields) + 3 // +1 for load config, +1 for run type, +1 for prompt
		if v.showContext {
			totalFields++ // +1 for context
		}
		// Go to submit button (which is after all fields)
		v.focusIndex = totalFields
		v.submitButtonFocused = true
		v.backButtonFocused = false
		v.updateFocus()
		return v, nil
	case msg.String() == "g":
		if v.waitingForG {
			// This is the second 'g' in 'gg' - go to top (first field)
			v.waitingForG = false
			v.focusIndex = 0 // Go to load config field
			v.submitButtonFocused = false
			v.backButtonFocused = false
			v.updateFocus()
		} else {
			// First 'g' pressed - wait for second 'g'
			v.waitingForG = true
			v.lastGPressTime = time.Now()
			// Start a timer to cancel the 'gg' command after 1 second
			return v, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
				return gKeyTimeoutMsg{}
			})
		}
		return v, nil
	default:
		// Cancel any pending 'gg' command if another key is pressed
		if v.waitingForG {
			v.waitingForG = false
		}
		// Block vim navigation keys from doing anything else
	}

	return v, nil
}

// handleRunCreated handles the runCreatedMsg message
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: runCreatedMsg received - err=%v, runID='%s'\n",
		msg.err, func() string {
			if msg.err == nil {
				return msg.run.GetIDString()
			}
			return "N/A"
		}())

	v.submitting = false
	v.isSubmitting = false // Reset our new submitting state
	if msg.err != nil {
		// Check if this is a duplicate run error
		errorMsg := msg.err.Error()
		if strings.Contains(errorMsg, "Duplicate run detected") && strings.Contains(errorMsg, "Use --force to override") {
			// Extract the run ID from the error message
			// Pattern: "Duplicate run detected: A run with this file hash already exists (ID: 955). Use --force to override."
			re := regexp.MustCompile(`\(ID: (\d+)\)`)
			matches := re.FindStringSubmatch(errorMsg)
			if len(matches) > 1 {
				v.duplicateRunID = matches[1]
				v.isDuplicateConfirm = true
				debug.LogToFilef("DEBUG: Duplicate run detected - entering confirmation mode for run ID %s\n", v.duplicateRunID)
				return v, nil
			}
		}

		// Regular error handling for non-duplicate errors
		v.error = msg.err
		v.initErrorFocus()
		return v, nil
	}

	// Check if the run has a valid ID
	runID := msg.run.GetIDString()
	if runID == "" {
		v.error = fmt.Errorf("run created but received invalid ID from server")
		v.initErrorFocus()
		debug.LogToFile("DEBUG: Run created successfully but runID is empty, not navigating to details\n")
		return v, nil
	}

	// Clear form data on successful submission
	v.cache.SetFormData(nil)
	v.success = true
	v.createdRun = &msg.run

	// Add the file hash to cache if we have one
	if v.currentFileHash != "" {
		v.cache.SetFileHash(v.lastLoadedFile, v.currentFileHash)
		debug.LogToFilef("DEBUG: Added file hash %s to cache after successful submission\n", v.currentFileHash)
	}

	debug.LogToFilef("DEBUG: Run created successfully with ID='%s', navigating to details\n", runID)
	// Pass the cache data and current dimensions to the details view
	return NewRunDetailsViewWithCacheAndDimensions(v.client, msg.run, v.parentRuns, v.parentCached, v.parentCachedAt, v.parentDetailsCache, v.width, v.height), nil
}

// handleRepositorySelected handles the repositorySelectedMsg message
func (v *CreateRunView) handleRepositorySelected(msg repositorySelectedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		debug.LogToFilef("DEBUG: Repository selection error: %v\n", msg.err)
		v.error = msg.err
		v.initErrorFocus()
		return v, nil
	}

	// Set the selected repository in the repository field (now at index 0)
	if len(v.fields) >= 1 && msg.repository != "" {
		v.fields[0].SetValue(msg.repository)
		debug.LogToFilef("DEBUG: Repository field updated to: %s\n", msg.repository)

		// Add to manual repository list for future use
		v.repoSelector.AddManualRepository(msg.repository)
	}

	return v, nil
}

// handleErrorMode handles keyboard input when there's an error
func (v *CreateRunView) handleErrorMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "Q":
		// Capital Q to force quit from anywhere
		return v, tea.Quit
	case "y":
		// Copy error message to clipboard only if error row is selected
		if v.error != nil && v.errorRowFocused {
			return v, v.copyToClipboard(v.error.Error())
		}
	case "j", "down":
		// Navigate down from error row to back button
		if v.errorRowFocused {
			v.errorRowFocused = false
			v.errorButtonFocused = true
		}
	case "k", "up":
		// Navigate up from back button to error row
		if v.errorButtonFocused {
			v.errorButtonFocused = false
			v.errorRowFocused = true
		}
	case "enter":
		if v.errorButtonFocused {
			// Enter on back button goes back to form (clear error)
			v.error = nil
			v.restorePreviousFocus()
			return v, nil
		} else if v.errorRowFocused {
			// Enter on error row also goes back to form
			v.error = nil
			v.restorePreviousFocus()
			return v, nil
		}
	case "escape", "q":
		// ESC, q, b - go back to form (clear error)
		v.error = nil
		v.restorePreviousFocus()
		return v, nil
	case "r":
		// 'r' to retry (clear error and go back to form)
		v.error = nil
		v.restorePreviousFocus()
		return v, nil
	}
	return v, nil
}

func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: CreateRunView.Update() received message: %T\n", msg)
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		debug.LogToFilef("DEBUG: CreateRunView.Update() - handling WindowSizeMsg: %dx%d\n", msg.Width, msg.Height)
		v.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		// If submitting, only allow ESC and 'q' to quit
		if v.isSubmitting {
			switch msg.String() {
			case "esc":
				// Allow ESC to handle long running submit API
				debug.LogToFilef("DEBUG: ESC pressed during submission - cancelling\n")
				v.isSubmitting = false
				return v, nil
			case "q", "Q":
				// Allow quit during submission
				return v, tea.Quit
			default:
				// Ignore all other input during submission
				return v, nil
			}
		}

		// If in duplicate confirmation mode, only handle y/n
		if v.isDuplicateConfirm {
			switch msg.String() {
			case "y", "Y":
				// User confirmed - retry with force flag
				debug.LogToFilef("DEBUG: User confirmed duplicate override - retrying with force\n")
				v.isDuplicateConfirm = false
				return v, v.submitWithForce()
			case "n", "N", "esc":
				// User cancelled - exit confirmation mode
				debug.LogToFilef("DEBUG: User cancelled duplicate override\n")
				v.isDuplicateConfirm = false
				return v, nil
			case "q", "Q":
				// Allow quit during confirmation
				return v, tea.Quit
			default:
				// Ignore all other input during duplicate confirmation
				return v, nil
			}
		}

		// If enhanced config file selector is active, handle input there first
		if v.configFileSelector != nil && v.configFileSelector.IsActive() {
			newConfigFileSelector, cmd := v.configFileSelector.Update(msg)
			v.configFileSelector = newConfigFileSelector
			return v, cmd
		}

		// If old file selector is active, handle input there first (fallback)
		if v.fileSelector != nil && v.fileSelector.IsActive() {
			newFileSelector, cmd := v.fileSelector.Update(msg)
			v.fileSelector = newFileSelector
			return v, cmd
		}

		// If FZF mode is active, handle input there first
		if v.fzfMode != nil && v.fzfMode.IsActive() {
			newFzf, cmd := v.fzfMode.Update(msg)
			v.fzfMode = newFzf
			return v, cmd
		}

		switch v.inputMode {
		case components.InsertMode:
			return v.handleInsertMode(msg)
		case components.NormalMode:
			return v.handleNormalMode(msg)
		}

	case runCreatedMsg:
		return v.handleRunCreated(msg)

	case repositorySelectedMsg:
		return v.handleRepositorySelected(msg)

	case components.FZFSelectedMsg:
		// Handle FZF selection result
		if v.configFileSelectorActive {
			// Handle file selector result
			v.configFileSelectorActive = false
			if !msg.Result.Canceled && msg.Result.Selected != "" {
				// Clean the selected file path (remove any icon prefixes)
				filePath := msg.Result.Selected
				if idx := strings.Index(filePath, " "); idx > 0 {
					filePath = filePath[idx+1:] // Skip icon
				}

				// Clean up any remaining emoji or prefixes
				filePath = strings.TrimSpace(strings.TrimLeft(filePath, "📁📝🔍"))

				debug.LogToFilef("DEBUG: File selected from FZF: %s\n", filePath)
				return v, v.loadConfigFromFile(filePath)
			}
		} else if !msg.Result.Canceled && v.focusIndex == 2 {
			// Handle repository field selection (focusIndex 2 is repository now)
			if msg.Result.Selected != "" {
				// Extract just the repository name (remove any icons)
				repoName := msg.Result.Selected
				if idx := strings.Index(repoName, " "); idx > 0 {
					repoName = repoName[idx+1:] // Skip icon
				}
				v.fields[0].SetValue(repoName)
				v.repoSelector.AddManualRepository(repoName)
			}
		}
		// Deactivate FZF mode
		v.fzfActive = false
		v.fzfMode = nil
		return v, nil

	case yankBlinkMsg:
		// Single blink: toggle off after being on
		if v.yankBlink {
			v.yankBlink = false // Turn off after being on - completes the single blink
		}
		return v, nil

	case gKeyTimeoutMsg:
		// Cancel waiting for second 'g' after timeout
		v.waitingForG = false
		return v, nil

	case clipboardResultMsg:
		// Handle clipboard result
		if msg.success {
			// Show what's actually on the clipboard, truncated for display if needed
			displayText := msg.text
			maxLen := 30
			if len(displayText) > maxLen {
				displayText = displayText[:maxLen-3] + "..."
			}
			v.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("📋 Copied \"%s\"", displayText), components.MessageSuccess, 100*time.Millisecond)
		} else {
			v.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 100*time.Millisecond)
		}
		v.yankBlink = true
		v.yankBlinkTime = time.Now()
		// Start blink animation and clear timer
		return v, tea.Batch(
			v.startYankBlinkAnimation(),
			v.startClearStatusTimer(),
		)

	case configLoadedMsg:
		// Handle successful config loading
		v.populateFormFromConfig(msg.config, msg.filePath)

		// Store the file hash and check if it's a duplicate
		v.currentFileHash = msg.fileHash
		if msg.fileHash != "" {
			// Check if this file hash already exists
			existingHash := v.cache.GetFileHash(v.lastLoadedFile)
			v.isDuplicateRun = (existingHash == msg.fileHash)
			debug.LogToFilef("DEBUG: File hash %s - isDuplicate: %v\n", msg.fileHash, v.isDuplicateRun)
		} else {
			v.isDuplicateRun = false
		}

		// Debug: Log field values AFTER populating from config
		debug.LogToFilef("DEBUG: After populateFormFromConfig - fields[0]=%s, fields[1]=%s, fields[2]=%s, fields[3]=%s, promptArea=%d chars\n",
			v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value(), len(v.promptArea.Value()))

		// IMPORTANT: Save the loaded config data to cache immediately
		// so it persists when navigating away
		v.saveFormData()

		// Debug: Verify what was saved
		savedData := v.cache.GetFormData()
		if savedData != nil {
			debug.LogToFilef("DEBUG: Verified saved data - Repository=%s, Prompt=%d chars, Source=%s, Target=%s, Title=%s\n",
				savedData.Repository, len(savedData.Prompt), savedData.Source, savedData.Target, savedData.Title)
		} else {
			debug.LogToFile("DEBUG: WARNING - savedData is nil after saveFormData!\n")
		}

		v.statusLine.SetTemporaryMessageWithType(
			fmt.Sprintf("✅ Config loaded from %s", filepath.Base(msg.filePath)),
			components.MessageSuccess,
			500*time.Millisecond, // Show for only 500ms
		)
		return v, nil

	case configLoadErrorMsg:
		// Handle config loading error
		v.fileSelectorLoading = false
		v.error = msg.err
		v.initErrorFocus()
		debug.LogToFilef("DEBUG: Config loading failed: %v\n", msg.err)
		return v, nil

	case fileSelectorActivatedMsg:
		// File selector is now active, clear loading state
		v.fileSelectorLoading = false
		v.configFileSelectorActive = true
		// Start the cursor blink animation
		return v, v.tickCmd()

	case configSelectorTickMsg:
		// Forward tick to config file selector if it's active
		if v.configFileSelector != nil && v.configFileSelector.IsActive() {
			// Convert to the config file selector's tick message type
			newSelector, cmd := v.configFileSelector.Update(components.TickMsg(time.Time(msg)))
			v.configFileSelector = newSelector
			cmds = append(cmds, cmd)
			// Continue ticking
			cmds = append(cmds, v.tickCmd())
		}
		return v, tea.Batch(cmds...)
	}

	return v, tea.Batch(cmds...)
}

func (v *CreateRunView) updateFields(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// Load config is at index 0 (not editable via text input)
	// Run type is at index 1 (not editable via text input)
	// Repository is at index 2
	// Prompt is at index 3
	// Other fields start at index 4
	if v.focusIndex == 0 {
		// Load config field - no text input
	} else if v.focusIndex == 1 {
		// Run type field - no text input
	} else if v.focusIndex == 2 {
		// Repository field
		var cmd tea.Cmd
		v.fields[0], cmd = v.fields[0].Update(msg)
		cmds = append(cmds, cmd)
	} else if v.focusIndex == 3 {
		// Prompt area
		var cmd tea.Cmd
		v.promptArea, cmd = v.promptArea.Update(msg)
		cmds = append(cmds, cmd)
	} else if v.focusIndex >= 4 && v.focusIndex < len(v.fields)+3 {
		// Other fields (source, target, title, issue)
		fieldIdx := v.focusIndex - 3 // Adjust for load config at 0, run type at 1, and prompt at 3
		if fieldIdx < len(v.fields) {
			var cmd tea.Cmd
			v.fields[fieldIdx], cmd = v.fields[fieldIdx].Update(msg)
			cmds = append(cmds, cmd)
		}
	} else if v.showContext && v.focusIndex == len(v.fields)+3 {
		// Context area (only if visible)
		var cmd tea.Cmd
		v.contextArea, cmd = v.contextArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (v *CreateRunView) nextField() {
	// Handle cycling from back button to first field (load config)
	if v.backButtonFocused {
		v.backButtonFocused = false
		v.submitButtonFocused = false
		v.focusIndex = 0 // Go to load config field
		v.updateFocus()
		return
	}

	v.backButtonFocused = false
	v.submitButtonFocused = false
	v.focusIndex++

	// Calculate total fields: 1 (load config) + 1 (run type) + 1 (repo) + 1 (prompt) + 4 (other fields) + context (if shown)
	totalFields := len(v.fields) + 3 // +1 for load config, +1 for run type, +1 for prompt
	if v.showContext {
		totalFields++ // +1 for context
	}

	if v.focusIndex == totalFields {
		// After last field, go to submit button
		v.submitButtonFocused = true
	} else if v.focusIndex > totalFields {
		// After submit button, go to back button
		v.submitButtonFocused = false
		v.backButtonFocused = true
		v.focusIndex = 0
	}
	v.updateFocus()
}

func (v *CreateRunView) prevField() {
	if v.backButtonFocused {
		// From back button, go to submit button
		v.backButtonFocused = false
		v.submitButtonFocused = true
		totalFields := len(v.fields) + 3 // +1 for load config, +1 for run type, +1 for prompt
		if v.showContext {
			totalFields++
		}
		v.focusIndex = totalFields
	} else if v.submitButtonFocused {
		// From submit button, go to last field
		v.submitButtonFocused = false
		if v.showContext {
			v.focusIndex = len(v.fields) + 3 // context area (+3 for load config, run type and prompt)
		} else {
			v.focusIndex = len(v.fields) + 2 // last regular field (+2 for load config and run type, prompt counted in fields)
		}
	} else {
		v.focusIndex--
		if v.focusIndex < 0 {
			v.backButtonFocused = true
			v.focusIndex = 0
		}
	}
	v.updateFocus()
}

func (v *CreateRunView) updateFocus() {
	// Only focus fields when in insert mode
	if v.inputMode == components.InsertMode {
		// Blur all fields first
		for i := range v.fields {
			v.fields[i].Blur()
		}
		v.promptArea.Blur()
		v.contextArea.Blur()

		// Now focus the current field
		if v.focusIndex == 0 {
			// Load config field - not editable in insert mode
		} else if v.focusIndex == 1 {
			// Run type field - not editable in insert mode
		} else if v.focusIndex == 2 {
			// Repository field
			v.fields[0].Focus()
		} else if v.focusIndex == 3 {
			// Prompt area
			v.promptArea.Focus()
			// Expand if collapsed when focusing
			if v.promptCollapsed {
				v.promptCollapsed = false
				v.promptArea.SetHeight(5)
			}
		} else if v.focusIndex >= 4 && v.focusIndex < len(v.fields)+3 {
			// Other fields - collapse prompt when moving away if it has content
			if v.promptArea.Value() != "" && !v.promptCollapsed {
				v.promptCollapsed = true
				v.promptArea.SetHeight(2) // Minimum 2 lines even when collapsed
			}
			fieldIdx := v.focusIndex - 3 // Adjust for load config at 0, run type at 1, and prompt at 3
			if fieldIdx < len(v.fields) {
				v.fields[fieldIdx].Focus()
			}
		} else if v.showContext && v.focusIndex == len(v.fields)+3 {
			// Context area - also collapse prompt if needed
			if v.promptArea.Value() != "" && !v.promptCollapsed {
				v.promptCollapsed = true
				v.promptArea.SetHeight(2) // Minimum 2 lines even when collapsed
			}
			v.contextArea.Focus()
		}
	} else {
		// In normal mode, blur all fields
		v.blurAllFields()
	}
}

func (v *CreateRunView) blurAllFields() {
	for i := range v.fields {
		v.fields[i].Blur()
	}
	v.promptArea.Blur()
	v.contextArea.Blur()
	v.filePathInput.Blur()
}

func (v *CreateRunView) saveFormData() {
	formData := &cache.FormData{
		Repository: v.fields[0].Value(),
		Source:     v.fields[1].Value(),
		Target:     v.fields[2].Value(),
		Title:      v.fields[3].Value(),
		Issue:      v.fields[4].Value(),
		Prompt:     v.promptArea.Value(),
		Context:    v.contextArea.Value(),
		RunType:    string(v.runType),
	}

	// Debug logging to verify what we're saving
	debug.LogToFilef("DEBUG: Saving form data - Repository: %s, Prompt: %d chars, Source: %s, Target: %s, Title: %s\n",
		formData.Repository, len(formData.Prompt), formData.Source, formData.Target, formData.Title)

	v.cache.SetFormData(formData)
}

func (v *CreateRunView) clearAllFields() {
	for i := range v.fields {
		v.fields[i].SetValue("")
	}
	v.promptArea.SetValue("")
	v.contextArea.SetValue("")
	v.filePathInput.SetValue("")
	v.cache.SetFormData(nil)
}

func (v *CreateRunView) clearCurrentField() {
	if v.backButtonFocused {
		return // Can't clear back button
	}

	if v.useFileInput {
		v.filePathInput.SetValue("")
	} else if v.focusIndex == 0 {
		// Load config field - can clear loaded file info
		v.lastLoadedFile = ""
	} else if v.focusIndex == 1 {
		// Run type field - can't clear, just toggle
	} else if v.focusIndex == 2 {
		// Repository field
		v.fields[0].SetValue("")
	} else if v.focusIndex == 3 {
		// Prompt area
		v.promptArea.SetValue("")
		v.promptCollapsed = false
		v.promptArea.SetHeight(5) // Expanded height when cleared
	} else if v.focusIndex >= 4 && v.focusIndex < len(v.fields)+3 {
		// Other fields
		fieldIdx := v.focusIndex - 3
		if fieldIdx < len(v.fields) {
			v.fields[fieldIdx].SetValue("")
		}
	} else if v.showContext && v.focusIndex == len(v.fields)+3 {
		// Context area
		v.contextArea.SetValue("")
	}
}

func (v *CreateRunView) View() string {
	if v.width <= 0 || v.height <= 0 {
		// Return a styled loading message instead of plain text
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render("⟳ Initializing...")
	}

	// Calculate available height for content
	// We have v.height total, minus 1 for statusbar, minus 2 for margin
	availableHeight := v.height - 3
	if availableHeight < 5 {
		availableHeight = 5
	}

	var content string

	if v.error != nil && !v.submitting {
		// Error mode - render error in bordered box similar to form
		content = v.renderErrorLayout(availableHeight)
	} else if v.submitting {
		loadingContent := "⟳ Creating run...\n\nPlease wait..."
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true).
			Width(v.width).
			Align(lipgloss.Center).
			MarginTop((availableHeight - 2) / 2)
		content = loadingStyle.Render(loadingContent)
	} else if v.useFileInput {
		// File input mode - centered box
		fileContent := v.renderFileInputMode()
		boxWidth := 60
		boxHeight := 10

		fileBoxStyle := lipgloss.NewStyle().
			Width(boxWidth).
			Height(boxHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

		fileBox := fileBoxStyle.Render(fileContent)

		// Center the box
		content = lipgloss.Place(
			v.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			fileBox,
		)
	} else {
		// Form input mode - single panel layout
		content = v.renderSinglePanelLayout(availableHeight)
	}

	// Create status bar
	statusBar := v.renderStatusBar()

	// Join all components with status bar
	finalView := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		statusBar,
	)

	// If enhanced config file selector is active, show it as an overlay
	if v.configFileSelector != nil && v.configFileSelector.IsActive() {
		return v.configFileSelector.View()
	}

	// If file selector is active, show modal instead of overlay
	if v.fileSelector != nil && v.fileSelector.IsActive() {
		return v.renderFileSelectionModal(statusBar)
	}

	// If FZF mode is active, overlay the dropdown
	if v.fzfMode != nil && v.fzfMode.IsActive() {
		return v.renderWithFZFOverlay(finalView)
	}

	return finalView
}

// renderFileSelectionModal renders a modal for file selection, replacing the entire view
func (v *CreateRunView) renderFileSelectionModal(statusBar string) string {
	if v.fileSelector == nil || !v.fileSelector.IsActive() {
		return ""
	}

	// Create modal content
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		PaddingLeft(1).
		Render("📄 Select Config File (JSON/Markdown)")

	// Get file selector view
	fileSelectorView := v.fileSelector.View()

	// Create help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingLeft(1).
		Render("Type to search • ↑↓ to navigate • Enter to select • ESC to cancel")

	// Calculate available height
	availableHeight := v.height - 4 // title + help + statusbar + padding
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Create content area
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		modalTitle,
		"",
		fileSelectorView,
		"",
		helpText,
	)

	// Center the content vertically
	modalContent := lipgloss.Place(
		v.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Join with status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		modalContent,
		statusBar,
	)
}

// renderWithFZFOverlay renders the view with FZF dropdown overlay
func (v *CreateRunView) renderWithFZFOverlay(baseView string) string {
	if v.fzfMode == nil || !v.fzfMode.IsActive() {
		return baseView
	}

	// Calculate position for FZF dropdown (repository field)
	// Title + border + load config + run type + repository field = about 5 lines
	yOffset := 5
	xOffset := 19 // After "Repository:    " label and indicator

	return v.renderOverlayDropdown(baseView, v.fzfMode.View(), yOffset, xOffset)
}

// renderOverlayDropdown renders a dropdown overlay on the base view
func (v *CreateRunView) renderOverlayDropdown(baseView, overlayView string, yOffset, xOffset int) string {
	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")

	// Create overlay dropdown view lines
	overlayLines := strings.Split(overlayView, "\n")

	// Create a new view with the dropdown overlaid
	result := make([]string, max(len(baseLines), yOffset+len(overlayLines)))
	copy(result, baseLines)

	// Ensure we have enough lines
	for i := len(baseLines); i < len(result); i++ {
		result[i] = ""
	}

	// Insert dropdown at the calculated position
	for i, overlayLine := range overlayLines {
		lineIdx := yOffset + i
		if lineIdx >= 0 && lineIdx < len(result) {
			// Create the overlay line
			if xOffset < len(result[lineIdx]) {
				// Preserve part of the base line before the dropdown
				basePart := ""
				if xOffset > 0 {
					basePart = result[lineIdx][:min(xOffset, len(result[lineIdx]))]
				}
				// Add the overlay line
				result[lineIdx] = basePart + overlayLine
			} else {
				// Line is shorter than offset, pad and add overlay
				padding := strings.Repeat(" ", max(0, xOffset-len(result[lineIdx])))
				result[lineIdx] = result[lineIdx] + padding + overlayLine
			}
		}
	}

	return strings.Join(result, "\n")
}

// renderSinglePanelLayout renders the form in a single panel with compact fields
func (v *CreateRunView) renderSinglePanelLayout(availableHeight int) string {
	// Account for borders (2 chars for top/bottom) in the content dimensions
	// Width calculation: terminal width minus some padding for cleaner look
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}

	// Height should fill available space
	panelHeight := availableHeight
	if panelHeight < 3 {
		panelHeight = 3
	}

	// Content dimensions (accounting for border and padding)
	// Border takes 2 from width and height, padding takes another 2 from each
	contentWidth := panelWidth - 4
	contentHeight := panelHeight - 4

	if contentWidth < 40 {
		contentWidth = 40
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Create the single panel content
	panelContent := v.renderCompactForm(contentWidth, contentHeight)

	// Style for single panel - Width includes the border
	// Use Height to maintain consistent window size regardless of content
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight). // Use Height to maintain consistent size
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Wrap with margin top to prevent border cutoff
	panel := panelStyle.Render(panelContent)
	return lipgloss.NewStyle().MarginTop(2).Render(panel)
}

// renderCompactForm renders all fields in a compact single-column layout
func (v *CreateRunView) renderCompactForm(width, height int) string {
	var b strings.Builder

	// Add title header inside the form
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	b.WriteString(titleStyle.Render("Create New Run"))
	b.WriteString("\n")

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(24)

	// Load from Config field (new, at index 0)
	b.WriteString(labelStyle.Render("📄 Load Config:"))
	if v.focusIndex == 0 && !v.backButtonFocused && !v.submitButtonFocused {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}

	loadConfigValue := "Press Enter or 'f' to select config file"
	if v.fileSelectorLoading {
		loadConfigValue = "⟳ Loading file selector..."
	} else if v.lastLoadedFile != "" {
		loadConfigValue = fmt.Sprintf("Loaded: %s", v.lastLoadedFile)
	}

	// Style the load config value based on focus
	loadConfigStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(v.focusIndex == 0 && !v.backButtonFocused && !v.submitButtonFocused)

	if v.focusIndex == 0 && !v.backButtonFocused && !v.submitButtonFocused && v.inputMode == components.NormalMode {
		loadConfigStyle = loadConfigStyle.Background(lipgloss.Color("236"))
	}

	b.WriteString(loadConfigStyle.Render(loadConfigValue))
	b.WriteString("\n")

	// Run type field (selectable, now at index 1)
	b.WriteString(labelStyle.Render("⚙️ Run Type:"))
	if v.focusIndex == 1 && !v.backButtonFocused && !v.submitButtonFocused {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}

	runTypeValue := "▶️ Run (execution)"
	if v.runType == models.RunTypePlan {
		runTypeValue = "📋 Plan (pro-plan)"
	}

	// Style the run type value based on focus
	runTypeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(v.focusIndex == 1 && !v.backButtonFocused && !v.submitButtonFocused)

	if v.focusIndex == 1 && !v.backButtonFocused && !v.submitButtonFocused && v.inputMode == components.NormalMode {
		runTypeStyle = runTypeStyle.Background(lipgloss.Color("236"))
	}

	b.WriteString(runTypeStyle.Render(runTypeValue))
	b.WriteString("\n")

	// Repository field (now at index 2)
	b.WriteString(labelStyle.Render("📁 Repository:"))
	if v.focusIndex == 2 {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}
	v.fields[0].Width = min(width-22, 80)
	b.WriteString(v.fields[0].View())
	b.WriteString("\n")

	// Prompt area (now at index 3) - can be collapsed
	b.WriteString(labelStyle.Render("✏️ Prompt:"))
	if v.focusIndex == 3 {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}

	// Adjust prompt width
	v.promptArea.SetWidth(min(width-22, 100))

	// Show collapsed or full prompt
	if v.promptCollapsed && v.promptArea.Value() != "" {
		// Show first two lines when collapsed
		promptLines := strings.Split(v.promptArea.Value(), "\n")
		collapsedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Italic(true)

		// Show up to 2 lines
		linesToShow := 2
		if len(promptLines) < linesToShow {
			linesToShow = len(promptLines)
		}

		for i := 0; i < linesToShow; i++ {
			line := promptLines[i]
			if len(line) > width-24 {
				line = line[:width-27] + "..."
			}
			b.WriteString(collapsedStyle.Render(line))
			if i < linesToShow-1 {
				b.WriteString("\n                           ") // Indent continuation
			}
		}

		// Show [+] indicator if there's more content
		if len(promptLines) > 2 {
			b.WriteString(" [+]")
		}
	} else {
		b.WriteString(v.promptArea.View())
	}
	b.WriteString("\n")

	// Other fields in compact layout
	fieldInfo := []struct {
		label string
		index int
	}{
		{"🌿 Source (optional):", 1},
		{"🎯 Target (optional):", 2},
		{"📝 Title (optional):", 3},
		{"🔢 Issue (optional):", 4},
	}

	for _, field := range fieldInfo {
		b.WriteString(labelStyle.Render(field.label))
		adjustedIndex := field.index + 3 // +3 because load config is at 0, run type is at 1, repository is at 2, prompt is at index 3
		if v.focusIndex == adjustedIndex {
			b.WriteString(v.renderFieldIndicator())
		} else {
			b.WriteString("   ")
		}
		v.fields[field.index].Width = min(width-22, 60)
		b.WriteString(v.fields[field.index].View())
		b.WriteString("\n")
	}

	// Context area (optional, toggled with 'c')
	if v.showContext {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("💭 Context (optional):"))
		if v.focusIndex == len(v.fields)+3 { // +3 for load config, run type, and prompt
			b.WriteString(v.renderFieldIndicator())
		} else {
			b.WriteString("   ")
		}
		v.contextArea.SetWidth(min(width-22, 100))
		b.WriteString(v.contextArea.View())
	} else if v.contextArea.Value() != "" {
		// Show hint that context exists
		b.WriteString("\n")
		contextHint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("[Press 'c' to show context]")
		b.WriteString(contextHint)
	}

	// Submit button and validation on same line
	b.WriteString("\n\n")
	submitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	if v.submitButtonFocused {
		if v.inputMode == components.NormalMode {
			submitStyle = submitStyle.
				Background(lipgloss.Color("236")).
				Padding(0, 1)
		}
	}

	// Change text based on submission state
	submitText := "🚀 Submit Run"
	if v.isSubmitting {
		submitText = "⏳ SUBMITTING..."
	}
	b.WriteString(submitStyle.Render(submitText))

	// Validation indicator to the right of submit button
	isValid, validationError := v.validateForm()
	validationStyle := lipgloss.NewStyle().
		Padding(0, 0, 0, 2) // Add left padding to separate from submit button

	if isValid {
		// Show checkmark when valid
		validationStyle = validationStyle.Foreground(lipgloss.Color("82")) // Green
		b.WriteString(validationStyle.Render("✓ Ready to submit"))
	} else {
		// Show error message when invalid
		validationStyle = validationStyle.Foreground(lipgloss.Color("203")) // Red
		b.WriteString(validationStyle.Render("✗ " + validationError))
	}

	// Back button at bottom
	b.WriteString("\n\n")
	backStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	if v.backButtonFocused {
		if v.inputMode == components.InsertMode {
			backStyle = backStyle.
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255"))
		} else {
			backStyle = backStyle.
				Foreground(lipgloss.Color("63")).
				Bold(true)
		}
	}

	b.WriteString(backStyle.Render("← Back to Dashboard"))

	return b.String()
}

// renderErrorLayout renders the error message in a bordered box
func (v *CreateRunView) renderErrorLayout(availableHeight int) string {
	// Calculate box dimensions similar to form layout
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}

	panelHeight := availableHeight
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Error content
	var b strings.Builder

	// Add title header inside the error panel
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	b.WriteString(titleStyle.Render("Create New Run"))
	b.WriteString("\n")

	// Error header
	errorHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	b.WriteString(errorHeaderStyle.Render("❌ Error"))
	b.WriteString("\n\n")

	// Error message row (selectable)
	errorRowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(panelWidth - 6) // Account for padding and potential selection indicator

	if v.errorRowFocused {
		// Show selection indicator and highlight when focused
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Render(" ● "))

		// Add blinking effect if recently copied
		if v.yankBlink && !v.yankBlinkTime.IsZero() && time.Since(v.yankBlinkTime) < 2*time.Second {
			if v.yankBlink {
				// Bright green flash
				errorRowStyle = errorRowStyle.
					Background(lipgloss.Color("82")).
					Foreground(lipgloss.Color("0"))
			} else {
				// Normal focused style
				errorRowStyle = errorRowStyle.
					Background(lipgloss.Color("236"))
			}
		} else {
			errorRowStyle = errorRowStyle.
				Background(lipgloss.Color("236"))
		}
	} else {
		b.WriteString("   ")
	}

	b.WriteString(errorRowStyle.Render(v.error.Error()))
	b.WriteString("\n\n")

	// Back button
	backStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63"))

	if v.errorButtonFocused {
		// Show selection indicator and highlight when focused
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Render(" ● "))
		backStyle = backStyle.
			Bold(true).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
	} else {
		b.WriteString("   ")
		backStyle = backStyle.Foreground(lipgloss.Color("240"))
	}

	b.WriteString(backStyle.Render("← Back to Form"))
	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	var helpText string
	if v.errorRowFocused {
		helpText = "[j/k] navigate [Enter] back to form [y] copy error [q] back to form [r] retry"
	} else if v.errorButtonFocused {
		helpText = "[j/k] navigate [Enter] back to form [q] back to form [r] retry"
	} else {
		helpText = "[j/k] navigate [Enter] back to form [y] copy error [q] back to form [r] retry"
	}

	b.WriteString(helpStyle.Render(helpText))

	// Style for the panel
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1)

	return panelStyle.Render(b.String())
}

// initErrorFocus sets the default focus when entering error mode
func (v *CreateRunView) initErrorFocus() {
	// Save current focus state before entering error mode
	v.prevFocusIndex = v.focusIndex
	v.prevBackButtonFocused = v.backButtonFocused
	v.prevSubmitButtonFocused = v.submitButtonFocused

	// Default to focusing the error row first
	v.errorRowFocused = true
	v.errorButtonFocused = false
}

// restorePreviousFocus restores the focus state from before the error
func (v *CreateRunView) restorePreviousFocus() {
	v.focusIndex = v.prevFocusIndex
	v.backButtonFocused = v.prevBackButtonFocused
	v.submitButtonFocused = v.prevSubmitButtonFocused
	v.errorRowFocused = false
	v.errorButtonFocused = false
}

// copyToClipboard copies text to clipboard and shows feedback
func (v *CreateRunView) copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		if err := utils.WriteToClipboard(text); err != nil {
			debug.LogToFilef("DEBUG: Failed to copy to clipboard: %v\n", err)
			return clipboardResultMsg{success: false, text: text}
		}
		debug.LogToFilef("DEBUG: Successfully copied to clipboard: %s\n", text)
		return clipboardResultMsg{success: true, text: text}
	}
}

// renderFieldIndicator renders the field focus indicator
func (v *CreateRunView) renderFieldIndicator() string {
	if v.inputMode == components.InsertMode {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render(" ▶ ")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Render(" ● ")
}

// renderFileInputMode renders the file input interface
func (v *CreateRunView) renderFileInputMode() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Render("Load Task from File"))
	b.WriteString("\n\n")

	b.WriteString("File Path:\n")
	b.WriteString(v.filePathInput.View())
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	b.WriteString(helpStyle.Render("Ctrl+F: Manual input | Ctrl+S: Submit | ESC: Cancel"))

	return b.String()
}

func (v *CreateRunView) renderStatusBar() string {
	var statusText string

	// Handle reset confirmation mode - yellow styling like URL mode
	if v.resetConfirmMode {
		// Use yellow background color (226) to match URL opener style
		resetStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("0")).
			Width(v.width).
			Align(lipgloss.Center)

		statusContent := "[RESET] ⚠️  RESET ALL FIELDS? [y] confirm [n] cancel"
		return resetStyle.Render(statusContent)
	}

	// Handle error mode status
	if v.error != nil && !v.submitting {
		if v.errorRowFocused {
			statusText = "[Enter] back to form [j/k] navigate [y] copy error [q] back to form [r] retry [Q]uit"
		} else if v.errorButtonFocused {
			statusText = "[Enter] back to form [j/k] navigate [q] back to form [r] retry [Q]uit"
		} else {
			statusText = "[Enter] back to form [j/k] navigate [y] copy error [q] back to form [r] retry [Q]uit"
		}
		return components.DashboardStatusLine(v.width, "ERROR", "", statusText)
	}

	// Handle submitting state
	if v.isSubmitting {
		statusText = "[ESC] cancel submission [Q] quit"
		return components.DashboardStatusLine(v.width, "SUBMITTING", "", statusText)
	}

	// Handle duplicate confirmation mode
	if v.isDuplicateConfirm {
		// Create yellow status line similar to reset confirmation
		duplicateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).  // Black text
			Background(lipgloss.Color("11")). // Yellow background
			Bold(true).
			Width(v.width).
			Align(lipgloss.Center)

		statusContent := fmt.Sprintf("[DUPLICATE] ⚠️  DUPLICATE RUN DETECTED (ID: %s) - Override? [y] yes [n] no", v.duplicateRunID)
		return duplicateStyle.Render(statusContent)
	}

	if v.inputMode == components.InsertMode {
		if !v.useFileInput && v.focusIndex == 2 {
			// When repository field is focused (now at index 2), show FZF options
			statusText = "[Enter] exit insert [Tab] next [Ctrl+F] fuzzy [Ctrl+R] browse [Ctrl+S] submit"
		} else {
			statusText = "[Enter] exit insert [Tab] next [Ctrl+S] submit [Ctrl+X] clear"
		}
	} else {
		if v.submitButtonFocused {
			// Submit button in normal mode
			statusText = "[q]back [Enter] 🚀 SUBMIT RUN [j/k] navigate [Q]uit"
		} else if v.backButtonFocused {
			// Back button in normal mode
			statusText = "[Enter] back to dashboard [j/k] navigate [Q]uit"
		} else {
			switch v.focusIndex {
			case 0:
				// Load config field in normal mode (index 0)
				statusText = "[q]back [Enter] load config [f] file select [j/k] navigate [c] context [r] reset [Ctrl+S] submit [Q]uit"
			case 1:
				// Run type field in normal mode (index 1)
				statusText = "[q]back [Enter] toggle type [j/k] navigate [c] context [r] reset [Ctrl+S] submit [Q]uit"
			case 2:
				// Repository field in normal mode (index 2)
				statusText = "[q]back [Enter] edit [f] fuzzy [j/k] navigate [c] context [r] reset [Ctrl+S] submit [Q]uit"
			default:
				statusText = "[q]back [Enter] edit [j/k] navigate [c] context [r] reset [Ctrl+S] submit [?] help [Q]uit"
			}
		}
	}

	// Use DashboardStatusLine for consistent formatting with [CREATE] label
	return v.statusLine.
		SetWidth(v.width).
		SetLeft("[CREATE]").
		SetRight("").
		SetHelp(statusText).
		Render()
}

// prepareTaskFromFile loads and parses task from a JSON file
func (v *CreateRunView) prepareTaskFromFile(filePath string) (models.RunRequest, error) {
	if filePath == "" {
		return models.RunRequest{}, fmt.Errorf("file path is required")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return models.RunRequest{}, fmt.Errorf("failed to read file: %w", err)
	}

	var task models.RunRequest
	if err := json.Unmarshal(data, &task); err != nil {
		return models.RunRequest{}, fmt.Errorf("invalid JSON: %w", err)
	}

	return task, nil
}

// prepareTaskFromForm creates a task from form fields
func (v *CreateRunView) prepareTaskFromForm() models.RunRequest {
	task := models.RunRequest{
		Repository: v.fields[0].Value(),
		Source:     v.fields[1].Value(),
		Target:     v.fields[2].Value(),
		Title:      v.fields[3].Value(),
		Prompt:     v.promptArea.Value(),
		Context:    v.contextArea.Value(),
		RunType:    v.runType,
	}

	// Debug logging - check each field individually
	debugInfo := fmt.Sprintf("DEBUG: Raw field values - [0]='%s', [1]='%s', [2]='%s', [3]='%s', [4]='%s'\n",
		v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value(), v.fields[4].Value())
	debugInfo += fmt.Sprintf("DEBUG: Prompt='%s', Context='%s'\n", v.promptArea.Value(), v.contextArea.Value())
	debugInfo += fmt.Sprintf(
		"DEBUG: Submit values - Repository='%s', Source='%s', Target='%s', Title='%s', Prompt='%s'\n",
		task.Repository, task.Source, task.Target, task.Title, task.Prompt)
	debug.LogToFile(debugInfo)

	return task
}

// validateTask validates required fields in the task
func (v *CreateRunView) validateTask(task *models.RunRequest) error {
	if task.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if task.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	return nil
}

// autoDetectGitInfo fills in missing repository and branch information from git
func (v *CreateRunView) autoDetectGitInfo(task *models.RunRequest) {
	if task.Repository == "" {
		debug.LogToFile("DEBUG: Repository field empty, trying git auto-detect\n")
		if repo, _, err := pkgutils.GetGitInfo(); err == nil {
			task.Repository = repo
		}
	}

	if task.Source == "" {
		if _, branch, err := pkgutils.GetGitInfo(); err == nil {
			task.Source = branch
		}
		if task.Source == "" {
			task.Source = "main"
		}
	}

	if task.Target == "" {
		task.Target = fmt.Sprintf("repobird/%d", time.Now().Unix())
	}
}

// submitToAPI converts the task to API format and submits it
func (v *CreateRunView) submitToAPI(task models.RunRequest) (models.RunResponse, error) {
	// Convert to API-compatible format
	apiTask := task.ToAPIRequest()

	// Add file hash if we have one (from loaded config file)
	if v.currentFileHash != "" {
		apiTask.FileHash = v.currentFileHash
		debug.LogToFilef("DEBUG: Including file hash in API request: %s\n", v.currentFileHash)
	}

	// Debug: Log the final task object being sent to API
	debug.LogToFilef(
		"DEBUG: Final API task object - Title='%s', RepositoryName='%s', SourceBranch='%s', "+
			"TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s', FileHash='%s'\\n",
		apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch,
		apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType, apiTask.FileHash)

	runPtr, err := v.client.CreateRunAPI(apiTask)

	// Debug: Log the API response
	debug.LogToFilef("DEBUG: API response - err=%v, runPtr!=nil=%v\\n", err, runPtr != nil)

	if err != nil {
		return models.RunResponse{}, err
	}
	if runPtr == nil {
		return models.RunResponse{}, fmt.Errorf("API returned nil response")
	}

	return *runPtr, nil
}

// submitToAPIWithForce submits to API with the force flag to override duplicates
func (v *CreateRunView) submitToAPIWithForce(task models.RunRequest) (models.RunResponse, error) {
	// Convert to API-compatible format
	apiTask := task.ToAPIRequest()

	// Add file hash if we have one (from loaded config file)
	if v.currentFileHash != "" {
		apiTask.FileHash = v.currentFileHash
		debug.LogToFilef("DEBUG: Including file hash in API request with force override: %s\n", v.currentFileHash)
	}

	// Set force flag to override duplicate detection
	apiTask.Force = true

	// Debug: Log the final task object being sent to API
	debug.LogToFilef(
		"DEBUG: Final API task object WITH FORCE - Title='%s', RepositoryName='%s', SourceBranch='%s', "+
			"TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s', FileHash='%s', Force=%v\\n",
		apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch,
		apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType, apiTask.FileHash, apiTask.Force)

	runPtr, err := v.client.CreateRunAPI(apiTask)

	// Debug: Log the API response
	debug.LogToFilef("DEBUG: API response with force - err=%v, runPtr!=nil=%v\\n", err, runPtr != nil)

	if err != nil {
		return models.RunResponse{}, err
	}
	if runPtr == nil {
		return models.RunResponse{}, fmt.Errorf("API returned nil response")
	}

	return *runPtr, nil
}

// selectRepository triggers the repository selector
func (v *CreateRunView) selectRepository() tea.Cmd {
	return func() tea.Msg {
		// Suspend Bubble Tea temporarily and show fzf selector
		selectedRepo, err := v.repoSelector.SelectRepository()
		if err != nil {
			debug.LogToFilef("DEBUG: Repository selection failed: %v\n", err)
			return repositorySelectedMsg{repository: "", err: err}
		}

		debug.LogToFilef("DEBUG: Repository selected: %s\n", selectedRepo)
		return repositorySelectedMsg{repository: selectedRepo, err: nil}
	}
}

// prepareTask prepares a task from either file or form input
func (v *CreateRunView) prepareTask() (models.RunRequest, error) {
	var task models.RunRequest
	var err error

	if v.useFileInput {
		task, err = v.prepareTaskFromFile(v.filePathInput.Value())
		if err != nil {
			return task, err
		}
	} else {
		task = v.prepareTaskFromForm()
		v.autoDetectGitInfo(&task)

		if err := v.validateTask(&task); err != nil {
			return task, err
		}

		// Generate file hash for form-based submission if not already set
		if v.currentFileHash == "" {
			// Create a deterministic hash from the task content
			config := &models.RunConfig{
				Prompt:     task.Prompt,
				Repository: task.Repository,
				Source:     task.Source,
				Target:     task.Target,
				RunType:    string(task.RunType),
				Title:      task.Title,
				Context:    task.Context,
			}
			if hash, err := cache.CalculateConfigHash(config); err == nil && hash != "" {
				v.currentFileHash = hash
				debug.LogToFilef("DEBUG: Generated file hash for form-based submission: %s\n", hash)
			}
		}

		// Add repository to history after successful validation
		if task.Repository != "" {
			go func() {
				v.cache.AddRepositoryToHistory(task.Repository)
			}()
		}
	}

	return task, nil
}

func (v *CreateRunView) submitRun() tea.Cmd {
	// Set submitting state immediately
	v.isSubmitting = true
	v.submitStartTime = time.Now()

	return func() tea.Msg {
		debug.LogToFile("DEBUG: submitRun() called - starting submission process\n")

		// Save form data before submitting in case submission fails
		v.saveFormData()

		task, err := v.prepareTask()
		if err != nil {
			return runCreatedMsg{err: err}
		}

		run, err := v.submitToAPI(task)
		if err != nil {
			return runCreatedMsg{err: err}
		}

		return runCreatedMsg{run: run, err: nil}
	}
}

// submitWithForce submits the run with force flag to override duplicate detection
func (v *CreateRunView) submitWithForce() tea.Cmd {
	// Set submitting state immediately
	v.isSubmitting = true
	v.submitStartTime = time.Now()

	return func() tea.Msg {
		debug.LogToFile("DEBUG: submitWithForce() called - retrying submission with force override\n")

		task, err := v.prepareTask()
		if err != nil {
			return runCreatedMsg{err: err}
		}

		// Submit to API with force flag
		run, err := v.submitToAPIWithForce(task)
		if err != nil {
			return runCreatedMsg{err: err}
		}

		return runCreatedMsg{run: run, err: nil}
	}
}

// Message types and helper functions have been moved to create_messages.go

// startYankBlinkAnimation starts the blink animation for clipboard feedback
func (v *CreateRunView) startYankBlinkAnimation() tea.Cmd {
	return func() tea.Msg {
		// Single blink duration - quick flash (100ms)
		time.Sleep(100 * time.Millisecond)
		return yankBlinkMsg{}
	}
}

// startClearStatusTimer starts the timer to clear clipboard status
func (v *CreateRunView) startClearStatusTimer() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(3 * time.Second)
		return clearStatusMsg{}
	}
}

// tickCmd returns a command for the cursor blink animation
func (v *CreateRunView) tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return configSelectorTickMsg(t)
	})
}

// activateFZFMode activates FZF mode for repository selection
func (v *CreateRunView) activateFZFMode() {
	// Build list of repositories
	var items []string

	// Add current git repository if available
	if gitRepo, _, err := pkgutils.GetGitInfo(); err == nil && gitRepo != "" {
		items = append(items, fmt.Sprintf("📁 %s", gitRepo))
	}

	// Add repositories from history
	if history, err := v.cache.GetRepositoryHistory(); err == nil {
		for _, repoName := range history {
			if repoName != "" {
				// Skip if already added (git repo)
				skip := false
				for _, item := range items {
					if strings.Contains(item, repoName) {
						skip = true
						break
					}
				}
				if !skip {
					items = append(items, fmt.Sprintf("🔄 %s", repoName))
				}
			}
		}
	}

	// Add current value if not empty and not in list
	currentValue := v.fields[0].Value()
	if currentValue != "" {
		skip := false
		for _, item := range items {
			if strings.Contains(item, currentValue) {
				skip = true
				break
			}
		}
		if !skip {
			items = append([]string{fmt.Sprintf("✏️ %s", currentValue)}, items...)
		}
	}

	// Add example if no items
	if len(items) == 0 {
		items = []string{"📝 owner/repo"}
	}

	// Create FZF mode
	fieldWidth := 50 // Default width for repository field
	v.fzfMode = components.NewFZFMode(items, fieldWidth, 10)
	v.fzfMode.Activate()
	v.fzfActive = true
}

// activateConfigFileSelector activates the file selector for loading config files
func (v *CreateRunView) activateConfigFileSelector() tea.Cmd {
	return func() tea.Msg {
		// Set config file selector dimensions
		v.configFileSelector.SetDimensions(v.width, v.height)

		// Activate the enhanced config file selector with preview
		if err := v.configFileSelector.Activate(); err != nil {
			debug.LogToFilef("DEBUG: Failed to activate config file selector: %v\n", err)
			return configLoadErrorMsg{err: fmt.Errorf("failed to show file selector: %w", err)}
		}

		debug.LogToFile("DEBUG: Config file selector with preview activated\n")
		return fileSelectorActivatedMsg{}
	}
}

// loadConfigFromFile loads a configuration file and populates the form
func (v *CreateRunView) loadConfigFromFile(filePath string) tea.Cmd {
	return func() tea.Msg {
		debug.LogToFilef("DEBUG: Loading config from file: %s\n", filePath)

		// Load the config
		config, err := v.configLoader.LoadConfig(filePath)
		if err != nil {
			debug.LogToFilef("DEBUG: Failed to load config: %v\n", err)
			return configLoadErrorMsg{err: err}
		}

		// Calculate file hash for duplicate detection
		fileHash, hashErr := cache.CalculateFileHashFromPath(filePath)
		if hashErr != nil {
			debug.LogToFilef("DEBUG: Failed to calculate file hash: %v\n", hashErr)
			// Continue without hash - not a critical error
			fileHash = ""
		}

		debug.LogToFilef("DEBUG: Config loaded successfully from %s with hash %s\n", filePath, fileHash)
		return configLoadedMsg{
			config:   config,
			filePath: filePath,
			fileHash: fileHash,
		}
	}
}

// validateForm checks if the form is valid and returns any validation errors
func (v *CreateRunView) validateForm() (bool, string) {
	// Check required fields - only prompt and repository are required
	if strings.TrimSpace(v.promptArea.Value()) == "" {
		return false, "Prompt is required"
	}

	if strings.TrimSpace(v.fields[0].Value()) == "" {
		return false, "Repository is required"
	}

	// Check for duplicate file hash if a config file was loaded
	if v.isDuplicateRun && v.currentFileHash != "" {
		return false, "This task file has already been submitted (duplicate detected)"
	}

	return true, ""
}

// loadFileHashCache loads the file hash cache from the API
func (v *CreateRunView) loadFileHashCache() tea.Cmd {
	return func() tea.Msg {
		debug.LogToFile("DEBUG: loadFileHashCache - starting\n")

		// First ensure we have user info to set the correct cache directory
		userInfo := v.cache.GetUserInfo()
		if userInfo == nil {
			debug.LogToFile("DEBUG: loadFileHashCache - fetching user info first\n")
			// Fetch user info to get user ID for cache directory
			userInfo, err := v.client.GetUserInfo()
			if err != nil {
				debug.LogToFilef("DEBUG: loadFileHashCache - failed to get user info: %v\n", err)
			} else if userInfo != nil {
				v.cache.SetUserInfo(userInfo)
				debug.LogToFilef("DEBUG: loadFileHashCache - cached user info for user %d\n", userInfo.ID)
			}
		}

		// File hash cache is now embedded in the SimpleCache
		// No need to load separately
		debug.LogToFile("DEBUG: loadFileHashCache - using embedded cache\n")

		return nil
	}
}

// populateFormFromConfig populates the form fields from loaded config
func (v *CreateRunView) populateFormFromConfig(config *models.RunRequest, filePath string) {
	debug.LogToFilef("DEBUG: populateFormFromConfig START - config.Repository=%s, config.Source=%s, config.Target=%s, config.Title=%s, config.Prompt=%d chars\n",
		config.Repository, config.Source, config.Target, config.Title, len(config.Prompt))

	// Store the loaded file path
	v.lastLoadedFile = filepath.Base(filePath)

	// Keep focus on Load Config field (index 0) - no need to update
	// This prevents any layout shifts

	// Populate form fields
	if config.Repository != "" {
		v.fields[0].SetValue(config.Repository) // Repository field
		debug.LogToFilef("DEBUG: Set fields[0] to %s, actual value: %s\n", config.Repository, v.fields[0].Value())
	}

	if config.Prompt != "" {
		v.promptArea.SetValue(config.Prompt)
		debug.LogToFilef("DEBUG: Set promptArea to %d chars, actual value: %d chars\n", len(config.Prompt), len(v.promptArea.Value()))
		// Don't change collapsed state when loading config
		// This prevents layout shifts that hide the top rows
	}

	if config.Source != "" {
		v.fields[1].SetValue(config.Source) // Source field
		debug.LogToFilef("DEBUG: Set fields[1] to %s, actual value: %s\n", config.Source, v.fields[1].Value())
	}

	if config.Target != "" {
		v.fields[2].SetValue(config.Target) // Target field
		debug.LogToFilef("DEBUG: Set fields[2] to %s, actual value: %s\n", config.Target, v.fields[2].Value())
	}

	if config.Title != "" {
		v.fields[3].SetValue(config.Title) // Title field
		debug.LogToFilef("DEBUG: Set fields[3] to %s, actual value: %s\n", config.Title, v.fields[3].Value())
	}

	if config.Context != "" {
		v.contextArea.SetValue(config.Context)
		v.showContext = true // Show context field if it has content
		debug.LogToFilef("DEBUG: Set contextArea to %d chars, showContext=%v\n", len(config.Context), v.showContext)
	}

	// Set run type
	if config.RunType != "" {
		v.runType = config.RunType
		debug.LogToFilef("DEBUG: Set runType to %s\n", v.runType)
	}

	debug.LogToFilef("DEBUG: populateFormFromConfig END - fields[0]=%s, fields[1]=%s, fields[2]=%s, fields[3]=%s\n",
		v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value())
}

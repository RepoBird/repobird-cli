package views

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
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
	// Create new cache instance
	cache := cache.NewSimpleCache()
	_ = cache.LoadFromDisk()
	return NewCreateRunViewWithCache(client, cache)
}

// CreateRunViewConfig holds configuration for creating a new CreateRunView
type CreateRunViewConfig struct {
	Client             APIClient
	SelectedRepository string // Pre-selected repository from dashboard
}

// NewCreateRunViewWithConfig creates a new CreateRunView with the given configuration
func NewCreateRunViewWithConfig(cfg CreateRunViewConfig) *CreateRunView {
	debug.LogToFile("DEBUG: NewCreateRunViewWithConfig called\n")

	// Create own cache instance
	embeddedCache := cache.NewSimpleCache()
	_ = embeddedCache.LoadFromDisk()

	view := &CreateRunView{
		client:    cfg.Client,
		keys:      components.DefaultKeyMap,
		help:      help.New(),
		cache:     embeddedCache,
		runType:   models.RunTypeRun,
		inputMode: components.NormalMode,
	}

	// Initialize components
	view.initializeInputFields()
	view.loadFormData()

	// Auto-fill repository if selected from dashboard
	if cfg.SelectedRepository != "" && len(view.fields) >= 1 && view.fields[0].Value() == "" {
		view.fields[0].SetValue(cfg.SelectedRepository)
		debug.LogToFilef("DEBUG: Set repository from dashboard: %s\n", cfg.SelectedRepository)
	}

	// Initialize other components
	view.configLoader = config.NewConfigLoader()
	view.repoSelector = components.NewRepositorySelector()
	view.configFileSelector = components.NewConfigFileSelector(80, 24)

	return view
}

func NewCreateRunViewWithCache(
	client APIClient,
	embeddedCache *cache.SimpleCache,
) *CreateRunView {
	debug.LogToFile("DEBUG: Creating CreateView with cache\n")

	view := &CreateRunView{
		client:    client,
		keys:      components.DefaultKeyMap,
		help:      help.New(),
		cache:     embeddedCache,
		runType:   models.RunTypeRun,
		inputMode: components.NormalMode,
	}

	// Initialize components
	view.initializeInputFields()
	view.loadFormData()
	view.autofillRepository()

	// Initialize other components
	view.configLoader = config.NewConfigLoader()
	view.repoSelector = components.NewRepositorySelector()
	view.configFileSelector = components.NewConfigFileSelector(80, 24)

	return view
}

func (v *CreateRunView) Init() tea.Cmd {
	debug.LogToFile("DEBUG: CreateRunView.Init() called\n")

	// Ensure form data is loaded and fields are properly focused
	if v.inputMode == components.NormalMode && v.focusIndex < len(v.fields) {
		v.fields[v.focusIndex].Focus()
	}

	// Load file hash cache
	return tea.Batch(
		v.loadFileHashCache(),
		textinput.Blink,
		textarea.Blink,
	)
}

func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return v, v.handleWindowSizeMsg(msg)

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

		// Handle keyboard input based on current mode
		switch v.inputMode {
		case components.InsertMode:
			return v.handleInsertMode(msg)
		case components.NormalMode:
			return v.handleNormalMode(msg)
		default:
			// Handle error state if needed
			if v.error != nil {
				return v.handleErrorMode(msg)
			}
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
				v.cache.AddRepositoryToHistory(repoName)
			}
		}
		// Deactivate FZF mode
		v.fzfActive = false
		v.fzfMode = nil
		return v, nil

	case configLoadedMsg:
		// Handle successful config loading
		if config, ok := msg.config.(*models.RunRequest); ok {
			v.populateFormFromConfig(config, msg.path)
		}

		// IMPORTANT: Save the loaded config data to cache immediately
		// so it persists when navigating away
		v.saveFormData()

		if v.statusLine != nil {
			v.statusLine.SetTemporaryMessageWithType(
				fmt.Sprintf("✅ Config loaded from %s", filepath.Base(msg.path)),
				components.MessageSuccess,
				500*time.Millisecond, // Show for only 500ms
			)
		}
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
		return v, tickCmd()

	case configSelectorTickMsg:
		// Forward tick to config file selector if it's active
		if v.configFileSelector != nil && v.configFileSelector.IsActive() {
			// Convert to the config file selector's tick message type
			newSelector, cmd := v.configFileSelector.Update(components.TickMsg(time.Time(msg)))
			v.configFileSelector = newSelector
			cmds = append(cmds, cmd)
			// Continue ticking
			cmds = append(cmds, tickCmd())
		}
		return v, tea.Batch(cmds...)

	case clipboardResultMsg:
		if msg.success {
			v.yankBlink = true
			v.yankBlinkTime = time.Now()
			return v, tea.Batch(v.startYankBlinkAnimation(), v.startClearStatusTimer())
		}
		return v, nil

	case yankBlinkMsg:
		// Single blink: toggle off after being on
		if v.yankBlink {
			v.yankBlink = false // Turn off after being on - completes the single blink
		}
		return v, nil

	case clearStatusMsg:
		v.yankBlink = false
		return v, nil

	case gKeyTimeoutMsg:
		// Cancel waiting for second 'g' after timeout
		v.waitingForG = false
		return v, nil

	case fileHashCacheLoadedMsg:
		// File hash cache loaded - check if current file is a duplicate
		if v.currentFileHash != "" && msg.cache != nil {
			for _, hash := range msg.cache {
				if hash == v.currentFileHash {
					v.isDuplicateRun = true
					break
				}
			}
		}
		return v, nil
	}

	return v, tea.Batch(cmds...)
}

func (v *CreateRunView) handleWindowSizeMsg(msg tea.WindowSizeMsg) tea.Cmd {
	v.width = msg.Width
	v.height = msg.Height

	// Update component dimensions
	if v.configFileSelector != nil {
		v.configFileSelector.SetDimensions(v.width, v.height)
	}

	return nil
}

// clearAllFields clears all input fields
func (v *CreateRunView) clearAllFields() {
	for i := range v.fields {
		v.fields[i].SetValue("")
	}
	v.promptArea.SetValue("")
	v.contextArea.SetValue("")
}

// initErrorFocus saves the current focus state and switches to error mode
func (v *CreateRunView) initErrorFocus() {
	v.prevFocusIndex = v.focusIndex
	v.prevBackButtonFocused = v.backButtonFocused
	v.prevSubmitButtonFocused = v.submitButtonFocused
	v.inputMode = components.NormalMode // Error mode doesn't exist, keep in normal mode
	v.errorButtonFocused = true
	v.errorRowFocused = false
}

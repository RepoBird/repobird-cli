package views

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/pkg/utils"
)

type CreateRunView struct {
	client        *api.Client
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
	showHelp      bool
	useFileInput  bool
	filePathInput textinput.Model
	// Input mode tracking
	inputMode     components.InputMode
	exitRequested bool
	// Back button
	backButtonFocused bool
	// Cache from parent list view
	parentRuns         []models.RunResponse
	parentCached       bool
	parentCachedAt     time.Time
	parentDetailsCache map[string]*models.RunResponse
	// Repository selector
	repoSelector *components.RepositorySelector
	// FZF mode for repository selection
	fzfMode   *components.FZFMode
	fzfActive bool
}

func NewCreateRunView(client *api.Client) *CreateRunView {
	return NewCreateRunViewWithCache(client, nil, false, time.Time{}, nil)
}

// CreateRunViewConfig holds configuration for creating a new CreateRunView
type CreateRunViewConfig struct {
	Client             *api.Client
	ParentRuns         []models.RunResponse
	ParentCached       bool
	ParentCachedAt     time.Time
	ParentDetailsCache map[string]*models.RunResponse
}

// NewCreateRunViewWithConfig creates a new CreateRunView with the given configuration
func NewCreateRunViewWithConfig(config CreateRunViewConfig) *CreateRunView {
	v := &CreateRunView{
		client:             config.Client,
		keys:               components.DefaultKeyMap,
		help:               help.New(),
		inputMode:          components.InsertMode,
		parentRuns:         config.ParentRuns,
		parentCached:       config.ParentCached,
		parentCachedAt:     config.ParentCachedAt,
		parentDetailsCache: config.ParentDetailsCache,
	}

	v.repoSelector = components.NewRepositorySelector()
	v.initializeInputFields()
	v.loadFormData()
	v.autofillRepository()
	return v
}

// initializeInputFields sets up all the input fields
func (v *CreateRunView) initializeInputFields() {
	titleInput := textinput.New()
	titleInput.Placeholder = "Brief title for the run"
	titleInput.Focus()
	titleInput.CharLimit = 100
	titleInput.Width = 50

	repoInput := textinput.New()
	repoInput.Placeholder = "org/repo (leave empty to auto-detect)"
	repoInput.CharLimit = 100
	repoInput.Width = 50

	sourceInput := textinput.New()
	sourceInput.Placeholder = "main (leave empty to auto-detect)"
	sourceInput.CharLimit = 50
	sourceInput.Width = 30

	targetInput := textinput.New()
	targetInput.Placeholder = "feature/branch-name"
	targetInput.CharLimit = 50
	targetInput.Width = 30

	issueInput := textinput.New()
	issueInput.Placeholder = "#123 (optional)"
	issueInput.CharLimit = 20
	issueInput.Width = 20

	promptArea := textarea.New()
	promptArea.Placeholder = "Describe what you want the AI to do..."
	promptArea.SetWidth(60)
	promptArea.SetHeight(5)
	promptArea.CharLimit = 5000

	contextArea := textarea.New()
	contextArea.Placeholder = "Additional context (optional)..."
	contextArea.SetWidth(60)
	contextArea.SetHeight(3)
	contextArea.CharLimit = 2000

	filePathInput := textinput.New()
	filePathInput.Placeholder = "Path to task JSON file"
	filePathInput.CharLimit = 200
	filePathInput.Width = 50

	autoDetectGit(repoInput, sourceInput)

	v.fields = []textinput.Model{
		titleInput,
		repoInput,
		sourceInput,
		targetInput,
		issueInput,
	}
	v.promptArea = promptArea
	v.contextArea = contextArea
	v.filePathInput = filePathInput
	v.focusIndex = 0
}

// loadFormData loads saved form data from cache
func (v *CreateRunView) loadFormData() {
	savedData := cache.GetFormData()
	if savedData != nil && len(v.fields) >= 5 {
		v.fields[0].SetValue(savedData.Title)
		v.fields[1].SetValue(savedData.Repository)
		v.fields[2].SetValue(savedData.Source)
		v.fields[3].SetValue(savedData.Target)
		v.fields[4].SetValue(savedData.Issue)
		v.promptArea.SetValue(savedData.Prompt)
		v.contextArea.SetValue(savedData.Context)
	}
}

// autofillRepository sets the repository field with the most appropriate default
func (v *CreateRunView) autofillRepository() {
	// Only autofill if the repository field is empty
	if len(v.fields) >= 2 && v.fields[1].Value() == "" {
		defaultRepo := v.repoSelector.GetDefaultRepository()
		if defaultRepo != "" {
			v.fields[1].SetValue(defaultRepo)
		}
	}
}

// NewCreateRunViewWithCache maintains backward compatibility
func NewCreateRunViewWithCache(
	client *api.Client,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt time.Time,
	parentDetailsCache map[string]*models.RunResponse,
) *CreateRunView {
	debug.LogToFilef("DEBUG: Creating CreateView - parentRuns=%d, parentCached=%v, detailsCache=%d\n",
		len(parentRuns), parentCached, len(parentDetailsCache))

	config := CreateRunViewConfig{
		Client:             client,
		ParentRuns:         parentRuns,
		ParentCached:       parentCached,
		ParentCachedAt:     parentCachedAt,
		ParentDetailsCache: parentDetailsCache,
	}

	return NewCreateRunViewWithConfig(config)
}

func autoDetectGit(repoInput, sourceInput textinput.Model) {
	if repo, branch, err := utils.GetGitInfo(); err == nil {
		if repo != "" {
			repoInput.SetValue(repo)
		}
		if branch != "" {
			sourceInput.SetValue(branch)
		}
	}
}

func (v *CreateRunView) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Send a window size message with stored dimensions if we have them
	if v.width > 0 && v.height > 0 {
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: v.width, Height: v.height}
		})
	}

	cmds = append(cmds, textinput.Blink)
	return tea.Batch(cmds...)
}

// handleWindowSizeMsg handles window resize events
func (v *CreateRunView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height
	v.help.Width = msg.Width

	// Make text areas use most of the available width with some padding
	textAreaWidth := msg.Width - 10
	if textAreaWidth < 30 {
		textAreaWidth = 30 // Minimum usable width
	}

	// Update widths for all input fields to be responsive
	for i := range v.fields {
		v.fields[i].Width = min(textAreaWidth, 80) // Cap at 80 for readability
	}

	v.promptArea.SetWidth(textAreaWidth)
	v.contextArea.SetWidth(textAreaWidth)

	// Set appropriate heights for text areas based on available space
	// Reserve space for: title(2) + form fields(~15) + buttons(2) + status bar(1) + help(4 if shown)
	reservedHeight := 20
	if v.showHelp {
		reservedHeight += 4
	}

	availableHeight := msg.Height - reservedHeight
	if availableHeight > 10 {
		// Split available height between prompt and context areas
		promptHeight := availableHeight / 2
		contextHeight := availableHeight - promptHeight

		v.promptArea.SetHeight(max(3, promptHeight))
		v.contextArea.SetHeight(max(3, contextHeight))
	}
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
		// Activate FZF mode for repository field if focused on it
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
				v.focusIndex = 0
			}
		}
	case msg.String() == "ctrl+r":
		// Trigger repository selector when repository field is focused
		if !v.useFileInput && v.focusIndex == 1 {
			return v, v.selectRepository()
		}
	case msg.String() == "ctrl+s" || msg.String() == "ctrl+enter":
		if !v.submitting {
			debug.LogToFile("DEBUG: Ctrl+S pressed in INSERT MODE - submitting run\n")
			return v, v.submitRun()
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
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.Back) || msg.String() == "esc":
		if v.exitRequested {
			// Second ESC - actually exit
			if !v.submitting {
				v.saveFormData()
				debug.LogToFile("DEBUG: CreateView double ESC - returning to dashboard\n")
				// Return to dashboard view
				dashboard := NewDashboardView(v.client)
				dashboard.width = v.width
				dashboard.height = v.height
				return dashboard, dashboard.Init()
			}
		} else {
			// First ESC in normal mode - prepare to exit
			v.exitRequested = true
		}
	case key.Matches(msg, v.keys.Help):
		v.showHelp = !v.showHelp
	case msg.String() == "i" || msg.String() == "enter":
		if v.backButtonFocused {
			v.saveFormData()
			debug.LogToFile("DEBUG: Back button pressed - returning to dashboard\n")
			// Return to dashboard view
			dashboard := NewDashboardView(v.client)
			dashboard.width = v.width
			dashboard.height = v.height
			return dashboard, dashboard.Init()
		} else {
			v.inputMode = components.InsertMode
			v.exitRequested = false
			v.updateFocus()
		}
	case key.Matches(msg, v.keys.Up) || msg.String() == "k":
		v.prevField()
	case key.Matches(msg, v.keys.Down) || msg.String() == "j":
		v.nextField()
	case msg.String() == "ctrl+s":
		if !v.submitting {
			debug.LogToFile("DEBUG: Ctrl+S pressed in NORMAL MODE - submitting run\n")
			return v, v.submitRun()
		}
	case msg.String() == "ctrl+l":
		v.clearAllFields()
	case msg.String() == "ctrl+x":
		v.clearCurrentField()
	case msg.String() == "f":
		// In normal mode, 'f' activates FZF for repository field
		if v.focusIndex == 1 && !v.fzfActive {
			v.activateFZFMode()
			return v, nil
		}
	default:
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
	if msg.err != nil {
		v.error = msg.err
		return v, nil
	}

	// Check if the run has a valid ID
	runID := msg.run.GetIDString()
	if runID == "" {
		v.error = fmt.Errorf("run created but received invalid ID from server")
		debug.LogToFile("DEBUG: Run created successfully but runID is empty, not navigating to details\n")
		return v, nil
	}

	// Clear form data on successful submission
	cache.ClearFormData()
	v.success = true
	v.createdRun = &msg.run

	debug.LogToFilef("DEBUG: Run created successfully with ID='%s', navigating to details\n", runID)
	// Pass the cache data and current dimensions to the details view
	return NewRunDetailsViewWithCacheAndDimensions(v.client, msg.run, v.parentRuns, v.parentCached, v.parentCachedAt, v.parentDetailsCache, v.width, v.height), nil
}

// handleRepositorySelected handles the repositorySelectedMsg message
func (v *CreateRunView) handleRepositorySelected(msg repositorySelectedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		debug.LogToFilef("DEBUG: Repository selection error: %v\n", msg.err)
		v.error = msg.err
		return v, nil
	}

	// Set the selected repository in the repository field
	if len(v.fields) >= 2 && msg.repository != "" {
		v.fields[1].SetValue(msg.repository)
		debug.LogToFilef("DEBUG: Repository field updated to: %s\n", msg.repository)

		// Add to manual repository list for future use
		v.repoSelector.AddManualRepository(msg.repository)
	}

	return v, nil
}

func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
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
		if !msg.Result.Canceled && v.focusIndex == 1 {
			// Update repository field with selected value
			if msg.Result.Selected != "" {
				// Extract just the repository name (remove any icons)
				repoName := msg.Result.Selected
				if idx := strings.Index(repoName, " "); idx > 0 {
					repoName = repoName[idx+1:] // Skip icon
				}
				v.fields[1].SetValue(repoName)
				v.repoSelector.AddManualRepository(repoName)
			}
		}
		// Deactivate FZF mode
		v.fzfActive = false
		v.fzfMode = nil
		return v, nil
	}

	return v, tea.Batch(cmds...)
}

func (v *CreateRunView) updateFields(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	if v.focusIndex < len(v.fields) {
		var cmd tea.Cmd
		v.fields[v.focusIndex], cmd = v.fields[v.focusIndex].Update(msg)
		cmds = append(cmds, cmd)
	} else if v.focusIndex == len(v.fields) {
		var cmd tea.Cmd
		v.promptArea, cmd = v.promptArea.Update(msg)
		cmds = append(cmds, cmd)
	} else if v.focusIndex == len(v.fields)+1 {
		var cmd tea.Cmd
		v.contextArea, cmd = v.contextArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (v *CreateRunView) nextField() {
	v.backButtonFocused = false
	v.focusIndex++
	totalFields := len(v.fields) + 2 // fields + prompt + context
	if v.focusIndex >= totalFields {
		// After last field, go to back button
		v.backButtonFocused = true
		v.focusIndex = 0
	}
	v.updateFocus()
}

func (v *CreateRunView) prevField() {
	if v.backButtonFocused {
		// From back button, go to last field (context)
		v.backButtonFocused = false
		v.focusIndex = len(v.fields) + 1 // context area
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
		for i := range v.fields {
			if i == v.focusIndex {
				v.fields[i].Focus()
			} else {
				v.fields[i].Blur()
			}
		}

		if v.focusIndex == len(v.fields) {
			v.promptArea.Focus()
			v.contextArea.Blur()
		} else if v.focusIndex == len(v.fields)+1 {
			v.promptArea.Blur()
			v.contextArea.Focus()
		} else {
			v.promptArea.Blur()
			v.contextArea.Blur()
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
		Title:      v.fields[0].Value(),
		Repository: v.fields[1].Value(),
		Source:     v.fields[2].Value(),
		Target:     v.fields[3].Value(),
		Issue:      v.fields[4].Value(),
		Prompt:     v.promptArea.Value(),
		Context:    v.contextArea.Value(),
	}
	cache.SaveFormData(formData)
}

func (v *CreateRunView) clearAllFields() {
	for i := range v.fields {
		v.fields[i].SetValue("")
	}
	v.promptArea.SetValue("")
	v.contextArea.SetValue("")
	v.filePathInput.SetValue("")
	cache.ClearFormData()
}

func (v *CreateRunView) clearCurrentField() {
	if v.backButtonFocused {
		return // Can't clear back button
	}

	if v.useFileInput {
		v.filePathInput.SetValue("")
	} else if v.focusIndex < len(v.fields) {
		v.fields[v.focusIndex].SetValue("")
	} else if v.focusIndex == len(v.fields) {
		v.promptArea.SetValue("")
	} else if v.focusIndex == len(v.fields)+1 {
		v.contextArea.SetValue("")
	}
}

func (v *CreateRunView) View() string {
	var s strings.Builder

	title := styles.TitleStyle.Render("Create New Run")
	s.WriteString(title)
	s.WriteString("\n\n")

	if v.error != nil {
		s.WriteString(styles.ErrorStyle.Render("Error: " + v.error.Error()))
		s.WriteString("\n\n")
	}

	if v.useFileInput {
		s.WriteString("═══ Load from File ═══\n\n")
		s.WriteString("File Path: ")
		s.WriteString(v.filePathInput.View())
		s.WriteString("\n\n")
		s.WriteString("Press Ctrl+F to switch to manual input\n")
		s.WriteString("Press Ctrl+S to submit\n")
	} else {
		s.WriteString("═══ Run Configuration ═══\n\n")

		fieldLabels := []string{
			"Title:      ",
			"Repository: ",
			"Source:     ",
			"Target:     ",
			"Issue:      ",
		}

		for i, label := range fieldLabels {
			s.WriteString(label)
			if i == v.focusIndex && v.inputMode == components.InsertMode {
				s.WriteString(styles.SelectedStyle.Render("▶ "))
			} else if i == v.focusIndex {
				s.WriteString(styles.ProcessingStyle.Render("● "))
			} else {
				s.WriteString("  ")
			}
			s.WriteString(v.fields[i].View())
			s.WriteString("\n")
		}

		s.WriteString("\nPrompt:\n")
		if v.focusIndex == len(v.fields) && v.inputMode == components.InsertMode {
			s.WriteString(styles.SelectedStyle.Render("▶ "))
		} else if v.focusIndex == len(v.fields) {
			s.WriteString(styles.ProcessingStyle.Render("● "))
		} else {
			s.WriteString("  ")
		}
		s.WriteString(v.promptArea.View())
		s.WriteString("\n")

		s.WriteString("\nContext (optional):\n")
		if v.focusIndex == len(v.fields)+1 && v.inputMode == components.InsertMode {
			s.WriteString(styles.SelectedStyle.Render("▶ "))
		} else if v.focusIndex == len(v.fields)+1 {
			s.WriteString(styles.ProcessingStyle.Render("● "))
		} else {
			s.WriteString("  ")
		}
		s.WriteString(v.contextArea.View())
		s.WriteString("\n\n")

		// Back button
		if v.backButtonFocused {
			if v.inputMode == components.InsertMode {
				s.WriteString(styles.SelectedStyle.Render("← [Back to Runs]"))
			} else {
				s.WriteString(styles.ProcessingStyle.Render("← [Back to Runs]"))
			}
		} else {
			s.WriteString("← [Back to Runs]")
		}
		s.WriteString("\n\n")

		// Show mode-specific instructions
		if v.inputMode == components.InsertMode {
			s.WriteString("INSERT MODE | ESC: normal mode | Tab: next field | Ctrl+S: submit\n")
		} else {
			if v.exitRequested {
				s.WriteString("Press ESC again to exit | Enter: select | j/k: navigate | Ctrl+S: submit\n")
			} else {
				s.WriteString("NORMAL MODE | ESC: exit | Enter: select | j/k: navigate | Ctrl+S: submit\n")
			}
		}
	}

	if v.submitting {
		s.WriteString("\n")
		s.WriteString(styles.ProcessingStyle.Render("⟳ Creating run..."))
		s.WriteString("\n")
	}

	// Show error at bottom if present (make it more prominent)
	if v.error != nil {
		s.WriteString("\n")
		s.WriteString(styles.ErrorStyle.Render("❌ " + v.error.Error()))
		s.WriteString("\n")
	}

	statusBar := v.renderStatusBar()
	s.WriteString("\n")
	s.WriteString(statusBar)

	if v.showHelp {
		helpView := v.help.View(v.keys)
		s.WriteString("\n" + helpView)
	}

	// If FZF mode is active, overlay the dropdown
	if v.fzfMode != nil && v.fzfMode.IsActive() {
		return v.renderWithFZFOverlay(s.String())
	}

	return s.String()
}

// renderWithFZFOverlay renders the view with FZF dropdown overlay
func (v *CreateRunView) renderWithFZFOverlay(baseView string) string {
	if v.fzfMode == nil || !v.fzfMode.IsActive() {
		return baseView
	}

	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")

	// Calculate position for FZF dropdown (repository field is 2nd field)
	// Title + spacing + first field = about 5 lines
	// Repository field is around line 6
	yOffset := 6
	xOffset := 14 // After "Repository: " label

	// Create FZF dropdown view
	fzfView := v.fzfMode.View()
	fzfLines := strings.Split(fzfView, "\n")

	// Create a new view with the FZF dropdown overlaid
	result := make([]string, max(len(baseLines), yOffset+len(fzfLines)))
	copy(result, baseLines)

	// Ensure we have enough lines
	for i := len(baseLines); i < len(result); i++ {
		result[i] = ""
	}

	// Insert FZF dropdown at the calculated position
	for i, fzfLine := range fzfLines {
		lineIdx := yOffset + i
		if lineIdx >= 0 && lineIdx < len(result) {
			// Create the overlay line
			if xOffset < len(result[lineIdx]) {
				// Preserve part of the base line before the dropdown
				basePart := ""
				if xOffset > 0 {
					basePart = result[lineIdx][:min(xOffset, len(result[lineIdx]))]
				}
				// Add the FZF line
				result[lineIdx] = basePart + fzfLine
			} else {
				// Line is shorter than offset, pad and add FZF
				padding := strings.Repeat(" ", max(0, xOffset-len(result[lineIdx])))
				result[lineIdx] = result[lineIdx] + padding + fzfLine
			}
		}
	}

	return strings.Join(result, "\n")
}

func (v *CreateRunView) renderStatusBar() string {
	var statusText string
	if v.inputMode == components.InsertMode {
		if !v.useFileInput && v.focusIndex == 1 {
			// When repository field is focused, show FZF options
			statusText = "[ESC] normal [Tab] next [Ctrl+F] fuzzy search [Ctrl+R] browse repos [Ctrl+S] submit"
		} else {
			statusText = "[ESC] normal mode [Tab] next [Ctrl+S] submit [Ctrl+X] clear field [Ctrl+L] clear all"
		}
	} else {
		if v.exitRequested {
			statusText = "[ESC] exit [Enter] select [j/k] navigate [Ctrl+S] submit [Ctrl+X] clear field"
		} else if v.focusIndex == 1 {
			// Repository field in normal mode
			statusText = "[ESC] exit [Enter] edit [f] fuzzy search [j/k] navigate [Ctrl+S] submit [?] help"
		} else {
			statusText = "[ESC] exit [Enter] select [j/k] navigate [Ctrl+S] submit [Ctrl+X] clear field [?] help"
		}
	}

	return styles.StatusBarStyle.Width(v.width).Render(statusText)
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
		Title:      v.fields[0].Value(),
		Repository: v.fields[1].Value(),
		Source:     v.fields[2].Value(),
		Target:     v.fields[3].Value(),
		Prompt:     v.promptArea.Value(),
		Context:    v.contextArea.Value(),
		RunType:    models.RunTypeRun,
	}

	// Debug logging - check each field individually
	debugInfo := fmt.Sprintf("DEBUG: Raw field values - [0]='%s', [1]='%s', [2]='%s', [3]='%s', [4]='%s'\n",
		v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value(), v.fields[4].Value())
	debugInfo += fmt.Sprintf("DEBUG: Prompt='%s', Context='%s'\n", v.promptArea.Value(), v.contextArea.Value())
	debugInfo += fmt.Sprintf(
		"DEBUG: Submit values - Title='%s', Repository='%s', Source='%s', Target='%s', Prompt='%s'\n",
		task.Title, task.Repository, task.Source, task.Target, task.Prompt)
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
		if repo, _, err := utils.GetGitInfo(); err == nil {
			task.Repository = repo
		}
	}

	if task.Source == "" {
		if _, branch, err := utils.GetGitInfo(); err == nil {
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

	// Debug: Log the final task object being sent to API
	debug.LogToFilef(
		"DEBUG: Final API task object - Title='%s', RepositoryName='%s', SourceBranch='%s', "+
			"TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s'\\n",
		apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch,
		apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType)

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

func (v *CreateRunView) submitRun() tea.Cmd {
	return func() tea.Msg {
		debug.LogToFile("DEBUG: submitRun() called - starting submission process\n")

		// Save form data before submitting in case submission fails
		v.saveFormData()

		var task models.RunRequest
		var err error

		if v.useFileInput {
			task, err = v.prepareTaskFromFile(v.filePathInput.Value())
			if err != nil {
				return runCreatedMsg{err: err}
			}
		} else {
			task = v.prepareTaskFromForm()
			v.autoDetectGitInfo(&task)

			if err := v.validateTask(&task); err != nil {
				return runCreatedMsg{err: err}
			}

			// Add repository to history after successful validation
			if task.Repository != "" {
				go func() {
					_ = cache.AddRepositoryToHistory(task.Repository)
				}()
			}
		}

		run, err := v.submitToAPI(task)
		if err != nil {
			return runCreatedMsg{err: err}
		}

		return runCreatedMsg{run: run, err: nil}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type runCreatedMsg struct {
	run models.RunResponse
	err error
}

type repositorySelectedMsg struct {
	repository string
	err        error
}

// activateFZFMode activates FZF mode for repository selection
func (v *CreateRunView) activateFZFMode() {
	// Build list of repositories
	var items []string

	// Add current git repository if available
	if gitRepo, _, err := utils.GetGitInfo(); err == nil && gitRepo != "" {
		items = append(items, fmt.Sprintf("📁 %s", gitRepo))
	}

	// Add repositories from history
	if history, err := cache.GetRepositoryHistory(); err == nil {
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
	currentValue := v.fields[1].Value()
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

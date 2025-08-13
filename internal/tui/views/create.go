package views

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	tuicache "github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/pkg/utils"
)

// CreateRunView implements a form-based view for creating new runs
type CreateRunView struct {
	client APIClient
	cache  *tuicache.SimpleCache
	layout *components.WindowLayout
	form   *CustomCreateForm // Using custom form now
	width  int
	height int

	// State
	submitting bool
	error      error
}

// NewCreateRunView creates a new create run view with proper dependencies
func NewCreateRunView(client APIClient, cache *tuicache.SimpleCache) *CreateRunView {
	debug.LogToFilef("🆕 CREATE VIEW: Creating new CreateRunView")

	v := &CreateRunView{
		client: client,
		cache:  cache,
		layout: components.NewWindowLayout(80, 24), // Default dimensions
		form:   NewCustomCreateForm(),               // Use custom form
	}

	return v
}

// Init initializes the create view and loads navigation context
func (v *CreateRunView) Init() tea.Cmd {
	debug.LogToFilef("📤 CREATE VIEW: Initializing CreateRunView")

	// Check for navigation context to pre-populate repository field
	if selectedRepo := v.cache.GetNavigationContext("selected_repo"); selectedRepo != nil {
		if repoStr, ok := selectedRepo.(string); ok {
			v.form.SetValue("repository", repoStr)
			debug.LogToFilef("📋 CREATE VIEW: Pre-populated repository from context: %s", repoStr)
		}
	}

	// Try to detect current git repository
	if currentRepo, err := utils.DetectRepository(); err == nil && currentRepo != "" {
		// Only set if not already set from context
		values := v.form.GetValues()
		if values["repository"] == "" {
			v.form.SetValue("repository", currentRepo)
			debug.LogToFilef("🔍 CREATE VIEW: Auto-detected repository: %s", currentRepo)
		}
	}

	// Try to detect current branch for source
	if currentBranch, err := utils.GetCurrentBranch(); err == nil && currentBranch != "" {
		v.form.SetValue("source", currentBranch)
		debug.LogToFilef("🌿 CREATE VIEW: Auto-detected branch: %s", currentBranch)
	}

	// Load saved form data if available (takes precedence over auto-detection)
	if savedFormData := v.cache.GetFormData(); savedFormData != nil {
		debug.LogToFilef("💾 CREATE VIEW: Loading saved form data")
		v.form.SetValue("title", savedFormData.Title)
		v.form.SetValue("repository", savedFormData.Repository)
		v.form.SetValue("source", savedFormData.Source)
		v.form.SetValue("target", savedFormData.Target)
		v.form.SetValue("prompt", savedFormData.Prompt)
		v.form.SetValue("context", savedFormData.Context)
		
		// Restore the runtype
		if savedFormData.RunType != "" {
			v.form.SetValue("runtype", savedFormData.RunType)
			debug.LogToFilef("⚙️ CREATE VIEW: Restored runtype: %s", savedFormData.RunType)
		}
		
		// Restore the focus index if available in Fields map
		if savedFormData.Fields != nil {
			if focusIndexStr, ok := savedFormData.Fields["_focusIndex"]; ok {
				// Parse the focus index (stored as string)
				if focusIndex, err := strconv.Atoi(focusIndexStr); err == nil {
					v.form.SetFocusIndex(focusIndex)
					debug.LogToFilef("🎯 CREATE VIEW: Restored focus index: %d", focusIndex)
				}
			}
		}
	}

	return v.form.Init()
}

// Update handles all messages and form interactions
func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return v.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case components.FormSubmitMsg:
		return v.handleFormSubmit(msg)
		
	case CustomFormSubmitMsg:
		return v.handleCustomFormSubmit(msg)

	case runCreatedMsg:
		return v.handleRunCreated(msg)

	default:
		// Delegate to form component
		newForm, cmd := v.form.Update(msg)
		v.form = newForm.(*CustomCreateForm)
		return v, cmd
	}
}

// handleWindowSizeMsg updates layout and form dimensions
func (v *CreateRunView) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	v.width = msg.Width
	v.height = msg.Height
	v.layout.Update(msg.Width, msg.Height)

	// Update form dimensions based on layout
	contentWidth, contentHeight := v.layout.GetContentDimensions()
	newForm, _ := v.form.Update(tea.WindowSizeMsg{
		Width:  contentWidth,
		Height: contentHeight,
	})
	v.form = newForm.(*CustomCreateForm)

	debug.LogToFilef("📐 CREATE VIEW: Updated dimensions: terminal=%dx%d, content=%dx%d", 
		msg.Width, msg.Height, contentWidth, contentHeight)

	return v, nil
}

// handleKeyMsg processes keyboard input
func (v *CreateRunView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Most key handling is done in HandleKey() via CoreViewKeymap
	// Here we only handle keys that HandleKey doesn't process
	
	// Handle navigation keys in normal mode
	if !v.form.IsInsertMode() {
		switch msg.String() {
		case "q", "b":
			debug.LogToFilef("🔙 CREATE VIEW: User requested back navigation")
			// Save form data before navigating away
			v.saveFormData()
			return v, func() tea.Msg {
				return messages.NavigateBackMsg{}
			}

		case "ctrl+c":
			// Force quit - handled by app layer
			return v, tea.Quit
		}
	}

	// Delegate to form component for any remaining keys
	newForm, cmd := v.form.Update(msg)
	v.form = newForm.(*CustomCreateForm)
	
	// Auto-save form data when values change
	v.saveFormData()
	
	return v, cmd
}

// handleFormSubmit processes form submission (legacy compatibility)
func (v *CreateRunView) handleFormSubmit(msg components.FormSubmitMsg) (tea.Model, tea.Cmd) {
	if v.submitting {
		return v, nil // Prevent double submission
	}

	v.submitting = true
	v.error = nil

	debug.LogToFilef("📝 CREATE VIEW: Form submitted with values: %+v", msg.Values)

	// Create run request from form values
	request := &models.APIRunRequest{
		Title:          msg.Values["title"],
		RepositoryName: msg.Values["repository"],
		SourceBranch:   msg.Values["source"],
		TargetBranch:   msg.Values["target"],
		Prompt:         msg.Values["prompt"],
		Context:        msg.Values["context"],
		RunType:        models.RunType("run"), // Default to run type
	}

	// Submit asynchronously
	return v, v.submitRunCmd(request)
}

// handleCustomFormSubmit processes custom form submission
func (v *CreateRunView) handleCustomFormSubmit(msg CustomFormSubmitMsg) (tea.Model, tea.Cmd) {
	if v.submitting {
		return v, nil // Prevent double submission
	}

	v.submitting = true
	v.error = nil

	debug.LogToFilef("📝 CREATE VIEW: Custom form submitted with values: %+v", msg.Values)

	// Get runtype from form values, default to "run" if not specified
	runType := msg.Values["runtype"]
	if runType == "" {
		runType = "run"
	}

	// Create run request from form values
	request := &models.APIRunRequest{
		Title:          msg.Values["title"],
		RepositoryName: msg.Values["repository"],
		SourceBranch:   msg.Values["source"],
		TargetBranch:   msg.Values["target"],
		Prompt:         msg.Values["prompt"],
		Context:        msg.Values["context"],
		RunType:        models.RunType(runType), // Use the selected run type
	}

	// Submit asynchronously
	return v, v.submitRunCmd(request)
}

// handleRunCreated processes the result of run creation
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
	v.submitting = false

	if msg.err != nil {
		v.error = msg.err
		debug.LogToFilef("❌ CREATE VIEW: Run creation failed: %v", msg.err)
		return v, nil
	}

	debug.LogToFilef("✅ CREATE VIEW: Run created successfully: %s", msg.run.GetIDString())

	// Cache the new run
	v.cache.SetRun(*msg.run)

	// Clear form data since run was successfully created
	v.cache.ClearFormData()
	debug.LogToFilef("🗑️ CREATE VIEW: Cleared form data after successful run creation")

	// Navigate to details view
	return v, func() tea.Msg {
		return messages.NavigateToDetailsMsg{
			RunID:      msg.run.GetIDString(),
			FromCreate: true,
		}
	}
}

// View renders the create view
func (v *CreateRunView) View() string {
	if !v.layout.IsValidDimensions() {
		return v.layout.GetMinimalView("Create Run - Terminal too small")
	}

	// Create styled box using layout
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()
	contentStyle := v.layout.CreateContentStyle()

	// Build content
	var content strings.Builder

	// Add title
	content.WriteString(titleStyle.Render("Create New Run"))
	content.WriteString("\n\n")

	// Add form
	content.WriteString(v.form.View())

	// Add submission status
	if v.submitting {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))
		content.WriteString("\n")
		content.WriteString(statusStyle.Render("⟳ Creating run..."))
	} else if v.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error: %v", v.error)))
	}

	// Wrap in styled box
	boxedContent := boxStyle.Render(contentStyle.Render(content.String()))
	
	// Add statusline
	statusLine := v.renderStatusLine("CREATE")
	
	// Join box and statusline
	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
}

// renderStatusLine renders the status line with help text based on form mode
func (v *CreateRunView) renderStatusLine(layoutName string) string {
	// Add mode indicator
	var modeIndicator string
	var helpText string
	
	if v.form.IsInsertMode() {
		modeIndicator = " [INPUT]"
		helpText = "[esc]normal [tab]next [shift+tab]prev [ctrl+s]submit"
	} else {
		modeIndicator = ""
		helpText = "[i]insert [d]delete [c]change [j/k/↑↓]nav [b/q]back [ctrl+s]submit"
	}
	
	// Compose the left side with layout name and mode indicator
	leftText := fmt.Sprintf("[%s]%s", layoutName, modeIndicator)
	
	// Create statusline component
	statusLine := components.NewStatusLine().
		SetWidth(v.width).
		SetLeft(leftText).
		SetRight("").
		SetHelp(helpText).
		ResetStyle().
		SetLoading(v.submitting)
	
	return statusLine.Render()
}

// saveFormData saves the current form state to cache
func (v *CreateRunView) saveFormData() {
	values := v.form.GetValues()
	
	// Create fields map to store additional state
	fields := make(map[string]string)
	fields["_focusIndex"] = fmt.Sprintf("%d", v.form.GetFocusIndex())
	
	// Get runtype from form values
	runType := values["runtype"]
	if runType == "" {
		runType = "run" // Default
	}
	
	formData := &tuicache.FormData{
		Title:      values["title"],
		Repository: values["repository"],
		Source:     values["source"],
		Target:     values["target"],
		Prompt:     values["prompt"],
		Context:    values["context"],
		RunType:    runType, // Save the selected run type
		Fields:     fields, // Store focus index and other metadata
	}
	
	v.cache.SetFormData(formData)
	debug.LogToFilef("💾 CREATE VIEW: Form data saved to cache (focus: %d, runtype: %s)", v.form.GetFocusIndex(), runType)
}

// submitRunCmd creates a command to submit the run asynchronously
func (v *CreateRunView) submitRunCmd(request *models.APIRunRequest) tea.Cmd {
	return func() tea.Msg {
		run, err := v.client.CreateRunAPI(request)
		return runCreatedMsg{run: run, err: err}
	}
}

// runCreatedMsg is sent when run creation completes
type runCreatedMsg struct {
	run *models.RunResponse
	err error
}


// Backward compatibility constructor - redirects to proper constructor
func NewCreateRunViewWithCache(
	client APIClient,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt interface{},
	parentDetailsCache interface{},
	embeddedCache interface{},
) *CreateRunView {
	// Extract cache if provided, otherwise create new one
	var simpleCache *tuicache.SimpleCache
	if cacheInstance, ok := embeddedCache.(*tuicache.SimpleCache); ok {
		simpleCache = cacheInstance
	} else {
		simpleCache = tuicache.NewSimpleCache()
	}

	return NewCreateRunView(client, simpleCache)
}

// IsKeyDisabled implements CoreViewKeymap interface to control key behavior
func (v *CreateRunView) IsKeyDisabled(keyString string) bool {
	// We don't disable any keys - we handle them properly in HandleKey
	return false
}

// HandleKey implements CoreViewKeymap interface for custom key handling
func (v *CreateRunView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	keyString := keyMsg.String()
	debug.LogToFilef("🔑 CREATE VIEW HandleKey: key='%s', insertMode=%v", keyString, v.form.IsInsertMode())
	
	// Handle keys differently based on mode
	if v.form.IsInsertMode() {
		// In INSERT mode, we need to handle special keys but pass through typing
		switch keyString {
		case "esc":
			// ESC exits insert mode
			debug.LogToFilef("⬅️ CREATE VIEW: ESC pressed - exiting insert mode")
			v.form.SetInsertMode(false)
			v.saveFormData() // Save when exiting insert mode
			return true, v, nil
			
		case "backspace":
			// Pass backspace to form for text deletion
			debug.LogToFilef("⌫ CREATE VIEW: Backspace in insert mode - passing to form")
			newForm, cmd := v.form.Update(keyMsg)
			v.form = newForm.(*CustomCreateForm)
			v.saveFormData() // Auto-save on change
			return true, v, cmd
			
		case "q", "b":
			// In insert mode, these are just characters to type
			debug.LogToFilef("⌨️ CREATE VIEW: Typing '%s' in insert mode", keyString)
			newForm, cmd := v.form.Update(keyMsg)
			v.form = newForm.(*CustomCreateForm)
			v.saveFormData() // Auto-save on change
			return true, v, cmd
			
		default:
			// For all other keys in insert mode, let the form handle them
			// This includes regular typing, tab, shift+tab, etc.
			// We return false so the key goes through normal processing
			return false, v, nil
		}
	} else {
		// In NORMAL mode, handle navigation and vim commands
		switch keyString {
		case "esc":
			// In normal mode, ESC doesn't navigate back - it's already in normal mode
			debug.LogToFilef("ℹ️ CREATE VIEW: ESC in normal mode - no action")
			return true, v, nil
			
		case "backspace":
			// In normal mode, backspace should NOT navigate back
			debug.LogToFilef("🚫 CREATE VIEW: Blocking backspace navigation in normal mode")
			return true, v, nil
			
		case "up", "down":
			// Convert arrow keys to j/k for form navigation
			var newKeyMsg tea.KeyMsg
			if keyString == "up" {
				newKeyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
			} else { // down
				newKeyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			}
			
			// Let form handle the converted key
			newForm, cmd := v.form.Update(newKeyMsg)
			v.form = newForm.(*CustomCreateForm)
			return true, v, cmd
			
		case "d":
			// Delete current field's text
			debug.LogToFilef("✂️ CREATE VIEW: Delete command - clearing current field")
			v.clearCurrentField()
			return true, v, nil
			
		case "c":
			// Change - clear current field and enter insert mode
			debug.LogToFilef("✏️ CREATE VIEW: Change command - clearing field and entering insert mode")
			v.clearCurrentField()
			v.form.SetInsertMode(true)
			return true, v, nil
			
		case "q", "b":
			// In normal mode, these are navigation keys - let handleKeyMsg handle them
			debug.LogToFilef("🔙 CREATE VIEW: Navigation key '%s' in normal mode - not handling", keyString)
			return false, v, nil
			
		default:
			// Other keys in normal mode - let default handling occur
			return false, v, nil
		}
	}
}

// clearCurrentField clears the text of the currently focused field
func (v *CreateRunView) clearCurrentField() {
	// Clear the current field using the form's new method
	v.form.ClearCurrentField()
	
	// Log which field was cleared
	fieldName := v.form.GetCurrentFieldName()
	debug.LogToFilef("🗑️ CREATE VIEW: Cleared field '%s'", fieldName)
	
	// Auto-save the form state after clearing
	v.saveFormData()
}
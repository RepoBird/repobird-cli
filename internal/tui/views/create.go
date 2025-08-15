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
		form:   NewCustomCreateForm(),              // Use custom form
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

	// Try to use last used repository from cache instead of git detection
	values := v.form.GetValues()
	if values["repository"] == "" {
		// First try to get last used repository from permanent cache
		if lastRepo, found := v.cache.GetLastUsedRepository(); found && lastRepo != "" {
			v.form.SetValue("repository", lastRepo)
			debug.LogToFilef("💾 CREATE VIEW: Using last used repository from cache: %s", lastRepo)
		} else {
			// Fallback: try to get repository from most recent run in cache
			runs := v.cache.GetRuns()
			if len(runs) > 0 && runs[0].Repository != "" {
				v.form.SetValue("repository", runs[0].Repository)
				debug.LogToFilef("📝 CREATE VIEW: Using repository from most recent run: %s", runs[0].Repository)
			}
		}
	}

	// Don't auto-fill source branch - keep it as default "main"
	// User requested to not do any git detection for branches

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
	// Log message type for debugging
	debug.LogToFilef("📨 CREATE VIEW Update: received %T", msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return v.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case components.FormSubmitMsg:
		return v.handleFormSubmit(msg)

	case CustomFormSubmitMsg:
		return v.handleCustomFormSubmit(msg)

	case CustomFormNavigateBackMsg:
		debug.LogToFilef("🔙 CREATE VIEW: Received navigation back message from form")
		// Save form data before navigating away
		v.saveFormData()
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

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
	keyString := msg.String()
	debug.LogToFilef("🎹 CREATE VIEW handleKeyMsg: key='%s', insertMode=%v", keyString, v.form.IsInsertMode())

	// Note: Most key handling is done in HandleKey() via CoreViewKeymap
	// HandleKey returns handled=true for keys it processes
	// Those keys should NOT reach here, but if they do, we handle them

	// Handle force quit
	if keyString == "ctrl+c" {
		debug.LogToFilef("⛔ CREATE VIEW: Force quit requested")
		return v, tea.Quit
	}

	// Skip keys that should have been handled by HandleKey
	// This prevents double processing
	switch keyString {
	case "esc":
		debug.LogToFilef("⚠️ CREATE VIEW handleKeyMsg: ESC reached handleKeyMsg (should have been handled by HandleKey)")
		// Don't process it again
		return v, nil
	case "q", "b":
		if !v.form.IsInsertMode() {
			debug.LogToFilef("⚠️ CREATE VIEW handleKeyMsg: '%s' in normal mode reached handleKeyMsg (should have been handled by HandleKey)", keyString)
			// Don't process it again
			return v, nil
		}
	}

	// Delegate to the form for updating its internal state
	newForm, cmd := v.form.Update(msg)
	v.form = newForm.(*CustomCreateForm)

	// Auto-save form data when values change
	if v.form.IsInsertMode() || keyString == "d" || keyString == "c" || keyString == "i" {
		// Save when typing, deleting, changing, or entering insert mode
		v.saveFormData()
	}

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

	// Save the repository name to permanent cache for future use
	if msg.run.Repository != "" {
		if err := v.cache.SetLastUsedRepository(msg.run.Repository); err != nil {
			debug.LogToFilef("⚠️ CREATE VIEW: Failed to save last used repository: %v", err)
		} else {
			debug.LogToFilef("💾 CREATE VIEW: Saved last used repository: %s", msg.run.Repository)
		}
	}

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
	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter(layoutName, v.width)

	// Determine mode and help text
	var mode string
	var helpText string

	if v.form.IsInsertMode() {
		mode = "INPUT"
		helpText = "[esc]normal [tab]next [shift+tab]prev [ctrl+s]submit"
	} else {
		mode = ""
		helpText = "[i]insert [d]delete [c]change [j/k/↑↓]nav [h]back [q]dashboard [ctrl+s]submit"
	}

	// Format left content consistently
	leftContent := formatter.FormatViewNameWithMode(mode)

	// Create status line using formatter
	statusLine := formatter.StandardStatusLine(leftContent, "", helpText)
	return statusLine.
		SetLoading(v.submitting).
		Render()
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
		Fields:     fields,  // Store focus index and other metadata
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
	// In insert mode, disable ALL navigation keys except ESC and ctrl+c
	// When a key is disabled, it bypasses navigation but still reaches Update() for typing
	if v.form.IsInsertMode() {
		// Only allow ESC and ctrl+c to be processed by navigation system
		// Everything else should be disabled to allow normal typing
		switch keyString {
		case "esc", "ctrl+c":
			// These keys should NOT be disabled - they need special handling
			debug.LogToFilef("✅ CREATE VIEW IsKeyDisabled: Allowing '%s' in insert mode for mode switching/quit", keyString)
			return false
		default:
			// Disable ALL other keys from navigation processing in insert mode
			// This includes h, j, k, l, q, b, backspace, etc.
			debug.LogToFilef("🚫 CREATE VIEW IsKeyDisabled: Disabling '%s' navigation in insert mode to allow typing", keyString)
			return true // Disable navigation processing - key will reach Update() for typing
		}
	}
	return false
}

// HandleKey implements CoreViewKeymap interface for custom key handling
func (v *CreateRunView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	keyString := keyMsg.String()
	debug.LogToFilef("🔑 CREATE VIEW HandleKey: key='%s', insertMode=%v", keyString, v.form.IsInsertMode())

	// IMPORTANT: Handle ESC specially to prevent navigation
	if keyString == "esc" {
		if v.form.IsInsertMode() {
			// In insert mode, ESC should exit to normal mode (not navigate)
			debug.LogToFilef("⬅️ CREATE VIEW HandleKey: ESC in insert mode - exiting to normal mode")
			v.form.SetInsertMode(false)
			v.saveFormData()
			return true, v, nil // Return handled=true to prevent navigation
		} else {
			// In normal mode, ESC does nothing (not navigate)
			debug.LogToFilef("ℹ️ CREATE VIEW HandleKey: ESC in normal mode - no action")
			return true, v, nil // Return handled=true to prevent navigation
		}
	}

	// In normal mode, block backspace navigation
	if keyString == "backspace" && !v.form.IsInsertMode() {
		debug.LogToFilef("🚫 CREATE VIEW HandleKey: Blocking backspace navigation in normal mode")
		return true, v, nil
	}

	// Let everything else go through - the keymap registry will handle navigation keys in normal mode
	// In insert mode, returning false means the key goes to Update() for typing
	// In normal mode, returning false means the keymap registry handles navigation
	debug.LogToFilef("➡️ CREATE VIEW HandleKey: Not handling '%s', letting system decide", keyString)
	return false, v, nil
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

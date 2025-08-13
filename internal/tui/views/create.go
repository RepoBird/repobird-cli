package views

import (
	"fmt"
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
	form   *components.FormComponent
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
		form:   createForm(),
	}

	return v
}

// createForm initializes the form component with all required fields
func createForm() *components.FormComponent {
	fields := []components.FormField{
		{
			Name:        "title",
			Label:       "Title",
			Type:        components.TextInput,
			Placeholder: "Brief description of the task",
			Required:    true,
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        components.TextInput,
			Placeholder: "org/repo",
			Required:    true,
		},
		{
			Name:        "source",
			Label:       "Source Branch",
			Type:        components.TextInput,
			Placeholder: "main",
			Value:       "main", // Default value
			Required:    true,
		},
		{
			Name:        "target",
			Label:       "Target Branch",
			Type:        components.TextInput,
			Placeholder: "feature/new-feature",
			Required:    false,
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        components.TextArea,
			Placeholder: "Describe what you want the AI to do...",
			Required:    true,
		},
		{
			Name:        "context",
			Label:       "Additional Context",
			Type:        components.TextArea,
			Placeholder: "Any additional context or requirements...",
			Required:    false,
		},
	}

	return components.NewForm(
		components.WithFields(fields),
		components.WithFormKeymaps(components.DefaultFormKeyMap()),
	)
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

	case runCreatedMsg:
		return v.handleRunCreated(msg)

	default:
		// Delegate to form component
		newForm, cmd := v.form.Update(msg)
		v.form = newForm.(*components.FormComponent)
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
	v.form = newForm.(*components.FormComponent)

	debug.LogToFilef("📐 CREATE VIEW: Updated dimensions: terminal=%dx%d, content=%dx%d", 
		msg.Width, msg.Height, contentWidth, contentHeight)

	return v, nil
}

// handleKeyMsg processes keyboard input
func (v *CreateRunView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// Delegate to form component
	newForm, cmd := v.form.Update(msg)
	v.form = newForm.(*components.FormComponent)
	
	// Auto-save form data when values change (debounced)
	v.saveFormData()
	
	return v, cmd
}

// handleFormSubmit processes form submission
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
		RunType:        "run", // Default to run type
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
	// Dynamic help text based on form mode
	var helpText string
	if v.form.IsInsertMode() {
		helpText = "[i]edit [esc]normal [tab]next [shift+tab]prev [ctrl+s]submit"
	} else {
		helpText = "[i]edit [j/k/↑↓]navigate [b/q]back [ctrl+s]submit"
	}
	
	// Create statusline component
	statusLine := components.NewStatusLine().
		SetWidth(v.width).
		SetLeft(fmt.Sprintf("[%s]", layoutName)).
		SetRight("").
		SetHelp(helpText).
		ResetStyle().
		SetLoading(v.submitting)
	
	return statusLine.Render()
}

// saveFormData saves the current form state to cache
func (v *CreateRunView) saveFormData() {
	values := v.form.GetValues()
	
	formData := &tuicache.FormData{
		Title:      values["title"],
		Repository: values["repository"],
		Source:     values["source"],
		Target:     values["target"],
		Prompt:     values["prompt"],
		Context:    values["context"],
		RunType:    "run", // Default run type
	}
	
	v.cache.SetFormData(formData)
	debug.LogToFilef("💾 CREATE VIEW: Form data saved to cache")
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
	// Disable backspace navigation when in insert mode to prevent unwanted navigation
	if keyString == "backspace" && v.form.IsInsertMode() {
		debug.LogToFilef("🚫 CREATE VIEW: Disabling backspace navigation in insert mode")
		return true
	}
	
	// Note: esc is not disabled in insert mode - form handles it to exit insert mode
	// Only back navigation keys (b/q) are disabled in normal mode
	
	return false
}

// HandleKey implements CoreViewKeymap interface for custom key handling
func (v *CreateRunView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	keyString := keyMsg.String()
	
	// Handle arrow key navigation in normal mode (like j/k)
	if !v.form.IsInsertMode() {
		switch keyString {
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
			v.form = newForm.(*components.FormComponent)
			return true, v, cmd
		}
	}
	
	return false, v, nil
}
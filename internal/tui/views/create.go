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
}

func NewCreateRunView(client *api.Client) *CreateRunView {
	return NewCreateRunViewWithCache(client, nil, false, time.Time{}, nil)
}

func NewCreateRunViewWithCache(client *api.Client, parentRuns []models.RunResponse, parentCached bool, parentCachedAt time.Time, parentDetailsCache map[string]*models.RunResponse) *CreateRunView {
	// DEBUG: Log cache info when creating create view
	debugInfo := fmt.Sprintf("DEBUG: Creating CreateView - parentRuns=%d, parentCached=%v, detailsCache=%d\n",
		len(parentRuns), parentCached, len(parentDetailsCache))
	if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.WriteString(debugInfo)
		f.Close()
	}
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

	// Load saved form data if available
	savedData := cache.GetFormData()
	if savedData != nil {
		titleInput.SetValue(savedData.Title)
		repoInput.SetValue(savedData.Repository)
		sourceInput.SetValue(savedData.Source)
		targetInput.SetValue(savedData.Target)
		issueInput.SetValue(savedData.Issue)
		promptArea.SetValue(savedData.Prompt)
		contextArea.SetValue(savedData.Context)
	}

	return &CreateRunView{
		client: client,
		keys:   components.DefaultKeyMap,
		help:   help.New(),
		fields: []textinput.Model{
			titleInput,
			repoInput,
			sourceInput,
			targetInput,
			issueInput,
		},
		promptArea:         promptArea,
		contextArea:        contextArea,
		filePathInput:      filePathInput,
		focusIndex:         0,
		inputMode:          components.InsertMode, // Start in insert mode
		exitRequested:      false,
		parentRuns:         parentRuns,
		parentCached:       parentCached,
		parentCachedAt:     parentCachedAt,
		parentDetailsCache: parentDetailsCache,
	}
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
	return textinput.Blink
}

func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.help.Width = msg.Width
		v.promptArea.SetWidth(min(60, msg.Width-10))
		v.contextArea.SetWidth(min(60, msg.Width-10))

	case tea.KeyMsg:
		// Handle modal input logic
		switch v.inputMode {
		case components.InsertMode:
			// In insert mode, handle ESC to enter normal mode
			if msg.String() == "esc" {
				v.inputMode = components.NormalMode
				v.exitRequested = false
				// Blur current field to show we're in normal mode
				v.blurAllFields()
				return v, nil
			}

			// In insert mode, handle text input and field navigation
			switch {
			case msg.String() == "ctrl+f":
				v.useFileInput = !v.useFileInput
				if v.useFileInput {
					v.filePathInput.Focus()
				} else {
					v.fields[0].Focus()
					v.focusIndex = 0
				}
			case msg.String() == "ctrl+s" || msg.String() == "ctrl+enter":
				if !v.submitting {
					// Debug: Log when Ctrl+S is pressed
					debugInfo := "DEBUG: Ctrl+S pressed in INSERT MODE - submitting run\n"
					if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
						f.WriteString(debugInfo)
						f.Close()
					}
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

		case components.NormalMode:
			// In normal mode, handle navigation and commands
			switch {
			case key.Matches(msg, v.keys.Quit):
				return v, tea.Quit
			case key.Matches(msg, v.keys.Back) || msg.String() == "esc":
				if v.exitRequested {
					// Second ESC - actually exit
					if !v.submitting {
						// Save form data before exiting
						v.saveFormData()
						// DEBUG: Log when returning from create view
						debugInfo := "DEBUG: CreateView double ESC - returning to list view\n"
						if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
							f.WriteString(debugInfo)
							f.Close()
						}
						return NewRunListView(v.client), nil
					}
				} else {
					// First ESC in normal mode - prepare to exit
					v.exitRequested = true
				}
			case key.Matches(msg, v.keys.Help):
				v.showHelp = !v.showHelp
			case msg.String() == "i" || msg.String() == "enter":
				if v.backButtonFocused {
					// Enter pressed on back button - save form and go back
					v.saveFormData()
					debugInfo := "DEBUG: Back button pressed - returning to list view\n"
					if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
						f.WriteString(debugInfo)
						f.Close()
					}
					return NewRunListView(v.client), nil
				} else {
					// Enter insert mode and focus current field
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
					// Debug: Log when Ctrl+S is pressed
					debugInfo := "DEBUG: Ctrl+S pressed in NORMAL MODE - submitting run\n"
					if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
						f.WriteString(debugInfo)
						f.Close()
					}
					return v, v.submitRun()
				}
			case msg.String() == "ctrl+l":
				// Clear entire form
				v.clearAllFields()
			case msg.String() == "ctrl+x":
				// Clear current field/area
				v.clearCurrentField()
			default:
				// Block vim navigation keys from doing anything else
				// This prevents 'b' from accidentally triggering actions
			}
		}

	case runCreatedMsg:
		// Debug: Log when run creation response is received
		debugInfo := fmt.Sprintf("DEBUG: runCreatedMsg received - err=%v, runID='%s'\n", 
			msg.err, func() string {
				if msg.err == nil {
					return msg.run.GetIDString()
				}
				return "N/A"
			}())
		if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			f.WriteString(debugInfo)
			f.Close()
		}
		
		v.submitting = false
		if msg.err != nil {
			v.error = msg.err
		} else {
			// Clear form data on successful submission
			cache.ClearFormData()
			v.success = true
			v.createdRun = &msg.run
			return NewRunDetailsView(v.client, msg.run), nil
		}
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
				s.WriteString("Press ESC again to exit | Enter: select | j/k: navigate\n")
			} else {
				s.WriteString("NORMAL MODE | ESC: exit | Enter: select | j/k: navigate\n")
			}
		}
	}

	if v.submitting {
		s.WriteString("\n")
		s.WriteString(styles.ProcessingStyle.Render("⟳ Creating run..."))
		s.WriteString("\n")
	}

	statusBar := v.renderStatusBar()
	s.WriteString("\n")
	s.WriteString(statusBar)

	if v.showHelp {
		helpView := v.help.View(v.keys)
		s.WriteString("\n" + helpView)
	}

	return s.String()
}

func (v *CreateRunView) renderStatusBar() string {
	var statusText string
	if v.inputMode == components.InsertMode {
		statusText = "[ESC] normal mode [Tab] next [Ctrl+S] submit [Ctrl+X] clear field [Ctrl+L] clear all"
	} else {
		if v.exitRequested {
			statusText = "[ESC] exit [Enter] select [j/k] navigate [Ctrl+X] clear field [Ctrl+L] clear all"
		} else {
			statusText = "[ESC] exit [Enter] select [j/k] navigate [Ctrl+X] clear field [?] help"
		}
	}

	return styles.StatusBarStyle.Width(v.width).Render(statusText)
}

func (v *CreateRunView) submitRun() tea.Cmd {
	return func() tea.Msg {
		// Save form data before submitting in case submission fails
		v.saveFormData()

		var task models.RunRequest

		if v.useFileInput {
			filePath := v.filePathInput.Value()
			if filePath == "" {
				return runCreatedMsg{err: fmt.Errorf("file path is required")}
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return runCreatedMsg{err: fmt.Errorf("failed to read file: %w", err)}
			}

			if err := json.Unmarshal(data, &task); err != nil {
				return runCreatedMsg{err: fmt.Errorf("invalid JSON: %w", err)}
			}
		} else {
			task = models.RunRequest{
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
			debugInfo += fmt.Sprintf("DEBUG: Submit values - Title='%s', Repository='%s', Source='%s', Target='%s', Prompt='%s'\n",
				task.Title, task.Repository, task.Source, task.Target, task.Prompt)
			if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				f.WriteString(debugInfo)
				f.Close()
			}

			if task.Prompt == "" {
				return runCreatedMsg{err: fmt.Errorf("prompt is required")}
			}

			if task.Repository == "" {
				debugInfo = "DEBUG: Repository field empty, trying git auto-detect\n"
				if f, err := os.OpenFile("/tmp/repobird_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					f.WriteString(debugInfo)
					f.Close()
				}
				if repo, _, err := utils.GetGitInfo(); err == nil {
					task.Repository = repo
				}
				if task.Repository == "" {
					return runCreatedMsg{err: fmt.Errorf("repository is required")}
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

		runPtr, err := v.client.CreateRun(&task)
		if err != nil {
			return runCreatedMsg{err: err}
		}
		return runCreatedMsg{run: *runPtr, err: nil}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type runCreatedMsg struct {
	run models.RunResponse
	err error
}

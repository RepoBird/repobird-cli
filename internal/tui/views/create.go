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
				// Enter insert mode and focus current field
				v.inputMode = components.InsertMode
				v.exitRequested = false
				v.updateFocus()
			case key.Matches(msg, v.keys.Up) || msg.String() == "k":
				v.prevField()
			case key.Matches(msg, v.keys.Down) || msg.String() == "j":
				v.nextField()
			case msg.String() == "ctrl+s":
				if !v.submitting {
					return v, v.submitRun()
				}
			default:
				// Block vim navigation keys from doing anything else
				// This prevents 'b' from accidentally triggering actions
			}
		}

	case runCreatedMsg:
		v.submitting = false
		if msg.err != nil {
			v.error = msg.err
		} else {
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
	v.focusIndex++
	totalFields := len(v.fields) + 2
	if v.focusIndex >= totalFields {
		v.focusIndex = 0
	}
	v.updateFocus()
}

func (v *CreateRunView) prevField() {
	v.focusIndex--
	if v.focusIndex < 0 {
		v.focusIndex = len(v.fields) + 1
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

		// Show mode-specific instructions
		if v.inputMode == components.InsertMode {
			s.WriteString("INSERT MODE | ESC: normal mode | Tab: next field | Ctrl+S: submit\n")
		} else {
			if v.exitRequested {
				s.WriteString("Press ESC again to exit | i/Enter: edit field | j/k: navigate\n")
			} else {
				s.WriteString("NORMAL MODE | ESC: exit | i/Enter: edit field | j/k: navigate\n")
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
		statusText = "[ESC] normal mode [Tab] next field [Ctrl+S] submit [Ctrl+F] file"
	} else {
		if v.exitRequested {
			statusText = "[ESC] exit [i/Enter] edit field [j/k] navigate [Ctrl+S] submit"
		} else {
			statusText = "[ESC] exit [i/Enter] edit field [j/k] navigate [?] help"
		}
	}
	
	return styles.StatusBarStyle.Width(v.width).Render(statusText)
}

func (v *CreateRunView) submitRun() tea.Cmd {
	return func() tea.Msg {
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

			if task.Prompt == "" {
				return runCreatedMsg{err: fmt.Errorf("prompt is required")}
			}

			if task.Repository == "" {
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

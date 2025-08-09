package views

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	createdRun    *models.Run
	showHelp      bool
	useFileInput  bool
	filePathInput textinput.Model
}

func NewCreateRunView(client *api.Client) *CreateRunView {
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
		promptArea:    promptArea,
		contextArea:   contextArea,
		filePathInput: filePathInput,
		focusIndex:    0,
	}
}

func autoDetectGit(repoInput, sourceInput textinput.Model) {
	if utils.IsGitRepository() {
		repo, _ := utils.GetRepositoryInfo()
		branch, _ := utils.GetCurrentBranch()
		
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
		switch {
		case key.Matches(msg, v.keys.Quit):
			return v, tea.Quit
		case key.Matches(msg, v.keys.Back):
			if !v.submitting {
				return NewRunListView(v.client), nil
			}
		case key.Matches(msg, v.keys.Help):
			v.showHelp = !v.showHelp
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
		}

		if v.useFileInput {
			var cmd tea.Cmd
			v.filePathInput, cmd = v.filePathInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			cmds = append(cmds, v.updateFields(msg)...)
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

	totalFields := len(v.fields) + 2

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
			if i == v.focusIndex {
				s.WriteString(styles.SelectedStyle.Render("▶ "))
			} else {
				s.WriteString("  ")
			}
			s.WriteString(v.fields[i].View())
			s.WriteString("\n")
		}

		s.WriteString("\nPrompt:\n")
		if v.focusIndex == len(v.fields) {
			s.WriteString(styles.SelectedStyle.Render("▶ "))
		} else {
			s.WriteString("  ")
		}
		s.WriteString(v.promptArea.View())
		s.WriteString("\n")

		s.WriteString("\nContext (optional):\n")
		if v.focusIndex == len(v.fields)+1 {
			s.WriteString(styles.SelectedStyle.Render("▶ "))
		} else {
			s.WriteString("  ")
		}
		s.WriteString(v.contextArea.View())
		s.WriteString("\n\n")

		s.WriteString("Press Ctrl+F to load from file | Ctrl+S to submit\n")
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
	return styles.StatusBarStyle.Width(v.width).Render(
		"[Tab/Shift+Tab] navigate [Ctrl+S] submit [Ctrl+F] file [Esc] back [?] help",
	)
}

func (v *CreateRunView) submitRun() tea.Cmd {
	return func() tea.Msg {
		var task models.TaskRequest

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
			task = models.TaskRequest{
				Title:        v.fields[0].Value(),
				Repository:   v.fields[1].Value(),
				SourceBranch: v.fields[2].Value(),
				TargetBranch: v.fields[3].Value(),
				Issue:        v.fields[4].Value(),
				Prompt:       v.promptArea.Value(),
				Context:      v.contextArea.Value(),
				RunType:      "run",
			}

			if task.Prompt == "" {
				return runCreatedMsg{err: fmt.Errorf("prompt is required")}
			}

			if task.Repository == "" {
				if utils.IsGitRepository() {
					repo, _ := utils.GetRepositoryInfo()
					task.Repository = repo
				}
				if task.Repository == "" {
					return runCreatedMsg{err: fmt.Errorf("repository is required")}
				}
			}

			if task.SourceBranch == "" {
				if utils.IsGitRepository() {
					branch, _ := utils.GetCurrentBranch()
					task.SourceBranch = branch
				}
				if task.SourceBranch == "" {
					task.SourceBranch = "main"
				}
			}

			if task.TargetBranch == "" {
				task.TargetBranch = fmt.Sprintf("repobird/%d", time.Now().Unix())
			}
		}

		run, err := v.client.CreateRun(task)
		return runCreatedMsg{run: run, err: err}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type runCreatedMsg struct {
	run models.Run
	err error
}
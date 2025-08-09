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
		switch {
		case key.Matches(msg, v.keys.Quit):
			return v, tea.Quit
		case key.Matches(msg, v.keys.Back):
			if !v.submitting {
				return NewRunListViewWithCache(v.client, v.parentRuns, v.parentCached, v.parentCachedAt, v.parentDetailsCache), nil
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
			return NewRunDetailsViewWithCache(v.client, msg.run, v.parentRuns, v.parentCached, v.parentCachedAt, v.parentDetailsCache), nil
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

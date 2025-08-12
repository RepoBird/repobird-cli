package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

type CreateRunView struct {
	client APIClient
	width  int
	height int
}

func NewCreateRunView(client APIClient) *CreateRunView {
	return &CreateRunView{client: client}
}

func (v *CreateRunView) Init() tea.Cmd {
	return nil
}

func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

func (v *CreateRunView) View() string {
	if v.width <= 0 || v.height <= 0 {
		return "⟳ Initializing..."
	}
	return "Create Run View (minimal) - Ready!"
}

// Minimal backward compatibility functions
func NewCreateRunViewWithCache(
	client APIClient,
	parentRuns []models.RunResponse,
	parentCached bool,
	parentCachedAt interface{},
	parentDetailsCache interface{},
	embeddedCache interface{},
) *CreateRunView {
	return NewCreateRunView(client)
}
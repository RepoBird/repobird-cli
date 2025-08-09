package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/tui/views"
)

type App struct {
	client *api.Client
}

func NewApp(client *api.Client) *App {
	return &App{
		client: client,
	}
}

func (a *App) Run() error {
	listView := views.NewRunListView(a.client)
	p := tea.NewProgram(listView, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
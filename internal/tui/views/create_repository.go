package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/components"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
)

// Repository selection and FZF integration
func (m *CreateRunView) selectRepository() tea.Cmd {
	if m.repoSelector == nil {
		return nil
	}
	return m.repoSelector.Show()
}

func (m *CreateRunView) autofillRepository() {
	if len(m.fields) > 0 {
		repoPath, err := pkgutils.GetCurrentRepository()
		if err == nil {
			m.fields[0].SetValue(repoPath)
		}
	}
}

func (m *CreateRunView) handleRepositorySelected(msg repositorySelectedMsg) (tea.Model, tea.Cmd) {
	if len(m.fields) > 0 {
		m.fields[0].SetValue(msg.repository)
	}
	return m, nil
}

// FZF mode operations
func (m *CreateRunView) activateFZFMode() {
	if m.fzfMode == nil {
		return
	}
	
	// Get current repository value for pre-selection
	currentRepo := ""
	if len(m.fields) > 0 {
		currentRepo = m.fields[0].Value()
	}
	
	// Initialize FZF mode with repository data
	options := []components.FZFOption{}
	
	// Add current git repository if available
	if repoPath, err := pkgutils.GetCurrentRepository(); err == nil {
		options = append(options, components.FZFOption{
			Value:   repoPath,
			Display: "📁 " + repoPath + " (current)",
		})
	}
	
	// Add repository history from cache if available
	if m.cache != nil {
		// TODO: Add repository history from cache
	}
	
	m.fzfMode.SetOptions(options)
	m.fzfMode.SetCurrentValue(currentRepo)
	m.fzfActive = true
}
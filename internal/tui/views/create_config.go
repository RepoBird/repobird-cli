package views

import (
	"encoding/json"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

// Configuration file operations and form data persistence

func (m *CreateRunView) loadConfigFromFile(filePath string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return configLoadErrorMsg{err: err, filePath: filePath}
		}

		var config models.RunRequest
		if err := json.Unmarshal(data, &config); err != nil {
			return configLoadErrorMsg{err: err, filePath: filePath}
		}

		return configLoadedMsg{config: &config, filePath: filePath}
	}
}

func (m *CreateRunView) populateFormFromConfig(config *models.RunRequest, filePath string) {
	if config == nil {
		return
	}

	// Populate form fields
	if len(m.fields) >= 4 {
		m.fields[0].SetValue(config.Repository)
		m.fields[1].SetValue(config.Source)
		m.fields[2].SetValue(config.Target)
		m.fields[3].SetValue(config.Title)
	}

	// Populate text areas
	m.promptArea.SetValue(config.Prompt)
	m.contextArea.SetValue(config.Context)

	// Set run type
	m.runType = config.RunType

	// Update last loaded file
	m.lastLoadedFile = filePath
}

func (m *CreateRunView) activateConfigFileSelector() tea.Cmd {
	if m.configFileSelector == nil {
		return nil
	}
	
	m.configFileSelectorActive = true
	return m.configFileSelector.Init()
}

// Form data persistence
func (m *CreateRunView) loadFormData() {
	if m.cache == nil {
		return
	}
	
	// Load cached form data if available
	// TODO: Implement form data loading from cache
}

func (m *CreateRunView) saveFormData() {
	if m.cache == nil {
		return
	}
	
	// Save current form data to cache
	// TODO: Implement form data saving to cache
}
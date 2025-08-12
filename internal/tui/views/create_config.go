package views

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// Configuration file operations and form data persistence

func (v *CreateRunView) loadConfigFromFile(filePath string) tea.Cmd {
	return func() tea.Msg {
		debug.LogToFilef("DEBUG: Loading config from file: %s\n", filePath)

		// Load the config
		config, err := v.configLoader.LoadConfig(filePath)
		if err != nil {
			debug.LogToFilef("DEBUG: Failed to load config: %v\n", err)
			return configLoadErrorMsg{err: err}
		}

		// Calculate file hash for duplicate detection
		fileHash, hashErr := cache.CalculateFileHashFromPath(filePath)
		if hashErr != nil {
			debug.LogToFilef("DEBUG: Failed to calculate file hash: %v\n", hashErr)
			// Continue without hash - not a critical error
			fileHash = ""
		}

		debug.LogToFilef("DEBUG: Config loaded successfully from %s with hash %s\n", filePath, fileHash)
		return configLoadedMsg{
			config:   config,
			filePath: filePath,
			fileHash: fileHash,
		}
	}
}

func (v *CreateRunView) populateFormFromConfig(config *models.RunRequest, filePath string) {
	debug.LogToFilef("DEBUG: populateFormFromConfig START - config.Repository=%s, config.Source=%s, config.Target=%s, config.Title=%s, config.Prompt=%d chars\n",
		config.Repository, config.Source, config.Target, config.Title, len(config.Prompt))

	// Store the loaded file path
	v.lastLoadedFile = filepath.Base(filePath)

	// Keep focus on Load Config field (index 0) - no need to update
	// This prevents any layout shifts

	// Populate form fields
	if config.Repository != "" {
		v.fields[0].SetValue(config.Repository) // Repository field
		debug.LogToFilef("DEBUG: Set fields[0] to %s, actual value: %s\n", config.Repository, v.fields[0].Value())
	}

	if config.Prompt != "" {
		v.promptArea.SetValue(config.Prompt)
		debug.LogToFilef("DEBUG: Set promptArea to %d chars, actual value: %d chars\n", len(config.Prompt), len(v.promptArea.Value()))
		// Don't change collapsed state when loading config
		// This prevents layout shifts that hide the top rows
	}

	if config.Source != "" {
		v.fields[1].SetValue(config.Source) // Source field
		debug.LogToFilef("DEBUG: Set fields[1] to %s, actual value: %s\n", config.Source, v.fields[1].Value())
	}

	if config.Target != "" {
		v.fields[2].SetValue(config.Target) // Target field
		debug.LogToFilef("DEBUG: Set fields[2] to %s, actual value: %s\n", config.Target, v.fields[2].Value())
	}

	if config.Title != "" {
		v.fields[3].SetValue(config.Title) // Title field
		debug.LogToFilef("DEBUG: Set fields[3] to %s, actual value: %s\n", config.Title, v.fields[3].Value())
	}

	if config.Context != "" {
		v.contextArea.SetValue(config.Context)
		v.showContext = true // Show context field if it has content
		debug.LogToFilef("DEBUG: Set contextArea to %d chars, showContext=%v\n", len(config.Context), v.showContext)
	}

	// Set run type
	if config.RunType != "" {
		v.runType = config.RunType
		debug.LogToFilef("DEBUG: Set runType to %s\n", v.runType)
	}

	debug.LogToFilef("DEBUG: populateFormFromConfig END - fields[0]=%s, fields[1]=%s, fields[2]=%s, fields[3]=%s\n",
		v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value())
}

func (v *CreateRunView) activateConfigFileSelector() tea.Cmd {
	return func() tea.Msg {
		// Set config file selector dimensions
		v.configFileSelector.SetDimensions(v.width, v.height)

		// Activate the enhanced config file selector with preview
		if err := v.configFileSelector.Activate(); err != nil {
			debug.LogToFilef("DEBUG: Failed to activate config file selector: %v\n", err)
			return configLoadErrorMsg{err: fmt.Errorf("failed to show file selector: %w", err)}
		}

		debug.LogToFile("DEBUG: Config file selector with preview activated\n")
		return fileSelectorActivatedMsg{}
	}
}

// Form data persistence
func (v *CreateRunView) loadFormData() {
	savedData := v.cache.GetFormData()
	if savedData != nil && len(v.fields) >= 5 {
		debug.LogToFilef("DEBUG: loadFormData START - Repository: %s, Prompt: %d chars, Source: %s, Target: %s, Title: %s\n",
			savedData.Repository, len(savedData.Prompt), savedData.Source, savedData.Target, savedData.Title)

		v.fields[0].SetValue(savedData.Repository)
		v.fields[1].SetValue(savedData.Source)
		v.fields[2].SetValue(savedData.Target)
		v.fields[3].SetValue(savedData.Title)
		v.fields[4].SetValue(savedData.Issue)
		v.promptArea.SetValue(savedData.Prompt)
		v.contextArea.SetValue(savedData.Context)
		if savedData.Context != "" {
			v.showContext = true
		}
		if savedData.RunType != "" {
			v.runType = models.RunType(savedData.RunType)
		}

		debug.LogToFilef("DEBUG: loadFormData AFTER SET - fields[0]=%s, fields[1]=%s, fields[2]=%s, fields[3]=%s, promptArea=%d chars\n",
			v.fields[0].Value(), v.fields[1].Value(), v.fields[2].Value(), v.fields[3].Value(), len(v.promptArea.Value()))
	} else {
		if savedData == nil {
			debug.LogToFile("DEBUG: loadFormData - savedData is nil!\n")
		} else {
			debug.LogToFilef("DEBUG: loadFormData - fields not initialized, len(v.fields)=%d\n", len(v.fields))
		}
	}
}

func (v *CreateRunView) saveFormData() {
	formData := &cache.FormData{
		Repository: v.fields[0].Value(),
		Source:     v.fields[1].Value(),
		Target:     v.fields[2].Value(),
		Title:      v.fields[3].Value(),
		Issue:      v.fields[4].Value(),
		Prompt:     v.promptArea.Value(),
		Context:    v.contextArea.Value(),
		RunType:    string(v.runType),
	}

	// Debug logging to verify what we're saving
	debug.LogToFilef("DEBUG: Saving form data - Repository: %s, Prompt: %d chars, Source: %s, Target: %s, Title: %s\n",
		formData.Repository, len(formData.Prompt), formData.Source, formData.Target, formData.Title)

	v.cache.SetFormData(formData)
}
package views

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// activateConfigFileSelector activates the file selector for loading config files
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

// loadConfigFromFile loads a configuration file and populates the form
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
			config: config,
			path:   filePath,
		}
	}
}

// populateFormFromConfig populates the form fields from a loaded config
func (v *CreateRunView) populateFormFromConfig(config *models.RunRequest, filePath string) {
	debug.LogToFilef("DEBUG: Populating form from config file: %s\n", filePath)

	// Set the loaded file path
	v.lastLoadedFile = filePath

	// Populate fields from config
	if config.Repository != "" {
		v.fields[0].SetValue(config.Repository)
	}
	if config.Source != "" {
		v.fields[1].SetValue(config.Source)
	}
	if config.Target != "" {
		v.fields[2].SetValue(config.Target)
	}
	if config.Title != "" {
		v.fields[3].SetValue(config.Title)
	}
	if len(config.Files) > 0 {
		v.fields[4].SetValue(strings.Join(config.Files, ","))
	}
	if config.Prompt != "" {
		v.promptArea.SetValue(config.Prompt)
	}
	if config.Context != "" {
		v.contextArea.SetValue(config.Context)
		v.showContext = true // Auto-show context field if it has content
	}
	if config.RunType != "" {
		v.runType = config.RunType
	}

	debug.LogToFile("DEBUG: Form populated from config\n")
}

// loadFormData loads saved form data from cache
func (v *CreateRunView) loadFormData() {
	data := v.cache.GetFormData()
	if data == nil {
		debug.LogToFile("DEBUG: No cached form data found\n")
		return
	}

	debug.LogToFile("DEBUG: Loading form data from cache\n")

	// Restore field values from old cache format
	if data.Repository != "" && len(v.fields) > 0 {
		v.fields[0].SetValue(data.Repository)
	}
	if data.Source != "" && len(v.fields) > 1 {
		v.fields[1].SetValue(data.Source)
	}
	if data.Target != "" && len(v.fields) > 2 {
		v.fields[2].SetValue(data.Target)
	}
	if data.Title != "" && len(v.fields) > 3 {
		v.fields[3].SetValue(data.Title)
	}
	if data.Issue != "" && len(v.fields) > 4 {
		v.fields[4].SetValue(data.Issue)
	}

	v.promptArea.SetValue(data.Prompt)
	v.contextArea.SetValue(data.Context)
	v.showContext = data.ShowContext
	if data.RunType != "" {
		v.runType = models.RunType(data.RunType)
	}

	debug.LogToFilef("DEBUG: Form data restored - Repository: %s, RunType: %s\n",
		func() string {
			if len(v.fields) > 0 {
				return v.fields[0].Value()
			}
			return ""
		}(),
		v.runType,
	)
}

// saveFormData saves the current form data to cache
func (v *CreateRunView) saveFormData() {
	// Don't save if we're in error state or submitting
	if v.error != nil || v.submitting || v.isSubmitting {
		return
	}

	data := &cache.FormData{
		Prompt:      v.promptArea.Value(),
		Context:     v.contextArea.Value(),
		ShowContext: v.showContext,
		RunType:     string(v.runType),
	}

	// Set individual field values
	if len(v.fields) > 0 {
		data.Repository = v.fields[0].Value()
	}
	if len(v.fields) > 1 {
		data.Source = v.fields[1].Value()
	}
	if len(v.fields) > 2 {
		data.Target = v.fields[2].Value()
	}
	if len(v.fields) > 3 {
		data.Title = v.fields[3].Value()
	}
	if len(v.fields) > 4 {
		data.Issue = v.fields[4].Value()
	}

	v.cache.SetFormData(data)
	debug.LogToFile("DEBUG: Form data saved to cache\n")
}

// loadFileHashCache loads the file hash cache for duplicate detection
func (v *CreateRunView) loadFileHashCache() tea.Cmd {
	return func() tea.Msg {
		// For now, return empty cache as GetFileHashes doesn't exist
		hashes := make(map[string]string)
		debug.LogToFilef("DEBUG: Loaded %d file hashes from cache\n", len(hashes))
		return fileHashCacheLoadedMsg{cache: hashes}
	}
}

// validateForm checks if the form is valid and returns any validation errors
func (v *CreateRunView) validateForm() (bool, string) {
	// Check required fields - only prompt and repository are required
	if strings.TrimSpace(v.promptArea.Value()) == "" {
		return false, "Prompt is required"
	}

	if strings.TrimSpace(v.fields[0].Value()) == "" {
		return false, "Repository is required"
	}

	// Check for duplicate file hash if a config file was loaded
	if v.isDuplicateRun && v.currentFileHash != "" {
		// This is a duplicate run, but not necessarily an error
		// The user will be prompted to confirm
		debug.LogToFile("DEBUG: Duplicate run detected - will prompt for confirmation\n")
	}

	return true, ""
}

// prepareTaskFromFile prepares a task from a config file
func (v *CreateRunView) prepareTaskFromFile(filePath string) (models.RunRequest, error) {
	debug.LogToFilef("DEBUG: Preparing task from file: %s\n", filePath)

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return models.RunRequest{}, fmt.Errorf("failed to read file: %w", err)
	}

	var task models.RunRequest
	if err := json.Unmarshal(data, &task); err != nil {
		return models.RunRequest{}, fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate required fields
	if task.Prompt == "" {
		return models.RunRequest{}, fmt.Errorf("prompt is required")
	}
	if task.Repository == "" {
		return models.RunRequest{}, fmt.Errorf("repository is required")
	}

	// Set default run type if not specified
	if task.RunType == "" {
		task.RunType = models.RunTypeRun
	}

	// Add to repository history if not empty
	if task.Repository != "" {
		go func() {
			v.cache.AddRepositoryToHistory(task.Repository)
		}()
	}

	return task, nil
}

// prepareTaskFromForm prepares a task from the form fields
func (v *CreateRunView) prepareTaskFromForm() models.RunRequest {
	// Parse files input (comma-separated)
	filesStr := strings.TrimSpace(v.fields[4].Value())
	var files []string
	if filesStr != "" {
		for _, f := range strings.Split(filesStr, ",") {
			trimmed := strings.TrimSpace(f)
			if trimmed != "" {
				files = append(files, trimmed)
			}
		}
	}

	task := models.RunRequest{
		Repository: strings.TrimSpace(v.fields[0].Value()),
		Source:     strings.TrimSpace(v.fields[1].Value()),
		Target:     strings.TrimSpace(v.fields[2].Value()),
		Title:      strings.TrimSpace(v.fields[3].Value()),
		Prompt:     strings.TrimSpace(v.promptArea.Value()),
		Context:    strings.TrimSpace(v.contextArea.Value()),
		Files:      files,
		RunType:    v.runType,
	}

	// Set defaults if empty
	if task.Source == "" {
		task.Source = "main"
	}
	if task.Target == "" {
		// Generate a target branch name based on the title or prompt
		if task.Title != "" {
			task.Target = generateBranchName(task.Title)
		} else {
			task.Target = generateBranchName(task.Prompt)
		}
	}
	if task.RunType == "" {
		task.RunType = models.RunTypeRun
	}

	return task
}

// generateBranchName generates a branch name from a string
func generateBranchName(input string) string {
	// Take first 50 characters
	if len(input) > 50 {
		input = input[:50]
	}

	// Convert to lowercase and replace non-alphanumeric with hyphens
	result := strings.ToLower(input)
	result = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, result)

	// Remove leading/trailing hyphens and collapse multiple hyphens
	result = strings.Trim(result, "-")
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Add prefix
	result = "repobird/" + result

	return result
}

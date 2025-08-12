package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
)

// Task preparation, validation, and API submission

func (m *CreateRunView) submitRun() tea.Cmd {
	return func() tea.Msg {
		task, err := m.prepareTask()
		if err != nil {
			return err
		}

		run, err := m.submitToAPI(task)
		if err != nil {
			return err
		}

		return runCreatedMsg{run: run}
	}
}

func (m *CreateRunView) submitWithForce() tea.Cmd {
	return func() tea.Msg {
		task, err := m.prepareTask()
		if err != nil {
			return err
		}

		run, err := m.submitToAPIWithForce(task)
		if err != nil {
			return err
		}

		return runCreatedMsg{run: run}
	}
}

func (m *CreateRunView) submitToAPI(task models.RunRequest) (models.RunResponse, error) {
	if m.client == nil {
		return models.RunResponse{}, models.NewError("API client not available", models.ErrorTypeClient)
	}
	
	return m.client.CreateRun(task)
}

func (m *CreateRunView) submitToAPIWithForce(task models.RunRequest) (models.RunResponse, error) {
	if m.client == nil {
		return models.RunResponse{}, models.NewError("API client not available", models.ErrorTypeClient)
	}
	
	return m.client.CreateRunWithForce(task)
}

func (m *CreateRunView) prepareTask() (models.RunRequest, error) {
	task := m.prepareTaskFromForm()
	
	// Auto-detect git info
	m.autoDetectGitInfo(&task)
	
	// Validate the task
	if err := m.validateTask(&task); err != nil {
		return task, err
	}
	
	return task, nil
}

func (m *CreateRunView) prepareTaskFromForm() models.RunRequest {
	task := models.RunRequest{
		RunType: m.runType,
		Prompt:  m.promptArea.Value(),
		Context: m.contextArea.Value(),
	}

	// Populate from form fields
	if len(m.fields) >= 4 {
		task.Repository = m.fields[0].Value()
		task.Source = m.fields[1].Value()
		task.Target = m.fields[2].Value()
		task.Title = m.fields[3].Value()
	}

	return task
}

func (m *CreateRunView) prepareTaskFromFile(filePath string) (models.RunRequest, error) {
	// TODO: Implement file-based task preparation
	return models.RunRequest{}, nil
}

func (m *CreateRunView) validateTask(task *models.RunRequest) error {
	if task.Repository == "" {
		return models.NewError("Repository is required", models.ErrorTypeValidation)
	}
	
	if task.Prompt == "" {
		return models.NewError("Prompt is required", models.ErrorTypeValidation)
	}
	
	if task.Source == "" {
		return models.NewError("Source branch is required", models.ErrorTypeValidation)
	}
	
	if task.Target == "" {
		return models.NewError("Target branch is required", models.ErrorTypeValidation)
	}
	
	return nil
}

func (m *CreateRunView) validateForm() (bool, string) {
	if len(m.fields) < 4 {
		return false, "Form not properly initialized"
	}

	if m.fields[0].Value() == "" {
		return false, "Repository is required"
	}

	if m.promptArea.Value() == "" {
		return false, "Prompt is required"
	}

	if m.fields[1].Value() == "" {
		return false, "Source branch is required"
	}

	if m.fields[2].Value() == "" {
		return false, "Target branch is required"
	}

	return true, ""
}

func (m *CreateRunView) autoDetectGitInfo(task *models.RunRequest) {
	// Try to auto-detect git information if not provided
	if task.Repository == "" {
		if repo, err := pkgutils.GetCurrentRepository(); err == nil {
			task.Repository = repo
		}
	}
	
	if task.Source == "" {
		if branch, err := pkgutils.GetCurrentBranch(); err == nil {
			task.Source = branch
		}
	}
}

func (m *CreateRunView) loadFileHashCache() tea.Cmd {
	// TODO: Implement file hash cache loading for duplicate detection
	return nil
}

func (m *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
	m.success = true
	m.createdRun = &msg.run
	m.submitting = false
	m.isSubmitting = false
	
	// Update cache if available
	if m.cache != nil {
		// TODO: Cache the created run
	}
	
	return m, startClearStatusTimer()
}
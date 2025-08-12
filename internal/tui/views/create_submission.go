package views

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
)

// submitRun handles the run submission process
func (v *CreateRunView) submitRun() tea.Cmd {
	// Set submitting state immediately
	v.isSubmitting = true
	v.submitStartTime = time.Now()

	return func() tea.Msg {
		debug.LogToFile("DEBUG: submitRun() called - starting submission process\n")

		// Save form data before submitting in case submission fails
		v.saveFormData()

		task, err := v.prepareTask()
		if err != nil {
			return runCreatedMsg{err: err}
		}

		run, err := v.submitToAPI(task)
		if err != nil {
			return runCreatedMsg{err: err}
		}

		return runCreatedMsg{run: run, err: nil}
	}
}

// submitWithForce submits the run with force flag to override duplicate detection
func (v *CreateRunView) submitWithForce() tea.Cmd {
	// Set submitting state immediately
	v.isSubmitting = true
	v.submitStartTime = time.Now()

	return func() tea.Msg {
		debug.LogToFile("DEBUG: submitWithForce() called - retrying submission with force override\n")

		task, err := v.prepareTask()
		if err != nil {
			return runCreatedMsg{err: err}
		}

		// Submit to API with force flag
		run, err := v.submitToAPIWithForce(task)
		if err != nil {
			return runCreatedMsg{err: err}
		}

		return runCreatedMsg{run: run, err: nil}
	}
}

// prepareTask prepares a task from either file or form input
func (v *CreateRunView) prepareTask() (models.RunRequest, error) {
	var task models.RunRequest
	var err error

	if v.useFileInput {
		task, err = v.prepareTaskFromFile(v.filePathInput.Value())
		if err != nil {
			return task, err
		}
	} else {
		task = v.prepareTaskFromForm()
		v.autoDetectGitInfo(&task)

		if err := v.validateTask(&task); err != nil {
			return task, err
		}

		// Generate file hash for form-based submission if not already set
		if v.currentFileHash == "" {
			// Create a deterministic hash from the task content
			config := &models.RunConfig{
				Prompt:     task.Prompt,
				Repository: task.Repository,
				Source:     task.Source,
				Target:     task.Target,
				RunType:    string(task.RunType),
				Title:      task.Title,
				Context:    task.Context,
			}
			if hash, err := cache.CalculateConfigHash(config); err == nil && hash != "" {
				v.currentFileHash = hash
				debug.LogToFilef("DEBUG: Generated file hash for form-based submission: %s\n", hash)
			}
		}

		// Add repository to history after successful validation
		if task.Repository != "" {
			go func() {
				v.cache.AddRepositoryToHistory(task.Repository)
			}()
		}
	}

	return task, nil
}

// validateTask validates required fields in the task
func (v *CreateRunView) validateTask(task *models.RunRequest) error {
	if task.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if task.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	return nil
}

// autoDetectGitInfo fills in missing repository and branch information from git
func (v *CreateRunView) autoDetectGitInfo(task *models.RunRequest) {
	if task.Repository == "" {
		debug.LogToFile("DEBUG: Repository field empty, trying git auto-detect\n")
		if repo, _, err := pkgutils.GetGitInfo(); err == nil {
			task.Repository = repo
		}
	}

	if task.Source == "" {
		if _, branch, err := pkgutils.GetGitInfo(); err == nil {
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

// submitToAPI converts the task to API format and submits it
func (v *CreateRunView) submitToAPI(task models.RunRequest) (models.RunResponse, error) {
	// Convert to API-compatible format
	apiTask := task.ToAPIRequest()

	// Add file hash if we have one (from loaded config file)
	if v.currentFileHash != "" {
		apiTask.FileHash = v.currentFileHash
		debug.LogToFilef("DEBUG: Including file hash in API request: %s\n", v.currentFileHash)
	}

	// Debug: Log the final task object being sent to API
	debug.LogToFilef(
		"DEBUG: Final API task object - Title='%s', RepositoryName='%s', SourceBranch='%s', "+
			"TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s', FileHash='%s'\n",
		apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch,
		apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType, apiTask.FileHash)

	runPtr, err := v.client.CreateRunAPI(apiTask)

	// Debug: Log the API response
	debug.LogToFilef("DEBUG: API response - err=%v, runPtr!=nil=%v\n", err, runPtr != nil)

	if err != nil {
		return models.RunResponse{}, err
	}
	if runPtr == nil {
		return models.RunResponse{}, fmt.Errorf("API returned nil response")
	}

	return *runPtr, nil
}

// submitToAPIWithForce submits to API with the force flag to override duplicates
func (v *CreateRunView) submitToAPIWithForce(task models.RunRequest) (models.RunResponse, error) {
	// Convert to API-compatible format
	apiTask := task.ToAPIRequest()

	// Add file hash if we have one (from loaded config file)
	if v.currentFileHash != "" {
		apiTask.FileHash = v.currentFileHash
		debug.LogToFilef("DEBUG: Including file hash in API request with force override: %s\n", v.currentFileHash)
	}

	// Set force flag to override duplicate detection
	apiTask.Force = true

	// Debug: Log the final task object being sent to API
	debug.LogToFilef(
		"DEBUG: Final API task object WITH FORCE - Title='%s', RepositoryName='%s', SourceBranch='%s', "+
			"TargetBranch='%s', Prompt='%s', Context='%s', RunType='%s', FileHash='%s', Force=%v\n",
		apiTask.Title, apiTask.RepositoryName, apiTask.SourceBranch,
		apiTask.TargetBranch, apiTask.Prompt, apiTask.Context, apiTask.RunType, apiTask.FileHash, apiTask.Force)

	runPtr, err := v.client.CreateRunAPI(apiTask)

	// Debug: Log the API response
	debug.LogToFilef("DEBUG: API response with force - err=%v, runPtr!=nil=%v\n", err, runPtr != nil)

	if err != nil {
		return models.RunResponse{}, err
	}
	if runPtr == nil {
		return models.RunResponse{}, fmt.Errorf("API returned nil response")
	}

	return *runPtr, nil
}

// handleRunCreated handles the runCreatedMsg message
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
	debug.LogToFilef("DEBUG: runCreatedMsg received - err=%v, runID='%s'\n",
		msg.err, func() string {
			if msg.err == nil {
				return msg.run.GetIDString()
			}
			return "N/A"
		}())

	v.submitting = false
	v.isSubmitting = false // Reset our new submitting state
	if msg.err != nil {
		// Check if this is a duplicate run error
		errorMsg := msg.err.Error()
		if strings.Contains(errorMsg, "Duplicate run detected") && strings.Contains(errorMsg, "Use --force to override") {
			// Extract the run ID from the error message
			// Pattern: "Duplicate run detected: A run with this file hash already exists (ID: 955). Use --force to override."
			re := regexp.MustCompile(`\(ID: (\d+)\)`)
			matches := re.FindStringSubmatch(errorMsg)
			if len(matches) > 1 {
				v.duplicateRunID = matches[1]
				v.isDuplicateConfirm = true
				debug.LogToFilef("DEBUG: Duplicate run detected - entering confirmation mode for run ID %s\n", v.duplicateRunID)
				return v, nil
			}
		}

		// Regular error handling for non-duplicate errors
		v.error = msg.err
		v.initErrorFocus()
		return v, nil
	}

	// Check if the run has a valid ID
	runID := msg.run.GetIDString()
	if runID == "" {
		v.error = fmt.Errorf("run created but received invalid ID from server")
		v.initErrorFocus()
		debug.LogToFile("DEBUG: Run created successfully but runID is empty, not navigating to details\n")
		return v, nil
	}

	// Clear form data on successful submission
	v.cache.SetFormData(nil)
	v.success = true
	v.createdRun = &msg.run

	// Add the file hash to cache if we have one
	if v.currentFileHash != "" {
		v.cache.SetFileHash(v.lastLoadedFile, v.currentFileHash)
		debug.LogToFilef("DEBUG: Added file hash %s to cache after successful submission\n", v.currentFileHash)
	}

	debug.LogToFilef("DEBUG: Run created successfully with ID='%s', navigating to details\n", runID)

	// Navigate to details view
	return v, func() tea.Msg {
		return messages.NavigateToDetailsMsg{
			RunID:      msg.run.GetIDString(),
			FromCreate: true,
		}
	}
}

// Message for run creation result
type runCreatedMsg struct {
	run models.RunResponse
	err error
}

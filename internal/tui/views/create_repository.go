package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
)

// Repository selection and FZF integration
func (v *CreateRunView) selectRepository() tea.Cmd {
	return func() tea.Msg {
		// Suspend Bubble Tea temporarily and show fzf selector
		selectedRepo, err := v.repoSelector.SelectRepository()
		if err != nil {
			debug.LogToFilef("DEBUG: Repository selection failed: %v\n", err)
			return repositorySelectedMsg{repository: "", err: err}
		}

		debug.LogToFilef("DEBUG: Repository selected: %s\n", selectedRepo)
		return repositorySelectedMsg{repository: selectedRepo, err: nil}
	}
}

func (v *CreateRunView) autofillRepository() {
	// Only autofill if the repository field is empty (now at index 0)
	if len(v.fields) >= 2 {
		// Auto-detect from git if fields are empty
		if repo, branch, err := pkgutils.GetGitInfo(); err == nil {
			if v.fields[0].Value() == "" && repo != "" {
				v.fields[0].SetValue(repo)
				debug.LogToFilef("DEBUG: Auto-filled repository from git: %s\n", repo)
			}
			if v.fields[1].Value() == "" && branch != "" {
				v.fields[1].SetValue(branch)
				debug.LogToFilef("DEBUG: Auto-filled source branch from git: %s\n", branch)
			}
		} else if v.fields[0].Value() == "" {
			// Fallback to repository selector if git detection fails
			defaultRepo := v.repoSelector.GetDefaultRepository()
			if defaultRepo != "" {
				v.fields[0].SetValue(defaultRepo)
				debug.LogToFilef("DEBUG: Auto-filled repository from selector: %s\n", defaultRepo)
			}
		}
	}
}

func (v *CreateRunView) handleRepositorySelected(msg repositorySelectedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		debug.LogToFilef("DEBUG: Repository selection error: %v\n", msg.err)
		v.error = msg.err
		v.initErrorFocus()
		return v, nil
	}

	// Set the selected repository in the repository field (now at index 0)
	if len(v.fields) >= 1 && msg.repository != "" {
		v.fields[0].SetValue(msg.repository)
		debug.LogToFilef("DEBUG: Repository field updated to: %s\n", msg.repository)

		// Add to manual repository list for future use
		v.repoSelector.AddManualRepository(msg.repository)
	}

	return v, nil
}

// FZF mode operations
func (v *CreateRunView) activateFZFMode() {
	// Build list of repositories
	var items []string

	// Add current git repository if available
	if gitRepo, _, err := pkgutils.GetGitInfo(); err == nil && gitRepo != "" {
		items = append(items, fmt.Sprintf("📁 %s", gitRepo))
	}

	// Add repositories from history
	if history, err := v.cache.GetRepositoryHistory(); err == nil {
		for _, repoName := range history {
			if repoName != "" {
				// Skip if already added (git repo)
				skip := false
				for _, item := range items {
					if strings.Contains(item, repoName) {
						skip = true
						break
					}
				}
				if !skip {
					items = append(items, fmt.Sprintf("🔄 %s", repoName))
				}
			}
		}
	}

	// Add current value if not empty and not in list
	currentValue := v.fields[0].Value()
	if currentValue != "" {
		skip := false
		for _, item := range items {
			if strings.Contains(item, currentValue) {
				skip = true
				break
			}
		}
		if !skip {
			items = append([]string{fmt.Sprintf("✏️ %s", currentValue)}, items...)
		}
	}

	// Add example if no items
	if len(items) == 0 {
		items = []string{"📝 owner/repo"}
	}

	// Create FZF mode
	fieldWidth := 50 // Default width for repository field
	v.fzfMode = components.NewFZFMode(items, fieldWidth, 10)
	v.fzfMode.Activate()
	v.fzfActive = true
}
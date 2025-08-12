package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
)

// selectRepository opens the repository selector
func (v *CreateRunView) selectRepository() tea.Cmd {
	return func() tea.Msg {
		// Use the repository selector component
		if v.repoSelector != nil {
			repo, err := v.repoSelector.SelectRepository()
			if err != nil {
				// User cancelled or error occurred
				debug.LogToFilef("DEBUG: Repository selection cancelled or failed: %v\n", err)
				return nil
			}
			return repositorySelectedMsg{repository: repo}
		}
		return nil
	}
}

// autofillRepository tries to automatically fill the repository field from git
func (v *CreateRunView) autofillRepository() {
	// Only autofill if repository field is empty
	if v.fields[0].Value() == "" {
		// Try to get git repository info
		if gitRepo, _, err := pkgutils.GetGitInfo(); err == nil && gitRepo != "" {
			v.fields[0].SetValue(gitRepo)
			debug.LogToFilef("DEBUG: Auto-filled repository from git: %s\n", gitRepo)
		}
	}
}

// handleRepositorySelected handles the repository selection result
func (v *CreateRunView) handleRepositorySelected(msg repositorySelectedMsg) (tea.Model, tea.Cmd) {
	if msg.repository != "" {
		v.fields[0].SetValue(msg.repository)
		// Add to history for future use
		go func() {
			v.cache.AddRepositoryToHistory(msg.repository)
		}()
	}
	return v, nil
}

// activateFZFMode activates FZF mode for repository selection
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

package components

import (
	"fmt"
	"os"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/pkg/utils"
	"golang.org/x/term"
)

// Repository represents a repository option in the selector
type Repository struct {
	Name        string
	Description string
}

// RepositorySelector provides fuzzy finding functionality for repository selection
type RepositorySelector struct {
	repositories []Repository
}

// NewRepositorySelector creates a new repository selector with history and git detection
func NewRepositorySelector() *RepositorySelector {
	selector := &RepositorySelector{
		repositories: []Repository{},
	}
	selector.loadRepositories()
	return selector
}

// loadRepositories loads repositories from various sources
func (rs *RepositorySelector) loadRepositories() {
	var repos []Repository
	repoNames := make(map[string]bool) // for deduplication

	// 1. Add current git repository if available
	if gitRepo, _, err := utils.GetGitInfo(); err == nil && gitRepo != "" {
		repos = append(repos, Repository{
			Name:        gitRepo,
			Description: "Current git repository",
		})
		repoNames[gitRepo] = true
	}

	// 2. Add repositories from history
	if history, err := cache.GetRepositoryHistory(); err == nil {
		for _, repoName := range history {
			if repoName != "" && !repoNames[repoName] {
				repos = append(repos, Repository{
					Name:        repoName,
					Description: "From history",
				})
				repoNames[repoName] = true
			}
		}
	}

	// 3. Add some common repository patterns as examples if list is empty
	if len(repos) == 0 {
		repos = []Repository{
			{Name: "owner/repo", Description: "Example repository format"},
		}
	}

	rs.repositories = repos
}

// SelectRepository shows the fuzzy finder and returns the selected repository
func (rs *RepositorySelector) SelectRepository() (string, error) {
	if len(rs.repositories) == 0 {
		return "", fmt.Errorf("no repositories available")
	}

	// Check if we're running in a terminal that supports fzf
	if !isTerminalInteractive() {
		// Fallback: return the most recent repository or current git repo
		if len(rs.repositories) > 0 {
			return rs.repositories[0].Name, nil
		}
		return "", fmt.Errorf("no interactive terminal available for selection")
	}

	idx, err := fuzzyfinder.Find(
		rs.repositories,
		func(i int) string {
			return rs.repositories[i].Name
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			repo := rs.repositories[i]
			return fmt.Sprintf("Repository: %s\nSource: %s\n\nPress Tab to select this repository",
				repo.Name, repo.Description)
		}),
		fuzzyfinder.WithHeader("Select Repository (↑↓ to navigate, Tab to select, Esc to cancel)"),
	)

	if err != nil {
		return "", err
	}

	selectedRepo := rs.repositories[idx].Name

	// Add selected repository to history (async to avoid blocking)
	go func() {
		_ = cache.AddRepositoryToHistory(selectedRepo)
	}()

	return selectedRepo, nil
}

// GetDefaultRepository returns the most appropriate default repository
func (rs *RepositorySelector) GetDefaultRepository() string {
	if len(rs.repositories) == 0 {
		return ""
	}

	// Return the first repository (current git repo or most recent from history)
	return rs.repositories[0].Name
}

// AddManualRepository adds a manually entered repository to the selector
func (rs *RepositorySelector) AddManualRepository(repo string) {
	if repo == "" {
		return
	}

	// Check if already exists
	for _, existing := range rs.repositories {
		if existing.Name == repo {
			return
		}
	}

	// Add to front of list
	newRepo := Repository{
		Name:        repo,
		Description: "Manually entered",
	}
	rs.repositories = append([]Repository{newRepo}, rs.repositories...)

	// Add to persistent history
	go func() {
		_ = cache.AddRepositoryToHistory(repo)
	}()
}

// isTerminalInteractive checks if we're running in an interactive terminal
func isTerminalInteractive() bool {
	// Check if stdin, stdout, stderr are terminals
	if !isatty(os.Stdin) || !isatty(os.Stdout) || !isatty(os.Stderr) {
		return false
	}

	// Check if TERM is set (indicates terminal capabilities)
	if os.Getenv("TERM") == "" {
		return false
	}

	// Additional check: ensure we're not running in a CI environment
	ci_envs := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE", "CIRCLECI"}
	for _, env := range ci_envs {
		if os.Getenv(env) != "" {
			return false
		}
	}

	return true
}

// isatty checks if a file descriptor is a terminal
func isatty(f *os.File) bool {
	if f == nil {
		return false
	}

	return term.IsTerminal(int(f.Fd()))
}

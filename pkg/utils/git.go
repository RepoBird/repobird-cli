package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func DetectRepository() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not a git repository or no remote configured")
	}

	remoteURL := strings.TrimSpace(out.String())
	if remoteURL == "" {
		return "", fmt.Errorf("no remote origin found")
	}

	repo := parseGitURL(remoteURL)
	if repo == "" {
		return "", fmt.Errorf("unable to parse repository from URL: %s", remoteURL)
	}

	return repo, nil
}

func parseGitURL(url string) string {
	url = strings.TrimSpace(url)

	if strings.HasPrefix(url, "git@github.com:") {
		repo := strings.TrimPrefix(url, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return repo
	}

	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		if len(parts) > 1 {
			repo := parts[1]
			repo = strings.TrimSuffix(repo, ".git")
			return repo
		}
	}

	return ""
}

func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(out.String())
	if branch == "" {
		return "", fmt.Errorf("unable to determine current branch")
	}

	return branch, nil
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package services

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/pkg/utils"
)

// gitService implements the domain.GitService interface
type gitService struct{}

// NewGitService creates a new git service
func NewGitService() domain.GitService {
	return &gitService{}
}

// GetCurrentBranch returns the current git branch
func (g *gitService) GetCurrentBranch() (string, error) {
	return utils.GetCurrentBranch()
}

// GetRepositoryName returns the repository name from the git remote
func (g *gitService) GetRepositoryName() (string, error) {
	return utils.DetectRepository()
}

// IsGitRepository checks if the current directory is a git repository
func (g *gitService) IsGitRepository() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) != ""
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"context"
	"os/exec"
	"time"
)

func GetGitInfo() (string, string, error) {
	// Create context with timeout for quick git operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return GetGitInfoWithContext(ctx)
}

func GetGitInfoWithContext(ctx context.Context) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return "", "", nil
	}

	repo, repoErr := DetectRepositoryWithContext(ctx)
	branch, branchErr := GetCurrentBranchWithContext(ctx)

	if repoErr != nil && branchErr != nil {
		if repoErr != nil {
			return "", "", repoErr
		}
		return "", "", branchErr
	}

	return repo, branch, nil
}

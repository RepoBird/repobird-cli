// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import "os/exec"

func GetGitInfo() (string, string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return "", "", nil
	}

	repo, repoErr := DetectRepository()
	branch, branchErr := GetCurrentBranch()

	if repoErr != nil && branchErr != nil {
		if repoErr != nil {
			return "", "", repoErr
		}
		return "", "", branchErr
	}

	return repo, branch, nil
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

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
	if url == "" {
		return ""
	}

	// Handle SSH URLs with ssh:// protocol
	if strings.HasPrefix(url, "ssh://") {
		// Format: ssh://git@host:port/path or ssh://git@host/path
		url = strings.TrimPrefix(url, "ssh://")
		// Remove git@ prefix if present
		url = strings.TrimPrefix(url, "git@")

		// Now we have host:port/path or host/path
		// Find the path part (after the first /)
		idx := strings.Index(url, "/")
		if idx == -1 {
			return ""
		}

		path := url[idx+1:]
		path = strings.TrimSuffix(path, ".git")
		path = strings.Trim(path, "/")

		// Validate we have at least org/repo
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return ""
		}

		return path
	}

	// Handle SSH URLs (git@host:path format)
	if strings.HasPrefix(url, "git@") {
		// Format: git@host:org/repo.git or git@host:port:org/repo.git
		parts := strings.SplitN(url, ":", 2)
		if len(parts) < 2 {
			return ""
		}

		// Get the path part after the colon
		path := parts[1]

		// Handle SSH with custom port (git@host:port:path)
		// This would have 3 parts when split by colon
		if strings.Contains(parts[0], ":") || strings.Contains(path, ":") {
			// Re-split to handle port
			allParts := strings.Split(url, ":")
			if len(allParts) >= 3 {
				// Last part is the path
				path = allParts[len(allParts)-1]
			}
		}

		// Clean up the path
		path = strings.TrimSuffix(path, ".git")
		path = strings.Trim(path, "/")

		// Validate we have at least org/repo
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return ""
		}

		return path
	}

	// Handle HTTPS URLs
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// Remove query parameters and fragments
		if idx := strings.Index(url, "?"); idx != -1 {
			url = url[:idx]
		}
		if idx := strings.Index(url, "#"); idx != -1 {
			url = url[:idx]
		}

		// Parse the URL to extract the path
		// Remove protocol
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")

		// Remove credentials if present (user:pass@host)
		if idx := strings.Index(url, "@"); idx != -1 {
			url = url[idx+1:]
		}

		// Split by first slash to separate host from path
		parts := strings.SplitN(url, "/", 2)
		if len(parts) < 2 {
			return ""
		}

		// Get the path part
		path := parts[1]
		path = strings.TrimSuffix(path, ".git")
		path = strings.Trim(path, "/")

		// Validate we have at least org/repo
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return ""
		}

		return path
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

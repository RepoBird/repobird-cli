// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"strings"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/utils"
)

// ConfigLoader handles loading and validating run configuration files
type ConfigLoader struct{}

// NewConfigLoader creates a new config loader instance
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// LoadConfig loads a run configuration from a JSON, YAML, or Markdown file
func (c *ConfigLoader) LoadConfig(filePath string) (*models.RunRequest, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Use the unified loader that supports JSON, YAML, and Markdown
	// Use the non-prompting version for TUI compatibility
	runConfig, additionalContext, err := utils.LoadConfigFromFileNoPrompts(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Convert RunConfig to RunRequest
	runRequest := &models.RunRequest{
		Prompt:     runConfig.Prompt,
		Repository: runConfig.Repository,
		Source:     runConfig.Source,
		Target:     runConfig.Target,
		RunType:    models.RunType(runConfig.RunType),
		Title:      runConfig.Title,
		Context:    runConfig.Context,
		Files:      runConfig.Files,
	}

	// Append markdown body to context if present
	if additionalContext != "" {
		if runRequest.Context != "" {
			runRequest.Context = runRequest.Context + "\n\n" + additionalContext
		} else {
			runRequest.Context = additionalContext
		}
	}

	// Validate the configuration
	if err := c.ValidateConfig(runRequest); err != nil {
		return nil, err
	}

	return runRequest, nil
}

// ValidateConfig validates a run configuration
func (c *ConfigLoader) ValidateConfig(config *models.RunRequest) error {
	var errors []string

	// Check required fields
	if config.Prompt == "" {
		errors = append(errors, "prompt is required and cannot be empty")
	}

	if config.Repository == "" {
		errors = append(errors, "repository identification is required")
	}

	// Validate runType
	switch config.RunType {
	case models.RunTypeRun, models.RunTypePlan:
		// Valid
	case "":
		config.RunType = models.RunTypeRun // Set default
	default:
		errors = append(errors, fmt.Sprintf("invalid runType '%s', must be 'run' or 'plan'", config.RunType))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
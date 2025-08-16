package config

import (
	"fmt"
	"strconv"
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

// parseConfig converts the raw config data to RunRequest
func (c *ConfigLoader) parseConfig(data map[string]interface{}) (*models.RunRequest, error) {
	config := &models.RunRequest{}

	// Handle prompt (required)
	if prompt, ok := data["prompt"]; ok {
		if promptStr, ok := prompt.(string); ok {
			config.Prompt = strings.TrimSpace(promptStr)
		}
	}

	// Handle repository identification (flexible)
	if err := c.parseRepository(data, config); err != nil {
		return nil, err
	}

	// Handle optional string fields
	if source, ok := data["source"]; ok {
		if sourceStr, ok := source.(string); ok {
			config.Source = strings.TrimSpace(sourceStr)
		}
	}

	if target, ok := data["target"]; ok {
		if targetStr, ok := target.(string); ok {
			config.Target = strings.TrimSpace(targetStr)
		}
	}

	if title, ok := data["title"]; ok {
		if titleStr, ok := title.(string); ok {
			config.Title = strings.TrimSpace(titleStr)
		}
	}

	if context, ok := data["context"]; ok {
		if contextStr, ok := context.(string); ok {
			config.Context = strings.TrimSpace(contextStr)
		}
	}

	// Handle runType with validation
	if runType, ok := data["runType"]; ok {
		if runTypeStr, ok := runType.(string); ok {
			runTypeStr = strings.TrimSpace(runTypeStr)
			switch runTypeStr {
			case "run", "plan", "approval":
				config.RunType = models.RunType(runTypeStr)
			default:
				return nil, fmt.Errorf("invalid runType '%s', must be one of: run, plan, approval", runTypeStr)
			}
		}
	} else {
		config.RunType = models.RunTypeRun // Default
	}

	// Handle files array
	if files, ok := data["files"]; ok {
		if filesArray, ok := files.([]interface{}); ok {
			config.Files = make([]string, 0, len(filesArray))
			for i, file := range filesArray {
				if fileStr, ok := file.(string); ok {
					if trimmed := strings.TrimSpace(fileStr); trimmed != "" {
						config.Files = append(config.Files, trimmed)
					}
				} else {
					return nil, fmt.Errorf("files[%d] must be a string", i)
				}
			}
		} else {
			return nil, fmt.Errorf("files must be an array of strings")
		}
	}

	return config, nil
}

// parseRepository handles the flexible repository field parsing
func (c *ConfigLoader) parseRepository(data map[string]interface{}, config *models.RunRequest) error {
	// Priority order: repoId -> repository -> repositoryName

	// Check for repoId (integer)
	if repoId, ok := data["repoId"]; ok {
		switch v := repoId.(type) {
		case float64:
			if v > 0 && v == float64(int(v)) {
				// For now, we'll convert repoId to string format since the API expects repositoryName
				// In future, we might want to resolve repoId to actual repository name via API
				config.Repository = fmt.Sprintf("repo-%d", int(v))
				return nil
			} else {
				return fmt.Errorf("repoId must be a positive integer")
			}
		case int:
			if v > 0 {
				config.Repository = fmt.Sprintf("repo-%d", v)
				return nil
			} else {
				return fmt.Errorf("repoId must be a positive integer")
			}
		default:
			return fmt.Errorf("repoId must be an integer")
		}
	}

	// Check for repository field
	if repository, ok := data["repository"]; ok {
		if repoStr, ok := repository.(string); ok {
			repoStr = strings.TrimSpace(repoStr)
			if repoStr != "" {
				if err := c.validateRepositoryFormat(repoStr); err != nil {
					return err
				}
				config.Repository = repoStr
				return nil
			}
		}
	}

	// Check for repositoryName field
	if repositoryName, ok := data["repositoryName"]; ok {
		if repoStr, ok := repositoryName.(string); ok {
			repoStr = strings.TrimSpace(repoStr)
			if repoStr != "" {
				if err := c.validateRepositoryFormat(repoStr); err != nil {
					return err
				}
				config.Repository = repoStr
				return nil
			}
		}
	}

	// No repository field found
	return fmt.Errorf("must specify one of: 'repository', 'repositoryName', or 'repoId'")
}

// validateRepositoryFormat validates the repository string format
func (c *ConfigLoader) validateRepositoryFormat(repo string) error {
	// Allow repo-ID format from repoId conversion
	if strings.HasPrefix(repo, "repo-") {
		if _, err := strconv.Atoi(repo[5:]); err == nil {
			return nil
		}
	}

	// Standard owner/repo format
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("repository must be in format 'owner/repo', got '%s'", repo)
	}
	return nil
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

// LoadConfigPartial loads a config file and returns both the config and any validation errors
// This allows partial loading where some fields might be invalid but others are usable
func (c *ConfigLoader) LoadConfigPartial(filePath string) (*models.RunRequest, []error) {
	config, err := c.LoadConfig(filePath)
	if err != nil {
		return nil, []error{err}
	}
	return config, nil
}

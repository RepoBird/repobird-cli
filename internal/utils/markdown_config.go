package utils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/repobird/repobird-cli/internal/models"
)

// MarkdownConfig represents the frontmatter structure in markdown task files
type MarkdownConfig struct {
	Prompt      string                 `yaml:"prompt" json:"prompt"`
	Repository  string                 `yaml:"repository" json:"repository"`
	Source      string                 `yaml:"source" json:"source"`
	Target      string                 `yaml:"target" json:"target"`
	RunType     string                 `yaml:"runType" json:"runType"`
	Title       string                 `yaml:"title" json:"title"`
	Context     string                 `yaml:"context" json:"context"`
	Files       []string               `yaml:"files" json:"files"`
	PullRequest *PullRequestConfig     `yaml:"pullRequest,omitempty" json:"pullRequest,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// PullRequestConfig represents pull request configuration
type PullRequestConfig struct {
	Create bool `yaml:"create" json:"create"`
	Draft  bool `yaml:"draft" json:"draft"`
}

// ParseMarkdownConfig reads and parses a markdown file with frontmatter
func ParseMarkdownConfig(filepath string) (*models.RunConfig, string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open markdown file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return ParseMarkdownConfigFromReader(file)
}

// ParseMarkdownConfigFromReader parses markdown config from an io.Reader
func ParseMarkdownConfigFromReader(r io.Reader) (*models.RunConfig, string, error) {
	var config MarkdownConfig
	rest, err := frontmatter.Parse(r, &config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Validate required fields
	if err := validateMarkdownConfig(&config); err != nil {
		return nil, "", err
	}

	// Convert to RunConfig
	runConfig := &models.RunConfig{
		Prompt:     config.Prompt,
		Repository: config.Repository,
		Source:     config.Source,
		Target:     config.Target,
		RunType:    config.RunType,
		Title:      config.Title,
		Context:    config.Context,
		Files:      config.Files,
	}

	// Extract markdown content (rest) as additional context if present
	markdownContent := strings.TrimSpace(string(rest))

	return runConfig, markdownContent, nil
}

// validateMarkdownConfig validates required fields in the markdown configuration
func validateMarkdownConfig(config *MarkdownConfig) error {
	// Convert to RunConfig for validation
	runConfig := &models.RunConfig{
		Prompt:     config.Prompt,
		Repository: config.Repository,
		Source:     config.Source,
		Target:     config.Target,
		RunType:    config.RunType,
		Title:      config.Title,
		Context:    config.Context,
		Files:      config.Files,
	}

	// Use shared validation
	if err := ValidateRunConfig(runConfig); err != nil {
		return err
	}

	// Apply defaults back to the markdown config
	config.Source = runConfig.Source
	config.RunType = runConfig.RunType

	return nil
}

// LoadMarkdownOrJSONConfig loads configuration from either markdown or JSON file
func LoadMarkdownOrJSONConfig(filepath string) (*models.RunConfig, string, error) {
	// Check file extension
	if strings.HasSuffix(strings.ToLower(filepath), ".md") ||
		strings.HasSuffix(strings.ToLower(filepath), ".markdown") {
		return ParseMarkdownConfig(filepath)
	}

	// Fall back to JSON parsing for .json files
	config, err := models.LoadRunConfigFromFile(filepath)
	if err != nil {
		return nil, "", err
	}

	return config, "", nil
}

// ValidateRunConfig validates a RunConfig structure and applies defaults
func ValidateRunConfig(config *models.RunConfig) error {
	var errors []string

	// Required fields validation
	if config.Prompt == "" {
		errors = append(errors, "prompt is required")
	}

	if config.Repository == "" {
		errors = append(errors, "repository is required")
	}

	// Apply defaults
	if config.Source == "" {
		config.Source = "main" // Default to main if not specified
	}

	// Target branch is optional - server will handle defaults

	if config.RunType == "" {
		config.RunType = "run" // Default to "run" if not specified
	} else if config.RunType != "run" && config.RunType != "plan" {
		errors = append(errors, fmt.Sprintf("invalid runType '%s', must be 'run' or 'plan'", config.RunType))
	}

	// Title is optional - server will generate if not provided

	// Validate repository format (basic check)
	if config.Repository != "" && !strings.Contains(config.Repository, "/") {
		errors = append(errors, "repository must be in format 'owner/repo'")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

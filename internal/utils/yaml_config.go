package utils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/repobird/repobird-cli/internal/models"
	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the structure of a YAML configuration file
type YAMLConfig struct {
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

// ParseYAMLConfig reads and parses a YAML configuration file
func ParseYAMLConfig(filepath string) (*models.RunConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open YAML file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return ParseYAMLConfigFromReader(file)
}

// ParseYAMLConfigFromReader parses YAML config from an io.Reader
func ParseYAMLConfigFromReader(r io.Reader) (*models.RunConfig, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML content: %w", err)
	}

	var config YAMLConfig
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // This will cause an error if unknown fields are present

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if err := validateYAMLConfig(&config); err != nil {
		return nil, err
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

	return runConfig, nil
}

// validateYAMLConfig validates required fields in the YAML configuration
func validateYAMLConfig(config *YAMLConfig) error {
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

	// Apply defaults back to the YAML config
	config.Source = runConfig.Source
	config.RunType = runConfig.RunType

	return nil
}

// LoadConfigFromFile loads configuration from JSON, YAML, or Markdown files
func LoadConfigFromFile(filepath string) (*models.RunConfig, string, error) {
	lowercasePath := strings.ToLower(filepath)

	// Check file extension
	switch {
	case strings.HasSuffix(lowercasePath, ".yaml") || strings.HasSuffix(lowercasePath, ".yml"):
		// Parse pure YAML file
		config, err := ParseYAMLConfig(filepath)
		return config, "", err

	case strings.HasSuffix(lowercasePath, ".md") || strings.HasSuffix(lowercasePath, ".markdown"):
		// Parse markdown file with YAML frontmatter
		return ParseMarkdownConfig(filepath)

	case strings.HasSuffix(lowercasePath, ".json"):
		// Parse JSON file
		config, err := models.LoadRunConfigFromFile(filepath)
		return config, "", err

	default:
		// Try to detect format by content
		return detectAndParseConfig(filepath)
	}
}

// detectAndParseConfig attempts to detect the file format and parse accordingly
func detectAndParseConfig(filepath string) (*models.RunConfig, string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read first few bytes to detect format
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		return nil, "", fmt.Errorf("failed to reset file position: %w", err)
	}

	content := string(buf[:n])
	trimmed := strings.TrimSpace(content)

	// Detect format
	switch {
	case strings.HasPrefix(trimmed, "{"):
		// Likely JSON
		config, err := models.LoadRunConfigFromFile(filepath)
		return config, "", err

	case strings.HasPrefix(trimmed, "---"):
		// Could be YAML frontmatter or pure YAML
		if strings.Contains(content, "\n---\n") || strings.Contains(content, "\r\n---\r\n") {
			// Likely markdown with frontmatter
			return ParseMarkdownConfig(filepath)
		}
		// Pure YAML
		config, err := ParseYAMLConfig(filepath)
		return config, "", err

	default:
		// Try YAML first (most flexible)
		config, yamlErr := ParseYAMLConfig(filepath)
		if yamlErr == nil {
			return config, "", nil
		}

		// Try JSON
		config, jsonErr := models.LoadRunConfigFromFile(filepath)
		if jsonErr == nil {
			return config, "", nil
		}

		// Return the YAML error as it's more likely
		return nil, "", fmt.Errorf("unable to parse file as YAML or JSON: %w", yamlErr)
	}
}

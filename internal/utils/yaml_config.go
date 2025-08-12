package utils

import (
	"encoding/json"
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
	config, _, err := ParseYAMLConfigWithPrompts(filepath)
	return config, err
}

// ParseYAMLConfigWithPrompts reads and parses a YAML configuration file with validation prompts
func ParseYAMLConfigWithPrompts(filepath string) (*models.RunConfig, *ValidationPromptHandler, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open YAML file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return ParseYAMLConfigFromReaderWithPrompts(file)
}

// ParseYAMLConfigFromReader parses YAML config from an io.Reader
func ParseYAMLConfigFromReader(r io.Reader) (*models.RunConfig, error) {
	config, _, err := ParseYAMLConfigFromReaderWithPrompts(r)
	return config, err
}

// ParseYAMLConfigFromReaderWithPrompts parses YAML config from an io.Reader with validation prompts
func ParseYAMLConfigFromReaderWithPrompts(r io.Reader) (*models.RunConfig, *ValidationPromptHandler, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read YAML content: %w", err)
	}

	return parseYAMLWithUnknownFieldsAndPrompts(data)
}

// parseYAMLWithUnknownFields parses YAML allowing unknown fields and warns about them
func parseYAMLWithUnknownFields(data []byte) (*models.RunConfig, error) {
	// First, parse into a generic map to detect unknown fields
	var genericMap map[string]interface{}
	if err := yaml.Unmarshal(data, &genericMap); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Parse into our known structure without strict field checking
	var config YAMLConfig
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	// Remove the KnownFields(true) restriction to allow unknown fields
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Check for unsupported fields and warn
	unsupportedFields := findUnsupportedYAMLFields(genericMap)
	if len(unsupportedFields) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Task file has unsupported fields: %s\n",
			strings.Join(unsupportedFields, ", "))
	}

	// Validate required fields
	if err := validateYAMLConfig(&config); err != nil {
		return nil, err
	}

	// Convert to RunConfig (only supported fields are included)
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

// findUnsupportedYAMLFields identifies fields not in the supported YAMLConfig struct
func findUnsupportedYAMLFields(data map[string]interface{}) []string {
	supportedFields := map[string]bool{
		"prompt":      true,
		"repository":  true,
		"source":      true,
		"target":      true,
		"runType":     true,
		"title":       true,
		"context":     true,
		"files":       true,
		"pullRequest": true,
		"metadata":    true,
	}

	supportedFieldsList := []string{
		"prompt", "repository", "source", "target", "runType",
		"title", "context", "files", "pullRequest", "metadata",
	}

	var unsupported []string
	for field := range data {
		if !supportedFields[field] {
			unsupported = append(unsupported, field)

			// Check for similar field names and suggest
			if suggestion := SuggestFieldName(field, supportedFieldsList); suggestion != "" {
				fmt.Fprintf(os.Stderr, "Warning: Unknown field '%s' - did you mean '%s'?\n", field, suggestion)
			}
		}
	}

	return unsupported
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
	config, additionalContext, promptHandler, err := LoadConfigFromFileWithPrompts(filepath)
	if err != nil {
		return nil, "", err
	}
	
	// Process any validation prompts before proceeding
	if promptHandler != nil && promptHandler.HasPrompts() {
		shouldContinue, err := promptHandler.ProcessPrompts()
		if err != nil {
			return nil, "", err
		}
		if !shouldContinue {
			return nil, "", fmt.Errorf("operation cancelled by user")
		}
	}
	
	return config, additionalContext, nil
}

// LoadConfigFromFileWithPrompts loads configuration and returns validation prompts for processing
func LoadConfigFromFileWithPrompts(filepath string) (*models.RunConfig, string, *ValidationPromptHandler, error) {
	lowercasePath := strings.ToLower(filepath)

	// Check file extension
	switch {
	case strings.HasSuffix(lowercasePath, ".yaml") || strings.HasSuffix(lowercasePath, ".yml"):
		// Parse pure YAML file
		config, promptHandler, err := ParseYAMLConfigWithPrompts(filepath)
		return config, "", promptHandler, err

	case strings.HasSuffix(lowercasePath, ".md") || strings.HasSuffix(lowercasePath, ".markdown"):
		// Parse markdown file with YAML frontmatter
		config, additionalContext, err := ParseMarkdownConfig(filepath)
		// Markdown doesn't currently have unknown field detection, so no prompts
		return config, additionalContext, nil, err

	case strings.HasSuffix(lowercasePath, ".json"):
		// Parse JSON file
		config, promptHandler, err := models.LoadRunConfigFromFileWithPrompts(filepath)
		return config, "", promptHandler, err

	default:
		// Try to detect format by content
		return detectAndParseConfigWithPrompts(filepath)
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

// ParseJSONFromStdin parses JSON from stdin with unknown field support and warnings
func ParseJSONFromStdin() (*models.RunConfig, error) {
	// First, read all stdin data
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	// Parse into a generic map to detect unknown fields
	var genericMap map[string]interface{}
	if err := json.Unmarshal(data, &genericMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check for unsupported fields and warn
	unsupportedFields := findUnsupportedJSONFieldsForStdin(genericMap)
	if len(unsupportedFields) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Task JSON has unsupported fields: %s\n",
			strings.Join(unsupportedFields, ", "))
	}

	// Parse into our known structure (unknown fields will be ignored)
	var runReq models.RunRequest
	if err := json.Unmarshal(data, &runReq); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert RunRequest to RunConfig (only supported fields are included)
	return &models.RunConfig{
		Prompt:     runReq.Prompt,
		Repository: runReq.Repository,
		Source:     runReq.Source,
		Target:     runReq.Target,
		RunType:    string(runReq.RunType),
		Title:      runReq.Title,
		Context:    runReq.Context,
		Files:      runReq.Files,
	}, nil
}

// findUnsupportedJSONFieldsForStdin identifies fields not supported for stdin JSON
func findUnsupportedJSONFieldsForStdin(data map[string]interface{}) []string {
	supportedFields := map[string]bool{
		"prompt":     true,
		"repository": true,
		"source":     true,
		"target":     true,
		"runType":    true,
		"title":      true,
		"context":    true,
		"files":      true,
	}

	supportedFieldsList := []string{
		"prompt", "repository", "source", "target", "runType",
		"title", "context", "files",
	}

	var unsupported []string
	for field := range data {
		if !supportedFields[field] {
			unsupported = append(unsupported, field)

			// Check for similar field names and suggest
			if suggestion := SuggestFieldName(field, supportedFieldsList); suggestion != "" {
				fmt.Fprintf(os.Stderr, "Warning: Unknown field '%s' - did you mean '%s'?\n", field, suggestion)
			}
		}
	}

	return unsupported
}

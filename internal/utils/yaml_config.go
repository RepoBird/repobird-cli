package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/prompts"
	"github.com/repobird/repobird-cli/internal/tui/debug"
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
func ParseYAMLConfigWithPrompts(filepath string) (*models.RunConfig, *prompts.ValidationPromptHandler, error) {
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
func ParseYAMLConfigFromReaderWithPrompts(r io.Reader) (*models.RunConfig, *prompts.ValidationPromptHandler, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read YAML content: %w", err)
	}

	return parseYAMLWithUnknownFieldsAndPrompts(data)
}

// parseYAMLWithUnknownFields parses YAML allowing unknown fields and warns about them
func parseYAMLWithUnknownFields(data []byte) (*models.RunConfig, error) {
	config, _, err := parseYAMLWithUnknownFieldsAndPrompts(data)
	return config, err
}

// parseYAMLWithUnknownFieldsAndPrompts parses YAML allowing unknown fields and creates validation prompts
func parseYAMLWithUnknownFieldsAndPrompts(data []byte) (*models.RunConfig, *prompts.ValidationPromptHandler, error) {
	// First, parse into a generic map to detect unknown fields
	var genericMap map[string]interface{}
	if err := yaml.Unmarshal(data, &genericMap); err != nil {
		debug.LogToFilef("parseYAMLWithUnknownFieldsAndPrompts: Failed to unmarshal into generic map: %v\n", err)
		return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Extract known fields from the map instead of using decoder.Decode
	// This allows us to ignore unknown fields completely
	var config YAMLConfig

	// Manually extract the known fields from the map
	if prompt, ok := genericMap["prompt"].(string); ok {
		config.Prompt = prompt
	}
	if repository, ok := genericMap["repository"].(string); ok {
		config.Repository = repository
	}
	if source, ok := genericMap["source"].(string); ok {
		config.Source = source
	}
	if target, ok := genericMap["target"].(string); ok {
		config.Target = target
	}
	if runType, ok := genericMap["runType"].(string); ok {
		config.RunType = runType
	}
	if title, ok := genericMap["title"].(string); ok {
		config.Title = title
	}
	if context, ok := genericMap["context"].(string); ok {
		config.Context = context
	}
	if filesInterface, ok := genericMap["files"]; ok {
		if filesArray, ok := filesInterface.([]interface{}); ok {
			config.Files = make([]string, 0, len(filesArray))
			for _, file := range filesArray {
				if fileStr, ok := file.(string); ok {
					config.Files = append(config.Files, fileStr)
				}
			}
		}
	}

	// Create validation prompt handler
	promptHandler := prompts.NewValidationPromptHandler()

	// Check for unsupported fields and create prompts
	unsupportedFields, suggestions := findUnsupportedYAMLFieldsWithSuggestions(genericMap)

	// Add field suggestion prompts
	for field, suggestion := range suggestions {
		promptHandler.AddFieldSuggestionPrompt(field, suggestion)
	}

	// Add general unknown field warning if there are fields without suggestions
	var fieldsWithoutSuggestions []string
	for _, field := range unsupportedFields {
		if suggestions[field] == "" {
			fieldsWithoutSuggestions = append(fieldsWithoutSuggestions, field)
		}
	}
	if len(fieldsWithoutSuggestions) > 0 {
		promptHandler.AddUnknownFieldWarning(fieldsWithoutSuggestions)
	}

	// Check for validation errors but don't add them as prompts
	// Validation errors can't be fixed interactively
	validationErr := validateYAMLConfigForPrompts(&config)
	if validationErr != nil {
		// Log for debugging but don't add as prompt
		debug.LogToFilef("YAML has validation errors that will be caught later: %v\n", validationErr)
		// Don't add validation errors as prompts - they'll be caught by ValidateRunConfig later
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

	// Return the config and prompts (but no validation error prompts)
	// Actual validation will happen later via ValidateRunConfig
	return runConfig, promptHandler, nil
}

// findUnsupportedYAMLFields identifies fields not in the supported YAMLConfig struct
func findUnsupportedYAMLFields(data map[string]interface{}) []string {
	unsupportedFields, _ := findUnsupportedYAMLFieldsWithSuggestions(data)
	return unsupportedFields
}

// findUnsupportedYAMLFieldsWithSuggestions identifies fields and returns suggestions
func findUnsupportedYAMLFieldsWithSuggestions(data map[string]interface{}) ([]string, map[string]string) {
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
	suggestions := make(map[string]string)

	for field := range data {
		if !supportedFields[field] {
			unsupported = append(unsupported, field)

			// Check for similar field names and store suggestion
			if suggestion := SuggestFieldName(field, supportedFieldsList); suggestion != "" {
				suggestions[field] = suggestion
			}
		}
	}

	return unsupported, suggestions
}

// validateYAMLConfigForPrompts validates and applies defaults without failing on validation errors
func validateYAMLConfigForPrompts(config *YAMLConfig) error {
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

	// Apply defaults first
	// DO NOT default source - let the server handle it based on repo's default branch
	if runConfig.RunType == "" {
		runConfig.RunType = "run"
		config.RunType = "run"
	}

	// Check for validation errors and return them (but don't fail)
	validationErr := ValidateRunConfig(runConfig)

	// Apply any additional defaults that were set during validation
	config.RunType = runConfig.RunType

	return validationErr
}

// LoadConfigFromFile loads configuration from JSON, YAML, or Markdown files
func LoadConfigFromFile(filepath string) (*models.RunConfig, string, error) {
	// For TUI compatibility, use the non-prompting version
	return LoadConfigFromFileNoPrompts(filepath)
}

// LoadConfigFromFileNoPrompts loads config without showing validation prompts
func LoadConfigFromFileNoPrompts(filepath string) (*models.RunConfig, string, error) {
	config, additionalContext, _, err := LoadConfigFromFileWithPrompts(filepath)
	// Ignore validation prompts for TUI compatibility
	return config, additionalContext, err
}

// LoadConfigFromFileWithPrompts loads configuration and returns validation prompts for processing
func LoadConfigFromFileWithPrompts(filepath string) (*models.RunConfig, string, *prompts.ValidationPromptHandler, error) {
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

// detectAndParseConfigWithPrompts attempts to detect the file format and parse accordingly with validation prompts
func detectAndParseConfigWithPrompts(filepath string) (*models.RunConfig, string, *prompts.ValidationPromptHandler, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read first few bytes to detect format
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		return nil, "", nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	content := string(buf[:n])
	trimmed := strings.TrimSpace(content)

	// Detect format
	switch {
	case strings.HasPrefix(trimmed, "{"):
		// Likely JSON
		config, promptHandler, err := models.LoadRunConfigFromFileWithPrompts(filepath)
		return config, "", promptHandler, err

	case strings.HasPrefix(trimmed, "---"):
		// Could be YAML frontmatter or pure YAML
		if strings.Contains(content, "\n---\n") || strings.Contains(content, "\r\n---\r\n") {
			// Likely markdown with frontmatter - no prompts currently
			config, additionalContext, err := ParseMarkdownConfig(filepath)
			return config, additionalContext, nil, err
		}
		// Pure YAML
		config, promptHandler, err := ParseYAMLConfigWithPrompts(filepath)
		return config, "", promptHandler, err

	default:
		// Try YAML first (most flexible)
		config, promptHandler, yamlErr := ParseYAMLConfigWithPrompts(filepath)
		if yamlErr == nil {
			return config, "", promptHandler, nil
		}

		// Try JSON
		config, promptHandler2, jsonErr := models.LoadRunConfigFromFileWithPrompts(filepath)
		if jsonErr == nil {
			return config, "", promptHandler2, nil
		}

		// Return the YAML error as it's more likely
		return nil, "", nil, fmt.Errorf("unable to parse file as YAML or JSON: %w", yamlErr)
	}
}

// ParseJSONFromStdin parses JSON from stdin with unknown field support and warnings
func ParseJSONFromStdin() (*models.RunConfig, error) {
	config, promptHandler, err := ParseJSONFromStdinWithPrompts()
	if err != nil {
		return nil, err
	}

	// Process any validation prompts before proceeding
	if promptHandler != nil && promptHandler.HasPrompts() {
		shouldContinue, err := promptHandler.ProcessPrompts()
		if err != nil {
			return nil, err
		}
		if !shouldContinue {
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	return config, nil
}

// ParseJSONFromStdinWithPrompts parses JSON from stdin with validation prompts
func ParseJSONFromStdinWithPrompts() (*models.RunConfig, *prompts.ValidationPromptHandler, error) {
	// First, read all stdin data
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	// Check if stdin data is empty
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("stdin is empty - no JSON data received")
	}

	// Parse into a generic map to detect unknown fields
	var genericMap map[string]interface{}
	if err := json.Unmarshal(data, &genericMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create validation prompt handler
	promptHandler := prompts.NewValidationPromptHandler()

	// Check for unsupported fields and create prompts
	unsupportedFields, suggestions := findUnsupportedJSONFieldsForStdinWithSuggestions(genericMap)

	// Add field suggestion prompts
	for field, suggestion := range suggestions {
		promptHandler.AddFieldSuggestionPrompt(field, suggestion)
	}

	// Add general unknown field warning if there are fields without suggestions
	var fieldsWithoutSuggestions []string
	for _, field := range unsupportedFields {
		if suggestions[field] == "" {
			fieldsWithoutSuggestions = append(fieldsWithoutSuggestions, field)
		}
	}
	if len(fieldsWithoutSuggestions) > 0 {
		promptHandler.AddUnknownFieldWarning(fieldsWithoutSuggestions)
	}

	// Parse into our known structure (unknown fields will be ignored)
	var runReq models.RunRequest
	if err := json.Unmarshal(data, &runReq); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert RunRequest to RunConfig (only supported fields are included)
	runConfig := &models.RunConfig{
		Prompt:     runReq.Prompt,
		Repository: runReq.Repository,
		Source:     runReq.Source,
		Target:     runReq.Target,
		RunType:    string(runReq.RunType),
		Title:      runReq.Title,
		Context:    runReq.Context,
		Files:      runReq.Files,
	}

	return runConfig, promptHandler, nil
}

// findUnsupportedJSONFieldsForStdin identifies fields not supported for stdin JSON
func findUnsupportedJSONFieldsForStdin(data map[string]interface{}) []string {
	unsupportedFields, _ := findUnsupportedJSONFieldsForStdinWithSuggestions(data)
	return unsupportedFields
}

// findUnsupportedJSONFieldsForStdinWithSuggestions identifies fields and returns suggestions
func findUnsupportedJSONFieldsForStdinWithSuggestions(data map[string]interface{}) ([]string, map[string]string) {
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
	suggestions := make(map[string]string)

	for field := range data {
		if !supportedFields[field] {
			unsupported = append(unsupported, field)

			// Check for similar field names and store suggestion
			if suggestion := SuggestFieldName(field, supportedFieldsList); suggestion != "" {
				suggestions[field] = suggestion
			}
		}
	}

	return unsupported, suggestions
}

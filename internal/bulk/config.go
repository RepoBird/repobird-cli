package bulk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/repobird/repobird-cli/internal/utils"
	"gopkg.in/yaml.v3"
)

const MaxBulkBatchSize = 40

// BulkConfig represents a bulk run configuration
type BulkConfig struct {
	Repository string          `json:"repository,omitempty" yaml:"repository,omitempty"`
	RepoID     int             `json:"repoId,omitempty" yaml:"repoId,omitempty"`
	BatchTitle string          `json:"batchTitle,omitempty" yaml:"batchTitle,omitempty"`
	Source     string          `json:"source,omitempty" yaml:"source,omitempty"`
	RunType    string          `json:"runType,omitempty" yaml:"runType,omitempty"`
	Force      bool            `json:"force,omitempty" yaml:"force,omitempty"`
	Runs       []BulkRunConfig `json:"runs" yaml:"runs"`
}

// BulkRunConfig represents a single run within a bulk configuration
type BulkRunConfig struct {
	Prompt  string `json:"prompt" yaml:"prompt"`
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`
	Target  string `json:"target,omitempty" yaml:"target,omitempty"`
	Context string `json:"context,omitempty" yaml:"context,omitempty"`
}

// BulkRunRequest is what gets sent to the API (includes generated fields)
type BulkRunRequest struct {
	BulkConfig
	RunHashes []string `json:"runHashes,omitempty"`
}

// LoadBulkConfig loads bulk configuration from file(s)
func LoadBulkConfig(paths []string) (*BulkConfig, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no files specified")
	}

	if len(paths) == 1 {
		// Single file - could be bulk or single run config
		return ParseBulkConfig(paths[0])
	}

	// Multiple files - combine into bulk config
	return CreateBulkFromSingleConfigs(paths)
}

// ParseBulkConfig parses a single file as bulk configuration
func ParseBulkConfig(path string) (*BulkConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))

	// Check if it's a JSONL file
	if ext == ".jsonl" {
		return parseJSONL(path)
	}

	// Check if it's a markdown file
	if ext == ".md" || ext == ".markdown" {
		return parseMarkdown(content)
	}

	// Try to determine if it's a bulk config or single config
	var test map[string]interface{}
	if err := json.Unmarshal(content, &test); err != nil {
		// Try YAML
		if err := yaml.Unmarshal(content, &test); err != nil {
			return nil, fmt.Errorf("failed to parse %s: not valid JSON or YAML", path)
		}
	}

	// Check if it has "runs" array (bulk)
	if _, hasBulk := test["runs"]; hasBulk {
		// It's a bulk config
		return parseFile(path, content)
	}

	// It's a single config - convert to bulk
	runConfig, additionalContext, err := utils.LoadConfigFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse single config: %w", err)
	}

	// Append additional context if present
	context := runConfig.Context
	if additionalContext != "" {
		if context != "" {
			context = context + "\n\n" + additionalContext
		} else {
			context = additionalContext
		}
	}

	bulk := &BulkConfig{
		Repository: runConfig.Repository,
		Source:     runConfig.Source,
		RunType:    runConfig.RunType,
		Runs: []BulkRunConfig{{
			Prompt:  runConfig.Prompt,
			Title:   runConfig.Title,
			Target:  runConfig.Target,
			Context: context,
		}},
	}

	return validateBulkConfig(bulk)
}

// CreateBulkFromSingleConfigs creates a bulk config from multiple single-run files
func CreateBulkFromSingleConfigs(paths []string) (*BulkConfig, error) {
	var runs []BulkRunConfig
	var repository string
	var repoID int
	var source string
	var runType string

	for _, path := range paths {
		// Try to parse as bulk config first
		bulkConfig, bulkErr := ParseBulkConfig(path)
		if bulkErr == nil {
			// It's a bulk config - add all runs
			runs = append(runs, bulkConfig.Runs...)
			if repository == "" {
				repository = bulkConfig.Repository
				repoID = bulkConfig.RepoID
				source = bulkConfig.Source
				runType = bulkConfig.RunType
			}
		} else {
			// Try as single run config
			runConfig, additionalContext, err := utils.LoadConfigFromFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %v", path, err)
			}

			// Append additional context if present
			context := runConfig.Context
			if additionalContext != "" {
				if context != "" {
					context = context + "\n\n" + additionalContext
				} else {
					context = additionalContext
				}
			}

			// Convert single run to bulk run
			run := BulkRunConfig{
				Prompt:  runConfig.Prompt,
				Title:   runConfig.Title,
				Target:  runConfig.Target,
				Context: context,
			}
			runs = append(runs, run)

			// Use repository from first file if not set
			if repository == "" {
				repository = runConfig.Repository
				source = runConfig.Source
				runType = runConfig.RunType
			}
		}
	}

	// Create final bulk config
	bulk := &BulkConfig{
		Repository: repository,
		RepoID:     repoID,
		Source:     source,
		RunType:    runType,
		Runs:       runs,
		BatchTitle: fmt.Sprintf("Batch of %d tasks", len(runs)),
	}

	return validateBulkConfig(bulk)
}

// parseFile parses a bulk config file (JSON or YAML)
func parseFile(path string, content []byte) (*BulkConfig, error) {
	var config BulkConfig
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		if err := json.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	return validateBulkConfig(&config)
}

// parseJSONL parses a JSONL file into bulk config
func parseJSONL(path string) (*BulkConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSONL file: %w", err)
	}
	defer file.Close()

	var runs []BulkRunConfig
	var repository string
	var repoID int
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var item struct {
			Repository string `json:"repository,omitempty"`
			RepoID     int    `json:"repoId,omitempty"`
			Prompt     string `json:"prompt"`
			Title      string `json:"title,omitempty"`
			Target     string `json:"target,omitempty"`
			Context    string `json:"context,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("failed to parse JSONL line: %w", err)
		}

		// Use first repository if not set
		if repository == "" && item.Repository != "" {
			repository = item.Repository
		}
		if repoID == 0 && item.RepoID != 0 {
			repoID = item.RepoID
		}

		runs = append(runs, BulkRunConfig{
			Prompt:  item.Prompt,
			Title:   item.Title,
			Target:  item.Target,
			Context: item.Context,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading JSONL file: %w", err)
	}

	bulk := &BulkConfig{
		Repository: repository,
		RepoID:     repoID,
		Runs:       runs,
		BatchTitle: fmt.Sprintf("Batch of %d tasks", len(runs)),
	}

	return validateBulkConfig(bulk)
}

// parseMarkdown parses a markdown file into bulk config
func parseMarkdown(content []byte) (*BulkConfig, error) {
	lines := strings.Split(string(content), "\n")
	var config BulkConfig
	var currentRun *BulkRunConfig
	var inFrontMatter bool
	var frontMatterLines []string
	var inContext bool
	var contextLines []string

	for i, line := range lines {
		// Check for front matter
		if i == 0 && line == "---" {
			inFrontMatter = true
			continue
		}
		if inFrontMatter {
			if line == "---" {
				inFrontMatter = false
				// Parse front matter
				frontMatter := strings.Join(frontMatterLines, "\n")
				if err := yaml.Unmarshal([]byte(frontMatter), &config); err != nil {
					return nil, fmt.Errorf("failed to parse front matter: %w", err)
				}
				continue
			}
			frontMatterLines = append(frontMatterLines, line)
			continue
		}

		// Check for run headers
		if strings.HasPrefix(line, "## Run ") || strings.HasPrefix(line, "## ") {
			// Save previous run if exists
			if currentRun != nil && currentRun.Prompt != "" {
				config.Runs = append(config.Runs, *currentRun)
			}
			currentRun = &BulkRunConfig{}
			// Extract title from header
			title := strings.TrimPrefix(line, "## ")
			if colonIdx := strings.Index(title, ":"); colonIdx > 0 {
				title = strings.TrimSpace(title[colonIdx+1:])
			}
			currentRun.Title = title
			inContext = false
			continue
		}

		// Check for target
		if strings.HasPrefix(line, "**Target**:") {
			if currentRun != nil {
				target := strings.TrimPrefix(line, "**Target**:")
				target = strings.TrimSpace(target)
				// Remove optional marker if present
				target = strings.ReplaceAll(target, "(Optional)", "")
				target = strings.TrimSpace(target)
				currentRun.Target = target
			}
			continue
		}

		// Check for context section
		if strings.HasPrefix(line, "### Context") {
			inContext = true
			contextLines = []string{}
			continue
		}

		// Check for section separator
		if line == "---" && currentRun != nil {
			if inContext && len(contextLines) > 0 {
				currentRun.Context = strings.TrimSpace(strings.Join(contextLines, "\n"))
			}
			inContext = false
			continue
		}

		// Collect prompt or context lines
		if currentRun != nil {
			if inContext {
				if line != "" {
					contextLines = append(contextLines, line)
				}
			} else if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "**") {
				if currentRun.Prompt == "" {
					currentRun.Prompt = line
				} else {
					currentRun.Prompt += "\n" + line
				}
			}
		}
	}

	// Save last run
	if currentRun != nil && currentRun.Prompt != "" {
		if inContext && len(contextLines) > 0 {
			currentRun.Context = strings.TrimSpace(strings.Join(contextLines, "\n"))
		}
		config.Runs = append(config.Runs, *currentRun)
	}

	return validateBulkConfig(&config)
}

// validateBulkConfig validates a bulk configuration
func validateBulkConfig(config *BulkConfig) (*BulkConfig, error) {
	// Validate batch size
	if len(config.Runs) > MaxBulkBatchSize {
		return nil, fmt.Errorf("batch size exceeds maximum of %d runs (got %d)",
			MaxBulkBatchSize, len(config.Runs))
	}

	// Validate required fields
	if config.Repository == "" && config.RepoID == 0 {
		return nil, fmt.Errorf("either repository or repoId is required")
	}

	// Validate each run has a prompt
	for i, run := range config.Runs {
		if run.Prompt == "" {
			return nil, fmt.Errorf("run %d is missing required prompt field", i+1)
		}
	}

	// Set defaults
	if config.RunType == "" {
		config.RunType = "run"
	}

	return config, nil
}

// IsBulkConfig checks if a file contains bulk configuration
func IsBulkConfig(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Check for JSONL
	if strings.ToLower(filepath.Ext(path)) == ".jsonl" {
		return true, nil
	}

	// Check for markdown with front matter
	if strings.ToLower(filepath.Ext(path)) == ".md" || strings.ToLower(filepath.Ext(path)) == ".markdown" {
		lines := strings.Split(string(content), "\n")
		if len(lines) > 0 && lines[0] == "---" {
			return true, nil
		}
	}

	// Check for "runs" field in JSON/YAML
	var test map[string]interface{}
	if err := json.Unmarshal(content, &test); err != nil {
		if err := yaml.Unmarshal(content, &test); err != nil {
			return false, nil
		}
	}

	_, hasBulk := test["runs"]
	return hasBulk, nil
}

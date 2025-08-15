package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/repobird/repobird-cli/internal/prompts"
)

type RunType string

const (
	RunTypeRun      RunType = "run"
	RunTypePlan     RunType = "plan"
	RunTypeApproval RunType = "approval"
)

type RunStatus string

const (
	StatusQueued       RunStatus = "QUEUED"
	StatusInitializing RunStatus = "INITIALIZING"
	StatusProcessing   RunStatus = "PROCESSING"
	StatusPostProcess  RunStatus = "POST_PROCESS"
	StatusDone         RunStatus = "DONE"
	StatusFailed       RunStatus = "FAILED"
)

type RunRequest struct {
	Prompt     string   `json:"prompt"`
	Repository string   `json:"repository"` // User-facing field name
	Source     string   `json:"source"`     // User-facing field name
	Target     string   `json:"target"`     // User-facing field name
	RunType    RunType  `json:"runType"`
	Title      string   `json:"title,omitempty"`
	Context    string   `json:"context,omitempty"`
	Files      []string `json:"files,omitempty"`
}

// RunConfig is a unified configuration structure for both JSON and Markdown configs
type RunConfig struct {
	Prompt     string   `json:"prompt" yaml:"prompt"`
	Repository string   `json:"repository" yaml:"repository"`
	Source     string   `json:"source" yaml:"source"`
	Target     string   `json:"target" yaml:"target"`
	RunType    string   `json:"runType" yaml:"runType"`
	Title      string   `json:"title,omitempty" yaml:"title,omitempty"`
	Context    string   `json:"context,omitempty" yaml:"context,omitempty"`
	Files      []string `json:"files,omitempty" yaml:"files,omitempty"`
}

// APIRunRequest is the structure that matches the actual API expectations
type APIRunRequest struct {
	Prompt         string   `json:"prompt"`
	RepositoryName string   `json:"repositoryName"`
	SourceBranch   string   `json:"sourceBranch"`
	TargetBranch   string   `json:"targetBranch"`
	RunType        RunType  `json:"runType"`
	Title          string   `json:"title,omitempty"`
	Context        string   `json:"context,omitempty"`
	Files          []string `json:"files,omitempty"`
	FileHash       string   `json:"fileHash,omitempty"`
	Force          bool     `json:"force,omitempty"`
}

// ToAPIRequest converts user-facing RunRequest to API-compatible structure
func (r *RunRequest) ToAPIRequest() *APIRunRequest {
	return &APIRunRequest{
		Prompt:         r.Prompt,
		RepositoryName: r.Repository,
		SourceBranch:   r.Source,
		TargetBranch:   r.Target,
		RunType:        r.RunType,
		Title:          r.Title,
		Context:        r.Context,
		Files:          r.Files,
	}
}

type RunResponse struct {
	ID             string    `json:"id"` // Now stored as string internally
	Status         RunStatus `json:"status"`
	Repository     string    `json:"repository,omitempty"`     // Legacy field
	RepositoryName string    `json:"repositoryName,omitempty"` // New API field
	RepoID         int       `json:"repoId,omitempty"`
	Source         string    `json:"source"`
	Target         string    `json:"target"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Prompt         string    `json:"prompt"`
	Title          string    `json:"title,omitempty"`
	Description    string    `json:"description,omitempty"`
	Context        string    `json:"context,omitempty"`
	Error          string    `json:"error,omitempty"`
	PrURL          *string   `json:"prUrl,omitempty"`
	TriggerSource  *string   `json:"triggerSource,omitempty"`
	RunType        string    `json:"runType,omitempty"`
	Plan           string    `json:"plan,omitempty"`
	FileHash       string    `json:"fileHash,omitempty"`
}

// GetIDString returns the ID as a string
func (r *RunResponse) GetIDString() string {
	if r.ID == "" || r.ID == "null" {
		return ""
	}
	return r.ID
}

// GetRepositoryName returns the repository name from either field
func (r *RunResponse) GetRepositoryName() string {
	if r.RepositoryName != "" {
		return r.RepositoryName
	}
	return r.Repository
}

// UnmarshalJSON custom unmarshaler to handle ID field that can be string or number
func (r *RunResponse) UnmarshalJSON(data []byte) error {
	type Alias RunResponse
	aux := &struct {
		ID interface{} `json:"id"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert ID to string regardless of its type
	if aux.ID != nil {
		switch v := aux.ID.(type) {
		case string:
			r.ID = v
		case float64:
			r.ID = strconv.FormatFloat(v, 'f', 0, 64)
		case int:
			r.ID = strconv.Itoa(v)
		default:
			r.ID = fmt.Sprintf("%v", v)
		}
	}

	return nil
}

type UserInfo struct {
	ID                int    `json:"id,omitempty"`
	StringID          string `json:"stringId,omitempty"` // Original string ID from API
	Email             string `json:"email"`
	Name              string `json:"name,omitempty"`
	GithubUsername    string `json:"githubUsername,omitempty"`
	RemainingRuns     int    `json:"remainingRuns"`     // Deprecated: use RemainingProRuns
	TotalRuns         int    `json:"totalRuns"`         // Deprecated: use ProTotalRuns
	RemainingProRuns  int    `json:"remainingProRuns"`  // Pro runs remaining (displayed as "Runs Left")
	RemainingPlanRuns int    `json:"remainingPlanRuns"` // Plan runs remaining (displayed as "Plan Runs Left")
	ProTotalRuns      int    `json:"proTotalRuns"`      // Total pro runs in tier (displayed as "Run Total")
	PlanTotalRuns     int    `json:"planTotalRuns"`     // Total plan runs in tier (displayed as "Plan Run Total")
	Tier              string `json:"tier"`
	TierDetails       *Tier  `json:"tierDetails,omitempty"`
}

type ListRunsResponse struct {
	Data     []*RunResponse      `json:"data"`
	Metadata *PaginationMetadata `json:"metadata"`
}

type SingleRunResponse struct {
	Data     *RunResponse        `json:"data"`
	Metadata *PaginationMetadata `json:"metadata"`
}

type PaginationMetadata struct {
	CurrentPage int `json:"currentPage"`
	Total       int `json:"total"`
	TotalPages  int `json:"totalPages"`
}

// FileHashEntry represents a file hash entry from the API
type FileHashEntry struct {
	IssueRunID int    `json:"issueRunId"`
	FileHash   string `json:"fileHash"`
}

// FileHashesResponse represents the response from /api/v1/runs/hashes
type FileHashesResponse struct {
	Data []FileHashEntry `json:"data"`
}

// LoadRunConfigFromFile loads a RunConfig from a JSON file
func LoadRunConfigFromFile(filepath string) (*RunConfig, error) {
	config, _, err := LoadRunConfigFromFileWithPrompts(filepath)
	return config, err
}

// LoadRunConfigFromFileWithPrompts loads a RunConfig from a JSON file with validation prompts
func LoadRunConfigFromFileWithPrompts(filepath string) (*RunConfig, *prompts.ValidationPromptHandler, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return parseJSONWithUnknownFieldsAndPrompts(file)
}

// parseJSONWithUnknownFieldsAndPrompts parses JSON allowing unknown fields and creates validation prompts
func parseJSONWithUnknownFieldsAndPrompts(file *os.File) (*RunConfig, *prompts.ValidationPromptHandler, error) {
	// Reset file position for multiple reads
	if _, err := file.Seek(0, 0); err != nil {
		return nil, nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	// First, parse into a generic map to detect unknown fields
	var genericMap map[string]interface{}
	if err := json.NewDecoder(file).Decode(&genericMap); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create validation prompt handler
	promptHandler := prompts.NewValidationPromptHandler()

	// Check for unsupported fields and create prompts
	unsupportedFields, suggestions := findUnsupportedJSONFieldsWithSuggestions(genericMap)

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

	// Reset file position for actual parsing
	if _, err := file.Seek(0, 0); err != nil {
		return nil, nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	// Parse into our known structure (unknown fields will be ignored)
	var runReq RunRequest
	if err := json.NewDecoder(file).Decode(&runReq); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert RunRequest to RunConfig (only supported fields are included)
	runConfig := &RunConfig{
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

// findUnsupportedJSONFields identifies fields not in the supported RunRequest struct
func findUnsupportedJSONFields(data map[string]interface{}) []string {
	unsupportedFields, _ := findUnsupportedJSONFieldsWithSuggestions(data)
	return unsupportedFields
}

// findUnsupportedJSONFieldsWithSuggestions identifies fields and returns suggestions
func findUnsupportedJSONFieldsWithSuggestions(data map[string]interface{}) ([]string, map[string]string) {
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
			if suggestion := suggestJSONFieldName(field, supportedFieldsList); suggestion != "" {
				suggestions[field] = suggestion
			}
		}
	}

	return unsupported, suggestions
}

// suggestJSONFieldName suggests similar field names for JSON parsing
func suggestJSONFieldName(input string, validFields []string) string {
	if input == "" || len(validFields) == 0 {
		return ""
	}

	input = strings.ToLower(input)
	bestMatch := ""
	minDistance := len(input) + 1

	// Only suggest if the distance is reasonable (≤ 2 for most cases)
	threshold := 2
	if len(input) <= 3 {
		threshold = 1 // Stricter for very short fields
	}

	for _, field := range validFields {
		field = strings.ToLower(field)
		distance := levenshteinDistance(input, field)

		// Skip exact matches - we only suggest for similar but not identical fields
		if distance == 0 {
			continue
		}

		if distance <= threshold && distance < minDistance {
			// Extra check: if the input is just the plural of a valid field, prioritize it
			if strings.HasSuffix(input, "s") && field == input[:len(input)-1] {
				return field
			}
			// Or if a valid field is just the plural of input
			if strings.HasSuffix(field, "s") && input == field[:len(field)-1] {
				return field
			}

			bestMatch = field
			minDistance = distance
		}
	}

	// Only return if we found a reasonable match
	if minDistance <= threshold {
		return bestMatch
	}

	return ""
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create a matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

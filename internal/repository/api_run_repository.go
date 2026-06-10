package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/errors"
)

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// apiRunRepository implements domain.RunRepository using the API
type apiRunRepository struct {
	httpClient HTTPClient
	baseURL    string
	apiKey     string
	debug      bool
}

// NewAPIRunRepository creates a new API-based run repository
func NewAPIRunRepository(httpClient HTTPClient, baseURL, apiKey string, debug bool) domain.RunRepository {
	if baseURL == "" {
		baseURL = "https://repobird.ai"
	}
	return &apiRunRepository{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiKey:     apiKey,
		debug:      debug,
	}
}

// Create creates a new run via the API
func (r *apiRunRepository) Create(ctx context.Context, req domain.CreateRunRequest) (*domain.Run, error) {
	// Convert domain request to API DTO
	apiReq := &dto.CreateRunRequest{
		Prompt:                req.Prompt,
		RepositoryName:        req.RepositoryName,
		SourceBranch:          req.SourceBranch,
		TargetBranch:          req.TargetBranch,
		BaseBranch:            req.BaseBranch,
		OutputMode:            req.OutputMode,
		OutputBranch:          req.OutputBranch,
		PRTargetBranch:        req.PRTargetBranch,
		OutputBranchPolicy:    req.OutputBranchPolicy,
		RunType:               req.RunType,
		Agent:                 agentOrDefault(req.Agent),
		OpenCodeModel:         req.OpenCodeModel,
		OpenCodeProvider:      req.OpenCodeProvider,
		Title:                 req.Title,
		Context:               req.Context,
		Files:                 req.Files,
		BranchOnly:            req.BranchOnly,
		AcknowledgePromptRisk: req.AcknowledgePromptRisk,
		IdempotencyKey:        req.IdempotencyKey,
	}

	// Make API request
	resp, err := r.doRequest(ctx, "POST", "/api/v1/runs", apiReq, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.ParseAPIError(resp.StatusCode, body)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as CreateRunResponse first
	var createResp dto.CreateRunResponse
	if err := json.Unmarshal(body, &createResp); err == nil && createResp.Data != nil && createResp.Data.ID.String() != "" {
		return r.createdRunToDomain(createResp.Data, req), nil
	}

	// Fall back to direct RunResponse decoding
	var runResp dto.RunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return r.toDomainRun(&runResp), nil
}

func agentOrDefault(agent string) string {
	if agent != "" {
		return agent
	}
	return "opencode"
}

// Get retrieves a run by ID
func (r *apiRunRepository) Get(ctx context.Context, id string) (*domain.Run, error) {
	resp, err := r.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/runs/%s", id), nil, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.ParseAPIError(resp.StatusCode, body)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug log raw response if debug is enabled
	if r.debug {
		fmt.Printf("DEBUG [api_run_repository.Get]: Raw API response for run %s:\n%s\n", id, string(body))
	}

	// Try to decode as SingleRunResponse first
	var singleResp dto.SingleRunResponse
	if err := json.Unmarshal(body, &singleResp); err == nil && singleResp.Data != nil {
		domainRun := r.toDomainRun(singleResp.Data)
		if r.debug {
			fmt.Printf("DEBUG [api_run_repository.Get]: Mapped to domain - Status: %s, PullRequestURL: '%s'\n", domainRun.Status, domainRun.PullRequestURL)
		}
		return domainRun, nil
	}

	// Fall back to direct decoding
	var runResp dto.RunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	domainRun := r.toDomainRun(&runResp)
	if r.debug {
		fmt.Printf("DEBUG [api_run_repository.Get]: Mapped to domain (fallback) - Status: %s, PullRequestURL: '%s'\n", domainRun.Status, domainRun.PullRequestURL)
	}
	return domainRun, nil
}

// List retrieves a list of runs
func (r *apiRunRepository) List(ctx context.Context, opts domain.ListOptions) ([]*domain.Run, error) {
	path := fmt.Sprintf("/api/v1/runs?limit=%d&page=%d", opts.Limit, pageFromListOptions(opts))
	resp, err := r.doRequest(ctx, "GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.ParseAPIError(resp.StatusCode, body)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as ListRunsResponse first
	var listResp dto.ListRunsResponse
	if err := json.Unmarshal(body, &listResp); err == nil && listResp.Data != nil {
		runs := make([]*domain.Run, len(listResp.Data))
		for i, runResp := range listResp.Data {
			runs[i] = r.toDomainRun(runResp)
		}
		return runs, nil
	}

	// Fall back to direct array decoding
	var runResps []*dto.RunResponse
	if err := json.Unmarshal(body, &runResps); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	runs := make([]*domain.Run, len(runResps))
	for i, runResp := range runResps {
		runs[i] = r.toDomainRun(runResp)
	}
	return runs, nil
}

func pageFromListOptions(opts domain.ListOptions) int {
	if opts.Limit <= 0 {
		return 1
	}
	if opts.Offset <= 0 {
		return 1
	}
	return opts.Offset/opts.Limit + 1
}

// doRequest performs an HTTP request
func (r *apiRunRepository) doRequest(ctx context.Context, method, path string, body interface{}, idempotencyKey string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	if r.debug {
		fmt.Printf("Request: %s %s\n", method, req.URL.String())
		if body != nil {
			b, _ := json.MarshalIndent(body, "", "  ")
			fmt.Printf("Body: %s\n", string(b))
		}
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{
			Err:       err,
			Operation: fmt.Sprintf("%s %s", method, path),
			URL:       r.baseURL + path,
		}
	}

	if r.debug {
		fmt.Printf("Response Status: %s\n", resp.Status)
	}

	return resp, nil
}

// toDomainRun converts a DTO RunResponse to a domain Run
// mapAPIStatusToDomain converts API status strings to domain status constants
func mapAPIStatusToDomain(apiStatus string) string {
	switch apiStatus {
	case "DONE":
		return domain.StatusCompleted
	case "QUEUED":
		return domain.StatusQueued
	case "INITIALIZING", "PROCESSING", "POST_PROCESS":
		return domain.StatusRunning
	case "FAILED":
		return domain.StatusFailed
	case "CANCELLED":
		return domain.StatusCancelled
	case "CREATED":
		return domain.StatusCreated
	default:
		// Return the status as-is if not mapped
		return apiStatus
	}
}

func (r *apiRunRepository) toDomainRun(resp *dto.RunResponse) *domain.Run {
	return &domain.Run{
		ID:                 resp.ID.String(),
		PublicID:           resp.PublicID,
		Status:             mapAPIStatusToDomain(resp.Status),
		StatusMessage:      resp.StatusMessage,
		Prompt:             resp.Prompt,
		RepositoryName:     resp.RepositoryName,
		SourceBranch:       resp.SourceBranch,
		TargetBranch:       resp.TargetBranch,
		BaseBranch:         resp.BaseBranch,
		OutputMode:         resp.OutputMode,
		OutputBranch:       resp.OutputBranch,
		PRTargetBranch:     resp.PRTargetBranch,
		OutputBranchPolicy: resp.OutputBranchPolicy,
		PullRequestURL:     resp.PullRequestURL,
		RunType:            resp.RunType,
		Title:              resp.Title,
		Context:            resp.Context,
		Files:              resp.Files,
		UserID:             resp.UserID,
		RepositoryID:       resp.RepositoryID,
		CreatedAt:          resp.CreatedAt,
		UpdatedAt:          resp.UpdatedAt,
		CompletedAt:        resp.CompletedAt,
		Cost:               resp.Cost,
		InputTokens:        resp.InputTokens,
		OutputTokens:       resp.OutputTokens,
		FileCount:          resp.FileCount,
		FilesChanged:       resp.FilesChanged,
		Summary:            resp.Summary,
		Error:              resp.Error,
	}
}

func (r *apiRunRepository) createdRunToDomain(resp *dto.CreateRunData, req domain.CreateRunRequest) *domain.Run {
	return &domain.Run{
		ID:                 resp.ID.String(),
		PublicID:           resp.PublicID,
		Status:             mapAPIStatusToDomain(resp.Status),
		Prompt:             req.Prompt,
		RepositoryName:     req.RepositoryName,
		SourceBranch:       req.SourceBranch,
		TargetBranch:       req.TargetBranch,
		BaseBranch:         firstNonEmpty(resp.BaseBranch, req.BaseBranch, req.SourceBranch),
		OutputMode:         firstNonEmpty(resp.OutputMode, req.OutputMode),
		OutputBranch:       firstNonEmpty(resp.OutputBranch, req.OutputBranch),
		PRTargetBranch:     firstNonEmpty(resp.PRTargetBranch, req.PRTargetBranch),
		OutputBranchPolicy: firstNonEmpty(resp.OutputBranchPolicy, req.OutputBranchPolicy),
		RunType:            req.RunType,
		Title:              req.Title,
		Context:            req.Context,
		Files:              req.Files,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

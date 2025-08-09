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
		baseURL = "https://api.repobird.ai"
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
		Prompt:         req.Prompt,
		RepositoryName: req.RepositoryName,
		SourceBranch:   req.SourceBranch,
		TargetBranch:   req.TargetBranch,
		RunType:        req.RunType,
		Title:          req.Title,
		Context:        req.Context,
		Files:          req.Files,
	}

	// Make API request
	resp, err := r.doRequest(ctx, "POST", "/api/v1/runs", apiReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	if err := json.Unmarshal(body, &createResp); err == nil && createResp.Data.ID.String() != "" {
		// Convert to domain model with minimal info from create response
		return &domain.Run{
			ID:     createResp.Data.ID.String(),
			Status: createResp.Data.Status,
		}, nil
	}

	// Fall back to direct RunResponse decoding
	var runResp dto.RunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return r.toDomainRun(&runResp), nil
}

// Get retrieves a run by ID
func (r *apiRunRepository) Get(ctx context.Context, id string) (*domain.Run, error) {
	resp, err := r.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/runs/%s", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.ParseAPIError(resp.StatusCode, body)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as SingleRunResponse first
	var singleResp dto.SingleRunResponse
	if err := json.Unmarshal(body, &singleResp); err == nil && singleResp.Data != nil {
		return r.toDomainRun(singleResp.Data), nil
	}

	// Fall back to direct decoding
	var runResp dto.RunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return r.toDomainRun(&runResp), nil
}

// List retrieves a list of runs
func (r *apiRunRepository) List(ctx context.Context, opts domain.ListOptions) ([]*domain.Run, error) {
	path := fmt.Sprintf("/api/v1/runs?limit=%d&offset=%d", opts.Limit, opts.Offset)
	resp, err := r.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

// doRequest performs an HTTP request
func (r *apiRunRepository) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
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
func (r *apiRunRepository) toDomainRun(resp *dto.RunResponse) *domain.Run {
	return &domain.Run{
		ID:             resp.ID.String(),
		Status:         resp.Status,
		StatusMessage:  resp.StatusMessage,
		Prompt:         resp.Prompt,
		RepositoryName: resp.RepositoryName,
		SourceBranch:   resp.SourceBranch,
		TargetBranch:   resp.TargetBranch,
		PullRequestURL: resp.PullRequestURL,
		RunType:        resp.RunType,
		Title:          resp.Title,
		Context:        resp.Context,
		Files:          resp.Files,
		UserID:         resp.UserID,
		RepositoryID:   resp.RepositoryID,
		CreatedAt:      resp.CreatedAt,
		UpdatedAt:      resp.UpdatedAt,
		CompletedAt:    resp.CompletedAt,
		Cost:           resp.Cost,
		InputTokens:    resp.InputTokens,
		OutputTokens:   resp.OutputTokens,
		FileCount:      resp.FileCount,
		FilesChanged:   resp.FilesChanged,
		Summary:        resp.Summary,
		Error:          resp.Error,
	}
}

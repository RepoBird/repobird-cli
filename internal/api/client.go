package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/retry"
	"github.com/repobird/repobird-cli/internal/utils"
	"github.com/repobird/repobird-cli/pkg/version"
)

const (
	DefaultAPIURL  = "https://api.repobird.ai"
	DefaultTimeout = 45 * time.Minute
)

type Client struct {
	httpClient     *http.Client
	baseURL        string
	apiKey         string
	debug          bool
	retryClient    *retry.Client
	circuitBreaker *retry.CircuitBreaker
}

func NewClient(apiKey, baseURL string, debug bool) *Client {
	if baseURL == "" {
		baseURL = DefaultAPIURL
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL:        baseURL,
		apiKey:         apiKey,
		debug:          debug,
		retryClient:    retry.NewClient(retry.DefaultConfig(), debug),
		circuitBreaker: retry.NewCircuitBreaker(5, 30*time.Second),
	}
}

// GetAPIEndpoint returns the API endpoint URL
func (c *Client) GetAPIEndpoint() string {
	return c.baseURL
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("repobird-cli/%s", version.GetVersion()))

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("API request",
			"method", method,
			"url", req.URL.String(),
			"authorization", utils.RedactAuthHeader(req.Header.Get("Authorization")),
			"has_body", body != nil,
		)
		if body != nil {
			b, _ := json.MarshalIndent(body, "", "  ")
			logger.Debug("Request body", "body", string(b))
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Wrap network errors
		return nil, &errors.NetworkError{
			Err:       err,
			Operation: fmt.Sprintf("%s %s", method, path),
			URL:       c.baseURL + path,
		}
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("API response", "status", resp.Status)
	}

	return resp, nil
}

func (c *Client) CreateRun(request *models.RunRequest) (*models.RunResponse, error) {
	resp, err := c.doRequest("POST", EndpointRuns, request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOKOrCreated(resp); err != nil {
		return nil, err
	}

	var runResp models.RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &runResp, nil
}

func (c *Client) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	resp, err := c.doRequest("POST", EndpointRuns, request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOKOrCreated(resp); err != nil {
		return nil, err
	}

	// Read the response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("CreateRunAPI response", "body", string(body))
	}

	// The CreateRun API returns a wrapped response: {data: {id, message, status}}
	// We need to extract the basic info and create a RunResponse
	var createResp struct {
		Data struct {
			ID      interface{} `json:"id"`
			Message string      `json:"message"`
			Status  string      `json:"status"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &createResp); err != nil {
		// Fall back to direct RunResponse decoding for backward compatibility
		var runResp models.RunResponse
		if err := json.Unmarshal(body, &runResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &runResp, nil
	}

	// Convert the create response to RunResponse format
	var idStr string
	switch v := createResp.Data.ID.(type) {
	case string:
		idStr = v
	case float64:
		idStr = fmt.Sprintf("%.0f", v)
	case int:
		idStr = fmt.Sprintf("%d", v)
	default:
		idStr = fmt.Sprintf("%v", v)
	}

	runResp := &models.RunResponse{
		ID:     idStr,
		Status: models.RunStatus(createResp.Data.Status),
	}

	return runResp, nil
}

func (c *Client) GetRun(id string) (*models.RunResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("run ID cannot be empty")
	}
	resp, err := c.doRequest("GET", RunDetailsURL(id), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	// Read the response body for debugging if needed
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("GetRun response", "body", string(body))
	}

	// Try to decode as SingleRunResponse first (wrapped response)
	var singleResp models.SingleRunResponse
	if err := json.Unmarshal(body, &singleResp); err == nil && singleResp.Data != nil {
		return singleResp.Data, nil
	}

	// Fall back to direct decoding for backward compatibility
	var runResp models.RunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		if c.debug {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			logger.Debug("Failed to decode GetRun response", "error", err)
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &runResp, nil
}

func (c *Client) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	path := RunsListURL(limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	// Read the response body for debugging if needed
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("ListRuns response", "body", string(body))
	}

	// Try to decode as ListRunsResponse first (paginated response)
	var listResp models.ListRunsResponse
	if err := json.Unmarshal(body, &listResp); err == nil && listResp.Data != nil {
		return listResp.Data, nil
	}

	// Fall back to direct array decoding for backward compatibility
	var runs []*models.RunResponse
	if err := json.Unmarshal(body, &runs); err != nil {
		if c.debug {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			logger.Debug("Failed to decode as array", "error", err)
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return runs, nil
}

func (c *Client) VerifyAuth() (*models.UserInfo, error) {
	resp, err := c.doRequest("GET", EndpointAuthVerify, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	// Try to decode as AuthVerifyResponse first (new API format)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("VerifyAuth response", "body", string(body))
	}

	// First try the new nested structure
	var authResponse models.AuthVerifyResponse
	if err := json.Unmarshal(body, &authResponse); err == nil && authResponse.Data.User.Email != "" {
		// Successfully parsed as new format
		if c.debug {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			logger.Debug("Parsed as AuthVerifyResponse", "email", authResponse.Data.User.Email)
		}
		return authResponse.ToUserInfo(), nil
	}

	// Fall back to legacy flat structure
	var userInfo models.UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		if c.debug {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			logger.Debug("Failed to parse response", "error", err, "body", string(body))
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("Parsed as legacy UserInfo", "email", userInfo.Email)
	}

	return &userInfo, nil
}

// ListRuns with context and page-based pagination (for dashboard compatibility)
func (c *Client) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	// Convert page to offset for existing endpoint
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	path := RunsListURL(limit, offset)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("repobird-cli/%s", version.GetVersion()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{
			Err:       err,
			Operation: fmt.Sprintf("GET %s", path),
			URL:       c.baseURL + path,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as ListRunsResponse first (wrapped response)
	var listResp models.ListRunsResponse
	if err := json.Unmarshal(body, &listResp); err == nil {
		return &listResp, nil
	}

	// Fall back to direct array decoding for backward compatibility
	var runs []*models.RunResponse
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Wrap in ListRunsResponse
	return &models.ListRunsResponse{
		Data: runs,
		Metadata: &models.PaginationMetadata{
			CurrentPage: page,
			Total:       len(runs),
			TotalPages:  1, // We don't have this info from the simple response
		},
	}, nil
}

// GetUserInfo gets user information (without context for backward compatibility)
func (c *Client) GetUserInfo() (*models.UserInfo, error) {
	return c.GetUserInfoWithContext(context.Background())
}

// GetUserInfoWithContext gets user information with context
func (c *Client) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+EndpointAuthVerify, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("repobird-cli/%s", version.GetVersion()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{
			Err:       err,
			Operation: "GET " + EndpointAuthVerify,
			URL:       c.baseURL + EndpointAuthVerify,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	// Try to decode as AuthVerifyResponse first (new API format)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// First try the new nested structure
	var authResponse models.AuthVerifyResponse
	if err := json.Unmarshal(body, &authResponse); err == nil && authResponse.Data.User.Email != "" {
		// Successfully parsed as new format
		return authResponse.ToUserInfo(), nil
	}

	// Fall back to legacy flat structure
	var userInfo models.UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		// If both fail, return error with body for debugging
		if c.debug {
			return nil, fmt.Errorf("failed to decode response (body: %s): %w", string(body), err)
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userInfo, nil
}

func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var resp *http.Response

	err := c.retryClient.DoWithRetry(ctx, func() error {
		return c.circuitBreaker.Call(func() error {
			var err error
			resp, err = c.doRequest(method, path, body)
			if err != nil {
				return err
			}

			// Check if response indicates a retryable error
			if resp.StatusCode >= 500 || resp.StatusCode == 429 || resp.StatusCode == 408 {
				defer func() { _ = resp.Body.Close() }()
				bodyBytes, _ := io.ReadAll(resp.Body)
				return errors.ParseAPIError(resp.StatusCode, bodyBytes)
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) CreateRunWithRetry(ctx context.Context, request *models.RunRequest) (*models.RunResponse, error) {
	resp, err := c.doRequestWithRetry(ctx, "POST", EndpointRuns, request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOKOrCreated(resp); err != nil {
		return nil, err
	}

	var runResp models.RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &runResp, nil
}

func (c *Client) GetRunWithRetry(ctx context.Context, id string) (*models.RunResponse, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", RunDetailsURL(id), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	// Read the response body for debugging if needed
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Debug("GetRunWithRetry response", "body", string(body))
	}

	// Try to decode as SingleRunResponse first (wrapped response)
	var singleResp models.SingleRunResponse
	if err := json.Unmarshal(body, &singleResp); err == nil && singleResp.Data != nil {
		return singleResp.Data, nil
	}

	// Fall back to direct decoding for backward compatibility
	var runResp models.RunResponse
	if err := json.Unmarshal(body, &runResp); err != nil {
		if c.debug {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			logger.Debug("Failed to decode GetRunWithRetry response", "error", err)
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &runResp, nil
}

// ListRepositories retrieves a list of repositories for the authenticated user
func (c *Client) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", EndpointRepositories, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	var repoListResp models.RepositoryListResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoListResp); err != nil {
		return nil, fmt.Errorf("failed to decode repositories response: %w", err)
	}

	return repoListResp.Data, nil
}

// GetFileHashes retrieves all file hashes for the authenticated user's runs
func (c *Client) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", EndpointRunsHashes, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	var hashesResp models.FileHashesResponse
	if err := json.NewDecoder(resp.Body).Decode(&hashesResp); err != nil {
		return nil, fmt.Errorf("failed to decode file hashes response: %w", err)
	}

	return hashesResp.Data, nil
}

// CreateBulkRuns creates multiple runs in a batch
// Note: This operation may take several minutes for large batches as the server
// processes each run sequentially. The server will return 207 Multi-Status if
// some runs are still being processed.
func (c *Client) CreateBulkRuns(ctx context.Context, req *dto.BulkRunRequest) (*dto.BulkRunResponse, error) {
	// For bulk operations, we use a longer timeout as the server may need
	// several minutes to process all runs
	bulkCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	resp, err := c.doRequestWithRetry(bulkCtx, "POST", EndpointBulkRuns, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Accept 200 OK, 201 Created, or 207 Multi-Status for bulk operations
	// 207 indicates partial completion - some runs created, others pending
	if err := ValidateResponse(resp, http.StatusOK, http.StatusCreated, http.StatusMultiStatus); err != nil {
		return nil, err
	}

	var bulkResp dto.BulkRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return nil, fmt.Errorf("failed to decode bulk runs response: %w", err)
	}

	// Add status code to response for caller to handle appropriately
	bulkResp.StatusCode = resp.StatusCode

	return &bulkResp, nil
}

// GetBulkStatus retrieves the status of a bulk run batch
func (c *Client) GetBulkStatus(ctx context.Context, batchID string) (*dto.BulkStatusResponse, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", EndpointBulkRuns, batchID)
	resp, err := c.doRequestWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return nil, err
	}

	var statusResp dto.BulkStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode bulk status response: %w", err)
	}

	return &statusResp, nil
}

// CancelBulkRuns cancels all runs in a batch
func (c *Client) CancelBulkRuns(ctx context.Context, batchID string) error {
	if batchID == "" {
		return fmt.Errorf("batch ID cannot be empty")
	}

	path := fmt.Sprintf("%s/%s", EndpointBulkRuns, batchID)
	resp, err := c.doRequestWithRetry(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := ValidateResponseOK(resp); err != nil {
		return err
	}

	return nil
}

// PollBulkStatus polls for bulk status updates at the specified interval
func (c *Client) PollBulkStatus(ctx context.Context, batchID string, interval time.Duration) (<-chan dto.BulkStatusResponse, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch ID cannot be empty")
	}

	statusChan := make(chan dto.BulkStatusResponse, 1)

	go func() {
		defer close(statusChan)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status, err := c.GetBulkStatus(ctx, batchID)
				if err != nil {
					if c.debug {
						logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
						logger.Debug("Failed to get bulk status", "error", err)
					}
					continue
				}

				select {
				case statusChan <- *status:
				case <-ctx.Done():
					return
				}

				// Check if batch is complete
				if status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled" {
					return
				}
			}
		}
	}()

	return statusChan, nil
}

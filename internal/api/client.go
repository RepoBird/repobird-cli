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

func (c *Client) ListRuns(limit, offset int) ([]*models.RunResponse, error) {
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

	var userInfo models.UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
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

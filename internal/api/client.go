package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

const (
	DefaultAPIURL = "https://api.repobird.ai"
	DefaultTimeout = 45 * time.Minute
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	debug      bool
}

func NewClient(apiKey, baseURL string, debug bool) *Client {
	if baseURL == "" {
		baseURL = DefaultAPIURL
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		debug:   debug,
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

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.debug {
		fmt.Printf("Request: %s %s\n", method, req.URL.String())
		if body != nil {
			b, _ := json.MarshalIndent(body, "", "  ")
			fmt.Printf("Body: %s\n", string(b))
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if c.debug {
		fmt.Printf("Response Status: %s\n", resp.Status)
	}

	return resp, nil
}

func (c *Client) CreateRun(request *models.RunRequest) (*models.RunResponse, error) {
	resp, err := c.doRequest("POST", "/api/v1/runs", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var runResp models.RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &runResp, nil
}

func (c *Client) GetRun(id string) (*models.RunResponse, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/runs/%s", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var runResp models.RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &runResp, nil
}

func (c *Client) ListRuns(limit int, offset int) ([]*models.RunResponse, error) {
	path := fmt.Sprintf("/api/v1/runs?limit=%d&offset=%d", limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var runs []*models.RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&runs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return runs, nil
}

func (c *Client) VerifyAuth() (*models.UserInfo, error) {
	resp, err := c.doRequest("GET", "/api/v1/auth/verify", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var userInfo models.UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userInfo, nil
}
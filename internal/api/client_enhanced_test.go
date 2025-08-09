package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/models"
)

func TestClient_doRequest_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
		contentType   string
	}{
		{
			name:          "400 Bad Request with JSON",
			statusCode:    400,
			responseBody:  `{"error": {"message": "Invalid request", "code": "BAD_REQUEST"}}`,
			expectedError: "API error (status 400): Invalid request",
			contentType:   "application/json",
		},
		{
			name:          "401 Unauthorized",
			statusCode:    401,
			responseBody:  "Unauthorized access",
			expectedError: "API error (status 401): Unauthorized access",
			contentType:   "text/plain",
		},
		{
			name:          "403 Forbidden",
			statusCode:    403,
			responseBody:  `{"error": {"message": "Forbidden", "details": "Insufficient permissions"}}`,
			expectedError: "API error (status 403): Forbidden",
			contentType:   "application/json",
		},
		{
			name:          "404 Not Found",
			statusCode:    404,
			responseBody:  "Not Found",
			expectedError: "API error (status 404): Not Found",
			contentType:   "text/plain",
		},
		{
			name:          "429 Rate Limited",
			statusCode:    429,
			responseBody:  `{"error": {"message": "Rate limit exceeded", "retry_after": 60}}`,
			expectedError: "API error (status 429): Rate limit exceeded",
			contentType:   "application/json",
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    500,
			responseBody:  "Internal Server Error",
			expectedError: "API error (status 500): Internal Server Error",
			contentType:   "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)

			resp, err := client.doRequest("GET", "/test", nil)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if resp != nil {
				defer resp.Body.Close()
			}
			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

func TestClient_doRequest_DebugMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Capture debug output
	client := NewClient("test-key", server.URL, true)

	resp, err := client.doRequest("GET", "/test", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	assert.NoError(t, err)

	// Note: In real implementation, debug output would go to a logger
	// This test validates the debug parameter is stored correctly
	assert.True(t, client.debug)
}

func TestClient_CreateRun_ValidationAndEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.RunRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		validateResp   func(t *testing.T, resp *models.RunResponse)
	}{
		{
			name: "Valid run request",
			request: &models.RunRequest{
				Prompt:     "Fix bug",
				Repository: "user/repo",
				Source:     "main",
				Target:     "fix/bug",
				RunType:    models.RunTypeRun,
				Files:      []string{"file1.go", "file2.go"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var req models.RunRequest
				json.NewDecoder(r.Body).Decode(&req)

				resp := models.RunResponse{
					ID:         "run-123",
					Status:     models.StatusQueued,
					Repository: req.Repository,
					Source:     req.Source,
					Target:     req.Target,
					Prompt:     req.Prompt,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(resp)
			},
			expectError: false,
			validateResp: func(t *testing.T, resp *models.RunResponse) {
				assert.Equal(t, "run-123", resp.ID)
				assert.Equal(t, models.StatusQueued, resp.Status)
				assert.Equal(t, "user/repo", resp.Repository)
			},
		},
		{
			name: "Approval request type",
			request: &models.RunRequest{
				Prompt:     "Review changes",
				Repository: "user/repo",
				Source:     "feature",
				Target:     "main",
				RunType:    models.RunTypeApproval,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var req models.RunRequest
				json.NewDecoder(r.Body).Decode(&req)

				assert.Equal(t, models.RunTypeApproval, req.RunType)

				resp := models.RunResponse{
					ID:     "approval-123",
					Status: models.StatusQueued,
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(resp)
			},
			expectError: false,
			validateResp: func(t *testing.T, resp *models.RunResponse) {
				assert.Equal(t, "approval-123", resp.ID)
			},
		},
		{
			name: "Server validation error",
			request: &models.RunRequest{
				Prompt:     "",
				Repository: "invalid",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "Validation failed",
						"details": map[string]string{
							"prompt":     "cannot be empty",
							"repository": "invalid format",
						},
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)
			resp, err := client.CreateRun(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				if tt.validateResp != nil {
					tt.validateResp(t, resp)
				}
			}
		})
	}
}

func TestClient_GetRun_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		runID      string
		serverResp func(w http.ResponseWriter, r *http.Request)
		expectErr  bool
		validate   func(t *testing.T, resp *models.RunResponse)
	}{
		{
			name:  "Completed run with results",
			runID: "completed-123",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				resp := models.RunResponse{
					ID:          "completed-123",
					Status:      models.StatusDone,
					Title:       "Test Run",
					Description: "Run completed successfully",
					CreatedAt:   time.Now().Add(-5 * time.Minute),
					UpdatedAt:   time.Now(),
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			},
			expectErr: false,
			validate: func(t *testing.T, resp *models.RunResponse) {
				assert.Equal(t, models.StatusDone, resp.Status)
				assert.Equal(t, "Test Run", resp.Title)
				assert.NotEmpty(t, resp.Description)
			},
		},
		{
			name:  "Failed run with error details",
			runID: "failed-123",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				resp := models.RunResponse{
					ID:        "failed-123",
					Status:    models.StatusFailed,
					Title:     "Failed Run",
					Error:     "Build failed: missing dependency",
					CreatedAt: time.Now().Add(-10 * time.Minute),
					UpdatedAt: time.Now(),
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			},
			expectErr: false,
			validate: func(t *testing.T, resp *models.RunResponse) {
				assert.Equal(t, models.StatusFailed, resp.Status)
				assert.Contains(t, resp.Error, "Build failed")
			},
		},
		{
			name:  "Non-existent run",
			runID: "non-existent",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Run not found"))
			},
			expectErr: true,
		},
		{
			name:  "Empty run ID",
			runID: "",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				// Should not be called
				t.Error("Server should not be called for empty run ID")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)
			resp, err := client.GetRun(tt.runID)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

func TestClient_ListRuns_Pagination(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		offset    int
		expectErr bool
		validate  func(t *testing.T, runs []*models.RunResponse)
	}{
		{
			name:   "Default pagination",
			limit:  10,
			offset: 0,
			validate: func(t *testing.T, runs []*models.RunResponse) {
				assert.Len(t, runs, 3) // Based on server response
			},
		},
		{
			name:   "Large limit",
			limit:  100,
			offset: 0,
			validate: func(t *testing.T, runs []*models.RunResponse) {
				assert.Len(t, runs, 3)
			},
		},
		{
			name:   "With offset",
			limit:  10,
			offset: 1,
			validate: func(t *testing.T, runs []*models.RunResponse) {
				// Should still get results, server handles pagination
				assert.NotEmpty(t, runs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate query parameters
				limit := r.URL.Query().Get("limit")
				offset := r.URL.Query().Get("offset")

				assert.Equal(t, fmt.Sprintf("%d", tt.limit), limit)
				assert.Equal(t, fmt.Sprintf("%d", tt.offset), offset)

				runs := []*models.RunResponse{
					{ID: "run-1", Status: models.StatusDone, CreatedAt: time.Now()},
					{ID: "run-2", Status: models.StatusProcessing, CreatedAt: time.Now()},
					{ID: "run-3", Status: models.StatusQueued, CreatedAt: time.Now()},
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(runs)
			}))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)
			runs, err := client.ListRuns(tt.limit, tt.offset)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, runs)
				}
			}
		})
	}
}

func TestClient_VerifyAuth_UserInfoValidation(t *testing.T) {
	tests := []struct {
		name       string
		serverResp func(w http.ResponseWriter, r *http.Request)
		expectErr  bool
		validate   func(t *testing.T, userInfo *models.UserInfo)
	}{
		{
			name: "Valid user info",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				userInfo := models.UserInfo{
					Email:         "user@example.com",
					RemainingRuns: 10,
					TotalRuns:     25,
					Tier:          "pro",
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(userInfo)
			},
			expectErr: false,
			validate: func(t *testing.T, userInfo *models.UserInfo) {
				assert.Equal(t, "user@example.com", userInfo.Email)
				assert.Equal(t, 10, userInfo.RemainingRuns)
				assert.Equal(t, 25, userInfo.TotalRuns)
				assert.Equal(t, "pro", userInfo.Tier)
			},
		},
		{
			name: "Free tier user",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				userInfo := models.UserInfo{
					Email:         "free@example.com",
					RemainingRuns: 2,
					TotalRuns:     3,
					Tier:          "free",
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(userInfo)
			},
			expectErr: false,
			validate: func(t *testing.T, userInfo *models.UserInfo) {
				assert.Equal(t, "free", userInfo.Tier)
				assert.Equal(t, 2, userInfo.RemainingRuns)
			},
		},
		{
			name: "Invalid API key",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid API key",
				})
			},
			expectErr: true,
		},
		{
			name: "Malformed response",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)
			userInfo, err := client.VerifyAuth()

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, userInfo)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, userInfo)
				if tt.validate != nil {
					tt.validate(t, userInfo)
				}
			}
		})
	}
}

func TestClient_Timeouts(t *testing.T) {
	// Test that requests respect timeouts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	// Set a very short timeout on the client
	client.httpClient.Timeout = 50 * time.Millisecond

	resp, err := client.doRequest("GET", "/timeout", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	// Should timeout
	assert.Error(t, err)
	assert.Nil(t, resp)
	if err != nil {
		assert.True(t, strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded"))
	}
}

func TestClient_RequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate headers
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("User-Agent"), "repobird-cli")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)

	// Test with POST request (should have Content-Type)
	_, err := client.doRequest("POST", "/test", map[string]bool{"test": true})
	assert.NoError(t, err)
}

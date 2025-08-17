// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateBulkRuns(t *testing.T) {
	tests := []struct {
		name           string
		request        *dto.BulkRunRequest
		mockResponse   interface{}
		mockStatusCode int
		expectedError  bool
		expectedResult *dto.BulkRunResponse
	}{
		{
			name: "successful bulk run creation",
			request: &dto.BulkRunRequest{
				RepositoryName: "org/repo",
				RunType:        "run",
				SourceBranch:   "main",
				BatchTitle:     "Test Batch",
				Runs: []dto.RunItem{
					{
						Prompt: "Fix bug",
						Title:  "Bug Fix",
						Target: "fix/bug",
					},
					{
						Prompt: "Add feature",
						Title:  "New Feature",
						Target: "feature/new",
					},
				},
			},
			mockResponse: dto.BulkRunResponse{
				Data: dto.BulkRunData{
					BatchID: "batch-123",
					Successful: []dto.RunCreatedItem{
						{
							ID:             1,
							Title:          "Bug Fix",
							Status:         "queued",
							RepositoryName: "org/repo",
							RequestIndex:   0,
						},
						{
							ID:             2,
							Title:          "New Feature",
							Status:         "queued",
							RepositoryName: "org/repo",
							RequestIndex:   1,
						},
					},
					Metadata: dto.BulkResponseMetadata{
						TotalRequested:  2,
						TotalSuccessful: 2,
						TotalFailed:     0,
					},
				},
			},
			mockStatusCode: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "partial success with errors",
			request: &dto.BulkRunRequest{
				RepositoryName: "org/repo",
				RunType:        "run",
				Runs: []dto.RunItem{
					{Prompt: "Task 1"},
					{Prompt: "Task 2"},
					{Prompt: "Task 3"},
				},
			},
			mockResponse: dto.BulkRunResponse{
				Data: dto.BulkRunData{
					BatchID: "batch-456",
					Successful: []dto.RunCreatedItem{
						{
							ID:             3,
							Title:          "Task 1",
							Status:         "queued",
							RepositoryName: "org/repo",
							RequestIndex:   0,
						},
					},
					Failed: []dto.RunError{
						{
							RequestIndex: 1,
							Prompt:       "Task 2",
							Error:        "DUPLICATE_RUN",
							Message:      "Duplicate run detected",
						},
						{
							RequestIndex: 2,
							Prompt:       "Task 3",
							Error:        "INVALID_BRANCH",
							Message:      "Invalid branch name",
						},
					},
					Metadata: dto.BulkResponseMetadata{
						TotalRequested:  3,
						TotalSuccessful: 1,
						TotalFailed:     2,
					},
				},
			},
			mockStatusCode: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "authentication error",
			request: &dto.BulkRunRequest{
				RepositoryName: "org/repo",
				Runs:           []dto.RunItem{{Prompt: "Test"}},
			},
			mockResponse:   map[string]string{"error": "Invalid API key"},
			mockStatusCode: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name: "repository not found",
			request: &dto.BulkRunRequest{
				RepositoryName: "org/nonexistent",
				Runs:           []dto.RunItem{{Prompt: "Test"}},
			},
			mockResponse:   map[string]string{"error": "Repository not found"},
			mockStatusCode: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name: "quota exceeded",
			request: &dto.BulkRunRequest{
				RepositoryName: "org/repo",
				Runs: []dto.RunItem{
					{Prompt: "Task 1"},
					{Prompt: "Task 2"},
				},
			},
			mockResponse:   map[string]string{"error": "Monthly quota exceeded"},
			mockStatusCode: http.StatusTooManyRequests,
			expectedError:  true,
		},
		{
			name: "with force flag",
			request: &dto.BulkRunRequest{
				RepositoryName: "org/repo",
				Force:          true,
				Runs: []dto.RunItem{
					{
						Prompt:   "Forced task",
						FileHash: "abc123",
					},
				},
			},
			mockResponse: dto.BulkRunResponse{
				Data: dto.BulkRunData{
					BatchID: "batch-789",
					Successful: []dto.RunCreatedItem{
						{
							ID:             4,
							Title:          "Forced task",
							Status:         "queued",
							RepositoryName: "org/repo",
							RequestIndex:   0,
						},
					},
					Metadata: dto.BulkResponseMetadata{
						TotalRequested:  1,
						TotalSuccessful: 1,
						TotalFailed:     0,
					},
				},
			},
			mockStatusCode: http.StatusCreated,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/runs/bulk", r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				var reqBody dto.BulkRunRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.request.RepositoryName, reqBody.RepositoryName)
				assert.Equal(t, len(tt.request.Runs), len(reqBody.Runs))

				// Send mock response
				w.WriteHeader(tt.mockStatusCode)
				_ = json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create client with mock server
			client := NewClient("test-key", server.URL, false)

			// Execute request
			result, err := client.CreateBulkRuns(context.Background(), tt.request)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if response, ok := tt.mockResponse.(dto.BulkRunResponse); ok {
					assert.Equal(t, response.Data.BatchID, result.Data.BatchID)
					assert.Equal(t, len(response.Data.Successful), len(result.Data.Successful))
					assert.Equal(t, len(response.Data.Failed), len(result.Data.Failed))
				}
			}
		})
	}
}

func TestClient_GetBulkStatus(t *testing.T) {
	tests := []struct {
		name           string
		batchID        string
		mockResponse   interface{}
		mockStatusCode int
		expectedError  bool
	}{
		{
			name:    "successful status retrieval",
			batchID: "batch-123",
			mockResponse: dto.BulkStatusResponse{
				BatchID: "batch-123",
				Status:  "processing",
				Runs: []dto.RunStatusItem{
					{
						ID:       1,
						Title:    "Task 1",
						Status:   "completed",
						Progress: 100,
						RunURL:   "https://repobird.ai/runs/1",
					},
					{
						ID:       2,
						Title:    "Task 2",
						Status:   "processing",
						Progress: 50,
						Message:  "Analyzing code...",
					},
				},
				Statistics: dto.BulkStatistics{
					Total:      2,
					Queued:     0,
					Processing: 1,
					Completed:  1,
					Failed:     0,
					Cancelled:  0,
				},
				CreatedAt: time.Now().Add(-5 * time.Minute),
				UpdatedAt: time.Now(),
			},
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:    "batch completed",
			batchID: "batch-456",
			mockResponse: dto.BulkStatusResponse{
				BatchID: "batch-456",
				Status:  "completed",
				Runs: []dto.RunStatusItem{
					{
						ID:          1,
						Title:       "Task 1",
						Status:      "completed",
						Progress:    100,
						CompletedAt: &[]time.Time{time.Now()}[0],
					},
					{
						ID:          2,
						Title:       "Task 2",
						Status:      "failed",
						Error:       "Build failed",
						CompletedAt: &[]time.Time{time.Now()}[0],
					},
				},
				Statistics: dto.BulkStatistics{
					Total:     2,
					Completed: 1,
					Failed:    1,
				},
			},
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "batch not found",
			batchID:        "nonexistent",
			mockResponse:   map[string]string{"error": "Batch not found"},
			mockStatusCode: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:          "empty batch ID",
			batchID:       "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.batchID == "" {
				// Test empty batch ID
				client := NewClient("test-key", "http://localhost", false)
				_, err := client.GetBulkStatus(context.Background(), tt.batchID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "batch ID cannot be empty")
				return
			}

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, fmt.Sprintf("/api/v1/runs/bulk/%s", tt.batchID), r.URL.Path)

				w.WriteHeader(tt.mockStatusCode)
				_ = json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)
			result, err := client.GetBulkStatus(context.Background(), tt.batchID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if response, ok := tt.mockResponse.(dto.BulkStatusResponse); ok {
					assert.Equal(t, response.BatchID, result.BatchID)
					assert.Equal(t, response.Status, result.Status)
					assert.Equal(t, len(response.Runs), len(result.Runs))
				}
			}
		})
	}
}

func TestClient_CancelBulkRuns(t *testing.T) {
	tests := []struct {
		name           string
		batchID        string
		mockStatusCode int
		expectedError  bool
	}{
		{
			name:           "successful cancellation",
			batchID:        "batch-123",
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "batch not found",
			batchID:        "nonexistent",
			mockStatusCode: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "batch already completed",
			batchID:        "completed-batch",
			mockStatusCode: http.StatusConflict,
			expectedError:  true,
		},
		{
			name:          "empty batch ID",
			batchID:       "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.batchID == "" {
				client := NewClient("test-key", "http://localhost", false)
				err := client.CancelBulkRuns(context.Background(), tt.batchID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "batch ID cannot be empty")
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, fmt.Sprintf("/api/v1/runs/bulk/%s", tt.batchID), r.URL.Path)

				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode != http.StatusOK {
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "Error message"})
				}
			}))
			defer server.Close()

			client := NewClient("test-key", server.URL, false)
			err := client.CancelBulkRuns(context.Background(), tt.batchID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_PollBulkStatus(t *testing.T) {
	t.Run("successful polling until completion", func(t *testing.T) {
		batchID := "batch-123"
		callCount := 0

		// Mock server that returns different statuses
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, fmt.Sprintf("/api/v1/runs/bulk/%s", batchID), r.URL.Path)

			callCount++

			var response dto.BulkStatusResponse
			switch callCount {
			case 1:
				// First call: processing
				response = dto.BulkStatusResponse{
					BatchID: batchID,
					Status:  "processing",
					Statistics: dto.BulkStatistics{
						Total:      2,
						Processing: 2,
					},
				}
			case 2:
				// Second call: still processing
				response = dto.BulkStatusResponse{
					BatchID: batchID,
					Status:  "processing",
					Statistics: dto.BulkStatistics{
						Total:      2,
						Processing: 1,
						Completed:  1,
					},
				}
			default:
				// Final call: completed
				response = dto.BulkStatusResponse{
					BatchID: batchID,
					Status:  "completed",
					Statistics: dto.BulkStatistics{
						Total:     2,
						Completed: 2,
					},
				}
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, false)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		statusChan, err := client.PollBulkStatus(ctx, batchID, 100*time.Millisecond)
		require.NoError(t, err)
		require.NotNil(t, statusChan)

		// Collect status updates
		var updates []dto.BulkStatusResponse
		for status := range statusChan {
			updates = append(updates, status)
			if status.Status == "completed" {
				break
			}
		}

		// Verify we got updates
		assert.Greater(t, len(updates), 0)
		lastUpdate := updates[len(updates)-1]
		assert.Equal(t, "completed", lastUpdate.Status)
		assert.Equal(t, 2, lastUpdate.Statistics.Completed)
	})

	t.Run("polling with context cancellation", func(t *testing.T) {
		batchID := "batch-456"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := dto.BulkStatusResponse{
				BatchID: batchID,
				Status:  "processing",
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, false)

		ctx, cancel := context.WithCancel(context.Background())

		statusChan, err := client.PollBulkStatus(ctx, batchID, 100*time.Millisecond)
		require.NoError(t, err)

		// Get first update
		select {
		case status := <-statusChan:
			assert.Equal(t, "processing", status.Status)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for status")
		}

		// Cancel context
		cancel()

		// Channel should be closed
		select {
		case _, ok := <-statusChan:
			assert.False(t, ok, "channel should be closed after context cancellation")
		case <-time.After(1 * time.Second):
			t.Fatal("channel not closed after context cancellation")
		}
	})

	t.Run("polling with failed status", func(t *testing.T) {
		batchID := "batch-789"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := dto.BulkStatusResponse{
				BatchID: batchID,
				Status:  "failed",
				Statistics: dto.BulkStatistics{
					Total:  2,
					Failed: 2,
				},
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, false)

		ctx := context.Background()
		statusChan, err := client.PollBulkStatus(ctx, batchID, 100*time.Millisecond)
		require.NoError(t, err)

		// Should get failed status and then channel closes
		status := <-statusChan
		assert.Equal(t, "failed", status.Status)

		// Channel should be closed
		_, ok := <-statusChan
		assert.False(t, ok)
	})

	t.Run("empty batch ID", func(t *testing.T) {
		client := NewClient("test-key", "http://localhost", false)
		_, err := client.PollBulkStatus(context.Background(), "", 1*time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch ID cannot be empty")
	})

	t.Run("polling with API errors", func(t *testing.T) {
		batchID := "batch-error"
		callCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++

			if callCount == 1 {
				// First call fails
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Server error"})
			} else {
				// Subsequent calls succeed
				response := dto.BulkStatusResponse{
					BatchID: batchID,
					Status:  "completed",
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, false)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		statusChan, err := client.PollBulkStatus(ctx, batchID, 100*time.Millisecond)
		require.NoError(t, err)

		// Should eventually get a successful response despite initial error
		select {
		case status := <-statusChan:
			assert.Equal(t, "completed", status.Status)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for successful status after error")
		}
	})
}

// Test retry behavior for bulk operations
func TestClient_BulkRetryBehavior(t *testing.T) {
	t.Run("retry on 5xx errors", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount < 3 {
				// Fail first two attempts
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Succeed on third attempt
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(dto.BulkRunResponse{
				Data: dto.BulkRunData{
					BatchID: "batch-retry",
				},
			})
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, false)

		result, err := client.CreateBulkRuns(context.Background(), &dto.BulkRunRequest{
			RepositoryName: "org/repo",
			Runs:           []dto.RunItem{{Prompt: "Test"}},
		})

		require.NoError(t, err)
		assert.Equal(t, "batch-retry", result.Data.BatchID)
		assert.Equal(t, 3, callCount)
	})

	t.Run("retry on rate limit", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Rate limited on first attempt
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			// Succeed on second attempt
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(dto.BulkStatusResponse{
				BatchID: "batch-123",
				Status:  "processing",
			})
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, false)

		result, err := client.GetBulkStatus(context.Background(), "batch-123")

		require.NoError(t, err)
		assert.Equal(t, "batch-123", result.BatchID)
		assert.Equal(t, 2, callCount)
	})
}

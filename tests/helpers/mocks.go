package helpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
)

// MockAPIServer creates a mock API server for testing
type MockAPIServer struct {
	*httptest.Server
	responses map[string]MockResponse
}

// MockResponse represents a mock API response
type MockResponse struct {
	StatusCode int
	Body       interface{}
	Headers    map[string]string
}

// NewMockAPIServer creates a new mock API server
func NewMockAPIServer(t *testing.T) *MockAPIServer {
	t.Helper()

	mock := &MockAPIServer{
		responses: make(map[string]MockResponse),
	}

	mux := http.NewServeMux()

	// Default handlers
	mux.HandleFunc("/api/v1/runs", mock.handleRuns)
	mux.HandleFunc("/api/v1/runs/", mock.handleRunDetails)
	mux.HandleFunc("/api/v1/auth/verify", mock.handleAuthVerify)

	mock.Server = httptest.NewServer(mux)

	t.Cleanup(mock.Close)

	return mock
}

// SetResponse configures a mock response for a specific endpoint
func (m *MockAPIServer) SetResponse(method, path string, response MockResponse) {
	key := method + " " + path
	m.responses[key] = response
}

// SetRunsListResponse sets up a mock response for listing runs
func (m *MockAPIServer) SetRunsListResponse(runs []models.RunResponse) {
	m.SetResponse("GET", "/api/v1/runs", MockResponse{
		StatusCode: 200,
		Body:       runs,
	})
}

// SetCreateRunResponse sets up a mock response for creating a run
func (m *MockAPIServer) SetCreateRunResponse(run models.RunResponse) {
	m.SetResponse("POST", "/api/v1/runs", MockResponse{
		StatusCode: 201,
		Body:       run,
	})
}

// SetGetRunResponse sets up a mock response for getting a specific run
func (m *MockAPIServer) SetGetRunResponse(runID string, run models.RunResponse) {
	m.SetResponse("GET", "/api/v1/runs/"+runID, MockResponse{
		StatusCode: 200,
		Body:       run,
	})
}

// SetAuthVerifyResponse sets up a mock response for auth verification
func (m *MockAPIServer) SetAuthVerifyResponse(valid bool) {
	statusCode := 200
	if !valid {
		statusCode = 401
	}

	m.SetResponse("GET", "/api/v1/auth/verify", MockResponse{
		StatusCode: statusCode,
		Body:       map[string]bool{"valid": valid},
	})
}

func (m *MockAPIServer) handleRuns(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	response, exists := m.responses[key]

	if !exists {
		// Default responses
		switch r.Method {
		case "GET":
			response = MockResponse{
				StatusCode: 200,
				Body:       []models.RunResponse{},
			}
		case "POST":
			response = MockResponse{
				StatusCode: 201,
				Body: models.RunResponse{
					ID:     "test-run-123",
					Status: "queued",
					Title:  "Test Run",
				},
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	m.sendResponse(w, response)
}

func (m *MockAPIServer) handleRunDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.Method + " " + r.URL.Path
	response, exists := m.responses[key]

	if !exists {
		// Extract run ID from path
		runID := r.URL.Path[len("/api/v1/runs/"):]

		response = MockResponse{
			StatusCode: 200,
			Body: models.RunResponse{
				ID:     runID,
				Status: "completed",
				Title:  "Test Run " + runID,
			},
		}
	}

	m.sendResponse(w, response)
}

func (m *MockAPIServer) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.Method + " " + r.URL.Path
	response, exists := m.responses[key]

	if !exists {
		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" || auth == "Bearer invalid-key" {
			response = MockResponse{
				StatusCode: 401,
				Body:       map[string]string{"error": "Invalid API key"},
			}
		} else {
			response = MockResponse{
				StatusCode: 200,
				Body:       map[string]bool{"valid": true},
			}
		}
	}

	m.sendResponse(w, response)
}

func (m *MockAPIServer) sendResponse(w http.ResponseWriter, response MockResponse) {
	// Set headers
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	// Set content type if not already set
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Send body
	if response.Body != nil {
		if bodyStr, ok := response.Body.(string); ok {
			w.Write([]byte(bodyStr))
		} else {
			json.NewEncoder(w).Encode(response.Body)
		}
	}
}

// URL returns the mock server's URL
func (m *MockAPIServer) URL() string {
	return m.Server.URL
}

// MockErrorResponse creates a standard error response
func MockErrorResponse(statusCode int, message string) MockResponse {
	return MockResponse{
		StatusCode: statusCode,
		Body: map[string]interface{}{
			"error": map[string]interface{}{
				"message": message,
				"code":    strconv.Itoa(statusCode),
			},
		},
	}
}

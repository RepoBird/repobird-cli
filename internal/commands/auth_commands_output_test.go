package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/models"
)

// TestAuthVerifyCommand_OutputFields tests that auth verify displays all required fields
func TestAuthVerifyCommand_OutputFields(t *testing.T) {
	tests := []struct {
		name           string
		userInfo       models.UserInfo
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "Free tier user with plan runs",
			userInfo: models.UserInfo{
				Email:             "free@example.com",
				Tier:              "Free Plan v1",
				RemainingProRuns:  0,
				ProTotalRuns:      0,
				RemainingPlanRuns: 4,
				PlanTotalRuns:     5,
			},
			expectedOutput: []string{
				"✓ API key is valid",
				"Email: free@example.com",
				"Tier: Free Plan v1",
				"Runs: 0/0",
				"Plan Runs: 4/5",
			},
		},
		{
			name: "Pro tier user with both run types",
			userInfo: models.UserInfo{
				Email:             "pro@example.com",
				Tier:              "Pro",
				RemainingProRuns:  80,
				ProTotalRuns:      100,
				RemainingPlanRuns: 10,
				PlanTotalRuns:     20,
			},
			expectedOutput: []string{
				"Email: pro@example.com",
				"Tier: Pro",
				"Runs: 80/100",
				"Plan Runs: 10/20",
			},
		},
		{
			name: "Pro tier user with only pro runs",
			userInfo: models.UserInfo{
				Email:             "pro2@example.com",
				Tier:              "Professional",
				RemainingProRuns:  50,
				ProTotalRuns:      100,
				RemainingPlanRuns: 0,
				PlanTotalRuns:     0,
			},
			expectedOutput: []string{
				"Email: pro2@example.com",
				"Tier: Professional",
				"Runs: 50/100",
			},
			notExpected: []string{
				"Plan Runs:",
			},
		},
		{
			name: "Free tier with zero totals (API not sending totals)",
			userInfo: models.UserInfo{
				Email:             "support@repobird.ai",
				Tier:              "Free Plan v1",
				RemainingProRuns:  0,
				ProTotalRuns:      0,
				RemainingPlanRuns: 4,
				PlanTotalRuns:     0, // API returning 0 for total
			},
			expectedOutput: []string{
				"Email: support@repobird.ai",
				"Tier: Free Plan v1",
				"Runs: 0/0",
				"Plan Runs: 4/0", // Should still show even with 0 total
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/auth/verify", r.URL.Path)
				
				// Return flattened response format
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":                "user-123",
						"email":             tt.userInfo.Email,
						"tier":              tt.userInfo.Tier,
						"remainingProRuns":  tt.userInfo.RemainingProRuns,
						"remainingPlanRuns": tt.userInfo.RemainingPlanRuns,
						"proTotalRuns":      tt.userInfo.ProTotalRuns,
						"planTotalRuns":     tt.userInfo.PlanTotalRuns,
					},
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set up config with test API key
			cfg = &config.SecureConfig{
				Config: &config.Config{
					APIKey: "test-key",
					APIURL: server.URL,
				},
			}

			// Run verify command
			err := verifyCmd.RunE(verifyCmd, []string{})
			
			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)
			output := buf.String()

			// Check no error
			assert.NoError(t, err)

			// Check expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}

			// Check not expected output
			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, output, notExpected, "Output should not contain: %s", notExpected)
			}
		})
	}
}

// TestAuthInfoCommand_OutputFields tests that auth info displays all required fields
func TestAuthInfoCommand_OutputFields(t *testing.T) {
	tests := []struct {
		name           string
		userInfo       models.UserInfo
		expectedOutput []string
	}{
		{
			name: "Free tier user shows both run types",
			userInfo: models.UserInfo{
				Email:             "free@example.com",
				Tier:              "Free Plan v1",
				RemainingProRuns:  0,
				ProTotalRuns:      0,
				RemainingPlanRuns: 3,
				PlanTotalRuns:     5,
			},
			expectedOutput: []string{
				"Account Information:",
				"Email: free@example.com",
				"Tier: Free Plan v1",
				"Runs: 0/0",
				"Plan Runs: 3/5",
			},
		},
		{
			name: "Pro user with both run types",
			userInfo: models.UserInfo{
				Email:             "pro@example.com",
				Tier:              "Pro",
				RemainingProRuns:  90,
				ProTotalRuns:      100,
				RemainingPlanRuns: 15,
				PlanTotalRuns:     20,
			},
			expectedOutput: []string{
				"Account Information:",
				"Email: pro@example.com",
				"Tier: Pro",
				"Runs: 90/100",
				"Plan Runs: 15/20",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/auth/verify", r.URL.Path)
				
				// Return flattened response format
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":                "user-123",
						"email":             tt.userInfo.Email,
						"tier":              tt.userInfo.Tier,
						"remainingProRuns":  tt.userInfo.RemainingProRuns,
						"remainingPlanRuns": tt.userInfo.RemainingPlanRuns,
						"proTotalRuns":      tt.userInfo.ProTotalRuns,
						"planTotalRuns":     tt.userInfo.PlanTotalRuns,
					},
				}
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create temp config file
			tmpDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", tmpDir)
			
			// Set up secure config
			secureConfig := &config.SecureConfig{
				Config: &config.Config{
					APIKey: "test-key",
					APIURL: server.URL,
				},
			}
			
			// Set global config for the test
			cfg = secureConfig

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run info command
			err := infoCmd.RunE(infoCmd, []string{})
			
			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)
			output := buf.String()

			// Check no error
			assert.NoError(t, err)

			// Check expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

// TestStatusCommand_OutputFields tests that status command displays run information
func TestStatusCommand_OutputFields(t *testing.T) {
	tests := []struct {
		name           string
		userInfo       models.UserInfo
		expectedOutput []string
	}{
		{
			name: "Free tier in status command",
			userInfo: models.UserInfo{
				Email:             "free@example.com",
				Tier:              "free",
				RemainingProRuns:  0,
				ProTotalRuns:      0,
				RemainingPlanRuns: 2,
				PlanTotalRuns:     5,
			},
			expectedOutput: []string{
				"Runs: 0/0 (free tier)",
				"Plan Runs: 2/5",
			},
		},
		{
			name: "Pro tier in status command",
			userInfo: models.UserInfo{
				Email:             "pro@example.com",
				Tier:              "pro",
				RemainingProRuns:  75,
				ProTotalRuns:      100,
				RemainingPlanRuns: 0,
				PlanTotalRuns:     0,
			},
			expectedOutput: []string{
				"Runs: 75/100 (pro tier)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "auth/verify") {
					// Return user info for auth verify
					response := map[string]interface{}{
						"data": map[string]interface{}{
							"id":                "user-123",
							"email":             tt.userInfo.Email,
							"tier":              tt.userInfo.Tier,
							"remainingProRuns":  tt.userInfo.RemainingProRuns,
							"remainingPlanRuns": tt.userInfo.RemainingPlanRuns,
							"proTotalRuns":      tt.userInfo.ProTotalRuns,
							"planTotalRuns":     tt.userInfo.PlanTotalRuns,
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else if strings.Contains(r.URL.Path, "runs") {
					// Return empty runs list
					response := []models.RunResponse{}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer server.Close()

			// Set up client
			client := api.NewClient("test-key", server.URL, false)
			
			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call listRuns function
			err := listRuns(client)
			
			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)
			output := buf.String()

			// Check no error
			assert.NoError(t, err)

			// Check expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

// TestAuthLogin_OutputFields tests that auth login displays the new fields
func TestAuthLogin_OutputFields(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/auth/verify", r.URL.Path)
		
		// Return Free tier user response
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":                "user-123",
				"email":             "test@example.com",
				"tier":              "Free Plan v1",
				"remainingProRuns":  0,
				"remainingPlanRuns": 3,
				"proTotalRuns":      0,
				"planTotalRuns":     5,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create temp dir for config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Provide API key as argument to avoid interactive prompt
	cfg = &config.SecureConfig{
		Config: &config.Config{
			APIURL: server.URL,
		},
	}

	// Run login command with API key as argument
	err := loginCmd.RunE(loginCmd, []string{"test-api-key"})
	
	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin
	io.Copy(&buf, r)
	output := buf.String()

	// Check no error
	assert.NoError(t, err)

	// Check expected output for Free tier
	expectedOutputs := []string{
		"✓ API key validated and stored successfully!",
		"Email: test@example.com",
		"Tier: Free Plan v1",
		"Runs: 0/0",
		"Plan Runs: 3/5",
	}

	for _, expected := range expectedOutputs {
		assert.Contains(t, output, expected, "Login output should contain: %s", expected)
	}
}

// TestAuthCommandsFieldOrder tests that fields are displayed in correct order
func TestAuthCommandsFieldOrder(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":                "user-123",
				"email":             "order@example.com",
				"tier":              "Free Plan v1",
				"remainingProRuns":  0,
				"remainingPlanRuns": 5,
				"proTotalRuns":      0,
				"planTotalRuns":     10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set up config
	cfg = &config.SecureConfig{
		Config: &config.Config{
			APIKey: "test-key",
			APIURL: server.URL,
		},
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run verify command
	err := verifyCmd.RunE(verifyCmd, []string{})
	
	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)

	// Check that Runs appears before Plan Runs
	runsIndex := strings.Index(output, "Runs: 0/0")
	planRunsIndex := strings.Index(output, "Plan Runs: 5/10")
	
	assert.True(t, runsIndex >= 0, "Should contain 'Runs: 0/0'")
	assert.True(t, planRunsIndex >= 0, "Should contain 'Plan Runs: 5/10'")
	assert.True(t, runsIndex < planRunsIndex, "Runs should appear before Plan Runs")
}

// TestFreeTierAlwaysShowsBothRunTypes verifies Free tier always shows both run types
func TestFreeTierAlwaysShowsBothRunTypes(t *testing.T) {
	testCases := []struct {
		name     string
		tier     string
		proRuns  int
		planRuns int
		expectBoth bool
	}{
		{"Free tier with no runs", "Free Plan v1", 0, 0, true},
		{"Free tier with plan runs only", "free", 0, 5, true},
		{"Free tier with pro runs only", "FREE", 2, 0, true},
		{"Pro tier with both", "Pro", 10, 5, true},
		{"Pro tier with pro only", "Professional", 10, 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":                "user-123",
						"email":             "test@example.com",
						"tier":              tc.tier,
						"remainingProRuns":  tc.proRuns,
						"remainingPlanRuns": tc.planRuns,
						"proTotalRuns":      100,
						"planTotalRuns":     0,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			cfg = &config.Config{
				APIKey: "test-key",
				APIURL: server.URL,
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := verifyCmd.RunE(verifyCmd, []string{})
			
			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)
			output := buf.String()

			assert.NoError(t, err)

			// Check if both run types are shown
			hasRuns := strings.Contains(output, "Runs:")
			hasPlanRuns := strings.Contains(output, "Plan Runs:")

			if tc.expectBoth {
				assert.True(t, hasRuns && hasPlanRuns, 
					"Tier %s should show both Runs and Plan Runs", tc.tier)
			} else {
				assert.True(t, hasRuns, "Should always show Runs")
				// Pro tier with 0 plan runs shouldn't show Plan Runs line
				if tc.planRuns == 0 && !strings.Contains(strings.ToLower(tc.tier), "free") {
					assert.False(t, hasPlanRuns, "Pro tier with 0 plan runs shouldn't show Plan Runs")
				}
			}
		})
	}
}
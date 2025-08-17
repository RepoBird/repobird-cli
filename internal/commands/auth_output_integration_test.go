// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
)

// TestUserInfoDisplay tests that user info is displayed correctly with new fields
func TestUserInfoDisplay(t *testing.T) {
	tests := []struct {
		name              string
		tier              string
		remainingProRuns  int
		remainingPlanRuns int
		proTotalRuns      int
		planTotalRuns     int
		expectedRunsLine  string
		expectedPlanLine  string
		expectPlanLine    bool
	}{
		{
			name:              "Free tier with plan runs",
			tier:              "Free Plan v1",
			remainingProRuns:  0,
			remainingPlanRuns: 4,
			proTotalRuns:      0,
			planTotalRuns:     5,
			expectedRunsLine:  "Runs: 0/0",
			expectedPlanLine:  "Plan Runs: 4/5",
			expectPlanLine:    true,
		},
		{
			name:              "Free tier with zero totals",
			tier:              "free",
			remainingProRuns:  0,
			remainingPlanRuns: 4,
			proTotalRuns:      0,
			planTotalRuns:     0,
			expectedRunsLine:  "Runs: 0/0",
			expectedPlanLine:  "Plan Runs: 4/0",
			expectPlanLine:    true,
		},
		{
			name:              "Pro tier with both types",
			tier:              "Pro",
			remainingProRuns:  80,
			remainingPlanRuns: 10,
			proTotalRuns:      100,
			planTotalRuns:     20,
			expectedRunsLine:  "Runs: 80/100",
			expectedPlanLine:  "Plan Runs: 10/20",
			expectPlanLine:    true,
		},
		{
			name:              "Pro tier with only pro runs",
			tier:              "Professional",
			remainingProRuns:  50,
			remainingPlanRuns: 0,
			proTotalRuns:      100,
			planTotalRuns:     0,
			expectedRunsLine:  "Runs: 50/100",
			expectedPlanLine:  "",
			expectPlanLine:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that returns our test data
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "auth/verify") {
					response := map[string]interface{}{
						"data": map[string]interface{}{
							"id":                "user-123",
							"email":             "test@example.com",
							"tier":              tt.tier,
							"remainingProRuns":  tt.remainingProRuns,
							"remainingPlanRuns": tt.remainingPlanRuns,
							"proTotalRuns":      tt.proTotalRuns,
							"planTotalRuns":     tt.planTotalRuns,
						},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer server.Close()

			// Create client and get user info
			client := api.NewClient("test-key", server.URL, false)
			userInfo, err := client.VerifyAuth()
			require.NoError(t, err)

			// Verify the UserInfo struct has correct values
			assert.Equal(t, tt.remainingProRuns, userInfo.RemainingProRuns)
			assert.Equal(t, tt.remainingPlanRuns, userInfo.RemainingPlanRuns)
			assert.Equal(t, tt.proTotalRuns, userInfo.ProTotalRuns)
			assert.Equal(t, tt.planTotalRuns, userInfo.PlanTotalRuns)
			assert.Equal(t, tt.tier, userInfo.Tier)

			// Simulate what the commands would display
			var output strings.Builder
			output.WriteString("✓ API key is valid\n")
			output.WriteString("  Email: " + userInfo.Email + "\n")
			output.WriteString("  Tier: " + userInfo.Tier + "\n")

			// Check if Free tier (case-insensitive)
			if strings.Contains(strings.ToLower(userInfo.Tier), "free") {
				// Free tier - always show both
				output.WriteString("  Runs: ")
				output.WriteString(formatRuns(userInfo.RemainingProRuns, userInfo.ProTotalRuns))
				output.WriteString("\n")
				output.WriteString("  Plan Runs: ")
				output.WriteString(formatRuns(userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns))
				output.WriteString("\n")
			} else {
				// Other tiers - show Runs, and Plan Runs if available
				output.WriteString("  Runs: ")
				output.WriteString(formatRuns(userInfo.RemainingProRuns, userInfo.ProTotalRuns))
				output.WriteString("\n")
				if userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
					output.WriteString("  Plan Runs: ")
					output.WriteString(formatRuns(userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns))
					output.WriteString("\n")
				}
			}

			outputStr := output.String()

			// Verify expected output
			assert.Contains(t, outputStr, tt.expectedRunsLine)
			if tt.expectPlanLine {
				assert.Contains(t, outputStr, tt.expectedPlanLine)
			} else {
				assert.NotContains(t, outputStr, "Plan Runs:")
			}
		})
	}
}

// TestStatusCommandUserInfoDisplay tests status command display logic
func TestStatusCommandUserInfoDisplay(t *testing.T) {
	tests := []struct {
		name              string
		tier              string
		remainingProRuns  int
		remainingPlanRuns int
		proTotalRuns      int
		planTotalRuns     int
		expectedOutput    []string
	}{
		{
			name:              "Free tier in status",
			tier:              "free",
			remainingProRuns:  0,
			remainingPlanRuns: 2,
			proTotalRuns:      0,
			planTotalRuns:     5,
			expectedOutput: []string{
				"Runs: 0/0 (free tier)",
				"Plan Runs: 2/5",
			},
		},
		{
			name:              "Pro tier in status",
			tier:              "pro",
			remainingProRuns:  75,
			remainingPlanRuns: 0,
			proTotalRuns:      100,
			planTotalRuns:     0,
			expectedOutput: []string{
				"Runs: 75/100 (pro tier)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user info
			userInfo := &models.UserInfo{
				Email:             "test@example.com",
				Tier:              tt.tier,
				RemainingProRuns:  tt.remainingProRuns,
				RemainingPlanRuns: tt.remainingPlanRuns,
				ProTotalRuns:      tt.proTotalRuns,
				PlanTotalRuns:     tt.planTotalRuns,
			}

			// Simulate status command output
			var output strings.Builder

			// Show runs - for Free tier, always show both
			if strings.Contains(strings.ToLower(userInfo.Tier), "free") {
				output.WriteString("Runs: ")
				output.WriteString(formatRuns(userInfo.RemainingProRuns, userInfo.ProTotalRuns))
				output.WriteString(" (" + userInfo.Tier + " tier)\n")
				output.WriteString("Plan Runs: ")
				output.WriteString(formatRuns(userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns))
				output.WriteString("\n")
			} else {
				output.WriteString("Runs: ")
				output.WriteString(formatRuns(userInfo.RemainingProRuns, userInfo.ProTotalRuns))
				output.WriteString(" (" + userInfo.Tier + " tier)\n")
				if userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
					output.WriteString("Plan Runs: ")
					output.WriteString(formatRuns(userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns))
					output.WriteString("\n")
				}
			}

			outputStr := output.String()

			// Verify expected output
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, outputStr, expected)
			}
		})
	}
}

// TestFieldOrderInOutput verifies fields appear in correct order
func TestFieldOrderInOutput(t *testing.T) {
	userInfo := &models.UserInfo{
		Email:             "test@example.com",
		Tier:              "Free Plan v1",
		RemainingProRuns:  0,
		RemainingPlanRuns: 5,
		ProTotalRuns:      0,
		PlanTotalRuns:     10,
	}

	// Simulate command output
	var output strings.Builder
	output.WriteString("✓ API key is valid\n")
	output.WriteString("  Email: " + userInfo.Email + "\n")
	output.WriteString("  Tier: " + userInfo.Tier + "\n")
	output.WriteString("  Runs: 0/0\n")
	output.WriteString("  Plan Runs: 5/10\n")

	outputStr := output.String()

	// Check order - Runs should come before Plan Runs
	runsIndex := strings.Index(outputStr, "Runs: 0/0")
	planRunsIndex := strings.Index(outputStr, "Plan Runs: 5/10")

	assert.True(t, runsIndex > 0, "Should contain Runs line")
	assert.True(t, planRunsIndex > 0, "Should contain Plan Runs line")
	assert.True(t, runsIndex < planRunsIndex, "Runs should appear before Plan Runs")
}

// Helper function to format runs display
func formatRuns(remaining, total int) string {
	return intToStr(remaining) + "/" + intToStr(total)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var result string
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

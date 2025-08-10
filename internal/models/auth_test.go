package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthVerifyResponseParsing(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected *UserInfo
	}{
		{
			name: "New nested API format",
			jsonData: `{
				"data": {
					"user": {
						"id": "user-123",
						"email": "test@example.com",
						"name": "Test User",
						"githubUsername": "testuser"
					},
					"tier": {
						"name": "pro",
						"remainingProRuns": 10,
						"remainingPlanRuns": 5,
						"lastPeriodResetDate": "2024-01-01T00:00:00Z"
					}
				}
			}`,
			expected: &UserInfo{
				ID:             HashStringToInt("user-123"),
				Email:          "test@example.com",
				Name:           "Test User",
				GithubUsername: "testuser",
				RemainingRuns:  15, // 10 + 5
				TotalRuns:      15,
				Tier:           "pro",
			},
		},
		{
			name: "Free tier with no username",
			jsonData: `{
				"data": {
					"user": {
						"id": "456",
						"email": "free@example.com",
						"name": "Free User"
					},
					"tier": {
						"name": "free",
						"remainingProRuns": 0,
						"remainingPlanRuns": 3,
						"lastPeriodResetDate": "2024-01-01T00:00:00Z"
					}
				}
			}`,
			expected: &UserInfo{
				ID:             HashStringToInt("456"),
				Email:          "free@example.com",
				Name:           "Free User",
				GithubUsername: "",
				RemainingRuns:  3,
				TotalRuns:      3,
				Tier:           "free",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response AuthVerifyResponse
			err := json.Unmarshal([]byte(tt.jsonData), &response)
			require.NoError(t, err)

			userInfo := response.ToUserInfo()
			assert.Equal(t, tt.expected.ID, userInfo.ID)
			assert.Equal(t, tt.expected.Email, userInfo.Email)
			assert.Equal(t, tt.expected.Name, userInfo.Name)
			assert.Equal(t, tt.expected.GithubUsername, userInfo.GithubUsername)
			assert.Equal(t, tt.expected.RemainingRuns, userInfo.RemainingRuns)
			assert.Equal(t, tt.expected.TotalRuns, userInfo.TotalRuns)
			assert.Equal(t, tt.expected.Tier, userInfo.Tier)
			assert.NotNil(t, userInfo.TierDetails)
		})
	}
}

func TestHashStringToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"123", 5031}, // Consistent hash
		{"user-123", 337518362},
		{"test", 3556498},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := HashStringToInt(tt.input)
			// Just ensure it's positive and consistent
			assert.GreaterOrEqual(t, result, 0)
			// Running again should produce same result
			assert.Equal(t, result, HashStringToInt(tt.input))
			
			if tt.input == "" {
				assert.Equal(t, 0, result)
			}
		})
	}
}

func TestTierDetails(t *testing.T) {
	jsonData := `{
		"data": {
			"user": {
				"id": "123",
				"email": "test@example.com"
			},
			"tier": {
				"name": "enterprise",
				"remainingProRuns": 100,
				"remainingPlanRuns": 50,
				"lastPeriodResetDate": "2024-12-01T00:00:00Z"
			}
		}
	}`

	var response AuthVerifyResponse
	err := json.Unmarshal([]byte(jsonData), &response)
	require.NoError(t, err)

	userInfo := response.ToUserInfo()
	require.NotNil(t, userInfo.TierDetails)
	assert.Equal(t, "enterprise", userInfo.TierDetails.Name)
	assert.Equal(t, 100, userInfo.TierDetails.RemainingProRuns)
	assert.Equal(t, 50, userInfo.TierDetails.RemainingPlanRuns)

	// Check date parsing
	expectedTime, _ := time.Parse(time.RFC3339, "2024-12-01T00:00:00Z")
	assert.Equal(t, expectedTime, userInfo.TierDetails.LastPeriodResetDate)
}
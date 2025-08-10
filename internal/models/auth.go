package models

import "time"

// AuthVerifyResponse represents the full response from /api/v1/auth/verify
type AuthVerifyResponse struct {
	Data struct {
		User User `json:"user"`
		Tier Tier `json:"tier"`
	} `json:"data"`
}

// User represents user information from the API
type User struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	GithubUsername string `json:"githubUsername"`
}

// Tier represents subscription tier information
type Tier struct {
	Name                string    `json:"name"` // free, pro, enterprise
	RemainingProRuns    int       `json:"remainingProRuns"`
	RemainingPlanRuns   int       `json:"remainingPlanRuns"`
	LastPeriodResetDate time.Time `json:"lastPeriodResetDate"`
}

// ToUserInfo converts AuthVerifyResponse to the legacy UserInfo structure
func (a *AuthVerifyResponse) ToUserInfo() *UserInfo {
	// Parse ID as int if possible, otherwise use 0
	var userID int
	if a.Data.User.ID != "" {
		// Try to parse as int, but don't fail if it's not
		// The ID might be a string UUID in the new API
		userID = HashStringToInt(a.Data.User.ID)
	}

	// Calculate total remaining runs
	remainingRuns := a.Data.Tier.RemainingProRuns + a.Data.Tier.RemainingPlanRuns

	return &UserInfo{
		ID:             userID,
		Email:          a.Data.User.Email,
		Name:           a.Data.User.Name,
		GithubUsername: a.Data.User.GithubUsername,
		RemainingRuns:  remainingRuns,
		TotalRuns:      remainingRuns, // This might need adjustment based on actual API
		Tier:           a.Data.Tier.Name,
		TierDetails:    &a.Data.Tier,
	}
}

// HashStringToInt creates a stable int hash from a string ID
// Used for backward compatibility when ID is a string
func HashStringToInt(s string) int {
	if s == "" {
		return 0
	}

	// Simple hash function for consistent int from string
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}

	// Ensure positive value
	if hash < 0 {
		hash = -hash
	}

	return hash
}

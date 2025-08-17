// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"encoding/json"
	"time"
)

// AuthVerifyResponse represents the full response from /api/v1/auth/verify
// Supports both flattened (new) and nested (legacy) API response formats
type AuthVerifyResponse struct {
	Data AuthVerifyData `json:"data"`
}

// AuthVerifyData holds the actual data from the auth verify response
type AuthVerifyData struct {
	// Flattened fields (new API format)
	ID             string `json:"id,omitempty"`
	Email          string `json:"email,omitempty"`
	Name           string `json:"name,omitempty"`
	GithubUsername string `json:"githubUsername,omitempty"`

	// Tier info
	TierName          string // Will be populated from either format
	RemainingProRuns  int    `json:"remainingProRuns"`
	RemainingPlanRuns int    `json:"remainingPlanRuns"`
	ProTotalRuns      int    `json:"proTotalRuns"`
	PlanTotalRuns     int    `json:"planTotalRuns"`

	// Nested structures (legacy API format)
	User    *User `json:"user,omitempty"`
	TierObj *Tier // Will be populated from tier object if present
}

// UnmarshalJSON custom unmarshaller to handle both formats
func (a *AuthVerifyData) UnmarshalJSON(data []byte) error {
	// First try to unmarshal into a generic map to detect format
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Helper struct for the simple fields
	type Alias AuthVerifyData
	aux := (*Alias)(a)

	// Unmarshal the basic fields
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Handle tier field which can be either string or object
	if tierData, ok := raw["tier"]; ok {
		// Try to unmarshal as string first (new format)
		var tierStr string
		if err := json.Unmarshal(tierData, &tierStr); err == nil {
			a.TierName = tierStr
		} else {
			// Try to unmarshal as object (legacy format)
			var tierObj Tier
			if err := json.Unmarshal(tierData, &tierObj); err == nil {
				a.TierObj = &tierObj
				a.TierName = tierObj.Name
				// Always use values from tier object in legacy format
				a.RemainingProRuns = tierObj.RemainingProRuns
				a.RemainingPlanRuns = tierObj.RemainingPlanRuns
				a.ProTotalRuns = tierObj.ProTotalRuns
				a.PlanTotalRuns = tierObj.PlanTotalRuns
			}
		}
	}

	return nil
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
	ProTotalRuns        int       `json:"proTotalRuns"`  // Total pro runs in tier
	PlanTotalRuns       int       `json:"planTotalRuns"` // Total plan runs in tier
	LastPeriodResetDate time.Time `json:"lastPeriodResetDate"`
}

// ToUserInfo converts AuthVerifyResponse to the legacy UserInfo structure
func (a *AuthVerifyResponse) ToUserInfo() *UserInfo {
	// Determine which format we're dealing with
	var id, email, name, githubUsername string
	var remainingProRuns, remainingPlanRuns, proTotalRuns, planTotalRuns int
	var tierName string

	// Check if we have nested user data (legacy format)
	if a.Data.User != nil {
		id = a.Data.User.ID
		email = a.Data.User.Email
		name = a.Data.User.Name
		githubUsername = a.Data.User.GithubUsername
	} else {
		// Use flattened fields (new format)
		id = a.Data.ID
		email = a.Data.Email
		name = a.Data.Name
		githubUsername = a.Data.GithubUsername
	}

	// Get tier information
	tierName = a.Data.TierName
	remainingProRuns = a.Data.RemainingProRuns
	remainingPlanRuns = a.Data.RemainingPlanRuns
	proTotalRuns = a.Data.ProTotalRuns
	planTotalRuns = a.Data.PlanTotalRuns

	// Parse ID as int if possible
	var userID int
	if id != "" {
		userID = HashStringToInt(id)
	}

	// Calculate totals for backward compatibility
	remainingRuns := remainingProRuns + remainingPlanRuns
	totalRuns := proTotalRuns + planTotalRuns

	// Create TierDetails
	var tierDetails *Tier
	if a.Data.TierObj != nil {
		tierDetails = a.Data.TierObj
	} else if tierName != "" {
		tierDetails = &Tier{
			Name:              tierName,
			RemainingProRuns:  remainingProRuns,
			RemainingPlanRuns: remainingPlanRuns,
			ProTotalRuns:      proTotalRuns,
			PlanTotalRuns:     planTotalRuns,
		}
	}

	return &UserInfo{
		ID:                userID,
		StringID:          id,
		Email:             email,
		Name:              name,
		GithubUsername:    githubUsername,
		RemainingRuns:     remainingRuns, // Deprecated
		TotalRuns:         totalRuns,     // Deprecated
		RemainingProRuns:  remainingProRuns,
		RemainingPlanRuns: remainingPlanRuns,
		ProTotalRuns:      proTotalRuns,
		PlanTotalRuns:     planTotalRuns,
		Tier:              tierName,
		TierDetails:       tierDetails,
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

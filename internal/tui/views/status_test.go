// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestStatusFieldsUseCreditsWhenAvailable(t *testing.T) {
	view := NewStatusView(nil)
	view.userInfo = &models.UserInfo{
		Email:            "test@example.com",
		Tier:             "pro",
		RemainingProRuns: 0,
		ProTotalRuns:     0,
		CreditBalance: &models.CreditBalance{
			AvailableCredits: 12.345,
			ReservedCredits:  1.25,
		},
	}

	view.initializeStatusFields()

	assert.Contains(t, view.statusKeys, "Credits:")
	assert.Contains(t, view.statusFields, "12.35 available (1.25 reserved)")
	assert.NotContains(t, view.statusKeys, "Runs:")
	assert.NotContains(t, view.statusFields, "0/0")
}

func TestStatusFieldsDoNotShowZeroRunQuotaWithoutCredits(t *testing.T) {
	view := NewStatusView(nil)
	view.userInfo = &models.UserInfo{
		Email: "test@example.com",
		Tier:  "free",
	}

	view.initializeStatusFields()

	assert.Contains(t, view.statusKeys, "Credits:")
	assert.Contains(t, view.statusFields, "unavailable")
	assert.NotContains(t, view.statusFields, "0/0")
}

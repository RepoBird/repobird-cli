// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

func printAccountUsage(userInfo *models.UserInfo) {
	if userInfo.CreditBalance != nil {
		fmt.Printf("  Credits: %d available", userInfo.CreditBalance.AvailableCredits)
		if userInfo.CreditBalance.ReservedCredits > 0 {
			fmt.Printf(" (%d reserved)", userInfo.CreditBalance.ReservedCredits)
		}
		fmt.Println()
		return
	}

	fmt.Printf("  Runs: %d/%d\n", userInfo.RemainingProRuns, userInfo.ProTotalRuns)
	if userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
		fmt.Printf("  Plan Runs: %d/%d\n", userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
	}
}

func printAccountReset(userInfo *models.UserInfo) {
	if userInfo.LastPeriodResetDate != nil {
		fmt.Printf("  Resets: %s\n", userInfo.LastPeriodResetDate.Format("2006-01-02"))
		return
	}

	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	resetDate := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	fmt.Printf("  Resets: %s\n", resetDate.Format("2006-01-02"))
}

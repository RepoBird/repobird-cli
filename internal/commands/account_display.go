// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

func printAccountUsage(userInfo *models.UserInfo) {
	fmt.Print(formatAccountUsage(userInfo, "  ", ""))
}

func printStatusAccountUsage(userInfo *models.UserInfo) {
	tierSuffix := ""
	if userInfo.Tier != "" {
		tierSuffix = fmt.Sprintf(" (%s tier)", userInfo.Tier)
	}
	fmt.Print(formatAccountUsage(userInfo, "", tierSuffix))
}

func formatAccountUsage(userInfo *models.UserInfo, indent, tierSuffix string) string {
	if userInfo.CreditBalance != nil {
		line := fmt.Sprintf("%sCredits: %s available", indent, models.FormatCredits(userInfo.CreditBalance.AvailableCredits))
		if userInfo.CreditBalance.ReservedCredits > 0 {
			line += fmt.Sprintf(" (%s reserved)", models.FormatCredits(userInfo.CreditBalance.ReservedCredits))
		}
		return line + tierSuffix + "\n"
	}

	if !hasLegacyRunUsage(userInfo) {
		return fmt.Sprintf("%sCredits: unavailable%s\n", indent, tierSuffix)
	}

	text := fmt.Sprintf("%sRuns: %d/%d%s\n", indent, userInfo.RemainingProRuns, userInfo.ProTotalRuns, tierSuffix)
	if userInfo.PlanTotalRuns > 0 || userInfo.RemainingPlanRuns > 0 {
		text += fmt.Sprintf("%sPlan Runs: %d/%d\n", indent, userInfo.RemainingPlanRuns, userInfo.PlanTotalRuns)
	}
	return text
}

func hasLegacyRunUsage(userInfo *models.UserInfo) bool {
	return userInfo.RemainingProRuns != 0 ||
		userInfo.ProTotalRuns != 0 ||
		userInfo.RemainingPlanRuns != 0 ||
		userInfo.PlanTotalRuns != 0 ||
		userInfo.RemainingRuns != 0 ||
		userInfo.TotalRuns != 0
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

package utils

import (
	"fmt"
	"time"
)

// FormatTimeAgo formats a time as a human-readable "ago" string
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
}

// FormatDurationLong formats a duration in a detailed human-readable format
func FormatDurationLong(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}

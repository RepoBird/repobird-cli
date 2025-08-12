package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// initializeStatusInfoFields initializes the selectable fields for the status info overlay
func (d *DashboardView) initializeStatusInfoFields() {
	d.statusInfoFields = []string{}
	d.statusInfoFieldLines = []int{}
	d.statusInfoKeys = []string{}
	d.statusInfoSelectedRow = 0

	lineNum := 0

	// User Info fields
	if d.userInfo != nil {
		if d.userInfo.Name != "" {
			d.statusInfoKeys = append(d.statusInfoKeys, "Name:")
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.Name)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}
		if d.userInfo.Email != "" {
			d.statusInfoKeys = append(d.statusInfoKeys, "Email:")
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.Email)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}
		if d.userInfo.GithubUsername != "" {
			d.statusInfoKeys = append(d.statusInfoKeys, "GitHub:")
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.GithubUsername)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}

		// Plan info
		lineNum++ // Skip one line for Plan section
		tierDisplay := strings.Title(strings.ToLower(d.userInfo.PlanTier))
		if tierDisplay == "" {
			tierDisplay = "Basic"
		}
		d.statusInfoKeys = append(d.statusInfoKeys, "Account Tier:")
		d.statusInfoFields = append(d.statusInfoFields, tierDisplay)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++

		// Usage info based on plan type
		if d.userInfo.PlanTier == "FREE" || d.userInfo.PlanTier == "BASIC" {
			// Show runs remaining for usage-based plans
			var runsRemaining string
			if d.userInfo.Usage.TotalCalls > 0 {
				remaining := d.userInfo.Usage.TotalCalls - d.userInfo.Usage.UsedCalls
				if remaining < 0 {
					remaining = 0
				}
				runsRemaining = fmt.Sprintf("%d / %d", remaining, d.userInfo.Usage.TotalCalls)
			} else {
				runsRemaining = "Unknown"
			}
			d.statusInfoKeys = append(d.statusInfoKeys, "Runs Remaining:")
			d.statusInfoFields = append(d.statusInfoFields, runsRemaining)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++

			// Also show percentage usage if we have the data
			if d.userInfo.Usage.TotalCalls > 0 {
				percentage := float64(d.userInfo.Usage.UsedCalls) / float64(d.userInfo.Usage.TotalCalls) * 100

				var usageValue string
				if percentage >= 90 {
					usageValue = fmt.Sprintf("%.1f%% ⚠️", percentage)
				} else if percentage >= 75 {
					usageValue = fmt.Sprintf("%.1f%% ⚡", percentage)
				} else {
					usageValue = fmt.Sprintf("%.1f%% ✅", percentage)
				}

				d.statusInfoKeys = append(d.statusInfoKeys, "Usage:")
				d.statusInfoFields = append(d.statusInfoFields, usageValue)
				d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
				lineNum++
			}
		} else if d.userInfo.PlanTier == "PRO" {
			// Show percentage for PRO plans
			if d.userInfo.Usage.TotalCalls > 0 {
				percentage := float64(d.userInfo.Usage.UsedCalls) / float64(d.userInfo.Usage.TotalCalls) * 100
				d.statusInfoKeys = append(d.statusInfoKeys, "Usage:")
				d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%.1f%%", percentage))
				d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
				lineNum++
			} else {
				d.statusInfoKeys = append(d.statusInfoKeys, "Usage:")
				d.statusInfoFields = append(d.statusInfoFields, "Unlimited")
				d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
				lineNum++
			}
		}
	}

	// System info
	lineNum++ // Skip one line for System section
	d.statusInfoKeys = append(d.statusInfoKeys, "Repositories:")
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%d", len(d.repositories)))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++
	d.statusInfoKeys = append(d.statusInfoKeys, "Total Runs:")
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%d", len(d.allRuns)))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	// Run status breakdown
	var running, completed, failed int
	for _, run := range d.allRuns {
		switch run.Status {
		case "RUNNING", "PENDING":
			running++
		case "DONE":
			completed++
		case "FAILED", "CANCELLED":
			failed++
		}
	}
	d.statusInfoKeys = append(d.statusInfoKeys, "Run Status:")
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("🔄 %d  ✅ %d  ❌ %d", running, completed, failed))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	// Last refresh time if available
	if !d.lastDataRefresh.IsZero() {
		refreshText := fmt.Sprintf("%s ago", time.Since(d.lastDataRefresh).Truncate(time.Second))
		d.statusInfoKeys = append(d.statusInfoKeys, "Last Refresh:")
		d.statusInfoFields = append(d.statusInfoFields, refreshText)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++
	}

	// API connection info
	lineNum++ // Skip one line for API section
	d.statusInfoKeys = append(d.statusInfoKeys, "API Endpoint:")
	d.statusInfoFields = append(d.statusInfoFields, d.client.GetAPIEndpoint())
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++
	d.statusInfoKeys = append(d.statusInfoKeys, "Status:")
	d.statusInfoFields = append(d.statusInfoFields, "Connected ✅")
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)

	// Ensure we have at least one field selected
	if len(d.statusInfoFields) > 0 {
		d.statusInfoSelectedRow = 0
	}
}

// handleStatusInfoNavigation handles navigation within the status info overlay
func (d *DashboardView) handleStatusInfoNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyDown, tea.KeyRunes:
		if msg.String() == "j" {
			if d.statusInfoSelectedRow < len(d.statusInfoFields)-1 {
				d.statusInfoSelectedRow++
				// Reset horizontal scroll when moving to a new row
				d.statusInfoKeyOffset = 0
				d.statusInfoValueOffset = 0
			}
		}
	case tea.KeyUp:
		if msg.String() == "k" || msg.Type == tea.KeyUp {
			if d.statusInfoSelectedRow > 0 {
				d.statusInfoSelectedRow--
				// Reset horizontal scroll when moving to a new row
				d.statusInfoKeyOffset = 0
				d.statusInfoValueOffset = 0
			}
		}
	case tea.KeyLeft:
		if d.statusInfoFocusColumn == 1 {
			// Move from value column to key column
			d.statusInfoFocusColumn = 0
		} else {
			// Scroll key column left
			if d.statusInfoKeyOffset > 0 {
				d.statusInfoKeyOffset--
			}
		}
	case tea.KeyRight:
		if d.statusInfoFocusColumn == 0 {
			// Move from key column to value column
			d.statusInfoFocusColumn = 1
		} else {
			// Scroll value column right
			if d.statusInfoSelectedRow >= 0 && d.statusInfoSelectedRow < len(d.statusInfoFields) {
				value := d.statusInfoFields[d.statusInfoSelectedRow]
				valueMaxWidth := 40 // Available width for value column

				// Debug logging
				debug.LogToFilef("DEBUG: StatusInfo scroll check - Row %d, Value len=%d, MaxWidth=%d, Offset=%d\n",
					d.statusInfoSelectedRow, len(value), valueMaxWidth, d.statusInfoValueOffset)

				// Only scroll if there's more content to show
				if len(value) > d.statusInfoValueOffset+valueMaxWidth {
					d.statusInfoValueOffset++
					debug.LogToFilef("DEBUG: Scrolling value to offset %d\n", d.statusInfoValueOffset)
				}
			}
		}
	case tea.KeyRunes:
		if msg.String() == "g" {
			d.statusInfoSelectedRow = 0
		} else if msg.String() == "G" {
			if len(d.statusInfoFields) > 0 {
				d.statusInfoSelectedRow = len(d.statusInfoFields) - 1
			}
		} else if msg.String() == "y" {
			// Copy current field to clipboard
			if d.statusInfoSelectedRow >= 0 && d.statusInfoSelectedRow < len(d.statusInfoFields) {
				var textToCopy string
				if d.statusInfoFocusColumn == 0 && d.statusInfoSelectedRow < len(d.statusInfoKeys) {
					// Copy the key (without the colon)
					textToCopy = strings.TrimSuffix(d.statusInfoKeys[d.statusInfoSelectedRow], ":")
				} else {
					// Copy the value
					textToCopy = d.statusInfoFields[d.statusInfoSelectedRow]
				}

				if err := d.copyToClipboard(textToCopy); err == nil {
					// Show success message temporarily
					d.copiedMessage = fmt.Sprintf("Copied: %s", textToCopy)
					if len(d.copiedMessage) > 50 {
						d.copiedMessage = d.copiedMessage[:47] + "..."
					}
					d.copiedMessageTime = time.Now()

					// Start the blink animation
					return d, tea.Batch(
						d.startYankBlinkAnimation(),
						d.startMessageClearTimer(2*time.Second),
					)
				}
			}
		}
	case tea.KeyEsc:
		// Exit status info overlay
		d.showStatusInfo = false
		return d, nil
	}

	return d, nil
}

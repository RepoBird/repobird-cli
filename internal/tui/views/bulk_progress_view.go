// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api/dto"
)

// BulkProgressView component handles progress display for bulk operations
type BulkProgressView struct {
	batchID    string
	statistics dto.BulkStatistics
	runs       []dto.RunStatusItem
	spinner    spinner.Model
}

func NewBulkProgressView() *BulkProgressView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BulkProgressView{
		spinner: s,
	}
}

func (v *BulkProgressView) UpdateProgressView(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return cmd
	}
	return nil
}

func (v *BulkProgressView) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Bulk Run Progress")

	// Progress bar
	progressBar := v.makeProgressBar()

	// Statistics
	stats := fmt.Sprintf(
		"Total: %d | Queued: %d | Processing: %d | Completed: %d | Failed: %d",
		v.statistics.Total,
		v.statistics.Queued,
		v.statistics.Processing,
		v.statistics.Completed,
		v.statistics.Failed,
	)

	// Run details
	var runDetails strings.Builder
	for _, run := range v.runs {
		statusIcon := v.getStatusIcon(run.Status)
		runDetails.WriteString(fmt.Sprintf("  %s %s (ID: %d)\n",
			statusIcon, run.Title, run.ID))
		if run.Message != "" {
			runDetails.WriteString(fmt.Sprintf("    %s\n", run.Message))
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		v.spinner.View()+" "+progressBar,
		"",
		stats,
		"",
		runDetails.String(),
	)
}

func (v *BulkProgressView) UpdateProgress(msg bulkProgressMsg) {
	v.batchID = msg.batchID
	v.statistics = msg.statistics
	v.runs = msg.runs
}

func (v *BulkProgressView) makeProgressBar() string {
	width := 40
	completed := v.statistics.Completed + v.statistics.Failed + v.statistics.Cancelled
	total := v.statistics.Total

	if total == 0 {
		return strings.Repeat("░", width)
	}

	progress := int(float64(completed) / float64(total) * float64(width))
	return fmt.Sprintf("[%s%s] %d/%d",
		strings.Repeat("█", progress),
		strings.Repeat("░", width-progress),
		completed, total,
	)
}

func (v *BulkProgressView) getStatusIcon(status string) string {
	switch status {
	case "completed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗")
	case "processing":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("●")
	case "queued":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("○")
	default:
		return "?"
	}
}

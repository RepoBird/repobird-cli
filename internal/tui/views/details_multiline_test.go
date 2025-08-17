// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"strings"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestRunDetailsView_MultilineFieldHandling(t *testing.T) {
	// Create a run with multi-line fields
	multilinePlan := `Step 1: Initialize the project
Step 2: Set up dependencies
Step 3: Create the main structure
Step 4: Implement core features
Step 5: Add tests and documentation`

	multilinePrompt := `Please help me create a new web application.
It should have user authentication.
And a dashboard for data visualization.`

	run := models.RunResponse{
		ID:         "multi-123",
		Title:      "Multi-line Test",
		Repository: "test/repo",
		Source:     "main",
		RunType:    "plan",
		Status:     models.StatusDone,
		Prompt:     multilinePrompt,
		Plan:       multilinePlan,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          80,
		height:         30,
	}

	// Initialize content
	view.updateContent()

	t.Run("MultilineFieldsTracked", func(t *testing.T) {
		// Check that we have the expected fields
		assert.Greater(t, len(view.fieldValues), 0, "Should have field values")
		assert.Equal(t, len(view.fieldValues), len(view.fieldRanges),
			"Should have same number of values and ranges")

		// Find the plan field
		planIndex := -1
		for i, label := range view.fieldLines {
			if label == "Plan" {
				planIndex = i
				break
			}
		}

		assert.NotEqual(t, -1, planIndex, "Should have Plan field")

		if planIndex >= 0 {
			// Check that the plan field has multiple lines
			fieldRange := view.fieldRanges[planIndex]
			startLine := fieldRange[0]
			endLine := fieldRange[1]

			// Plan has 5 lines, so range should span 5 lines
			assert.Equal(t, 4, endLine-startLine, "Plan should span 5 lines (0-indexed)")
		}

		// Find the prompt field
		promptIndex := -1
		for i, label := range view.fieldLines {
			if label == "Prompt" {
				promptIndex = i
				break
			}
		}

		assert.NotEqual(t, -1, promptIndex, "Should have Prompt field")

		if promptIndex >= 0 {
			// Check that the prompt field has multiple lines
			fieldRange := view.fieldRanges[promptIndex]
			startLine := fieldRange[0]
			endLine := fieldRange[1]

			// Prompt has 3 lines, so range should span 3 lines
			assert.Equal(t, 2, endLine-startLine, "Prompt should span 3 lines (0-indexed)")
		}
	})

	t.Run("NavigateMultilineFields", func(t *testing.T) {
		// Find and select the plan field
		planIndex := -1
		for i, label := range view.fieldLines {
			if label == "Plan" {
				planIndex = i
				break
			}
		}

		if planIndex >= 0 {
			view.selectedRow = planIndex

			// The selected field range should cover multiple lines
			fieldRange := view.fieldRanges[view.selectedRow]
			assert.Greater(t, fieldRange[1], fieldRange[0],
				"Selected multi-line field should have end > start")
		}
	})

	t.Run("HighlightMultilineField", func(t *testing.T) {
		// Create some test lines
		lines := []string{
			"Title: Multi-line Test",
			"Run ID: multi-123",
			"",
			"=== Plan ===",
			"Step 1: Initialize the project",
			"Step 2: Set up dependencies",
			"Step 3: Create the main structure",
			"Step 4: Implement core features",
			"Step 5: Add tests and documentation",
			"",
			"=== End ===",
		}

		// Set up field ranges to simulate the plan field (lines 4-8)
		view.fieldRanges = [][2]int{
			{0, 0}, // Title
			{1, 1}, // Run ID
			{4, 8}, // Plan (multi-line)
		}
		view.selectedRow = 2 // Select the Plan field

		highlighted := view.createHighlightedContent(lines)

		// Check that all plan lines are present
		for i := 4; i <= 8; i++ {
			assert.Contains(t, highlighted, lines[i],
				"Highlighted content should contain all plan lines")
		}
	})

	t.Run("YankMultilineField", func(t *testing.T) {
		// Find the plan field
		planIndex := -1
		for i, label := range view.fieldLines {
			if label == "Plan" {
				planIndex = i
				break
			}
		}

		if planIndex >= 0 {
			view.selectedRow = planIndex

			// The field value should contain the entire multi-line content
			fieldValue := view.fieldValues[view.selectedRow]
			assert.Contains(t, fieldValue, "Step 1:", "Should contain first line")
			assert.Contains(t, fieldValue, "Step 5:", "Should contain last line")
			assert.Equal(t, multilinePlan, fieldValue,
				"Field value should be the complete multi-line plan")
		}
	})
}

func TestRunDetailsView_ScrollingMultilineFields(t *testing.T) {
	// Create a run with a very long plan
	longPlan := strings.Repeat("Line of plan content\n", 50)

	run := models.RunResponse{
		ID:        "scroll-test",
		Title:     "Scroll Test",
		RunType:   "plan",
		Status:    models.StatusDone,
		Plan:      longPlan,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          80,
		height:         20,
	}

	// Set up a mock viewport
	view.viewport.Height = 10
	view.viewport.YOffset = 0

	// Initialize content
	view.updateContent()

	t.Run("ScrollToLargeField", func(t *testing.T) {
		// Find the plan field
		planIndex := -1
		for i, label := range view.fieldLines {
			if label == "Plan" {
				planIndex = i
				break
			}
		}

		if planIndex >= 0 {
			view.selectedRow = planIndex
			// Properly set up the field ranges based on actual field positions
			if planIndex < len(view.fieldRanges) {
				// Get actual range from the initialized content
				fieldRange := view.fieldRanges[planIndex]
				startLine := fieldRange[0]

				// Set viewport offset far away from the field
				view.viewport.YOffset = 100

				// Simulate scrolling to the field
				view.scrollToSelectedField()

				// When scrolling to a large field, it should show the beginning
				assert.Equal(t, startLine, view.viewport.YOffset,
					"Should scroll to show the start of the large field")
			}
		}
	})
}

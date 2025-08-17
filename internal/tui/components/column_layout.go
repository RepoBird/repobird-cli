// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ColumnData represents the data for a single column in the Miller Columns layout
type ColumnData struct {
	Title    string
	Items    []string
	Selected int
	Active   bool
	MinWidth int
	MaxWidth int
}

// ColumnLayout implements Miller Columns navigation pattern with three columns
type ColumnLayout struct {
	columns      [3]*ColumnData
	activeColumn int
	width        int
	height       int
	columnWidths [3]int
	showBorders  bool
}

// NewColumnLayout creates a new Miller Columns layout
func NewColumnLayout() *ColumnLayout {
	return &ColumnLayout{
		columns: [3]*ColumnData{
			{Title: "Repositories", Items: []string{}, Selected: 0, Active: true, MinWidth: 15, MaxWidth: 40},
			{Title: "Runs", Items: []string{}, Selected: 0, Active: false, MinWidth: 25, MaxWidth: 50},
			{Title: "Details", Items: []string{}, Selected: 0, Active: false, MinWidth: 30, MaxWidth: 60},
		},
		activeColumn: 0,
		showBorders:  true,
	}
}

// SetDimensions updates the layout dimensions and recalculates column widths
func (cl *ColumnLayout) SetDimensions(width, height int) {
	cl.width = width
	cl.height = height
	cl.calculateColumnWidths()
}

// calculateColumnWidths distributes available width across columns
func (cl *ColumnLayout) calculateColumnWidths() {
	if cl.width <= 0 {
		return
	}

	// Account for borders and spacing
	borderWidth := 2 // 1 char each side for borders
	spacing := 2     // 1 space between columns
	totalBorderWidth := 0
	totalSpacing := 0

	if cl.showBorders {
		totalBorderWidth = borderWidth * 3 // 3 columns
		totalSpacing = spacing * 2         // 2 gaps between 3 columns
	}

	availableWidth := cl.width - totalBorderWidth - totalSpacing

	// Default distribution: 25%, 35%, 40%
	defaultRatios := []float64{0.25, 0.35, 0.40}

	// Calculate initial widths
	for i := 0; i < 3; i++ {
		cl.columnWidths[i] = int(float64(availableWidth) * defaultRatios[i])
	}

	// Ensure minimum widths are respected
	for i := 0; i < 3; i++ {
		if cl.columnWidths[i] < cl.columns[i].MinWidth {
			cl.columnWidths[i] = cl.columns[i].MinWidth
		}
		if cl.columns[i].MaxWidth > 0 && cl.columnWidths[i] > cl.columns[i].MaxWidth {
			cl.columnWidths[i] = cl.columns[i].MaxWidth
		}
	}

	// Adjust if total exceeds available width
	totalUsed := cl.columnWidths[0] + cl.columnWidths[1] + cl.columnWidths[2]
	if totalUsed > availableWidth {
		// Proportionally reduce all columns
		ratio := float64(availableWidth) / float64(totalUsed)
		for i := 0; i < 3; i++ {
			cl.columnWidths[i] = int(float64(cl.columnWidths[i]) * ratio)
			if cl.columnWidths[i] < cl.columns[i].MinWidth {
				cl.columnWidths[i] = cl.columns[i].MinWidth
			}
		}
	}
}

// SetColumnData updates the data for a specific column
func (cl *ColumnLayout) SetColumnData(columnIndex int, title string, items []string) {
	if columnIndex >= 0 && columnIndex < 3 {
		cl.columns[columnIndex].Title = title
		cl.columns[columnIndex].Items = items
		if cl.columns[columnIndex].Selected >= len(items) && len(items) > 0 {
			cl.columns[columnIndex].Selected = len(items) - 1
		} else if len(items) == 0 {
			cl.columns[columnIndex].Selected = 0
		}
	}
}

// GetActiveColumn returns the index of the currently active column
func (cl *ColumnLayout) GetActiveColumn() int {
	return cl.activeColumn
}

// SetActiveColumn changes the active column and updates visual state
func (cl *ColumnLayout) SetActiveColumn(column int) {
	if column >= 0 && column < 3 {
		// Deactivate current column
		cl.columns[cl.activeColumn].Active = false

		// Activate new column
		cl.activeColumn = column
		cl.columns[cl.activeColumn].Active = true
	}
}

// MoveLeft switches to the previous column
func (cl *ColumnLayout) MoveLeft() bool {
	if cl.activeColumn > 0 {
		cl.SetActiveColumn(cl.activeColumn - 1)
		return true
	}
	return false
}

// MoveRight switches to the next column
func (cl *ColumnLayout) MoveRight() bool {
	if cl.activeColumn < 2 {
		cl.SetActiveColumn(cl.activeColumn + 1)
		return true
	}
	return false
}

// MoveUp moves selection up in the active column
func (cl *ColumnLayout) MoveUp() bool {
	activeCol := cl.columns[cl.activeColumn]
	if activeCol.Selected > 0 {
		activeCol.Selected--
		return true
	}
	return false
}

// MoveDown moves selection down in the active column
func (cl *ColumnLayout) MoveDown() bool {
	activeCol := cl.columns[cl.activeColumn]
	if activeCol.Selected < len(activeCol.Items)-1 {
		activeCol.Selected++
		return true
	}
	return false
}

// GetSelectedItem returns the selected item in the active column
func (cl *ColumnLayout) GetSelectedItem() string {
	activeCol := cl.columns[cl.activeColumn]
	if activeCol.Selected >= 0 && activeCol.Selected < len(activeCol.Items) {
		return activeCol.Items[activeCol.Selected]
	}
	return ""
}

// GetSelectedIndex returns the selected index in the active column
func (cl *ColumnLayout) GetSelectedIndex() int {
	return cl.columns[cl.activeColumn].Selected
}

// GetSelectedItemInColumn returns the selected item in a specific column
func (cl *ColumnLayout) GetSelectedItemInColumn(columnIndex int) string {
	if columnIndex >= 0 && columnIndex < 3 {
		col := cl.columns[columnIndex]
		if col.Selected >= 0 && col.Selected < len(col.Items) {
			return col.Items[col.Selected]
		}
	}
	return ""
}

// SetSelectedInColumn sets the selected index for a specific column
func (cl *ColumnLayout) SetSelectedInColumn(columnIndex, selectedIndex int) {
	if columnIndex >= 0 && columnIndex < 3 {
		col := cl.columns[columnIndex]
		if selectedIndex >= 0 && selectedIndex < len(col.Items) {
			col.Selected = selectedIndex
		}
	}
}

// Render renders the Miller Columns layout
func (cl *ColumnLayout) Render() string {
	if cl.width <= 0 || cl.height <= 0 {
		return ""
	}

	var columns []string

	// Render each column
	for i := 0; i < 3; i++ {
		columns = append(columns, cl.renderColumn(i))
	}

	// Join columns horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, columns...)
}

// renderColumn renders a single column
func (cl *ColumnLayout) renderColumn(columnIndex int) string {
	col := cl.columns[columnIndex]
	width := cl.columnWidths[columnIndex]

	// Create column style
	var style lipgloss.Style
	if cl.showBorders {
		if col.Active {
			style = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Width(width).
				Height(cl.height - 2) // Account for border
		} else {
			style = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Width(width).
				Height(cl.height - 2)
		}
	} else {
		style = lipgloss.NewStyle().
			Width(width).
			Height(cl.height)
	}

	// Render title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Align(lipgloss.Center).
		Width(width)

	title := titleStyle.Render(col.Title)

	// Calculate available height for items
	availableHeight := cl.height - 3 // Title + borders
	if !cl.showBorders {
		availableHeight = cl.height - 1 // Just title
	}

	// Render items with scrolling if needed
	items := cl.renderColumnItems(col, width-2, availableHeight) // Account for padding

	// Combine title and items
	content := lipgloss.JoinVertical(lipgloss.Left, title, items)

	return style.Render(content)
}

// renderColumnItems renders the items within a column with scrolling support
func (cl *ColumnLayout) renderColumnItems(col *ColumnData, width, height int) string {
	if len(col.Items) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Align(lipgloss.Center).
			Width(width)
		return emptyStyle.Render("No items")
	}

	// Calculate scroll offset to keep selected item visible
	scrollOffset := 0
	if col.Selected >= height {
		scrollOffset = col.Selected - height + 1
	}

	var renderedItems []string

	// Render visible items
	for i := scrollOffset; i < len(col.Items) && i < scrollOffset+height; i++ {
		item := col.Items[i]

		// Truncate item if too long
		if len(item) > width {
			if width > 3 {
				item = item[:width-3] + "..."
			} else {
				item = item[:width]
			}
		}

		// Style the item
		var itemStyle lipgloss.Style
		if i == col.Selected {
			// Selected item
			if col.Active {
				itemStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("255")).
					Width(width)
			} else {
				itemStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("240")).
					Foreground(lipgloss.Color("255")).
					Width(width)
			}
		} else {
			// Regular item
			itemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Width(width)
		}

		renderedItems = append(renderedItems, itemStyle.Render(item))
	}

	// Fill remaining space with empty lines
	for len(renderedItems) < height {
		emptyStyle := lipgloss.NewStyle().Width(width)
		renderedItems = append(renderedItems, emptyStyle.Render(""))
	}

	return strings.Join(renderedItems, "\n")
}

// SetShowBorders enables or disables borders around columns
func (cl *ColumnLayout) SetShowBorders(show bool) {
	cl.showBorders = show
	cl.calculateColumnWidths() // Recalculate widths since borders affect available space
}

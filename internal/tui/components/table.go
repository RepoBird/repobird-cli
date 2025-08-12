package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

type Column struct {
	Title    string
	Width    int
	MinWidth int // Minimum width for this column
	Flex     int // Flex weight for distributing extra space (0 = fixed width)
}

type Row []string

type Table struct {
	columns      []Column
	rows         []Row
	selectedRow  int
	showCursor   bool
	height       int
	width        int
	scrollOffset int
}

func NewTable(columns []Column) *Table {
	return &Table{
		columns:     columns,
		rows:        []Row{},
		selectedRow: 0,
		showCursor:  true,
		height:      20,
		width:       80,
	}
}

func (t *Table) SetRows(rows []Row) {
	t.rows = rows
	if t.selectedRow >= len(rows) && len(rows) > 0 {
		t.selectedRow = len(rows) - 1
	}
}

func (t *Table) SetDimensions(width, height int) {
	t.width = width
	t.height = height
	t.calculateColumnWidths()
	// Recalculate scroll position to ensure selected row is still visible
	// but don't scroll unnecessarily
	if t.scrollOffset > 0 {
		t.ensureVisible()
	}
}

// calculateColumnWidths dynamically calculates column widths based on available space
func (t *Table) calculateColumnWidths() {
	if t.width == 0 {
		return
	}

	// Account for spacing between columns (1 space per column gap)
	totalSpacing := len(t.columns) - 1
	availableWidth := t.width - totalSpacing

	// First pass: calculate total minimum width and flex weight
	totalMinWidth := 0
	totalFlex := 0
	for _, col := range t.columns {
		if col.MinWidth > 0 {
			totalMinWidth += col.MinWidth
		} else if col.Flex == 0 {
			// Fixed width column
			totalMinWidth += col.Width
		}
		totalFlex += col.Flex
	}

	// Calculate remaining space for flexible columns
	remainingWidth := availableWidth - totalMinWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	// Second pass: assign widths
	for i := range t.columns {
		if t.columns[i].Flex > 0 {
			// Flexible column - distribute remaining space proportionally
			if totalFlex > 0 {
				flexWidth := (remainingWidth * t.columns[i].Flex) / totalFlex
				t.columns[i].Width = t.columns[i].MinWidth + flexWidth
			} else {
				t.columns[i].Width = t.columns[i].MinWidth
			}
		} else if t.columns[i].MinWidth > 0 {
			// Fixed column with minimum width
			t.columns[i].Width = t.columns[i].MinWidth
		}
		// else: keep existing width for fixed columns
	}
}

func (t *Table) MoveUp() {
	if t.selectedRow > 0 {
		t.selectedRow--
		t.ensureVisible()
	}
}

func (t *Table) MoveDown() {
	if t.selectedRow < len(t.rows)-1 {
		t.selectedRow++
		t.ensureVisible()
	}
}

func (t *Table) PageUp() {
	t.selectedRow -= 10
	if t.selectedRow < 0 {
		t.selectedRow = 0
	}
	t.ensureVisible()
}

func (t *Table) PageDown() {
	t.selectedRow += 10
	if t.selectedRow >= len(t.rows) {
		t.selectedRow = len(t.rows) - 1
	}
	t.ensureVisible()
}

func (t *Table) GoToTop() {
	t.selectedRow = 0
	t.scrollOffset = 0
}

func (t *Table) GoToBottom() {
	t.selectedRow = len(t.rows) - 1
	t.ensureVisible()
}

func (t *Table) GetSelectedIndex() int {
	return t.selectedRow
}

func (t *Table) SetSelectedIndex(index int) {
	if index >= 0 && index < len(t.rows) {
		t.selectedRow = index
		t.ensureVisible()
	}
}

func (t *Table) ResetScroll() {
	t.scrollOffset = 0
}

func (t *Table) ensureVisible() {
	visibleRows := t.height - 3
	if t.selectedRow < t.scrollOffset {
		t.scrollOffset = t.selectedRow
	} else if t.selectedRow >= t.scrollOffset+visibleRows {
		t.scrollOffset = t.selectedRow - visibleRows + 1
	}
}

func (t *Table) View() string {
	if t.width == 0 || t.height == 0 {
		return ""
	}

	var s strings.Builder

	header := t.renderHeader()
	s.WriteString(header)
	s.WriteString("\n")
	s.WriteString(t.renderSeparator())
	s.WriteString("\n")

	visibleRows := t.height - 3
	endRow := t.scrollOffset + visibleRows
	if endRow > len(t.rows) {
		endRow = len(t.rows)
	}

	for i := t.scrollOffset; i < endRow; i++ {
		row := t.renderRow(i)
		s.WriteString(row)
		if i < endRow-1 {
			s.WriteString("\n")
		}
	}

	if len(t.rows) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No runs found")
		s.WriteString(lipgloss.Place(t.width, 3, lipgloss.Center, lipgloss.Center, emptyMsg))
	}

	return s.String()
}

func (t *Table) renderHeader() string {
	var cells []string
	for _, col := range t.columns {
		cell := truncate(col.Title, col.Width)
		cell = lipgloss.NewStyle().Width(col.Width).Render(cell)
		cells = append(cells, cell)
	}
	return styles.TableHeaderStyle.Render(strings.Join(cells, " "))
}

func (t *Table) renderSeparator() string {
	totalWidth := 0
	for _, col := range t.columns {
		totalWidth += col.Width + 1
	}
	return strings.Repeat("─", totalWidth-1)
}

func (t *Table) renderRow(index int) string {
	if index >= len(t.rows) {
		return ""
	}

	row := t.rows[index]
	var cells []string

	for i, col := range t.columns {
		cell := ""
		if i < len(row) {
			cell = truncate(row[i], col.Width)
		}
		cell = lipgloss.NewStyle().Width(col.Width).Render(cell)
		cells = append(cells, cell)
	}

	content := strings.Join(cells, " ")

	if t.showCursor && index == t.selectedRow {
		return styles.TableSelectedRowStyle.Render(content)
	}
	return styles.TableRowStyle.Render(content)
}

// truncate is now replaced by utils.TruncateSimple
// Keeping this as an alias for backward compatibility
// Note: This has slightly different behavior for width <= 3 (no ellipsis)
func truncate(s string, width int) string {
	// Preserve original behavior for width <= 3
	if width <= 3 && len(s) > width {
		return s[:width]
	}
	return utils.TruncateSimple(s, width)
}

func (t *Table) StatusLine() string {
	if len(t.rows) == 0 {
		return "No items"
	}
	return fmt.Sprintf("%d/%d", t.selectedRow+1, len(t.rows))
}

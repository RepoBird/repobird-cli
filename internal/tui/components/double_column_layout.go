package components

import (
	"github.com/charmbracelet/lipgloss"
)

// DoubleColumnLayout provides a consistent two-column layout system
// for views that need side-by-side content (like FZF + preview)
type DoubleColumnLayout struct {
	width  int
	height int

	// Column split ratios
	leftRatio  float64 // 0.0 to 1.0
	rightRatio float64 // 0.0 to 1.0

	// Calculated dimensions
	leftWidth    int
	rightWidth   int
	columnHeight int
	gap          int
}

// DoubleColumnConfig holds configuration for double column layout
type DoubleColumnConfig struct {
	LeftRatio  float64 // Default: 0.6 (60% left, 40% right)
	RightRatio float64 // Default: 0.4
	Gap        int     // Default: 1 (space between columns)
}

// NewDoubleColumnLayout creates a new double column layout system
func NewDoubleColumnLayout(width, height int, config *DoubleColumnConfig) *DoubleColumnLayout {
	if config == nil {
		config = &DoubleColumnConfig{
			LeftRatio:  0.6,
			RightRatio: 0.4,
			Gap:        1,
		}
	}

	layout := &DoubleColumnLayout{
		width:      width,
		height:     height,
		leftRatio:  config.LeftRatio,
		rightRatio: config.RightRatio,
		gap:        config.Gap,
	}

	layout.recalculateDimensions()
	return layout
}

// Update recalculates layout dimensions when terminal size changes
func (d *DoubleColumnLayout) Update(width, height int) {
	d.width = width
	d.height = height
	d.recalculateDimensions()
}

// recalculateDimensions calculates column dimensions based on ratios
func (d *DoubleColumnLayout) recalculateDimensions() {
	if d.width <= 0 || d.height <= 0 {
		return
	}

	// Calculate usable width (subtract gap)
	usableWidth := d.width - d.gap
	if usableWidth < 10 {
		usableWidth = 10
		d.gap = 0
	}

	// Calculate column widths
	d.leftWidth = int(float64(usableWidth) * d.leftRatio)
	d.rightWidth = usableWidth - d.leftWidth

	// Ensure minimum widths
	if d.leftWidth < 5 {
		d.leftWidth = 5
		d.rightWidth = usableWidth - d.leftWidth
	}
	if d.rightWidth < 5 {
		d.rightWidth = 5
		d.leftWidth = usableWidth - d.rightWidth
	}

	// Column height is full height minus space for title and status
	d.columnHeight = d.height - 3 // Title + status line
	if d.columnHeight < 5 {
		d.columnHeight = 5
	}
}

// GetLeftColumnDimensions returns dimensions for the left column
func (d *DoubleColumnLayout) GetLeftColumnDimensions() (width, height int) {
	return d.leftWidth, d.columnHeight
}

// GetRightColumnDimensions returns dimensions for the right column
func (d *DoubleColumnLayout) GetRightColumnDimensions() (width, height int) {
	return d.rightWidth, d.columnHeight
}

// GetLeftColumnBox creates a styled box for the left column
func (d *DoubleColumnLayout) GetLeftColumnBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(d.leftWidth).
		Height(d.columnHeight).
		Padding(0, 1)
}

// GetRightColumnBox creates a styled box for the right column
func (d *DoubleColumnLayout) GetRightColumnBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(d.rightWidth).
		Height(d.columnHeight).
		Padding(0, 1)
}

// CreateTitleStyle creates a consistent title style
func (d *DoubleColumnLayout) CreateTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Margin(0, 0, 1, 0)
}

// RenderColumns combines left and right content into a two-column layout
func (d *DoubleColumnLayout) RenderColumns(leftContent, rightContent string) string {
	leftBox := d.GetLeftColumnBox().Render(leftContent)
	rightBox := d.GetRightColumnBox().Render(rightContent)

	// Join columns horizontally with gap
	if d.gap > 0 {
		gap := lipgloss.NewStyle().Width(d.gap).Render("")
		return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, gap, rightBox)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}

// RenderWithTitle combines title, columns, and status line
func (d *DoubleColumnLayout) RenderWithTitle(title, leftContent, rightContent, statusLine string) string {
	titleStyle := d.CreateTitleStyle()
	styledTitle := titleStyle.Render(title)
	columns := d.RenderColumns(leftContent, rightContent)

	return lipgloss.JoinVertical(lipgloss.Left, styledTitle, columns, statusLine)
}

// IsValidDimensions checks if the layout has enough space to render properly
func (d *DoubleColumnLayout) IsValidDimensions() bool {
	return d.width >= 20 && d.height >= 8
}

// GetContentDimensions returns the actual content area dimensions for each column
// (subtracting borders and padding)
func (d *DoubleColumnLayout) GetContentDimensions() (leftWidth, leftHeight, rightWidth, rightHeight int) {
	// Account for borders (2 chars) and padding (2 chars) = 4 total width reduction
	// Account for borders (2 chars) = 2 total height reduction
	leftWidth = d.leftWidth - 4
	leftHeight = d.columnHeight - 2
	rightWidth = d.rightWidth - 4
	rightHeight = d.columnHeight - 2

	// Ensure minimums
	if leftWidth < 1 {
		leftWidth = 1
	}
	if leftHeight < 1 {
		leftHeight = 1
	}
	if rightWidth < 1 {
		rightWidth = 1
	}
	if rightHeight < 1 {
		rightHeight = 1
	}

	return leftWidth, leftHeight, rightWidth, rightHeight
}
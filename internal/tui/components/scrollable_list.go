package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ScrollableList is a reusable component for scrollable multi-column lists
type ScrollableList struct {
	viewport   viewport.Model
	items      [][]string // Multi-column data
	selected   int
	focusedCol int

	// Configuration
	columns      int
	keyNav       bool // Navigate between keys (like status view)
	valueNav     bool // Navigate between values (normal)
	keymaps      KeyMap
	width        int
	height       int
	columnWidths []int

	// Styling
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	headerStyle   lipgloss.Style
}

// ScrollableListOption is a functional option for configuring ScrollableList
type ScrollableListOption func(*ScrollableList)

// NewScrollableList creates a new scrollable list with the given options
func NewScrollableList(opts ...ScrollableListOption) *ScrollableList {
	s := &ScrollableList{
		viewport:      viewport.New(80, 20),
		items:         [][]string{},
		selected:      0,
		focusedCol:    0,
		columns:       1,
		columnWidths:  make([]int, 1), // Initialize column widths for default 1 column
		keyNav:        false,
		valueNav:      true,
		keymaps:       DefaultKeyMap,
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("240")),
		normalStyle:   lipgloss.NewStyle(),
		headerStyle:   lipgloss.NewStyle().Bold(true).Underline(true),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithColumns sets the number of columns
func WithColumns(n int) ScrollableListOption {
	return func(s *ScrollableList) {
		s.columns = n
		s.columnWidths = make([]int, n)
	}
}

// WithKeyNavigation enables navigation between keys (for status-like views)
func WithKeyNavigation(enabled bool) ScrollableListOption {
	return func(s *ScrollableList) {
		s.keyNav = enabled
	}
}

// WithValueNavigation enables navigation between values (normal list navigation)
func WithValueNavigation(enabled bool) ScrollableListOption {
	return func(s *ScrollableList) {
		s.valueNav = enabled
	}
}

// WithDimensions sets the width and height of the list
func WithDimensions(width, height int) ScrollableListOption {
	return func(s *ScrollableList) {
		s.width = width
		s.height = height
		s.viewport = viewport.New(width, height)
	}
}

// WithKeymaps sets custom keymaps for the list
func WithKeymaps(km KeyMap) ScrollableListOption {
	return func(s *ScrollableList) {
		s.keymaps = km
	}
}

// WithColumnWidths sets specific widths for each column
func WithColumnWidths(widths []int) ScrollableListOption {
	return func(s *ScrollableList) {
		if len(widths) == s.columns {
			s.columnWidths = widths
		}
	}
}

// Init initializes the scrollable list
func (s *ScrollableList) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the list state
func (s *ScrollableList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.viewport.Width = msg.Width
		s.viewport.Height = msg.Height - 2 // Leave room for borders/headers
		s.updateColumnWidths()

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.valueNav && s.selected > 0 {
				s.selected--
				s.ensureSelectedVisible()
			}

		case "down", "j":
			if s.valueNav && s.selected < len(s.items)-1 {
				s.selected++
				s.ensureSelectedVisible()
			}

		case "left", "h":
			if s.keyNav && s.focusedCol > 0 {
				s.focusedCol--
			}

		case "right", "l":
			if s.keyNav && s.focusedCol < s.columns-1 {
				s.focusedCol++
			}

		case "tab":
			// Move to next column
			if s.focusedCol < s.columns-1 {
				s.focusedCol++
			} else {
				s.focusedCol = 0
			}

		case "shift+tab":
			// Move to previous column
			if s.focusedCol > 0 {
				s.focusedCol--
			} else {
				s.focusedCol = s.columns - 1
			}

		case "pgup":
			s.viewport, cmd = s.viewport.Update(msg)
			cmds = append(cmds, cmd)

		case "pgdown":
			s.viewport, cmd = s.viewport.Update(msg)
			cmds = append(cmds, cmd)

		case "home":
			s.selected = 0
			s.ensureSelectedVisible()

		case "end":
			s.selected = len(s.items) - 1
			s.ensureSelectedVisible()
		}
	}

	// Update viewport
	s.viewport, cmd = s.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Update content
	s.viewport.SetContent(s.renderContent())

	return s, tea.Batch(cmds...)
}

// View renders the scrollable list
func (s *ScrollableList) View() string {
	if s.width == 0 || s.height == 0 {
		return ""
	}

	return s.viewport.View()
}

// SetItems sets the list items
func (s *ScrollableList) SetItems(items [][]string) {
	s.items = items
	if s.selected >= len(items) {
		s.selected = len(items) - 1
	}
	if s.selected < 0 {
		s.selected = 0
	}
	s.viewport.SetContent(s.renderContent())
}

// GetSelected returns the currently selected item
func (s *ScrollableList) GetSelected() []string {
	if s.selected >= 0 && s.selected < len(s.items) {
		return s.items[s.selected]
	}
	return nil
}

// GetSelectedIndex returns the currently selected index
func (s *ScrollableList) GetSelectedIndex() int {
	return s.selected
}

// GetFocusedColumn returns the currently focused column
func (s *ScrollableList) GetFocusedColumn() int {
	return s.focusedCol
}

// SetSelected sets the selected index
func (s *ScrollableList) SetSelected(index int) {
	if index >= 0 && index < len(s.items) {
		s.selected = index
		s.ensureSelectedVisible()
	}
}

// renderContent renders the list content
func (s *ScrollableList) renderContent() string {
	if len(s.items) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			Italic(true).
			Render("No items to display")
	}

	var lines []string
	for i, item := range s.items {
		line := s.renderRow(item, i == s.selected)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderRow renders a single row
func (s *ScrollableList) renderRow(row []string, isSelected bool) string {
	var cells []string

	for i, cell := range row {
		if i >= s.columns {
			break
		}

		width := s.columnWidths[i]
		if width == 0 {
			width = 20 // Default width
		}

		// Truncate or pad cell content to fit width
		if len(cell) > width {
			cell = cell[:width-3] + "..."
		} else {
			cell = fmt.Sprintf("%-*s", width, cell)
		}

		if isSelected {
			if s.keyNav && i == s.focusedCol {
				// Highlight the focused cell in key navigation mode
				cell = lipgloss.NewStyle().
					Background(lipgloss.Color("33")).
					Foreground(lipgloss.Color("255")).
					Render(cell)
			} else {
				cell = s.selectedStyle.Render(cell)
			}
		} else {
			cell = s.normalStyle.Render(cell)
		}

		cells = append(cells, cell)
	}

	return strings.Join(cells, " ")
}

// updateColumnWidths calculates column widths based on available space
func (s *ScrollableList) updateColumnWidths() {
	if s.columns == 0 || s.width == 0 {
		return
	}

	// If no specific widths set, distribute evenly
	if len(s.columnWidths) == 0 || s.columnWidths[0] == 0 {
		baseWidth := (s.width - (s.columns - 1)) / s.columns // Account for spaces
		for i := range s.columnWidths {
			s.columnWidths[i] = baseWidth
		}
	}
}

// ensureSelectedVisible scrolls the viewport to make the selected item visible
func (s *ScrollableList) ensureSelectedVisible() {
	lineHeight := 1
	selectedY := s.selected * lineHeight

	viewTop := s.viewport.YOffset
	viewBottom := viewTop + s.viewport.Height

	if selectedY < viewTop {
		// Scroll up
		s.viewport.SetYOffset(selectedY)
	} else if selectedY >= viewBottom {
		// Scroll down
		s.viewport.SetYOffset(selectedY - s.viewport.Height + 1)
	}
}

// Focused returns whether the list is focused
func (s *ScrollableList) Focused() bool {
	return true
}

// Focus sets the focus state
func (s *ScrollableList) Focus() {
	// No-op for now, but could be used for focus management
}

// Blur removes focus
func (s *ScrollableList) Blur() {
	// No-op for now, but could be used for focus management
}

// SetSize updates the dimensions of the scrollable list
func (s *ScrollableList) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.viewport.Width = width
	s.viewport.Height = height
	s.updateColumnWidths()
}

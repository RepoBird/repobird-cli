package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/utils"
)

// DocsView displays comprehensive help documentation with multiple pages
type DocsView struct {
	// Parent view to return to
	parentView tea.Model

	// View state
	width  int
	height int

	// Navigation
	currentPage int
	totalPages  int
	selectedRow int // Selected row in current page

	// Content
	pages []DocsPage

	// Status line
	statusLine *components.StatusLine

	// Keys
	keys components.KeyMap
}

// DocsPage represents a single page of documentation
type DocsPage struct {
	Title   string
	Content []DocsRow
}

// DocsRow represents a single row in the documentation
type DocsRow struct {
	Key         string // Key binding (left side)
	Description string // Description (right side)
	Copyable    bool   // Whether this row can be copied with 'y'
}

// NewDocsView creates a new documentation view
func NewDocsView(parentView tea.Model) *DocsView {
	d := &DocsView{
		parentView:  parentView,
		currentPage: 0,
		selectedRow: 0,
		statusLine:  components.NewStatusLine(),
		keys:        components.DefaultKeyMap,
	}

	// Initialize pages with content
	d.initializePages()
	d.totalPages = len(d.pages)

	return d
}

func (d *DocsView) initializePages() {
	// Page 1: Basic Navigation
	page1 := DocsPage{
		Title: "Basic Navigation",
		Content: []DocsRow{
			{Key: "↑/↓, j/k", Description: "Move up/down in current column", Copyable: false},
			{Key: "←/→, h/l", Description: "Move between columns", Copyable: false},
			{Key: "Tab", Description: "Cycle through columns", Copyable: false},
			{Key: "Enter", Description: "Select item and move to next column", Copyable: false},
			{Key: "Backspace", Description: "Move to previous column", Copyable: false},
			{Key: "g", Description: "Jump to first item", Copyable: false},
			{Key: "G", Description: "Jump to last item", Copyable: false},
			{Key: "gg", Description: "Jump to top (vim-style double tap)", Copyable: false},
			{Key: "Ctrl+u", Description: "Page up", Copyable: false},
			{Key: "Ctrl+d", Description: "Page down", Copyable: false},
		},
	}

	// Page 2: Fuzzy Search (FZF)
	page2 := DocsPage{
		Title: "Fuzzy Search (FZF)",
		Content: []DocsRow{
			{Key: "f", Description: "Activate FZF mode on current column", Copyable: false},
			{Key: "Type", Description: "Filter items in real-time", Copyable: false},
			{Key: "↑/↓", Description: "Navigate filtered items", Copyable: false},
			{Key: "Ctrl+j/k", Description: "Alternative navigation in FZF", Copyable: false},
			{Key: "Enter", Description: "Select item and proceed", Copyable: false},
			{Key: "ESC", Description: "Cancel FZF mode", Copyable: false},
			{Key: "", Description: "", Copyable: false},
			{Key: "In Create View:", Description: "", Copyable: false},
			{Key: "Ctrl+F", Description: "FZF for repository (insert mode)", Copyable: false},
			{Key: "f", Description: "FZF for repository (normal mode)", Copyable: false},
		},
	}

	// Page 3: View Controls
	page3 := DocsPage{
		Title: "View Controls",
		Content: []DocsRow{
			{Key: "n", Description: "Create new run", Copyable: false},
			{Key: "s", Description: "Show status/user info overlay", Copyable: false},
			{Key: "r", Description: "Refresh data", Copyable: false},
			{Key: "o", Description: "Open URL (when available)", Copyable: false},
			{Key: "?", Description: "Toggle help/documentation", Copyable: false},
			{Key: "q", Description: "Go back/quit (context-aware)", Copyable: false},
			{Key: "Q", Description: "Force quit from anywhere", Copyable: false},
			{Key: "ESC, b", Description: "Alternative back navigation", Copyable: false},
		},
	}

	// Page 4: Clipboard Operations
	page4 := DocsPage{
		Title: "Clipboard Operations",
		Content: []DocsRow{
			{Key: "y", Description: "Copy current selection to clipboard", Copyable: false},
			{Key: "Y", Description: "Copy all content (details view)", Copyable: false},
			{Key: "", Description: "", Copyable: false},
			{Key: "Visual Feedback:", Description: "", Copyable: false},
			{Key: "Green flash", Description: "Successful copy animation", Copyable: false},
			{Key: "Status message", Description: "Shows what was copied", Copyable: false},
			{Key: "", Description: "", Copyable: false},
			{Key: "Tip:", Description: "All selectable fields support copying", Copyable: false},
		},
	}

	// Page 5: Create Run Form
	page5 := DocsPage{
		Title: "Create Run Form",
		Content: []DocsRow{
			{Key: "Normal Mode:", Description: "", Copyable: false},
			{Key: "i, Enter", Description: "Enter insert mode", Copyable: false},
			{Key: "j/k", Description: "Navigate fields", Copyable: false},
			{Key: "ESC (2x)", Description: "Return to dashboard", Copyable: false},
			{Key: "", Description: "", Copyable: false},
			{Key: "Insert Mode:", Description: "", Copyable: false},
			{Key: "Tab/Shift+Tab", Description: "Navigate between fields", Copyable: false},
			{Key: "ESC", Description: "Switch to normal mode", Copyable: false},
			{Key: "Ctrl+S", Description: "Submit run", Copyable: false},
			{Key: "Ctrl+L", Description: "Clear all fields", Copyable: false},
			{Key: "Ctrl+X", Description: "Clear current field", Copyable: false},
			{Key: "Ctrl+F", Description: "Repository fuzzy search", Copyable: false},
		},
	}

	// Page 6: Dashboard Columns
	page6 := DocsPage{
		Title: "Dashboard Layout",
		Content: []DocsRow{
			{Key: "Left Column", Description: "Repositories with active runs", Copyable: false},
			{Key: "Middle Column", Description: "Runs for selected repository", Copyable: false},
			{Key: "Right Column", Description: "Details for selected run", Copyable: false},
			{Key: "", Description: "", Copyable: false},
			{Key: "Status Icons:", Description: "", Copyable: false},
			{Key: "🟢", Description: "Success", Copyable: false},
			{Key: "🔵", Description: "Running", Copyable: false},
			{Key: "🟡", Description: "Pending", Copyable: false},
			{Key: "🔴", Description: "Failed", Copyable: false},
			{Key: "⚪", Description: "Unknown", Copyable: false},
		},
	}

	// Page 7: Tips & Tricks
	page7 := DocsPage{
		Title: "Tips & Tricks",
		Content: []DocsRow{
			{Key: "Quick Find", Description: "Use 'f' instead of scrolling", Copyable: false},
			{Key: "Fast Navigation", Description: "Enter drills down, Backspace goes up", Copyable: false},
			{Key: "Context Aware", Description: "'q' behavior changes by view", Copyable: false},
			{Key: "Repository Memory", Description: "Recently used repos saved", Copyable: false},
			{Key: "Smart Icons", Description: "📁 current, 🔄 history, ✏️ edited", Copyable: false},
			{Key: "", Description: "", Copyable: false},
			{Key: "Pro Tip:", Description: "Chain 'f' + Enter for quick access", Copyable: false},
		},
	}

	// Page 8: Keyboard Shortcuts Reference
	page8 := DocsPage{
		Title: "Quick Reference",
		Content: []DocsRow{
			{Key: "Navigation", Description: "j/k h/l Tab Enter Backspace", Copyable: true},
			{Key: "Search", Description: "f (fuzzy) / (search)", Copyable: true},
			{Key: "Actions", Description: "n (new) r (refresh) s (status)", Copyable: true},
			{Key: "Clipboard", Description: "y (copy) Y (copy all)", Copyable: true},
			{Key: "View Control", Description: "? (help) q (back) Q (quit)", Copyable: true},
			{Key: "", Description: "", Copyable: false},
			{Key: "Vim Commands", Description: "gg G Ctrl+u Ctrl+d", Copyable: true},
			{Key: "Form Submit", Description: "Ctrl+S", Copyable: true},
		},
	}

	d.pages = []DocsPage{page1, page2, page3, page4, page5, page6, page7, page8}
}

// Init implements tea.Model
func (d *DocsView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (d *DocsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return d, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "?", "q", "esc":
			// Return to parent view
			if d.parentView != nil {
				return d.parentView, nil
			}
			return d, tea.Quit

		case "left", "h":
			// Previous page
			if d.currentPage > 0 {
				d.currentPage--
				d.selectedRow = 0
			}
			return d, nil

		case "right", "l":
			// Next page
			if d.currentPage < d.totalPages-1 {
				d.currentPage++
				d.selectedRow = 0
			}
			return d, nil

		case "up", "k":
			// Move selection up
			if d.selectedRow > 0 {
				d.selectedRow--
			}
			return d, nil

		case "down", "j":
			// Move selection down
			currentPageContent := d.pages[d.currentPage].Content
			if d.selectedRow < len(currentPageContent)-1 {
				d.selectedRow++
			}
			return d, nil

		case "g":
			// Jump to first row
			d.selectedRow = 0
			return d, nil

		case "G":
			// Jump to last row
			currentPageContent := d.pages[d.currentPage].Content
			if len(currentPageContent) > 0 {
				d.selectedRow = len(currentPageContent) - 1
			}
			return d, nil

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Direct page navigation
			pageNum := int(msg.String()[0] - '1')
			if pageNum >= 0 && pageNum < d.totalPages {
				d.currentPage = pageNum
				d.selectedRow = 0
			}
			return d, nil

		case "y":
			// Copy selected row if copyable
			currentPage := d.pages[d.currentPage]
			if d.selectedRow < len(currentPage.Content) {
				row := currentPage.Content[d.selectedRow]
				if row.Copyable {
					textToCopy := fmt.Sprintf("%s: %s", row.Key, row.Description)
					d.copyToClipboard(textToCopy)
					d.statusLine.SetTemporaryMessageWithType(
						fmt.Sprintf("📋 Copied: %s", truncateString(textToCopy, 50)),
						components.MessageSuccess,
						2*time.Second,
					)
				}
			}
			return d, nil

		case "Q":
			// Force quit
			return d, tea.Quit
		}
	}

	// Status line doesn't need updating as it's stateless

	return d, nil
}

// View implements tea.Model
func (d *DocsView) View() string {
	if d.width == 0 || d.height == 0 {
		return ""
	}

	// Calculate available height
	availableHeight := d.height - 1 // Reserve for status line

	// Create main content area
	contentHeight := availableHeight - 4 // Title, page indicator, borders
	contentWidth := d.width - 4          // Borders and padding

	// Render current page
	currentPage := d.pages[d.currentPage]

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	title := titleStyle.Render(currentPage.Title)

	// Content rows
	var rows []string
	for i, row := range currentPage.Content {
		rowStyle := lipgloss.NewStyle().Width(contentWidth)

		// Highlight selected row
		if i == d.selectedRow {
			rowStyle = rowStyle.
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("255"))
		}

		// Format row
		var formattedRow string
		if row.Key == "" && row.Description == "" {
			// Empty row for spacing
			formattedRow = " "
		} else if row.Description == "" {
			// Section header
			formattedRow = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214")).
				Render(row.Key)
		} else if row.Key == "" {
			// Description only
			formattedRow = "  " + row.Description
		} else {
			// Key-value pair
			keyStyle := lipgloss.NewStyle().
				Width(20).
				Foreground(lipgloss.Color("86"))
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

			formattedRow = lipgloss.JoinHorizontal(
				lipgloss.Left,
				keyStyle.Render(row.Key),
				descStyle.Render(row.Description),
			)
		}

		// Add copy indicator for copyable rows
		if row.Copyable && i == d.selectedRow {
			formattedRow += " [y to copy]"
		}

		rows = append(rows, rowStyle.Render(formattedRow))
	}

	// Join rows with proper height limit
	content := strings.Join(rows, "\n")

	// Ensure content fits in available space
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
		content = strings.Join(contentLines, "\n")
	}

	// Page indicator
	pageIndicator := d.renderPageIndicator()

	// Main panel
	panelStyle := lipgloss.NewStyle().
		Width(d.width - 2).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Combine all elements
	mainContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		"\n"+pageIndicator,
	)

	panel := panelStyle.Render(mainContent)

	// Status line
	statusText := d.getStatusText()
	statusBar := d.statusLine.
		SetWidth(d.width).
		SetLeft("[DOCS]").
		SetRight("").
		SetHelp(statusText).
		Render()

	// Final layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		panel,
		statusBar,
	)
}

func (d *DocsView) renderPageIndicator() string {
	indicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(d.width - 8).
		Align(lipgloss.Center)

	// Page dots
	var dots []string
	for i := 0; i < d.totalPages; i++ {
		if i == d.currentPage {
			dots = append(dots, "●")
		} else {
			dots = append(dots, "○")
		}
	}

	indicator := fmt.Sprintf("Page %d/%d  %s",
		d.currentPage+1,
		d.totalPages,
		strings.Join(dots, " "))

	// Add page numbers hint
	hint := "  (1-8: jump to page)"

	return indicatorStyle.Render(indicator + hint)
}

func (d *DocsView) getStatusText() string {
	navigation := "h/l:pages j/k:navigate"
	if d.selectedRow >= 0 && d.selectedRow < len(d.pages[d.currentPage].Content) {
		row := d.pages[d.currentPage].Content[d.selectedRow]
		if row.Copyable {
			navigation += " y:copy"
		}
	}
	return fmt.Sprintf("%s 1-8:page ?/q:back", navigation)
}

func (d *DocsView) copyToClipboard(text string) error {
	// Use the same clipboard implementation as other views
	return utils.WriteToClipboard(text)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

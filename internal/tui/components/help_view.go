package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
)

// HelpSection represents a section in the help documentation
type HelpSection struct {
	Title   string
	Content []string
}

// HelpView is a scrollable help view component
type HelpView struct {
	viewport   viewport.Model
	sections   []HelpSection
	width      int
	height     int
	ready      bool
	statusLine *StatusLine
	// For copy functionality
	copiedMessage string
	copiedTime    time.Time
	yankBlink     bool
	yankBlinkTime time.Time
	contentLines  []string    // All lines for easy copying
	lineToSection map[int]int // Map line number to section index
}

// NewHelpView creates a new scrollable help view
func NewHelpView() *HelpView {
	return &HelpView{
		viewport:      viewport.New(80, 20),
		statusLine:    NewStatusLine(),
		lineToSection: make(map[int]int),
		sections:      getDefaultHelpSections(),
	}
}

// getDefaultHelpSections returns the organized help content
func getDefaultHelpSections() []HelpSection {
	return []HelpSection{
		{
			Title: "🎯 Basic Navigation",
			Content: []string{
				"↑/↓, j/k     Move up/down in current column",
				"←/→, h/l     Move between columns",
				"Tab          Cycle through columns",
				"Enter        Select item and move to next column",
				"Backspace    Move to previous column",
				"gg           Jump to top (vim-style double tap)",
				"G            Jump to last item",
			},
		},
		{
			Title: "📜 Scrolling",
			Content: []string{
				"Ctrl+u       Half page up",
				"Ctrl+d       Half page down",
				"Page Up      Full page up",
				"Page Down    Full page down",
				"Home         Jump to top",
				"End          Jump to bottom",
			},
		},
		{
			Title: "🔍 Fuzzy Search (FZF)",
			Content: []string{
				"f            Activate FZF mode on current column",
				"Type         Filter items in real-time",
				"↑/↓          Navigate filtered items",
				"Ctrl+j/k     Alternative navigation in FZF",
				"Enter        Select item and proceed",
				"ESC          Cancel FZF mode",
				"",
				"In Create View:",
				"Ctrl+F       FZF for repository (insert mode)",
				"f            FZF for repository (normal mode)",
			},
		},
		{
			Title: "🎮 View Controls",
			Content: []string{
				"n            Create new run",
				"s            Show status/user info overlay",
				"r            Refresh data",
				"o            Open URL (when available)",
				"?            Toggle help/documentation",
				"q            Go back/quit (context-aware)",
				"Q            Force quit from anywhere",
				"ESC, b       Alternative back navigation",
			},
		},
		{
			Title: "📋 Clipboard Operations",
			Content: []string{
				"y            Copy current selection to clipboard",
				"Y            Copy all content (details view)",
				"",
				"Visual Feedback:",
				"• Green flash  Successful copy animation",
				"• Status msg   Shows what was copied",
				"",
				"💡 Tip: All selectable fields support copying",
			},
		},
		{
			Title: "📝 Create Run Form",
			Content: []string{
				"Normal Mode:",
				"i, Enter     Enter insert mode",
				"j/k          Navigate fields",
				"ESC (2x)     Return to dashboard",
				"",
				"Insert Mode:",
				"Tab/Shift+Tab Navigate between fields",
				"ESC          Switch to normal mode",
				"Ctrl+S       Submit run",
				"Ctrl+L       Clear all fields",
				"Ctrl+X       Clear current field",
				"Ctrl+F       Repository fuzzy search",
			},
		},
		{
			Title: "🗂️ Dashboard Layout",
			Content: []string{
				"Left Column  Repositories with active runs",
				"Middle       Runs for selected repository",
				"Right        Details for selected run",
				"",
				"Status Icons:",
				"🟢           Success / Done",
				"🔵           Running / In Progress",
				"🟡           Pending / Waiting",
				"🔴           Failed / Error",
				"⚪           Unknown / Other",
			},
		},
		{
			Title: "⚡ Tips & Tricks",
			Content: []string{
				"• Quick Find   Use 'f' instead of scrolling",
				"• Fast Nav     Enter drills down, Backspace goes up",
				"• Context      'q' behavior changes by view",
				"• Memory       Recently used repos saved",
				"• URLs         'o' opens PR/repo URLs intelligently",
				"• Multi-select Some views support batch operations",
				"• History      Previous runs are cached locally",
			},
		},
		{
			Title: "🔧 Troubleshooting",
			Content: []string{
				"Connection Issues:",
				"• Check API key: repobird config get api-key",
				"• Verify network: repobird status",
				"• Check API URL: REPOBIRD_API_URL env var",
				"",
				"Display Issues:",
				"• Resize terminal if content is cut off",
				"• Use fullscreen mode for best experience",
				"• Check terminal emulator settings",
				"",
				"Performance:",
				"• Use 'r' to manually refresh data",
				"• FZF mode ('f') for large lists",
				"• Clear cache if data seems stale",
			},
		},
	}
}

// SetSize updates the viewport size
func (h *HelpView) SetSize(width, height int) {
	h.width = width
	h.height = height

	// Account for status line, borders, and scrollbar
	innerHeight := height - 3 // 1 for status, 2 for border
	innerWidth := width - 6   // 2 for border, 2 for padding, 2 for scrollbar

	if innerHeight < 1 {
		innerHeight = 1
	}
	if innerWidth < 1 {
		innerWidth = 1
	}

	if !h.ready {
		h.viewport = viewport.New(innerWidth, innerHeight)
		h.buildContent()
		h.ready = true
	} else {
		h.viewport.Width = innerWidth
		h.viewport.Height = innerHeight
	}
}

// buildContent constructs the help content with proper formatting
func (h *HelpView) buildContent() {
	var lines []string
	h.contentLines = []string{}
	h.lineToSection = make(map[int]int)

	// Build formatted content
	for i, section := range h.sections {
		if i > 0 {
			// Add spacing between sections
			lines = append(lines, "")
			h.contentLines = append(h.contentLines, "")
		}

		// Add section title
		titleLine := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			Render(section.Title)
		lines = append(lines, titleLine)
		h.contentLines = append(h.contentLines, section.Title)
		h.lineToSection[len(h.contentLines)-1] = i

		lines = append(lines, strings.Repeat("─", 40))
		h.contentLines = append(h.contentLines, strings.Repeat("─", 40))

		// Add section content
		for _, line := range section.Content {
			// Format special lines
			formattedLine := line
			if strings.Contains(line, "     ") {
				// Command line - format with colors
				parts := strings.SplitN(line, "     ", 2)
				if len(parts) == 2 {
					cmd := lipgloss.NewStyle().
						Foreground(lipgloss.Color("220")).
						Render(parts[0])
					desc := lipgloss.NewStyle().
						Foreground(lipgloss.Color("245")).
						Render(parts[1])
					formattedLine = fmt.Sprintf("  %-12s %s", cmd, desc)
				}
			} else if strings.HasPrefix(line, "•") {
				// Bullet point
				formattedLine = lipgloss.NewStyle().
					Foreground(lipgloss.Color("245")).
					Render(line)
			} else if line != "" && !strings.HasPrefix(line, " ") && strings.Contains(line, ":") {
				// Subsection header
				formattedLine = lipgloss.NewStyle().
					Foreground(lipgloss.Color("111")).
					Italic(true).
					Render(line)
			}

			lines = append(lines, formattedLine)
			h.contentLines = append(h.contentLines, line)
			h.lineToSection[len(h.contentLines)-1] = i
		}
	}

	// Set viewport content
	content := strings.Join(lines, "\n")
	h.viewport.SetContent(content)
}

// Update handles tea messages
func (h *HelpView) Update(msg tea.Msg) (*HelpView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.SetSize(msg.Width, msg.Height)
		return h, nil

	case tea.KeyMsg:
		switch msg.String() {
		// Scrolling
		case "j", "down":
			h.viewport.ScrollDown(1)
		case "k", "up":
			h.viewport.ScrollUp(1)
		case "ctrl+d":
			h.viewport.HalfPageDown()
		case "ctrl+u":
			h.viewport.HalfPageUp()
		case "pgdown":
			h.viewport.PageDown()
		case "pgup":
			h.viewport.PageUp()
		case "g", "home":
			h.viewport.GotoTop()
		case "G", "end":
			h.viewport.GotoBottom()

		// Copy functionality
		case "y":
			// Copy current visible line at cursor position
			if h.viewport.YOffset < len(h.contentLines) {
				line := h.contentLines[h.viewport.YOffset]
				if err := utils.WriteToClipboard(line); err == nil {
					h.copiedMessage = fmt.Sprintf("📋 Copied: %s", truncateString(line, 40))
					h.copiedTime = time.Now()
					h.yankBlink = true
					h.yankBlinkTime = time.Now()
				}
			}
		case "Y":
			// Copy all content
			allContent := strings.Join(h.contentLines, "\n")
			if err := utils.WriteToClipboard(allContent); err == nil {
				h.copiedMessage = "📋 Copied all help content"
				h.copiedTime = time.Now()
				h.yankBlink = true
				h.yankBlinkTime = time.Now()
			}
		case "D":
			// Debug: Copy the entire rendered view to clipboard for debugging
			renderedView := h.View()
			if err := utils.WriteToClipboard(renderedView); err == nil {
				h.copiedMessage = "🐛 Copied debug snapshot to clipboard"
				h.copiedTime = time.Now()
				debug.LogToFilef("DEBUG SNAPSHOT: View copied to clipboard, %d chars\n", len(renderedView))
			}
		}
	}

	// Update viewport
	h.viewport, cmd = h.viewport.Update(msg)

	return h, cmd
}

// View renders the help view
func (h *HelpView) View() string {
	if !h.ready {
		return "Loading help..."
	}

	// Title bar
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Width(h.width).
		Align(lipgloss.Center).
		Render("📚 RepoBird CLI Help Documentation")

	// Viewport content
	viewportContent := h.viewport.View()

	// Add scroll indicator text
	scrollIndicator := ""
	percentScrolled := h.viewport.ScrollPercent()

	position := "TOP"
	if h.viewport.AtBottom() {
		position = "BOTTOM"
	} else if h.viewport.AtTop() {
		position = "TOP"
	} else if percentScrolled > 0 {
		position = fmt.Sprintf("%d%%", int(percentScrolled*100))
	}

	// Only show indicator if there's content to scroll
	if !h.viewport.AtTop() || !h.viewport.AtBottom() {
		scrollIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf(" [%s]", position))
	}

	// Box style with border (make it narrower to leave room for scrollbar)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(h.width - 4).  // Leave room for scrollbar
		Height(h.height - 3) // Account for title and status

	// Render the box with content
	boxedContent := boxStyle.Render(viewportContent)

	// Build scrollbar outside the box
	var finalContent string
	if !h.viewport.AtTop() || !h.viewport.AtBottom() {
		// The main box height is h.height - 3 (accounting for title and status)
		boxHeight := h.height - 3

		// Build scrollbar lines to match the box height
		scrollbarLines := h.buildScrollbarLines(boxHeight)

		// Join all lines into a single string
		scrollbarContent := strings.Join(scrollbarLines, "\n")

		// Use NormalBorder for proper connection between border and vertical lines
		// NormalBorder uses straight lines that connect properly to vertical bars
		scrollbarStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(false).
			BorderRight(false)

		scrollbarBox := scrollbarStyle.Render(scrollbarContent)

		// Join the box and scrollbar horizontally
		finalContent = lipgloss.JoinHorizontal(lipgloss.Top, boxedContent, scrollbarBox)
	} else {
		finalContent = boxedContent
	}

	// Status line
	shortHelp := "[↑↓/jk]scroll [Ctrl+u/d]halfpage [g/G]top/bottom [y]copy [?/q/ESC]back"

	// Show copy message if active
	statusText := shortHelp
	if h.copiedMessage != "" && time.Since(h.copiedTime) < 2*time.Second {
		statusText = h.copiedMessage
	}

	statusLine := h.statusLine.
		SetWidth(h.width).
		SetLeft("[HELP]").
		SetRight(scrollIndicator).
		SetHelp(statusText).
		Render()

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		finalContent,
		statusLine,
	)
}

// buildScrollbarLines creates scrollbar lines to match exact height
func (h *HelpView) buildScrollbarLines(totalHeight int) []string {
	// Don't subtract - we need to fill the exact height for proper alignment
	innerHeight := totalHeight

	if innerHeight <= 0 {
		return []string{}
	}

	// Calculate scrollbar metrics
	totalLines := len(h.contentLines)
	visibleLines := h.viewport.Height

	// Calculate thumb size and position
	thumbSize := max(1, (visibleLines*innerHeight)/totalLines)
	if thumbSize > innerHeight {
		thumbSize = innerHeight
	}

	percentScrolled := h.viewport.ScrollPercent()
	maxThumbPos := innerHeight - thumbSize
	thumbPos := int(float64(maxThumbPos) * percentScrolled)
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > maxThumbPos {
		thumbPos = maxThumbPos
	}

	// Build exactly innerHeight lines
	var lines []string
	for i := 0; i < innerHeight; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			// Thumb
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("63")).
				Render("█"))
		} else {
			// Track
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("238")).
				Render("│"))
		}
	}

	return lines
}

// buildSimpleScrollbar creates a standalone scrollbar as a single string
func (h *HelpView) buildSimpleScrollbar(height int) string {
	if height <= 0 {
		return ""
	}

	// Calculate scrollbar metrics
	totalLines := len(h.contentLines)
	visibleLines := h.viewport.Height

	// Calculate thumb size and position
	thumbSize := max(1, (visibleLines*height)/totalLines)
	if thumbSize > height {
		thumbSize = height
	}

	percentScrolled := h.viewport.ScrollPercent()
	maxThumbPos := height - thumbSize
	thumbPos := int(float64(maxThumbPos) * percentScrolled)
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > maxThumbPos {
		thumbPos = maxThumbPos
	}

	// Build scrollbar as array of lines
	var lines []string
	for i := 0; i < height; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			// Thumb
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("63")).
				Render("█"))
		} else {
			// Track
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("238")).
				Render("│"))
		}
	}

	return strings.Join(lines, "\n")
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

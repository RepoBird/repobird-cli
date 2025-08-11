package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/repobird/repobird-cli/internal/utils"
	"github.com/sahilm/fuzzy"
)

// TickMsg is sent periodically to update the cursor blink
type TickMsg time.Time

// tick returns a command that sends a TickMsg after a delay
func tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// ConfigFileSelector is an enhanced file selector with preview pane for config files
type ConfigFileSelector struct {
	files           []string
	filteredFiles   []string
	selectedIndex   int
	filterInput     string
	previewContent  string
	width           int
	height          int
	active          bool
	previewOffset   int
	maxPreviewLines int
	cursorVisible   bool // For blinking cursor animation
}

// NewConfigFileSelector creates a new config file selector with preview
func NewConfigFileSelector(width, height int) *ConfigFileSelector {
	return &ConfigFileSelector{
		width:           width,
		height:          height,
		maxPreviewLines: 100,
	}
}

// Activate activates the config file selector
func (cfs *ConfigFileSelector) Activate() error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find config files (JSON, YAML, and Markdown)
	configFiles, err := utils.FindConfigFiles(currentDir)
	if err != nil {
		return fmt.Errorf("failed to find config files: %w", err)
	}

	cfs.files = configFiles
	cfs.filteredFiles = configFiles
	cfs.active = true
	cfs.selectedIndex = 0
	cfs.filterInput = ""
	cfs.cursorVisible = true // Start with cursor visible

	if len(configFiles) > 0 {
		cfs.updatePreview()
	}

	return nil
}

// IsActive returns whether the selector is active
func (cfs *ConfigFileSelector) IsActive() bool {
	return cfs.active
}

// Deactivate deactivates the selector
func (cfs *ConfigFileSelector) Deactivate() {
	cfs.active = false
}

// SetDimensions sets the dimensions of the selector
func (cfs *ConfigFileSelector) SetDimensions(width, height int) {
	cfs.width = width
	cfs.height = height
}

// Update handles updates
func (cfs *ConfigFileSelector) Update(msg tea.Msg) (*ConfigFileSelector, tea.Cmd) {
	if !cfs.active {
		return cfs, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cfs.active = false
			return cfs, func() tea.Msg {
				return FZFSelectedMsg{
					Result: FZFResult{
						Canceled: true,
					},
				}
			}

		case "enter":
			if len(cfs.filteredFiles) > 0 {
				selected := cfs.filteredFiles[cfs.selectedIndex]
				cfs.active = false
				return cfs, func() tea.Msg {
					return FZFSelectedMsg{
						Result: FZFResult{
							Selected: selected,
							Index:    cfs.selectedIndex,
							Canceled: false,
						},
					}
				}
			}

		case "up", "ctrl+p":
			if cfs.selectedIndex > 0 {
				cfs.selectedIndex--
				cfs.updatePreview()
			}

		case "down", "ctrl+n":
			if cfs.selectedIndex < len(cfs.filteredFiles)-1 {
				cfs.selectedIndex++
				cfs.updatePreview()
			}

		case "pgup":
			cfs.selectedIndex = max(0, cfs.selectedIndex-10)
			cfs.updatePreview()

		case "pgdown":
			cfs.selectedIndex = min(len(cfs.filteredFiles)-1, cfs.selectedIndex+10)
			cfs.updatePreview()

		case "ctrl+u":
			cfs.previewOffset = max(0, cfs.previewOffset-10)

		case "ctrl+d":
			cfs.previewOffset = cfs.previewOffset + 10

		case "backspace":
			if len(cfs.filterInput) > 0 {
				cfs.filterInput = cfs.filterInput[:len(cfs.filterInput)-1]
				cfs.applyFilter()
			}

		case "ctrl+w":
			cfs.filterInput = ""
			cfs.applyFilter()

		default:
			// Handle all single character input for filtering, including 'j' and 'k'
			if len(msg.String()) == 1 {
				cfs.filterInput += msg.String()
				cfs.applyFilter()
			}
		}

	case tea.WindowSizeMsg:
		cfs.width = msg.Width
		cfs.height = msg.Height

	case TickMsg:
		// Toggle cursor visibility for blinking effect
		cfs.cursorVisible = !cfs.cursorVisible
		return cfs, tick()
	}

	return cfs, nil
}

func (cfs *ConfigFileSelector) applyFilter() {
	if cfs.filterInput == "" {
		cfs.filteredFiles = cfs.files
		cfs.selectedIndex = 0
		cfs.updatePreview()
		return
	}

	matches := fuzzy.Find(cfs.filterInput, cfs.files)
	cfs.filteredFiles = make([]string, len(matches))
	for i, match := range matches {
		cfs.filteredFiles[i] = match.Str
	}

	cfs.selectedIndex = 0
	cfs.updatePreview()
}

func (cfs *ConfigFileSelector) updatePreview() {
	if len(cfs.filteredFiles) == 0 || cfs.selectedIndex >= len(cfs.filteredFiles) {
		cfs.previewContent = "No file selected"
		return
	}

	filePath := cfs.filteredFiles[cfs.selectedIndex]
	content, err := os.ReadFile(filePath)
	if err != nil {
		cfs.previewContent = fmt.Sprintf("Error reading file: %v", err)
		return
	}

	// Apply syntax highlighting
	highlightedContent, err := highlightConfigFile(string(content), filePath)
	if err != nil {
		// Fall back to plain text if highlighting fails
		cfs.previewContent = string(content)
	} else {
		cfs.previewContent = highlightedContent
	}
	cfs.previewOffset = 0
}

func highlightConfigFile(code, filename string) (string, error) {
	// Try to match lexer by filename first
	lexer := lexers.Match(filename)
	if lexer == nil {
		// Try to determine lexer by extension
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".json":
			lexer = lexers.Get("json")
		case ".yaml", ".yml":
			lexer = lexers.Get("yaml")
		case ".md", ".markdown":
			lexer = lexers.Get("markdown")
		default:
			// Try to analyze the content
			lexer = lexers.Analyse(code)
			if lexer == nil {
				lexer = lexers.Fallback
			}
		}
	}

	// Use a terminal-friendly style
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal formatter for ANSI color output
	formatter := formatters.Get("terminal")
	if formatter == nil {
		// Try terminal256 as fallback
		formatter = formatters.Get("terminal256")
		if formatter == nil {
			formatter = formatters.Fallback
		}
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	// Format with colors
	var buf strings.Builder
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
}

// View renders the config file selector with split pane
func (cfs *ConfigFileSelector) View() string {
	if !cfs.active {
		return ""
	}

	// Styles
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	// Use full terminal dimensions
	// Status bar takes 1 line at bottom, 2 lines at top for border visibility
	availableHeight := cfs.height - 3
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Box height is the available height
	boxHeight := availableHeight

	// Account for border expansion: each box renders 2 chars wider than set width
	// So we need to subtract 4 total (2 per box) from terminal width
	totalWidth := cfs.width - 4

	// Split width between list and preview (40/60 split favoring preview)
	// No gap between panes to maximize space usage
	listWidth := int(float64(totalWidth) * 0.4)
	if listWidth < 30 {
		listWidth = 30
	}
	if listWidth > 50 {
		listWidth = 50
	}
	previewWidth := totalWidth - listWidth

	// Content height inside boxes (account for borders and headers)
	// boxHeight includes the borders, so content is boxHeight - 2
	contentHeight := boxHeight - 2
	// Reserve space for header and filter in list, just header in preview
	listContentHeight := contentHeight - 3    // Header + filter line + spacing
	previewContentHeight := contentHeight - 2 // Header + spacing

	// Build file list content
	var fileListContent []string

	// Add filter line with blinking cursor
	cursor := ""
	if cfs.cursorVisible {
		cursor = "█"
	} else {
		cursor = " "
	}
	filterLine := fmt.Sprintf("Filter: %s%s", cfs.filterInput, cursor)
	if len(filterLine) > listWidth-4 {
		// Ensure cursor is always visible by truncating the input, not the cursor
		maxInputLen := listWidth - 12 // "Filter: " + cursor + "..."
		if len(cfs.filterInput) > maxInputLen {
			filterLine = fmt.Sprintf("Filter: ...%s%s",
				cfs.filterInput[len(cfs.filterInput)-maxInputLen+3:], cursor)
		}
	}
	fileListContent = append(fileListContent, filterStyle.Render(filterLine))
	fileListContent = append(fileListContent, "") // Empty line

	// Show files or empty message
	if len(cfs.filteredFiles) == 0 {
		fileListContent = append(fileListContent, lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No config files found"))
	} else {
		// Calculate visible range
		visibleFiles := listContentHeight - 2 // Account for filter and empty line
		startIdx := 0
		if cfs.selectedIndex >= visibleFiles {
			startIdx = cfs.selectedIndex - visibleFiles + 1
		}
		endIdx := min(len(cfs.filteredFiles), startIdx+visibleFiles)

		for i := startIdx; i < endIdx; i++ {
			file := cfs.filteredFiles[i]

			// Add icon based on file type
			var icon string
			switch {
			case strings.HasSuffix(file, ".json"):
				icon = "📄 "
			case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
				icon = "📋 "
			case strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown"):
				icon = "📝 "
			default:
				icon = "📁 "
			}

			// Use base name for display
			displayName := filepath.Base(file)

			// Truncate if too long (account for icon)
			maxNameLen := listWidth - 6 - len(icon)
			if len(displayName) > maxNameLen {
				displayName = displayName[:maxNameLen-3] + "..."
			}

			line := icon + displayName

			if i == cfs.selectedIndex {
				fileListContent = append(fileListContent, selectedStyle.Render(line))
			} else {
				fileListContent = append(fileListContent, line)
			}
		}
	}

	// Pad file list to fixed height
	for len(fileListContent) < listContentHeight {
		fileListContent = append(fileListContent, "")
	}

	// Build preview content
	var previewContent []string

	if len(cfs.filteredFiles) > 0 && cfs.selectedIndex < len(cfs.filteredFiles) {
		lines := strings.Split(cfs.previewContent, "\n")
		previewStart := cfs.previewOffset
		if previewStart < 0 {
			previewStart = 0
		}
		if previewStart >= len(lines) {
			previewStart = max(0, len(lines)-1)
		}

		for i := 0; i < previewContentHeight && previewStart+i < len(lines); i++ {
			lineIdx := previewStart + i
			if lineIdx >= len(lines) {
				break
			}

			line := lines[lineIdx]

			// Replace tabs with spaces for consistent display
			line = strings.ReplaceAll(line, "\t", "    ")

			// Truncate long lines - ensure we don't exceed bounds
			// Use ANSI-aware truncation to preserve color codes
			// Account for box borders (2 chars) and minimal padding (2 chars)
			maxLineLen := previewWidth - 4
			if maxLineLen < 20 {
				maxLineLen = 20
			}

			// Use ANSI-aware width calculation and truncation
			if ansi.StringWidth(line) > maxLineLen {
				line = ansi.Truncate(line, maxLineLen, "...")
			}

			previewContent = append(previewContent, line)
		}

		// If no content was added, show placeholder
		if len(previewContent) == 0 {
			previewContent = append(previewContent, "")
		}
	} else {
		previewContent = append(previewContent, lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Select a file to preview"))
	}

	// Pad preview to fixed height
	for len(previewContent) < previewContentHeight {
		previewContent = append(previewContent, "")
	}

	// Create file list box
	fileListBox := borderStyle.
		Width(listWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("📁 Config Files"),
			strings.Join(fileListContent, "\n"),
		))

	// Create preview box
	previewBox := borderStyle.
		Width(previewWidth).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("👁️ Preview"),
			strings.Join(previewContent, "\n"),
		))

	// Combine both panes horizontally - no gap between them
	splitView := lipgloss.JoinHorizontal(lipgloss.Top, fileListBox, previewBox)

	// Add top margin to ensure borders are fully visible
	contentWithMargin := lipgloss.NewStyle().
		MarginTop(2).
		Render(splitView)

	// Status bar at the bottom (full width, like other views)
	statusBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("235")).
		Width(cfs.width).
		Padding(0, 1).
		Height(1)

	statusText := "[FZF] ↑↓: nav • Enter: select • ESC: cancel • Type to filter"
	statusBar := statusBarStyle.Render(statusText)

	// Join content and status bar vertically - content aligned to top
	fullView := lipgloss.JoinVertical(
		lipgloss.Left,
		contentWithMargin,
		statusBar,
	)

	return fullView
}

// GetSelectedFile returns the currently selected file
func (cfs *ConfigFileSelector) GetSelectedFile() string {
	if len(cfs.filteredFiles) > 0 && cfs.selectedIndex < len(cfs.filteredFiles) {
		return cfs.filteredFiles[cfs.selectedIndex]
	}
	return ""
}

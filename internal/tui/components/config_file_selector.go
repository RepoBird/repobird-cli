package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/utils"
	"github.com/sahilm/fuzzy"
)

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

		case "up", "k", "ctrl+p":
			if cfs.selectedIndex > 0 {
				cfs.selectedIndex--
				cfs.updatePreview()
			}

		case "down", "j", "ctrl+n":
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
			if len(msg.String()) == 1 {
				cfs.filterInput += msg.String()
				cfs.applyFilter()
			}
		}

	case tea.WindowSizeMsg:
		cfs.width = msg.Width
		cfs.height = msg.Height
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

	// For now, just use plain text to avoid ANSI issues
	// TODO: Implement proper ANSI-aware rendering
	cfs.previewContent = string(content)
	cfs.previewOffset = 0
}

func highlightConfigFile(code, filename string) (string, error) {
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
			lexer = lexers.Fallback
		}
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

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

	// Use full height minus status bar (like other views)
	availableHeight := cfs.height - 1 // Reserve 1 line for status bar
	if availableHeight < 10 {
		availableHeight = 10
	}
	
	// Use more of the available width
	totalWidth := cfs.width - 4
	halfWidth := totalWidth / 2
	listWidth := min(halfWidth-2, 50) // Cap at 50 for readability
	previewWidth := totalWidth - listWidth - 3
	
	// Content area height (account for borders and padding)
	boxHeight := availableHeight - 2 // Leave space for help text
	contentHeight := boxHeight - 4 // Account for borders and headers
	listContentHeight := contentHeight - 1 // Extra line for filter
	previewContentHeight := contentHeight

	// Build file list content
	var fileListContent []string
	
	// Add filter line
	filterLine := fmt.Sprintf("Filter: %s", cfs.filterInput)
	if len(filterLine) > listWidth-4 {
		filterLine = filterLine[:listWidth-7] + "..."
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
			maxLineLen := previewWidth - 6
			if maxLineLen < 10 {
				maxLineLen = 10
			}
			
			// Handle rune-safe truncation
			runes := []rune(line)
			if len(runes) > maxLineLen {
				line = string(runes[:maxLineLen-3]) + "..."
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

	// Create file list box with fixed dimensions
	fileListBox := borderStyle.
		Width(listWidth).
		Height(maxHeight).
		MaxWidth(listWidth).
		MaxHeight(maxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("📁 Config Files"),
			strings.Join(fileListContent, "\n"),
		))

	// Create preview box with fixed dimensions
	previewBox := borderStyle.
		Width(previewWidth).
		Height(maxHeight).
		MaxWidth(previewWidth).
		MaxHeight(maxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("👁️ Preview"),
			strings.Join(previewContent, "\n"),
		))

	// Combine both panes horizontally
	splitView := lipgloss.JoinHorizontal(lipgloss.Top, fileListBox, " ", previewBox)

	// Add help text at the bottom
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		MaxWidth(totalWidth)
	
	helpText := helpStyle.Render("↑↓/jk: navigate • Enter: select • ESC: cancel • Type to filter • Ctrl+u/d: scroll")

	// Use Place to center everything
	return lipgloss.Place(
		cfs.width,
		cfs.height,
		lipgloss.Center,
		lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Left, splitView, helpText),
	)
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	// For now, just return the original string since we're using plain text
	// This avoids any issues with ANSI code handling
	return str
}

// GetSelectedFile returns the currently selected file
func (cfs *ConfigFileSelector) GetSelectedFile() string {
	if len(cfs.filteredFiles) > 0 && cfs.selectedIndex < len(cfs.filteredFiles) {
		return cfs.filteredFiles[cfs.selectedIndex]
	}
	return ""
}
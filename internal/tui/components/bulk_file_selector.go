package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/utils"
	pkgutils "github.com/repobird/repobird-cli/pkg/utils"
	"github.com/sahilm/fuzzy"
)

// BulkFileSelector is a file selector for bulk operations supporting multiple selections
type BulkFileSelector struct {
	// Core data
	files         []FileItem
	filteredFiles []FileItem
	selectedFiles map[string]bool // Map of file paths to selected state

	// UI state
	cursor         int
	filterInput    string
	active         bool
	loading        bool
	previewContent string
	previewOffset  int

	// Dimensions
	width  int
	height int

	// Git root for relative paths
	gitRoot string

	// Error state
	loadError error
}

// FileItem represents a file in the selector
type FileItem struct {
	Path     string // Full absolute path
	RelPath  string // Relative path from git root
	Display  string // Display name (relative from git root)
	Selected bool   // Whether this file is selected
}

// BulkFileSelectedMsg is sent when files are selected
type BulkFileSelectedMsg struct {
	Files    []string // List of selected file paths
	Canceled bool
}

// filesLoadedMsg is sent when files are loaded
type filesLoadedMsg struct {
	files     []FileItem
	err       error
	isPartial bool // Indicates if more files are coming
}

// getGitRoot returns the git repository root path
func getGitRoot() string {
	// Use pkgutils.GetGitInfo which should be available
	repo, _, err := pkgutils.GetGitInfo()
	if err == nil && repo != "" {
		// Try to get the actual git root directory
		if cwd, err := os.Getwd(); err == nil {
			// Walk up from current directory to find .git
			dir := cwd
			for {
				if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
					return dir
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
		}
	}
	// Fallback to current directory
	cwd, _ := os.Getwd()
	return cwd
}

// NewBulkFileSelector creates a new bulk file selector
func NewBulkFileSelector(width, height int) *BulkFileSelector {
	gitRoot := getGitRoot()

	return &BulkFileSelector{
		width:         width,
		height:        height,
		selectedFiles: make(map[string]bool),
		gitRoot:       gitRoot,
		files:         []FileItem{},
		filteredFiles: []FileItem{},
	}
}

// SetActive sets the active state
func (b *BulkFileSelector) SetActive(active bool) {
	b.active = active
}

// IsActive returns whether the selector is active
func (b *BulkFileSelector) IsActive() bool {
	return b.active
}

// SetDimensions updates the dimensions
func (b *BulkFileSelector) SetDimensions(width, height int) {
	b.width = width
	b.height = height
}

// Activate activates the selector and starts loading files
func (b *BulkFileSelector) Activate() tea.Cmd {
	b.active = true
	b.loading = true
	b.loadError = nil
	// Return command to load files progressively
	return b.LoadFilesProgressiveCmd()
}

// LoadFilesCmd returns a command to load files asynchronously
func (b *BulkFileSelector) LoadFilesCmd() tea.Cmd {
	return func() tea.Msg {
		files, err := b.findConfigFiles()
		return filesLoadedMsg{files: files, err: err}
	}
}

// LoadFilesProgressiveCmd returns a command to load files progressively
func (b *BulkFileSelector) LoadFilesProgressiveCmd() tea.Cmd {
	return func() tea.Msg {
		currentDir, err := os.Getwd()
		if err != nil {
			return filesLoadedMsg{files: nil, err: err}
		}

		// Start with a reasonable depth to find files in subdirectories like run-tasks/
		opts := utils.FileDiscoveryOptions{
			MaxDepth:       5, // Increase depth to find files in subdirectories
			IgnorePatterns: utils.DefaultIgnorePatterns,
			FileExtensions: []string{".json", ".yaml", ".yml", ".jsonl", ".md", ".markdown"},
			SortByModTime:  true,
			MaxFiles:       200, // Increase file limit for more files
		}

		files, _ := utils.FindFiles(currentDir, opts)

		// Convert to FileItems
		items := make([]FileItem, 0, len(files))
		for _, file := range files {
			absPath, _ := filepath.Abs(file)
			relPath := b.getRelativePath(absPath)
			items = append(items, FileItem{
				Path:     absPath,
				RelPath:  relPath,
				Display:  relPath,
				Selected: b.selectedFiles[absPath],
			})
		}

		// Return initial batch immediately
		return filesLoadedMsg{files: items, err: nil}
	}
}

// findConfigFiles finds only the config files we care about
func (b *BulkFileSelector) findConfigFiles() ([]FileItem, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Use utils.FindFiles with proper options for config files
	opts := utils.FileDiscoveryOptions{
		MaxDepth:       5,                           // Increase depth to find files in subdirectories like run-tasks/
		IgnorePatterns: utils.DefaultIgnorePatterns, // Use the default ignore patterns
		FileExtensions: []string{".json", ".yaml", ".yml", ".jsonl", ".md", ".markdown"},
		SortByModTime:  true,
		MaxFiles:       500, // Increase limit for more files in subdirectories
	}

	configFiles, err := utils.FindFiles(currentDir, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}

	// Create FileItems
	items := make([]FileItem, 0, len(configFiles))
	for _, file := range configFiles {
		absPath, _ := filepath.Abs(file)
		relPath := b.getRelativePath(absPath)

		item := FileItem{
			Path:     absPath,
			RelPath:  relPath,
			Display:  relPath,
			Selected: b.selectedFiles[absPath],
		}
		items = append(items, item)
	}

	return items, nil
}

// getRelativePath gets the relative path from git root or current directory
func (b *BulkFileSelector) getRelativePath(absPath string) string {
	if b.gitRoot != "" {
		if rel, err := filepath.Rel(b.gitRoot, absPath); err == nil {
			return rel
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, absPath); err == nil {
			return rel
		}
	}

	return filepath.Base(absPath)
}

// Update handles updates
func (b *BulkFileSelector) Update(msg tea.Msg) (*BulkFileSelector, tea.Cmd) {
	if !b.active {
		return b, nil
	}

	switch msg := msg.(type) {
	case filesLoadedMsg:
		if msg.err != nil {
			b.loading = false
			b.loadError = msg.err
			b.files = []FileItem{}
			b.filteredFiles = []FileItem{}
		} else {
			if msg.isPartial {
				// Append to existing files for progressive loading
				b.files = append(b.files, msg.files...)
			} else {
				// Replace all files
				b.files = msg.files
				b.loading = false
			}
			// Re-apply filter
			b.applyFilter()
		}
		if b.cursor >= len(b.filteredFiles) {
			b.cursor = 0
		}
		// Update preview for the first file
		b.updatePreview()
		return b, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "ctrl+[":
			b.active = false
			return b, func() tea.Msg {
				return BulkFileSelectedMsg{Canceled: true}
			}

		case "enter":
			// Collect all selected files
			var selected []string
			for _, file := range b.files {
				if file.Selected {
					selected = append(selected, file.Path)
				}
			}

			// Allow 1 or more files (single config file can contain multiple runs)
			if len(selected) >= 1 {
				b.active = false
				return b, func() tea.Msg {
					return BulkFileSelectedMsg{
						Files:    selected,
						Canceled: false,
					}
				}
			}

		case " ", "space":
			// Toggle selection for current item
			if b.cursor < len(b.filteredFiles) {
				item := &b.filteredFiles[b.cursor]
				item.Selected = !item.Selected
				b.selectedFiles[item.Path] = item.Selected

				// Update the original item too
				for i := range b.files {
					if b.files[i].Path == item.Path {
						b.files[i].Selected = item.Selected
						break
					}
				}

				// Move to next item after selection
				if b.cursor < len(b.filteredFiles)-1 {
					b.cursor++
				}
			}

		case "ctrl+a":
			// Select all visible items
			for i := range b.filteredFiles {
				b.filteredFiles[i].Selected = true
				b.selectedFiles[b.filteredFiles[i].Path] = true

				// Update original items
				for j := range b.files {
					if b.files[j].Path == b.filteredFiles[i].Path {
						b.files[j].Selected = true
						break
					}
				}
			}

		case "ctrl+d":
			// Deselect all (ctrl+d for deselect)
			for i := range b.filteredFiles {
				b.filteredFiles[i].Selected = false
				b.selectedFiles[b.filteredFiles[i].Path] = false

				// Update original items
				for j := range b.files {
					if b.files[j].Path == b.filteredFiles[i].Path {
						b.files[j].Selected = false
						break
					}
				}
			}

		case "up", "ctrl+p", "ctrl+k":
			if b.cursor > 0 {
				b.cursor--
			} else if len(b.filteredFiles) > 0 {
				// Wraparound to bottom
				b.cursor = len(b.filteredFiles) - 1
			}
			b.updatePreview()

		case "down", "ctrl+n", "ctrl+j":
			if b.cursor < len(b.filteredFiles)-1 {
				b.cursor++
			} else if len(b.filteredFiles) > 0 {
				// Wraparound to top
				b.cursor = 0
			}
			b.updatePreview()

		case "pgup":
			b.cursor = max(0, b.cursor-10)

		case "pgdown":
			b.cursor = min(len(b.filteredFiles)-1, b.cursor+10)

		case "backspace":
			if len(b.filterInput) > 0 {
				b.filterInput = b.filterInput[:len(b.filterInput)-1]
				b.applyFilter()
			}

		case "ctrl+w":
			b.filterInput = ""
			b.applyFilter()

		default:
			// Handle single character input for filtering
			if len(msg.String()) == 1 {
				b.filterInput += msg.String()
				b.applyFilter()
			}
		}

	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height

	case TickMsg:
		// Ignore tick messages to prevent lag
		return b, nil
	}

	return b, nil
}

// applyFilter applies the fuzzy filter to files
func (b *BulkFileSelector) applyFilter() {
	if b.filterInput == "" {
		b.filteredFiles = b.files
		// Don't reset cursor to prevent jumping
		if b.cursor >= len(b.filteredFiles) {
			b.cursor = max(0, len(b.filteredFiles)-1)
		}
		b.updatePreview()
		return
	}

	// Create string slice for fuzzy matching
	fileNames := make([]string, len(b.files))
	for i, file := range b.files {
		fileNames[i] = file.Display
	}

	// Apply fuzzy filter
	matches := fuzzy.Find(b.filterInput, fileNames)
	b.filteredFiles = make([]FileItem, len(matches))
	for i, match := range matches {
		b.filteredFiles[i] = b.files[match.Index]
	}

	// Keep cursor in bounds but don't reset to 0
	if b.cursor >= len(b.filteredFiles) {
		b.cursor = max(0, len(b.filteredFiles)-1)
	}
	b.updatePreview()
}

// updatePreview updates the preview content for the currently selected file
func (b *BulkFileSelector) updatePreview() {
	if len(b.filteredFiles) == 0 || b.cursor >= len(b.filteredFiles) {
		b.previewContent = "No file selected"
		return
	}

	filePath := b.filteredFiles[b.cursor].Path
	content, err := os.ReadFile(filePath)
	if err != nil {
		b.previewContent = fmt.Sprintf("Error reading file: %v", err)
		return
	}

	// Limit preview size
	maxPreviewSize := 5000
	if len(content) > maxPreviewSize {
		content = content[:maxPreviewSize]
	}

	b.previewContent = string(content)
	b.previewOffset = 0
}

// View renders the file selector
func (b *BulkFileSelector) View(statusLine *StatusLine) string {
	if !b.active {
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

	checkboxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	// Calculate dimensions - split screen between file list and preview
	availableHeight := b.height - 3 // Reserve for statusline
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Split width between file list (60%) and preview (40%)
	fileListWidth := int(float64(b.width) * 0.6)
	previewWidth := b.width - fileListWidth - 1 // -1 for gap

	boxHeight := availableHeight
	contentHeight := boxHeight - 2            // Account for borders
	listContentHeight := contentHeight - 3    // Header + filter + spacing
	previewContentHeight := contentHeight - 2 // Header + spacing

	// Build content
	var content []string

	// Add filter line with cursor (always visible)
	cursor := "█"
	filterLine := fmt.Sprintf("Filter: %s%s", b.filterInput, cursor)
	content = append(content, filterStyle.Render(filterLine))
	content = append(content, "") // Empty line

	// Count selected files
	selectedCount := 0
	for _, file := range b.files {
		if file.Selected {
			selectedCount++
		}
	}

	// Show loading, error, or files
	if b.loading {
		content = append(content, lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Loading config files..."))
	} else if b.loadError != nil {
		content = append(content, lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(fmt.Sprintf("Error: %v", b.loadError)))
	} else if len(b.filteredFiles) == 0 {
		content = append(content, lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No config files found (JSON, YAML, JSONL, Markdown)"))
	} else {
		// Calculate visible range
		visibleFiles := listContentHeight - 2
		startIdx := 0
		if b.cursor >= visibleFiles {
			startIdx = b.cursor - visibleFiles + 1
		}
		endIdx := min(len(b.filteredFiles), startIdx+visibleFiles)

		for i := startIdx; i < endIdx; i++ {
			file := b.filteredFiles[i]

			// Checkbox
			checkbox := "[ ]"
			if file.Selected {
				checkbox = checkboxStyle.Render("[✓]")
			}

			// Icon based on file type
			var icon string
			switch {
			case strings.HasSuffix(file.Path, ".json"):
				icon = "📄"
			case strings.HasSuffix(file.Path, ".jsonl"):
				icon = "📋"
			case strings.HasSuffix(file.Path, ".yaml"), strings.HasSuffix(file.Path, ".yml"):
				icon = "📑"
			case strings.HasSuffix(file.Path, ".md"), strings.HasSuffix(file.Path, ".markdown"):
				icon = "📝"
			default:
				icon = "📁"
			}

			// Build line
			line := fmt.Sprintf("%s %s %s", checkbox, icon, file.Display)

			// Apply cursor style
			if i == b.cursor {
				// Preserve checkbox coloring
				if file.Selected {
					parts := strings.SplitN(line, " ", 2)
					line = checkboxStyle.Render(parts[0]) + " " + selectedStyle.Render(strings.Join(parts[1:], " "))
				} else {
					line = selectedStyle.Render(line)
				}
			}

			content = append(content, line)
		}

		// Show scroll indicators if needed
		if startIdx > 0 {
			content[2] = "↑ " + content[2] // Add up arrow to first file line
		}
		if endIdx < len(b.filteredFiles) {
			content[len(content)-1] = content[len(content)-1] + " ↓" // Add down arrow to last file line
		}
	}

	// Pad file list to fixed height
	for len(content) < listContentHeight {
		content = append(content, "")
	}

	// Create file list box
	fileListBox := borderStyle.
		Width(fileListWidth - 2).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(fmt.Sprintf("📁 Bulk Config Files (%d selected)", selectedCount)),
			strings.Join(content, "\n"),
		))

	// Build preview pane
	var previewContent []string
	if len(b.filteredFiles) > 0 && b.cursor < len(b.filteredFiles) {
		lines := strings.Split(b.previewContent, "\n")
		previewStart := b.previewOffset
		if previewStart < 0 {
			previewStart = 0
		}
		if previewStart >= len(lines) {
			previewStart = max(0, len(lines)-1)
		}

		for i := 0; i < previewContentHeight && previewStart+i < len(lines); i++ {
			line := lines[previewStart+i]
			// Truncate long lines for preview
			maxLineLen := previewWidth - 6
			if len(line) > maxLineLen {
				line = line[:maxLineLen] + "..."
			}
			previewContent = append(previewContent, line)
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

	// Create preview box
	previewBox := borderStyle.
		Width(previewWidth - 2).
		Height(boxHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("👁️ Preview"),
			strings.Join(previewContent, "\n"),
		))

	// Combine both panes horizontally
	combinedPanes := lipgloss.JoinHorizontal(lipgloss.Top, fileListBox, previewBox)

	// Add top margin for visibility
	contentWithMargin := lipgloss.NewStyle().
		MarginTop(2).
		Render(combinedPanes)

	// Setup statusline with better text visibility
	if statusLine != nil {
		// Build status text parts - selection count already shown in file list header
		leftStatus := "[FZF]"

		// Use SetHelp to put commands right after the label instead of far right
		statusLine.SetWidth(b.width).
			SetLeft(leftStatus).
			SetRight("").
			SetHelp("↑↓/Ctrl+K/J:nav | Space:toggle | Ctrl+A:all | Ctrl+D:none | Enter:submit | Esc:cancel")

		// Join content and status bar
		return lipgloss.JoinVertical(
			lipgloss.Left,
			contentWithMargin,
			statusLine.Render(),
		)
	}

	// Fallback without statusline (shouldn't happen)
	return contentWithMargin
}

// GetSelectedFiles returns the list of selected file paths
func (b *BulkFileSelector) GetSelectedFiles() []string {
	var selected []string
	for _, file := range b.files {
		if file.Selected {
			selected = append(selected, file.Path)
		}
	}
	return selected
}

package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/utils"
)

// FileSelector provides FZF-based file selection functionality
type FileSelector struct {
	fzfMode     *FZFMode
	currentPath string
	active      bool
	width       int
	height      int
}

// FileSelectorOptions configures file selection behavior
type FileSelectorOptions struct {
	MaxDepth       int
	IgnorePatterns []string
	FileExtensions []string
	MaxFiles       int
}

// NewFileSelector creates a new file selector
func NewFileSelector(width, height int) *FileSelector {
	return &FileSelector{
		currentPath: ".",
		active:      false,
		width:       width,
		height:      height,
	}
}

// ActivateJSONFileSelector activates FZF mode for JSON file selection
func (fs *FileSelector) ActivateJSONFileSelector() error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find config files (JSON and Markdown)
	configFiles, err := utils.FindConfigFiles(currentDir)
	if err != nil {
		return fmt.Errorf("failed to find config files: %w", err)
	}

	// Add file type indicators
	var formattedFiles []string
	for _, file := range configFiles {
		var icon string
		switch {
		case strings.HasSuffix(file, ".json"):
			icon = "📄"
		case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
			icon = "📋"
		case strings.HasSuffix(file, ".md") || strings.HasSuffix(file, ".markdown"):
			icon = "📝"
		default:
			icon = "📁"
		}
		formattedFiles = append(formattedFiles, fmt.Sprintf("%s %s", icon, file))
	}

	// If no config files found, create an empty list with a helpful message
	if len(formattedFiles) == 0 {
		formattedFiles = []string{"📝 No config files (.json, .yaml, .md) found - type full path"}
	}

	// Add manual entry hint at the top
	items := []string{"📁 Browse files or type full path..."}
	items = append(items, formattedFiles...)

	// Create and activate FZF mode
	fs.fzfMode = NewFZFMode(items, fs.width, fs.height)
	fs.fzfMode.Activate()
	fs.active = true

	return nil
}

// IsActive returns whether file selector is currently active
func (fs *FileSelector) IsActive() bool {
	return fs.active && fs.fzfMode != nil && fs.fzfMode.IsActive()
}

// Update handles file selector updates
func (fs *FileSelector) Update(msg tea.Msg) (*FileSelector, tea.Cmd) {
	if !fs.IsActive() {
		return fs, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle manual path entry
		if msg.String() == "enter" {
			// Check if user typed a custom path
			if fs.fzfMode != nil && fs.fzfMode.Input.Value() != "" {
				inputValue := strings.TrimSpace(fs.fzfMode.Input.Value())

				// If input looks like a file path and doesn't match any filtered item
				if fs.isManualPath(inputValue) {
					fs.deactivate()
					return fs, func() tea.Msg {
						return FZFSelectedMsg{
							Result: FZFResult{
								Selected: inputValue,
								Index:    -1, // Indicates manual entry
								Canceled: false,
							},
						}
					}
				}
			}
		}

		// Pass through to FZF mode
		if fs.fzfMode != nil {
			newFzf, cmd := fs.fzfMode.Update(msg)
			fs.fzfMode = newFzf

			// Check if FZF mode was deactivated
			if !fs.fzfMode.IsActive() {
				fs.active = false
			}

			return fs, cmd
		}

	case FZFSelectedMsg:
		// File was selected, deactivate
		fs.deactivate()
		return fs, nil
	}

	return fs, nil
}

// isManualPath determines if the input looks like a manual file path
func (fs *FileSelector) isManualPath(input string) bool {
	// Consider it a manual path if:
	// 1. It contains path separators
	// 2. It ends with .json, .yaml, .yml, .md, or .markdown
	// 3. It starts with / or . or ~
	// 4. It doesn't match any of the filtered items exactly

	if strings.Contains(input, string(filepath.Separator)) ||
		strings.HasSuffix(input, ".json") ||
		strings.HasSuffix(input, ".yaml") ||
		strings.HasSuffix(input, ".yml") ||
		strings.HasSuffix(input, ".md") ||
		strings.HasSuffix(input, ".markdown") ||
		strings.HasPrefix(input, "/") ||
		strings.HasPrefix(input, "./") ||
		strings.HasPrefix(input, "../") ||
		strings.HasPrefix(input, "~/") {
		// Check if it matches any filtered item exactly
		if fs.fzfMode != nil {
			for _, item := range fs.fzfMode.FilteredItems {
				// Remove any emoji prefixes for comparison
				cleanItem := strings.TrimSpace(strings.TrimLeft(item, "📁📄📝🔍"))
				if cleanItem == input {
					return false // It matches an existing item
				}
			}
		}
		return true
	}

	return false
}

// deactivate deactivates the file selector
func (fs *FileSelector) deactivate() {
	fs.active = false
	if fs.fzfMode != nil {
		fs.fzfMode.Deactivate()
	}
}

// View renders the file selector
func (fs *FileSelector) View() string {
	if !fs.IsActive() || fs.fzfMode == nil {
		return ""
	}
	return fs.fzfMode.View()
}

// Deactivate manually deactivates the file selector
func (fs *FileSelector) Deactivate() {
	fs.deactivate()
}

// SetDimensions updates the selector dimensions
func (fs *FileSelector) SetDimensions(width, height int) {
	fs.width = width
	fs.height = height
	if fs.fzfMode != nil {
		fs.fzfMode.Width = width
		fs.fzfMode.Height = height
	}
}

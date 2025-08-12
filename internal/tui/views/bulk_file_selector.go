package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/bulk"
)

// BulkFileSelector component handles file selection for bulk operations
type BulkFileSelector struct {
	files         []FileItem
	selected      map[string]bool
	filteredFiles []FileItem
	filterInput   textinput.Model
	mode          SelectMode
	cursor        int
	viewport      int
	height        int
}

type SelectMode int

const (
	SingleSelect SelectMode = iota
	MultiSelect
	DirectorySelect
)

type FileItem struct {
	Path        string
	Name        string
	Type        FileType
	Size        int64
	Modified    time.Time
	RunCount    int
	IsDirectory bool
}

type FileType int

const (
	FileTypeJSON FileType = iota
	FileTypeYAML
	FileTypeMarkdown
	FileTypeJSONL
	FileTypeUnknown
)

type filesLoadedMsg struct {
	files []FileItem
}

func NewBulkFileSelector() *BulkFileSelector {
	ti := textinput.New()
	ti.Placeholder = "Filter files..."
	ti.CharLimit = 100

	return &BulkFileSelector{
		files:       []FileItem{},
		selected:    make(map[string]bool),
		filterInput: ti,
		mode:        MultiSelect,
	}
}

func (s *BulkFileSelector) Init() tea.Cmd {
	return s.loadFiles()
}

func (s *BulkFileSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.filteredFiles)-1 {
				s.cursor++
			}
		case " ":
			// Toggle selection
			if s.cursor < len(s.filteredFiles) {
				file := s.filteredFiles[s.cursor]
				s.selected[file.Path] = !s.selected[file.Path]
			}
		case "enter":
			// Submit selected files
			var selectedFiles []string
			for path, isSelected := range s.selected {
				if isSelected {
					selectedFiles = append(selectedFiles, path)
				}
			}
			if len(selectedFiles) > 0 {
				return s, func() tea.Msg {
					return fileSelectedMsg{files: selectedFiles}
				}
			}
		case "a":
			// Select all
			for _, file := range s.filteredFiles {
				s.selected[file.Path] = true
			}
		case "n":
			// Select none
			s.selected = make(map[string]bool)
		}

	case filesLoadedMsg:
		s.files = msg.files
		s.filteredFiles = msg.files
	}

	// Update filter input
	var cmd tea.Cmd
	s.filterInput, cmd = s.filterInput.Update(msg)

	// Apply filter
	s.applyFilter()

	return s, cmd
}

func (s *BulkFileSelector) View() string {
	var b strings.Builder

	b.WriteString(s.filterInput.View() + "\n\n")

	// File list
	for i, file := range s.filteredFiles {
		prefix := "  "
		if i == s.cursor {
			prefix = "> "
		}

		checkbox := "[ ]"
		if s.selected[file.Path] {
			checkbox = "[✓]"
		}

		fileType := s.getFileTypeString(file.Type)
		line := fmt.Sprintf("%s %s %s (%s)", prefix, checkbox, file.Name, fileType)

		if i == s.cursor {
			selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
			line = selectedStyle.Render(line)
		}

		b.WriteString(line + "\n")
	}

	// Instructions
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"space: toggle | a: all | n: none | enter: submit",
	))

	return b.String()
}

func (s *BulkFileSelector) loadFiles() tea.Cmd {
	return func() tea.Msg {
		// Find configuration files in current directory
		var files []FileItem

		// Look for common patterns
		patterns := []string{
			"*.json",
			"*.yaml", "*.yml",
			"*.jsonl",
			"*.md", "*.markdown",
			"tasks/*.json",
			"tasks/*.yaml", "tasks/*.yml",
			"bulk/*.json",
			"bulk/*.yaml", "bulk/*.yml",
		}

		for _, pattern := range patterns {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}

			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil {
					continue
				}

				if info.IsDir() {
					continue
				}

				// Determine file type
				fileType := s.detectFileType(match)

				// Check if it's a bulk config
				isBulk, _ := bulk.IsBulkConfig(match)
				runCount := 1
				if isBulk {
					// Try to load and count runs
					if config, err := bulk.ParseBulkConfig(match); err == nil {
						runCount = len(config.Runs)
					}
				}

				files = append(files, FileItem{
					Path:     match,
					Name:     filepath.Base(match),
					Type:     fileType,
					Size:     info.Size(),
					Modified: info.ModTime(),
					RunCount: runCount,
				})
			}
		}

		return filesLoadedMsg{files: files}
	}
}

func (s *BulkFileSelector) applyFilter() {
	filter := strings.ToLower(s.filterInput.Value())
	if filter == "" {
		s.filteredFiles = s.files
		return
	}

	s.filteredFiles = []FileItem{}
	for _, file := range s.files {
		if strings.Contains(strings.ToLower(file.Name), filter) ||
			strings.Contains(strings.ToLower(file.Path), filter) {
			s.filteredFiles = append(s.filteredFiles, file)
		}
	}

	// Reset cursor if out of bounds
	if s.cursor >= len(s.filteredFiles) {
		s.cursor = len(s.filteredFiles) - 1
		if s.cursor < 0 {
			s.cursor = 0
		}
	}
}

func (s *BulkFileSelector) detectFileType(path string) FileType {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return FileTypeJSON
	case ".yaml", ".yml":
		return FileTypeYAML
	case ".md", ".markdown":
		return FileTypeMarkdown
	case ".jsonl":
		return FileTypeJSONL
	default:
		return FileTypeUnknown
	}
}

func (s *BulkFileSelector) getFileTypeString(t FileType) string {
	switch t {
	case FileTypeJSON:
		return "JSON"
	case FileTypeYAML:
		return "YAML"
	case FileTypeMarkdown:
		return "Markdown"
	case FileTypeJSONL:
		return "JSONL"
	default:
		return "Unknown"
	}
}
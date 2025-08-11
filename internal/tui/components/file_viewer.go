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
	"github.com/sahilm/fuzzy"
)

type FileViewer struct {
	files           []string
	filteredFiles   []string
	selectedIndex   int
	filterInput     string
	previewContent  string
	width           int
	height          int
	focused         bool
	showPreview     bool
	previewOffset   int
	maxPreviewLines int
}

func NewFileViewer(rootPath string) (*FileViewer, error) {
	files, err := collectFiles(rootPath)
	if err != nil {
		return nil, err
	}

	fv := &FileViewer{
		files:           files,
		filteredFiles:   files,
		showPreview:     true,
		maxPreviewLines: 100,
	}

	if len(files) > 0 {
		fv.updatePreview()
	}

	return fv, nil
}

func collectFiles(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") && path != rootPath {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common non-source directories
		dirName := filepath.Base(path)
		if info.IsDir() && (dirName == "vendor" || dirName == "node_modules" || dirName == "build" || dirName == "dist") {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				relPath = path
			}
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

func (fv *FileViewer) Init() tea.Cmd {
	return nil
}

func (fv *FileViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !fv.focused {
			return fv, nil
		}

		switch msg.String() {
		case "esc":
			fv.focused = false
			return fv, nil

		case "enter":
			if len(fv.filteredFiles) > 0 {
				return fv, tea.Quit
			}

		case "up", "k", "ctrl+p":
			if fv.selectedIndex > 0 {
				fv.selectedIndex--
				fv.updatePreview()
			}

		case "down", "j", "ctrl+n":
			if fv.selectedIndex < len(fv.filteredFiles)-1 {
				fv.selectedIndex++
				fv.updatePreview()
			}

		case "pgup":
			fv.selectedIndex = max(0, fv.selectedIndex-10)
			fv.updatePreview()

		case "pgdown":
			fv.selectedIndex = min(len(fv.filteredFiles)-1, fv.selectedIndex+10)
			fv.updatePreview()

		case "ctrl+u":
			fv.previewOffset = max(0, fv.previewOffset-10)

		case "ctrl+d":
			fv.previewOffset = fv.previewOffset + 10

		case "backspace":
			if len(fv.filterInput) > 0 {
				fv.filterInput = fv.filterInput[:len(fv.filterInput)-1]
				fv.applyFilter()
			}

		case "ctrl+w":
			fv.filterInput = ""
			fv.applyFilter()

		case "tab":
			fv.showPreview = !fv.showPreview

		default:
			if len(msg.String()) == 1 {
				fv.filterInput += msg.String()
				fv.applyFilter()
			}
		}

	case tea.WindowSizeMsg:
		fv.width = msg.Width
		fv.height = msg.Height
	}

	return fv, nil
}

func (fv *FileViewer) applyFilter() {
	if fv.filterInput == "" {
		fv.filteredFiles = fv.files
		fv.selectedIndex = 0
		fv.updatePreview()
		return
	}

	matches := fuzzy.Find(fv.filterInput, fv.files)
	fv.filteredFiles = make([]string, len(matches))
	for i, match := range matches {
		fv.filteredFiles[i] = match.Str
	}

	fv.selectedIndex = 0
	fv.updatePreview()
}

func (fv *FileViewer) updatePreview() {
	if len(fv.filteredFiles) == 0 || fv.selectedIndex >= len(fv.filteredFiles) {
		fv.previewContent = "No file selected"
		return
	}

	filePath := fv.filteredFiles[fv.selectedIndex]
	content, err := os.ReadFile(filePath)
	if err != nil {
		fv.previewContent = fmt.Sprintf("Error reading file: %v", err)
		return
	}

	// Apply syntax highlighting
	highlighted, err := highlightCode(string(content), filePath)
	if err != nil {
		// Fallback to plain text if highlighting fails
		fv.previewContent = string(content)
	} else {
		fv.previewContent = highlighted
	}

	fv.previewOffset = 0
}

func highlightCode(code, filename string) (string, error) {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
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

func (fv *FileViewer) View() string {
	if !fv.focused {
		return ""
	}

	// Styles
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Calculate dimensions
	listWidth := fv.width / 3
	previewWidth := fv.width - listWidth - 4
	contentHeight := fv.height - 4

	// Build file list
	var fileList strings.Builder
	fileList.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s", fv.filterInput)))
	fileList.WriteString("\n\n")

	startIdx := max(0, fv.selectedIndex-contentHeight/2)
	endIdx := min(len(fv.filteredFiles), startIdx+contentHeight-2)

	for i := startIdx; i < endIdx; i++ {
		file := fv.filteredFiles[i]
		if len(file) > listWidth-4 {
			file = file[:listWidth-7] + "..."
		}

		if i == fv.selectedIndex {
			fileList.WriteString(selectedStyle.Render(file))
		} else {
			fileList.WriteString(file)
		}
		fileList.WriteString("\n")
	}

	// Build preview pane
	var preview string
	if fv.showPreview {
		lines := strings.Split(fv.previewContent, "\n")
		previewStart := min(fv.previewOffset, len(lines)-1)
		previewEnd := min(previewStart+contentHeight-2, len(lines))

		var previewLines []string
		for i := previewStart; i < previewEnd; i++ {
			line := lines[i]
			if len(line) > previewWidth-4 {
				line = line[:previewWidth-4]
			}
			previewLines = append(previewLines, line)
		}

		preview = strings.Join(previewLines, "\n")
	}

	// Layout
	fileListBox := borderStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(fileList.String())

	previewBox := borderStyle.
		Width(previewWidth).
		Height(contentHeight).
		Render(preview)

	if fv.showPreview {
		return lipgloss.JoinHorizontal(lipgloss.Left, fileListBox, previewBox)
	}

	return fileListBox
}

func (fv *FileViewer) Focus() {
	fv.focused = true
}

func (fv *FileViewer) Blur() {
	fv.focused = false
}

func (fv *FileViewer) GetSelectedFile() string {
	if len(fv.filteredFiles) > 0 && fv.selectedIndex < len(fv.filteredFiles) {
		return fv.filteredFiles[fv.selectedIndex]
	}
	return ""
}

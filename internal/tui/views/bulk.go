package views

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	"github.com/repobird/repobird-cli/internal/cache"
)

// BulkMode represents the current mode of the bulk view
type BulkMode int

const (
	ModeFileSelect BulkMode = iota
	ModeRunList
	ModeRunEdit
	ModeProgress
	ModeResults
)

// BulkView represents the bulk runs TUI view
type BulkView struct {
	// API client
	client *api.Client

	// Configuration
	repository   string
	repoID       int
	sourceBranch string
	runType      string
	batchTitle   string
	force        bool

	// Runs collection
	runs        []BulkRunItem
	selectedRun int

	// UI state
	mode         BulkMode
	fileSelector *BulkFileSelector
	runEditor    *RunEditor
	progressView *BulkProgressView
	help         help.Model
	keys         bulkKeyMap
	width        int
	height       int

	// Submission state
	submitting bool
	batchID    string
	results    []BulkRunResult
	error      error

	// Components
	spinner spinner.Model
}

// BulkRunItem represents a single run in the bulk collection
type BulkRunItem struct {
	Prompt   string
	Title    string
	Target   string
	Context  string
	Selected bool
	Status   RunStatus
	Error    string
	FileHash string
}

// RunStatus represents the status of a run
type RunStatus int

const (
	StatusPending RunStatus = iota
	StatusQueued
	StatusProcessing
	StatusCompleted
	StatusFailed
	StatusCancelled
)

// BulkRunResult represents the result of a bulk run submission
type BulkRunResult struct {
	ID     int
	Title  string
	Status string
	Error  string
	URL    string
}

// bulkKeyMap defines key bindings for the bulk view
type bulkKeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Actions
	Select    key.Binding
	Submit    key.Binding
	Cancel    key.Binding
	Edit      key.Binding
	Add       key.Binding
	Delete    key.Binding
	Duplicate key.Binding
	ToggleAll key.Binding
	FZF       key.Binding

	// Mode switches
	FileMode key.Binding
	ListMode key.Binding

	// Global
	Help key.Binding
	Quit key.Binding
}

// NewBulkView creates a new bulk view
func NewBulkView(client *api.Client) *BulkView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BulkView{
		client:       client,
		mode:         ModeFileSelect,
		fileSelector: NewBulkFileSelector(),
		runEditor:    NewRunEditor(),
		progressView: NewBulkProgressView(),
		help:         help.New(),
		keys:         defaultBulkKeyMap(),
		spinner:      s,
		runType:      "run",
		runs:         []BulkRunItem{},
	}
}

func defaultBulkKeyMap() bulkKeyMap {
	return bulkKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup/ctrl+u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn/ctrl+d", "page down"),
		),
		Select: key.NewBinding(
			key.WithKeys(" ", "enter"),
			key.WithHelp("space/enter", "select"),
		),
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("ctrl+c", "esc"),
			key.WithHelp("esc", "cancel"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Duplicate: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "duplicate"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "toggle all"),
		),
		FZF: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fuzzy find"),
		),
		FileMode: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "file mode"),
		),
		ListMode: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "list mode"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+q"),
			key.WithHelp("q", "quit"),
		),
	}
}

// Init initializes the bulk view
func (v *BulkView) Init() tea.Cmd {
	return tea.Batch(
		v.spinner.Tick,
		v.fileSelector.Init(),
	)
}

// Update handles messages for the bulk view
func (v *BulkView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.help.Width = msg.Width

	case tea.KeyMsg:
		switch v.mode {
		case ModeFileSelect:
			return v.handleFileSelectKeys(msg)
		case ModeRunList:
			return v.handleRunListKeys(msg)
		case ModeRunEdit:
			return v.handleRunEditKeys(msg)
		case ModeProgress:
			return v.handleProgressKeys(msg)
		case ModeResults:
			return v.handleResultsKeys(msg)
		}

	case fileSelectedMsg:
		// File(s) selected, load configurations
		return v.loadFiles(msg.files)

	case bulkRunsLoadedMsg:
		// Runs loaded from files
		v.runs = msg.runs
		v.repository = msg.repository
		v.repoID = msg.repoID
		v.sourceBranch = msg.source
		v.runType = msg.runType
		v.batchTitle = msg.batchTitle
		v.mode = ModeRunList
		return v, nil

	case bulkSubmittedMsg:
		// Bulk submission completed
		v.submitting = false
		v.batchID = msg.batchID
		v.results = msg.results
		if msg.err != nil {
			v.error = msg.err
			v.mode = ModeResults
		} else {
			v.mode = ModeProgress
			// Start polling for progress
			return v, v.pollProgress()
		}

	case bulkProgressMsg:
		// Progress update received
		v.progressView.UpdateProgress(msg)
		if msg.completed {
			v.mode = ModeResults
		}
		return v, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update sub-components based on mode
	switch v.mode {
	case ModeFileSelect:
		newFileSelector, cmd := v.fileSelector.Update(msg)
		v.fileSelector = newFileSelector.(*BulkFileSelector)
		cmds = append(cmds, cmd)
	case ModeRunEdit:
		cmd := v.runEditor.UpdateRunEditor(msg)
		cmds = append(cmds, cmd)
	case ModeProgress:
		cmd := v.progressView.UpdateProgressView(msg)
		cmds = append(cmds, cmd)
	}

	return v, tea.Batch(cmds...)
}

// View renders the bulk view
func (v *BulkView) View() string {
	switch v.mode {
	case ModeFileSelect:
		return v.renderFileSelect()
	case ModeRunList:
		return v.renderRunList()
	case ModeRunEdit:
		return v.renderRunEdit()
	case ModeProgress:
		return v.renderProgress()
	case ModeResults:
		return v.renderResults()
	default:
		return "Unknown mode"
	}
}

func (v *BulkView) renderFileSelect() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Select Configuration Files")
	content := v.fileSelector.View()
	help := v.renderHelp()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		help,
	)
}

func (v *BulkView) renderRunList() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render(fmt.Sprintf("Bulk Runs (%d)", len(v.runs)))

	// Repository info
	repoInfo := fmt.Sprintf("Repository: %s | Source: %s | Type: %s",
		v.repository, v.sourceBranch, v.runType)
	repoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Runs list
	var runsList strings.Builder
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	for i, run := range v.runs {
		prefix := "  "
		if i == v.selectedRun {
			prefix = "> "
		}

		title := run.Title
		if title == "" {
			title = fmt.Sprintf("Run %d", i+1)
		}

		statusIcon := ""
		if run.Selected {
			statusIcon = "[✓] "
		} else {
			statusIcon = "[ ] "
		}

		line := fmt.Sprintf("%s%s%s", prefix, statusIcon, title)
		if i == v.selectedRun {
			line = selectedStyle.Render(line)
		}
		runsList.WriteString(line + "\n")
	}

	help := v.renderHelp()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		repoStyle.Render(repoInfo),
		"",
		runsList.String(),
		help,
	)
}

func (v *BulkView) renderRunEdit() string {
	return v.runEditor.View()
}

func (v *BulkView) renderProgress() string {
	if v.submitting {
		return fmt.Sprintf("%s Submitting bulk runs...", v.spinner.View())
	}
	return v.progressView.View()
}

func (v *BulkView) renderResults() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Bulk Run Results")

	var content strings.Builder

	if v.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", v.error)))
	}

	if len(v.results) > 0 {
		content.WriteString("\nCreated Runs:\n")
		for _, result := range v.results {
			statusIcon := "✓"
			statusColor := lipgloss.Color("10")

			if result.Status == "failed" {
				statusIcon = "✗"
				statusColor = lipgloss.Color("9")
			}

			style := lipgloss.NewStyle().Foreground(statusColor)
			content.WriteString(fmt.Sprintf("  %s %s (ID: %d)\n",
				style.Render(statusIcon),
				result.Title,
				result.ID,
			))

			if result.Error != "" {
				content.WriteString(fmt.Sprintf("    Error: %s\n", result.Error))
			}
			if result.URL != "" {
				content.WriteString(fmt.Sprintf("    URL: %s\n", result.URL))
			}
		}
	}

	if v.batchID != "" {
		content.WriteString(fmt.Sprintf("\nBatch ID: %s\n", v.batchID))
	}

	help := v.renderHelp()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content.String(),
		help,
	)
}

func (v *BulkView) renderHelp() string {
	// Return simple help text for now
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	return helpStyle.Render("? help | q quit | n new | e edit | d delete | space select | enter submit")
}

// Handle key events for different modes
func (v *BulkView) handleFileSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.ListMode):
		// Switch to list mode if we have runs
		if len(v.runs) > 0 {
			v.mode = ModeRunList
		}
		return v, nil
	default:
		// Pass to file selector
		newFileSelector, cmd := v.fileSelector.Update(msg)
		v.fileSelector = newFileSelector.(*BulkFileSelector)
		return v, cmd
	}
}

func (v *BulkView) handleRunListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.Up):
		if v.selectedRun > 0 {
			v.selectedRun--
		}
	case key.Matches(msg, v.keys.Down):
		if v.selectedRun < len(v.runs)-1 {
			v.selectedRun++
		}
	case key.Matches(msg, v.keys.Select):
		if v.selectedRun < len(v.runs) {
			v.runs[v.selectedRun].Selected = !v.runs[v.selectedRun].Selected
		}
	case key.Matches(msg, v.keys.ToggleAll):
		allSelected := true
		for _, run := range v.runs {
			if !run.Selected {
				allSelected = false
				break
			}
		}
		for i := range v.runs {
			v.runs[i].Selected = !allSelected
		}
	case key.Matches(msg, v.keys.Edit):
		if v.selectedRun < len(v.runs) {
			v.runEditor.SetRun(&v.runs[v.selectedRun])
			v.mode = ModeRunEdit
		}
	case key.Matches(msg, v.keys.Add):
		newRun := BulkRunItem{Selected: true}
		v.runs = append(v.runs, newRun)
		v.selectedRun = len(v.runs) - 1
		v.runEditor.SetRun(&v.runs[v.selectedRun])
		v.mode = ModeRunEdit
	case key.Matches(msg, v.keys.Delete):
		if v.selectedRun < len(v.runs) && len(v.runs) > 1 {
			v.runs = append(v.runs[:v.selectedRun], v.runs[v.selectedRun+1:]...)
			if v.selectedRun >= len(v.runs) {
				v.selectedRun = len(v.runs) - 1
			}
		}
	case key.Matches(msg, v.keys.Duplicate):
		if v.selectedRun < len(v.runs) {
			duplicate := v.runs[v.selectedRun]
			v.runs = append(v.runs, duplicate)
		}
	case key.Matches(msg, v.keys.Submit):
		return v, v.submitBulkRuns()
	case key.Matches(msg, v.keys.FileMode):
		v.mode = ModeFileSelect
	}
	return v, nil
}

func (v *BulkView) handleRunEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, v.keys.Cancel) {
		v.mode = ModeRunList
		return v, nil
	}
	// Pass to run editor
	cmd := v.runEditor.UpdateRunEditor(msg)
	return v, cmd
}

func (v *BulkView) handleProgressKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		// TODO: Confirm cancellation if in progress
		return v, tea.Quit
	case key.Matches(msg, v.keys.Cancel):
		// Cancel batch
		return v, v.cancelBatch()
	}
	return v, nil
}

func (v *BulkView) handleResultsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Quit):
		return v, tea.Quit
	case key.Matches(msg, v.keys.FileMode):
		// Start over with new files
		v.mode = ModeFileSelect
		v.runs = []BulkRunItem{}
		v.results = []BulkRunResult{}
		v.error = nil
		v.batchID = ""
	}
	return v, nil
}

// Command functions
func (v *BulkView) loadFiles(files []string) (tea.Model, tea.Cmd) {
	return v, func() tea.Msg {
		// Load bulk configuration from files
		bulkConfig, err := bulk.LoadBulkConfig(files)
		if err != nil {
			return errMsg{err}
		}

		// Convert to BulkRunItems
		var runs []BulkRunItem
		for _, run := range bulkConfig.Runs {
			runs = append(runs, BulkRunItem{
				Prompt:   run.Prompt,
				Title:    run.Title,
				Target:   run.Target,
				Context:  run.Context,
				Selected: true,
				Status:   StatusPending,
			})
		}

		return bulkRunsLoadedMsg{
			runs:       runs,
			repository: bulkConfig.Repository,
			repoID:     bulkConfig.RepoID,
			source:     bulkConfig.Source,
			runType:    bulkConfig.RunType,
			batchTitle: bulkConfig.BatchTitle,
		}
	}
}

func (v *BulkView) submitBulkRuns() tea.Cmd {
	v.submitting = true

	return func() tea.Msg {
		// Filter selected runs
		var selectedRuns []BulkRunItem
		for _, run := range v.runs {
			if run.Selected {
				selectedRuns = append(selectedRuns, run)
			}
		}

		if len(selectedRuns) == 0 {
			return errMsg{fmt.Errorf("no runs selected")}
		}

		// Generate file hashes
		var runItems []dto.RunItem

		for i, run := range selectedRuns {
			// Create hash from run content
			hashContent := fmt.Sprintf("%s-%s-%s-%s-%d",
				v.repository,
				run.Prompt,
				run.Target,
				run.Context,
				i,
			)
			hash := cache.CalculateStringHash(hashContent)
			run.FileHash = hash

			runItems = append(runItems, dto.RunItem{
				Prompt:   run.Prompt,
				Title:    run.Title,
				Target:   run.Target,
				Context:  run.Context,
				FileHash: hash,
			})
		}

		// Create bulk request
		req := &dto.BulkRunRequest{
			RepositoryName: v.repository,
			RepoID:         v.repoID,
			RunType:        v.runType,
			SourceBranch:   v.sourceBranch,
			BatchTitle:     v.batchTitle,
			Force:          v.force,
			Runs:           runItems,
			Options: dto.BulkOptions{
				Parallel: 5,
			},
		}

		// Submit to API
		ctx := context.Background()
		resp, err := v.client.CreateBulkRuns(ctx, req)
		if err != nil {
			return bulkSubmittedMsg{err: err}
		}

		// Convert response to results
		var results []BulkRunResult
		for _, run := range resp.Runs {
			results = append(results, BulkRunResult{
				ID:     run.ID,
				Title:  run.Title,
				Status: run.Status,
				URL:    run.RunURL,
			})
		}

		for _, runErr := range resp.Errors {
			results = append(results, BulkRunResult{
				Title:  runErr.Title,
				Status: "failed",
				Error:  runErr.Error,
			})
		}

		return bulkSubmittedMsg{
			batchID: resp.BatchID,
			results: results,
			err:     nil,
		}
	}
}

func (v *BulkView) pollProgress() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		statusChan, err := v.client.PollBulkStatus(ctx, v.batchID, 2*time.Second)
		if err != nil {
			return errMsg{err}
		}

		// Get first status update
		status := <-statusChan

		// Check if completed
		completed := status.Status == "completed" ||
			status.Status == "failed" ||
			status.Status == "cancelled"

		return bulkProgressMsg{
			batchID:    v.batchID,
			statistics: status.Statistics,
			runs:       status.Runs,
			completed:  completed,
		}
	}
}

func (v *BulkView) cancelBatch() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.CancelBulkRuns(ctx, v.batchID)
		if err != nil {
			return errMsg{err}
		}
		return bulkCancelledMsg{}
	}
}

// Message types
type fileSelectedMsg struct {
	files []string
}

type bulkRunsLoadedMsg struct {
	runs       []BulkRunItem
	repository string
	repoID     int
	source     string
	runType    string
	batchTitle string
}

type bulkSubmittedMsg struct {
	batchID string
	results []BulkRunResult
	err     error
}

type bulkProgressMsg struct {
	batchID    string
	statistics dto.BulkStatistics
	runs       []dto.RunStatusItem
	completed  bool
}

type bulkCancelledMsg struct{}

type errMsg struct {
	err error
}

// BulkFileSelector component
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

type filesLoadedMsg struct {
	files []FileItem
}

// RunEditor component
type RunEditor struct {
	run          *BulkRunItem
	promptInput  textinput.Model
	titleInput   textinput.Model
	targetInput  textinput.Model
	contextInput textinput.Model
	focusedField int
}

func NewRunEditor() *RunEditor {
	prompt := textinput.New()
	prompt.Placeholder = "Enter prompt (required)"
	prompt.CharLimit = 500
	prompt.Focus()

	title := textinput.New()
	title.Placeholder = "Enter title (optional)"
	title.CharLimit = 100

	target := textinput.New()
	target.Placeholder = "Enter target branch (optional)"
	target.CharLimit = 100

	context := textinput.New()
	context.Placeholder = "Enter context (optional)"
	context.CharLimit = 500

	return &RunEditor{
		promptInput:  prompt,
		titleInput:   title,
		targetInput:  target,
		contextInput: context,
		focusedField: 0,
	}
}

func (e *RunEditor) SetRun(run *BulkRunItem) {
	e.run = run
	e.promptInput.SetValue(run.Prompt)
	e.titleInput.SetValue(run.Title)
	e.targetInput.SetValue(run.Target)
	e.contextInput.SetValue(run.Context)
}

func (e *RunEditor) UpdateRunEditor(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			e.focusedField++
			if e.focusedField > 3 {
				e.focusedField = 0
			}
			e.updateFocus()
		case "shift+tab", "up":
			e.focusedField--
			if e.focusedField < 0 {
				e.focusedField = 3
			}
			e.updateFocus()
		case "enter":
			// Save changes
			if e.run != nil {
				e.run.Prompt = e.promptInput.Value()
				e.run.Title = e.titleInput.Value()
				e.run.Target = e.targetInput.Value()
				e.run.Context = e.contextInput.Value()
			}
			// Return to list mode (handled by parent)
		}
	}

	// Update inputs
	var cmd tea.Cmd
	e.promptInput, cmd = e.promptInput.Update(msg)
	cmds = append(cmds, cmd)

	e.titleInput, cmd = e.titleInput.Update(msg)
	cmds = append(cmds, cmd)

	e.targetInput, cmd = e.targetInput.Update(msg)
	cmds = append(cmds, cmd)

	e.contextInput, cmd = e.contextInput.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (e *RunEditor) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Edit Run")

	fields := []string{
		"Prompt (required):",
		e.promptInput.View(),
		"",
		"Title (optional):",
		e.titleInput.View(),
		"",
		"Target Branch (optional):",
		e.targetInput.View(),
		"",
		"Context (optional):",
		e.contextInput.View(),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, fields...)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"tab/↓: next field | shift+tab/↑: prev field | enter: save | esc: cancel",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		help,
	)
}

func (e *RunEditor) updateFocus() {
	e.promptInput.Blur()
	e.titleInput.Blur()
	e.targetInput.Blur()
	e.contextInput.Blur()

	switch e.focusedField {
	case 0:
		e.promptInput.Focus()
	case 1:
		e.titleInput.Focus()
	case 2:
		e.targetInput.Focus()
	case 3:
		e.contextInput.Focus()
	}
}

// BulkProgressView component
type BulkProgressView struct {
	batchID    string
	statistics dto.BulkStatistics
	runs       []dto.RunStatusItem
	spinner    spinner.Model
}

func NewBulkProgressView() *BulkProgressView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &BulkProgressView{
		spinner: s,
	}
}

func (v *BulkProgressView) UpdateProgressView(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		v.spinner, cmd = v.spinner.Update(msg)
		return cmd
	}
	return nil
}

func (v *BulkProgressView) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Bulk Run Progress")

	// Progress bar
	progressBar := v.makeProgressBar()

	// Statistics
	stats := fmt.Sprintf(
		"Total: %d | Queued: %d | Processing: %d | Completed: %d | Failed: %d",
		v.statistics.Total,
		v.statistics.Queued,
		v.statistics.Processing,
		v.statistics.Completed,
		v.statistics.Failed,
	)

	// Run details
	var runDetails strings.Builder
	for _, run := range v.runs {
		statusIcon := v.getStatusIcon(run.Status)
		runDetails.WriteString(fmt.Sprintf("  %s %s (ID: %d)\n",
			statusIcon, run.Title, run.ID))
		if run.Message != "" {
			runDetails.WriteString(fmt.Sprintf("    %s\n", run.Message))
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		v.spinner.View()+" "+progressBar,
		"",
		stats,
		"",
		runDetails.String(),
	)
}

func (v *BulkProgressView) UpdateProgress(msg bulkProgressMsg) {
	v.batchID = msg.batchID
	v.statistics = msg.statistics
	v.runs = msg.runs
}

func (v *BulkProgressView) makeProgressBar() string {
	width := 40
	completed := v.statistics.Completed + v.statistics.Failed + v.statistics.Cancelled
	total := v.statistics.Total

	if total == 0 {
		return strings.Repeat("░", width)
	}

	progress := int(float64(completed) / float64(total) * float64(width))
	return fmt.Sprintf("[%s%s] %d/%d",
		strings.Repeat("█", progress),
		strings.Repeat("░", width-progress),
		completed, total,
	)
}

func (v *BulkProgressView) getStatusIcon(status string) string {
	switch status {
	case "completed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗")
	case "processing":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("●")
	case "queued":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("○")
	default:
		return "?"
	}
}

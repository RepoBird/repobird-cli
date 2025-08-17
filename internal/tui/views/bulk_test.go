// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBulkAPIClient for testing bulk operations
type MockBulkAPIClient struct {
	mock.Mock
	*api.Client
}

func (m *MockBulkAPIClient) CreateBulkRuns(ctx context.Context, req *dto.BulkRunRequest) (*dto.BulkRunResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BulkRunResponse), args.Error(1)
}

func (m *MockBulkAPIClient) PollBulkStatus(ctx context.Context, batchID string, interval any) (<-chan *dto.BulkStatusResponse, error) {
	args := m.Called(ctx, batchID, interval)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *dto.BulkStatusResponse), args.Error(1)
}

func (m *MockBulkAPIClient) CancelBulkRuns(ctx context.Context, batchID string) error {
	args := m.Called(ctx, batchID)
	return args.Error(0)
}

func TestNewBulkView(t *testing.T) {
	// Create a real API client for now since BulkView requires *api.Client
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	assert.NotNil(t, view)
	assert.Equal(t, client, view.client)
	assert.Equal(t, ModeInstructions, view.mode) // Now starts with instructions
	assert.Nil(t, view.fileSelector)             // File selector not created until user presses 'f'
	assert.NotNil(t, view.help)
	assert.NotNil(t, view.keys)
	assert.NotNil(t, view.spinner)
	assert.NotNil(t, view.statusLine)
	assert.Equal(t, "run", view.runType)
	assert.Empty(t, view.runs)
	assert.Equal(t, 0, view.selectedRun)
	assert.False(t, view.submitting)
}

func TestBulkViewInit(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	cmd := view.Init()
	assert.NotNil(t, cmd)

	// Init should return a batch command
	// We can't easily test the contents without running it, but we can verify it's not nil
}

func TestBulkViewModeConstants(t *testing.T) {
	// Verify mode constants are correctly defined
	assert.Equal(t, BulkMode(0), ModeInstructions)
	assert.Equal(t, BulkMode(1), ModeFileBrowser)
	assert.Equal(t, BulkMode(2), ModeRunList)
	assert.Equal(t, BulkMode(3), ModeRunEdit)
	assert.Equal(t, BulkMode(4), ModeProgress)
	assert.Equal(t, BulkMode(5), ModeResults)
}

func TestBulkViewStatusConstants(t *testing.T) {
	// Verify status constants are correctly defined
	assert.Equal(t, RunStatus(0), StatusPending)
	assert.Equal(t, RunStatus(1), StatusQueued)
	assert.Equal(t, RunStatus(2), StatusProcessing)
	assert.Equal(t, RunStatus(3), StatusCompleted)
	assert.Equal(t, RunStatus(4), StatusFailed)
	assert.Equal(t, RunStatus(5), StatusCancelled)
}

func TestBulkViewWindowSizeMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	msg := tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	}

	model, cmd := view.Update(msg)
	updatedView := model.(*BulkView)

	assert.Equal(t, 120, updatedView.width)
	assert.Equal(t, 40, updatedView.height)
	assert.NotNil(t, updatedView.layout)
	assert.Nil(t, cmd)

	// File selector should not be created yet (only created when pressing 'f')
	assert.Nil(t, updatedView.fileSelector)
}

func TestBulkViewGlobalQuitKeys(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)
	view.width = 80
	view.height = 24

	tests := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'Q'}},
		{Type: tea.KeyCtrlC},
	}

	for _, keyMsg := range tests {
		model, cmd := view.Update(keyMsg)
		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)

		// Execute the command to check if it returns quit message
		msg := cmd()
		_, isQuitMsg := msg.(tea.QuitMsg)
		assert.True(t, isQuitMsg, "Expected quit message")
	}
}

func TestBulkViewFileSelectKeys(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)
	view.width = 80
	view.height = 24
	view.mode = ModeFileBrowser
	view.fileSelector = components.NewBulkFileSelector(80, 24)

	t.Run("Quit key returns to instructions mode", func(t *testing.T) {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		model, cmd := view.Update(keyMsg)

		assert.Equal(t, view, model)
		assert.Nil(t, cmd) // No command returned, just mode change

		// Should change mode back to instructions and clear file selector
		updatedView := model.(*BulkView)
		assert.Equal(t, ModeInstructions, updatedView.mode)
		assert.Nil(t, updatedView.fileSelector)
	})

	t.Run("ListMode key switches to run list when runs exist", func(t *testing.T) {
		// Reset view state for this test
		view.mode = ModeFileBrowser
		view.fileSelector = components.NewBulkFileSelector(80, 24)
		view.runs = []BulkRunItem{
			{Title: "Test Run", Selected: true},
		}

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, ModeRunList, updatedView.mode)
		assert.Nil(t, cmd)
	})

	t.Run("ListMode key does nothing when no runs", func(t *testing.T) {
		// Reset view state for this test
		view.mode = ModeFileBrowser
		view.fileSelector = components.NewBulkFileSelector(80, 24)
		view.runs = []BulkRunItem{}

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, ModeFileBrowser, updatedView.mode)
		assert.Nil(t, cmd)
	})
}

func TestBulkViewRunListKeys(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)
	view.width = 80
	view.height = 24
	view.mode = ModeRunList
	view.runs = []BulkRunItem{
		{Title: "Run 1", Selected: true},
		{Title: "Run 2", Selected: false},
		{Title: "Run 3", Selected: true},
	}
	view.selectedRun = 1
	view.focusMode = "runs" // Need to set focus mode for navigation to work

	t.Run("Quit key returns NavigateBackMsg", func(t *testing.T) {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		model, cmd := view.Update(keyMsg)

		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)

		navMsg := cmd()
		_, ok := navMsg.(messages.NavigateBackMsg)
		assert.True(t, ok, "Should return NavigateBackMsg")
	})

	t.Run("Up key moves selection up", func(t *testing.T) {
		keyMsg := tea.KeyMsg{Type: tea.KeyUp}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, 0, updatedView.selectedRun)
		assert.Nil(t, cmd)
	})

	t.Run("Up key at top does nothing", func(t *testing.T) {
		view.selectedRun = 0
		keyMsg := tea.KeyMsg{Type: tea.KeyUp}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, 0, updatedView.selectedRun)
		assert.Nil(t, cmd)
	})

	t.Run("Down key moves selection down", func(t *testing.T) {
		// Reset view state for this test
		view.selectedRun = 0
		view.focusMode = "runs"
		keyMsg := tea.KeyMsg{Type: tea.KeyDown}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, 1, updatedView.selectedRun)
		assert.Nil(t, cmd)
	})

	t.Run("Down key at bottom does nothing", func(t *testing.T) {
		view.selectedRun = 2
		keyMsg := tea.KeyMsg{Type: tea.KeyDown}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, 2, updatedView.selectedRun)
		assert.Nil(t, cmd)
	})

	t.Run("Space key toggles selection", func(t *testing.T) {
		// Reset view state for this test
		view.selectedRun = 1
		view.focusMode = "runs"
		originalSelection := view.runs[1].Selected

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, !originalSelection, updatedView.runs[1].Selected)
		assert.Nil(t, cmd)
	})

	t.Run("FileMode key switches to file select", func(t *testing.T) {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, ModeFileBrowser, updatedView.mode)
		assert.NotNil(t, cmd) // Should return fileSelector.Activate() command
	})

	t.Run("Submit key calls submitBulkRuns", func(t *testing.T) {
		// Reset state for this test
		view.mode = ModeRunList
		view.runs = []BulkRunItem{
			{Title: "Run 1", Selected: true},
			{Title: "Run 2", Selected: false},
			{Title: "Run 3", Selected: true},
		}
		view.selectedRun = 1

		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		model, cmd := view.Update(keyMsg)

		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)
		// cmd should be the result of submitBulkRuns() but we can't easily test it
	})
}

func TestBulkViewBulkFileSelectedMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	t.Run("Canceled selection does nothing", func(t *testing.T) {
		msg := components.BulkFileSelectedMsg{
			Files:    []string{"file1.json"},
			Canceled: true,
		}

		model, cmd := view.Update(msg)
		assert.Equal(t, view, model)
		assert.Nil(t, cmd)
	})

	t.Run("Empty files does nothing", func(t *testing.T) {
		msg := components.BulkFileSelectedMsg{
			Files:    []string{},
			Canceled: false,
		}

		model, cmd := view.Update(msg)
		assert.Equal(t, view, model)
		assert.Nil(t, cmd)
	})

	t.Run("Valid files triggers loadFiles", func(t *testing.T) {
		msg := components.BulkFileSelectedMsg{
			Files:    []string{"file1.json", "file2.yaml"},
			Canceled: false,
		}

		model, cmd := view.Update(msg)
		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)
		// cmd should be the result of loadFiles() but we can't easily test the contents
	})
}

func TestBulkViewBulkRunsLoadedMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	runs := []BulkRunItem{
		{Title: "Run 1", Prompt: "Test prompt 1"},
		{Title: "Run 2", Prompt: "Test prompt 2"},
	}

	msg := bulkRunsLoadedMsg{
		runs:       runs,
		repository: "test/repo",
		repoID:     123,
		source:     "main",
		runType:    "run",
		batchTitle: "Test Batch",
	}

	model, cmd := view.Update(msg)
	updatedView := model.(*BulkView)

	assert.Equal(t, runs, updatedView.runs)
	assert.Equal(t, "test/repo", updatedView.repository)
	assert.Equal(t, 123, updatedView.repoID)
	assert.Equal(t, "main", updatedView.sourceBranch)
	assert.Equal(t, "run", updatedView.runType)
	assert.Equal(t, "Test Batch", updatedView.batchTitle)
	assert.Equal(t, ModeRunList, updatedView.mode)
	assert.Equal(t, 0, updatedView.selectedRun)
	assert.Nil(t, cmd)
}

func TestBulkViewBulkSubmittedMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)
	view.submitting = true

	t.Run("Successful submission", func(t *testing.T) {
		results := []BulkRunResult{
			{ID: 1, Title: "Run 1", Status: "success"},
			{ID: 2, Title: "Run 2", Status: "success"},
		}

		msg := bulkSubmittedMsg{
			batchID: "batch-123",
			results: results,
			err:     nil,
		}

		model, cmd := view.Update(msg)
		updatedView := model.(*BulkView)

		assert.False(t, updatedView.submitting)
		assert.Equal(t, "batch-123", updatedView.batchID)
		assert.Equal(t, results, updatedView.results)
		assert.Nil(t, updatedView.error)
		// Mode doesn't change - navigation happens via command
		assert.NotNil(t, cmd) // Should be navigation command
	})

	t.Run("Submission with error", func(t *testing.T) {
		testError := assert.AnError

		msg := bulkSubmittedMsg{
			batchID: "batch-123",
			results: nil,
			err:     testError,
		}

		model, cmd := view.Update(msg)
		updatedView := model.(*BulkView)

		assert.False(t, updatedView.submitting)
		assert.Equal(t, "batch-123", updatedView.batchID)
		assert.Equal(t, testError, updatedView.error)
		// Mode doesn't change - navigation happens via command
		assert.NotNil(t, cmd) // Should be navigation command
	})
}

func TestBulkViewBulkProgressMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	t.Run("Progress update incomplete", func(t *testing.T) {
		msg := bulkProgressMsg{
			completed: false,
		}

		model, cmd := view.Update(msg)
		updatedView := model.(*BulkView)

		assert.NotEqual(t, ModeResults, updatedView.mode)
		assert.Nil(t, cmd)
	})

	t.Run("Progress update completed", func(t *testing.T) {
		msg := bulkProgressMsg{
			completed: true,
		}

		model, cmd := view.Update(msg)
		updatedView := model.(*BulkView)

		// Progress messages don't change mode anymore (navigation happens immediately on submission)
		assert.NotEqual(t, ModeResults, updatedView.mode)
		assert.Nil(t, cmd)
	})
}

func TestBulkViewBulkCancelledMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	msg := bulkCancelledMsg{}
	model, cmd := view.Update(msg)
	updatedView := model.(*BulkView)

	// Cancelled messages don't change mode anymore (navigation happens immediately on submission)
	assert.NotEqual(t, ModeResults, updatedView.mode)
	assert.Nil(t, cmd)
}

func TestBulkViewErrMsg(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	testError := assert.AnError
	msg := errMsg{err: testError}

	model, cmd := view.Update(msg)
	updatedView := model.(*BulkView)

	assert.Equal(t, testError, updatedView.error)
	assert.Nil(t, cmd)
}

func TestBulkViewRendering(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	t.Run("Before window size is set", func(t *testing.T) {
		output := view.View()
		assert.Equal(t, "⟳ Initializing Bulk View...", output)
	})

	t.Run("File select mode rendering", func(t *testing.T) {
		view.width = 80
		view.height = 24
		view.mode = ModeFileBrowser

		output := view.View()

		assert.Contains(t, output, "Select Configuration Files")
		assert.Contains(t, output, "[FZF-BULK]")
	})

	t.Run("Run list mode rendering", func(t *testing.T) {
		view.width = 80
		view.height = 24
		view.mode = ModeRunList
		view.runs = []BulkRunItem{
			{Title: "Test Run 1", Selected: true},
			{Title: "Test Run 2", Selected: false},
		}
		view.repository = "test/repo"
		view.sourceBranch = "main"
		view.runType = "run"

		output := view.View()

		assert.Contains(t, output, "Bulk Runs (2 total")
		assert.Contains(t, output, "[✓] Test Run 1")
		assert.Contains(t, output, "[ ] Test Run 2")
		assert.Contains(t, output, "[BULK]")
	})

	t.Run("Run edit mode rendering", func(t *testing.T) {
		view.width = 80
		view.height = 24
		view.mode = ModeRunEdit

		output := view.View()

		assert.Contains(t, output, "Edit Run")
		assert.Contains(t, output, "[BULK]")
	})

	t.Run("Progress mode rendering", func(t *testing.T) {
		view.width = 80
		view.height = 24
		view.mode = ModeProgress
		view.submitting = true

		output := view.View()

		assert.Contains(t, output, "Submitting bulk runs...")
		assert.Contains(t, output, "[BULK]")
	})

	t.Run("Results mode rendering", func(t *testing.T) {
		view.width = 80
		view.height = 24
		view.mode = ModeResults
		view.results = []BulkRunResult{
			{ID: 1, Title: "Success Run", Status: "success"},
			{ID: 2, Title: "Failed Run", Status: "failed", Error: "Test error"},
		}
		view.batchID = "batch-123"

		output := view.View()

		assert.Contains(t, output, "Bulk Run Results")
		assert.Contains(t, output, "Success Run")
		assert.Contains(t, output, "Failed Run")
		assert.Contains(t, output, "Batch ID: batch-123")
		assert.Contains(t, output, "[BULK]")
	})

	t.Run("Unknown mode rendering", func(t *testing.T) {
		view.width = 80
		view.height = 24
		view.mode = BulkMode(999) // Invalid mode

		output := view.View()

		assert.Equal(t, "Unknown mode", output)
	})
}

func TestBulkViewStatusLineHelp(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)
	view.width = 80
	view.height = 24

	tests := []struct {
		mode     BulkMode
		expected string
	}{
		{ModeFileBrowser, "↑↓/j/k:nav space:select i:input mode esc:back enter:confirm"},
		{ModeRunList, "↑↓/j/k:nav space:toggle enter:submit tab:buttons f:files"},
		{ModeRunEdit, "[h]back [q]dashboard ?:help"},
		{ModeProgress, "[h]back [q]dashboard ?:help"},
		{ModeResults, "[h]back [q]dashboard ?:help"},
	}

	for _, tt := range tests {
		view.mode = tt.mode
		statusLine := view.renderStatusLine("BULK")
		assert.Contains(t, statusLine, tt.expected)
	}
}

func TestBulkViewLayoutSystem(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	// Simulate window size update
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	view.Update(msg)

	// Layout should be created
	assert.NotNil(t, view.layout)

	// Test that rendering methods use the layout
	view.mode = ModeFileBrowser
	output := view.renderFileBrowser()
	assert.NotEmpty(t, output)

	view.mode = ModeRunList
	output = view.renderRunList()
	assert.NotEmpty(t, output)

	view.mode = ModeRunEdit
	output = view.renderRunEdit()
	assert.NotEmpty(t, output)

	view.mode = ModeProgress
	output = view.renderProgress()
	assert.NotEmpty(t, output)

	view.mode = ModeResults
	output = view.renderResults()
	assert.NotEmpty(t, output)
}

func TestBulkViewModelInterface(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	// Verify it implements tea.Model interface
	var _ tea.Model = view

	// Test all required methods
	assert.NotPanics(t, func() {
		_ = view.Init()
		_, _ = view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		_ = view.View()
	})
}

func TestBulkViewNavigationMessages(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)
	view.width = 80
	view.height = 24

	t.Run("File select quit returns to instructions mode", func(t *testing.T) {
		view.mode = ModeFileBrowser
		view.fileSelector = components.NewBulkFileSelector(80, 24)
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

		model, cmd := view.Update(keyMsg)
		updatedView := model.(*BulkView)

		assert.Equal(t, ModeInstructions, updatedView.mode)
		assert.Nil(t, updatedView.fileSelector)
		assert.Nil(t, cmd)
	})

	t.Run("Run list quit returns NavigateBackMsg", func(t *testing.T) {
		view.mode = ModeRunList
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

		_, cmd := view.Update(keyMsg)
		assert.NotNil(t, cmd)

		navMsg := cmd()
		_, ok := navMsg.(messages.NavigateBackMsg)
		assert.True(t, ok)
	})
}

func TestBulkViewDimensions(t *testing.T) {
	client := &api.Client{}
	testCache := cache.NewSimpleCache()
	view := NewBulkView(client, testCache)

	tests := []struct {
		width  int
		height int
	}{
		{80, 24},
		{120, 40},
		{60, 20},
		{200, 50},
	}

	for _, tt := range tests {
		msg := tea.WindowSizeMsg{
			Width:  tt.width,
			Height: tt.height,
		}

		model, _ := view.Update(msg)
		updatedView := model.(*BulkView)

		assert.Equal(t, tt.width, updatedView.width)
		assert.Equal(t, tt.height, updatedView.height)

		// View should render with these dimensions
		output := updatedView.View()
		assert.NotEmpty(t, output)
	}
}

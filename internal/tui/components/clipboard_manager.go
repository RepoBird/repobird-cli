package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/utils"
)

// ClipboardManager handles clipboard operations with consistent visual feedback
type ClipboardManager struct {
	isBlinking     bool
	blinkStartTime time.Time
}

// ClipboardBlinkMsg is sent to trigger blink animation updates
type ClipboardBlinkMsg struct{}

// NewClipboardManager creates a new clipboard manager
func NewClipboardManager() ClipboardManager {
	return ClipboardManager{}
}

// CopyWithBlink copies text to clipboard and starts visual feedback animation
func (c *ClipboardManager) CopyWithBlink(text, description string) (tea.Cmd, error) {
	if text == "" {
		return nil, utils.NewClipboardError("no content to copy")
	}

	// Attempt to copy to clipboard
	if err := utils.WriteToClipboard(text); err != nil {
		return nil, err
	}

	// Start blink animation
	c.isBlinking = true
	c.blinkStartTime = time.Now()

	// Return command to trigger blink message after 200ms
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return ClipboardBlinkMsg{}
	}), nil
}

// Update handles clipboard-related messages
func (c ClipboardManager) Update(msg tea.Msg) (ClipboardManager, tea.Cmd) {
	switch msg.(type) {
	case ClipboardBlinkMsg:
		// End blink animation after 200ms
		c.isBlinking = false
		return c, nil
	}
	return c, nil
}

// IsBlinking returns true if currently in blink animation
func (c *ClipboardManager) IsBlinking() bool {
	return c.isBlinking
}

// ShouldHighlight returns true if visual highlight should be applied
// This provides the 200ms highlight window for visual feedback
func (c *ClipboardManager) ShouldHighlight() bool {
	if !c.isBlinking || c.blinkStartTime.IsZero() {
		return false
	}
	
	// Highlight for exactly 200ms from start time
	return time.Since(c.blinkStartTime) < 200*time.Millisecond
}

// Reset clears any active blink state
func (c *ClipboardManager) Reset() {
	c.isBlinking = false
	c.blinkStartTime = time.Time{}
}
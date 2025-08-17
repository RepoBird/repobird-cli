// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewClipboardManager(t *testing.T) {
	cm := NewClipboardManager()

	assert.False(t, cm.IsBlinking())
	assert.False(t, cm.ShouldHighlight())
	assert.True(t, cm.blinkStartTime.IsZero())
}

func TestCopyWithBlink_EmptyText(t *testing.T) {
	cm := NewClipboardManager()

	cmd, err := cm.CopyWithBlink("", "test")

	assert.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "no content to copy")
	assert.False(t, cm.IsBlinking())
}

func TestCopyWithBlink_Success(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// Test with valid text
	cmd, err := cm.CopyWithBlink("test content", "description")

	// Should succeed and start blink animation
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.True(t, cm.IsBlinking())
	assert.True(t, cm.ShouldHighlight())
	assert.False(t, cm.blinkStartTime.IsZero())
}

func TestUpdate_ClipboardBlinkMsg(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// Start blink animation
	_, _ = cm.CopyWithBlink("test", "")
	assert.True(t, cm.IsBlinking())

	// Process blink message to end animation
	updatedCm, cmd := cm.Update(ClipboardBlinkMsg{})

	assert.False(t, updatedCm.IsBlinking())
	assert.Nil(t, cmd)
}

func TestUpdate_OtherMessage(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// Start blink animation
	_, _ = cm.CopyWithBlink("test", "")
	initialState := cm.IsBlinking()

	// Process unrelated message
	updatedCm, cmd := cm.Update(tea.KeyMsg{})

	// State should remain unchanged
	assert.Equal(t, initialState, updatedCm.IsBlinking())
	assert.Nil(t, cmd)
}

func TestShouldHighlight_Timing(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// Before blink starts
	assert.False(t, cm.ShouldHighlight())

	// Start blink
	_, _ = cm.CopyWithBlink("test", "")

	// Should highlight immediately after starting
	assert.True(t, cm.ShouldHighlight())

	// Simulate time passage beyond 200ms
	cm.blinkStartTime = time.Now().Add(-250 * time.Millisecond)

	// Should no longer highlight after 200ms
	assert.False(t, cm.ShouldHighlight())
}

func TestShouldHighlight_NoBlinkStarted(t *testing.T) {
	cm := NewClipboardManager()

	// Manually set blinking without proper start time
	cm.isBlinking = true

	// Should not highlight without valid start time
	assert.False(t, cm.ShouldHighlight())
}

func TestReset(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// Start blink animation
	_, _ = cm.CopyWithBlink("test", "")
	assert.True(t, cm.IsBlinking())
	assert.False(t, cm.blinkStartTime.IsZero())

	// Reset state
	cm.Reset()

	assert.False(t, cm.IsBlinking())
	assert.False(t, cm.ShouldHighlight())
	assert.True(t, cm.blinkStartTime.IsZero())
}

func TestBlinkAnimation_FullCycle(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// Start blink
	cmd, err := cm.CopyWithBlink("test content", "test description")
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.True(t, cm.IsBlinking())
	assert.True(t, cm.ShouldHighlight())

	// Execute the command to get the blink message
	msg := cmd()
	assert.IsType(t, ClipboardBlinkMsg{}, msg)

	// Process the blink message to end animation
	updatedCm, endCmd := cm.Update(msg)
	assert.False(t, updatedCm.IsBlinking())
	assert.Nil(t, endCmd)
}

func TestMultipleCopyOperations(t *testing.T) {
	// Skip clipboard tests in CI/headless environments where clipboard may not be available
	if err := utils.WriteToClipboard("test"); err != nil {
		t.Skipf("Skipping clipboard test: %v", err)
	}

	cm := NewClipboardManager()

	// First copy
	cmd1, err1 := cm.CopyWithBlink("first", "")
	assert.NoError(t, err1)
	assert.NotNil(t, cmd1)
	firstStartTime := cm.blinkStartTime

	// Second copy before first completes (should restart)
	cmd2, err2 := cm.CopyWithBlink("second", "")
	assert.NoError(t, err2)
	assert.NotNil(t, cmd2)
	assert.True(t, cm.blinkStartTime.After(firstStartTime))
	assert.True(t, cm.IsBlinking())
	assert.True(t, cm.ShouldHighlight())
}

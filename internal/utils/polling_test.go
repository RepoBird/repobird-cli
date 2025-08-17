// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPollConfig(t *testing.T) {
	config := DefaultPollConfig()

	assert.NotNil(t, config)
	assert.Equal(t, DefaultPollInterval, config.Interval)
	assert.Equal(t, 45*time.Minute, config.MaxDuration)
	assert.True(t, config.ShowProgress)
	assert.False(t, config.Debug)
}

func TestNewPoller(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		config := &PollConfig{
			Interval:     10 * time.Second,
			MaxDuration:  1 * time.Hour,
			ShowProgress: false,
			Debug:        true,
		}

		poller := NewPoller(config)

		assert.NotNil(t, poller)
		assert.Equal(t, config, poller.config)
		assert.WithinDuration(t, time.Now(), poller.startTime, 1*time.Second)
	})

	t.Run("with nil config uses default", func(t *testing.T) {
		poller := NewPoller(nil)

		assert.NotNil(t, poller)
		assert.NotNil(t, poller.config)
		assert.Equal(t, DefaultPollInterval, poller.config.Interval)
	})
}

func TestPoller_Poll(t *testing.T) {
	t.Run("returns immediately on terminal status", func(t *testing.T) {
		config := &PollConfig{
			Interval:     100 * time.Millisecond,
			MaxDuration:  1 * time.Second,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			return &models.RunResponse{
				ID:     "run-1",
				Status: models.StatusDone,
			}, nil
		}

		updateCalled := false
		onUpdate := func(run *models.RunResponse) {
			updateCalled = true
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, onUpdate)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.StatusDone, result.Status)
		assert.True(t, updateCalled)
	})

	t.Run("polls until terminal status", func(t *testing.T) {
		config := &PollConfig{
			Interval:     50 * time.Millisecond,
			MaxDuration:  1 * time.Second,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		callCount := 0
		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			callCount++
			status := models.StatusProcessing
			if callCount >= 3 {
				status = models.StatusDone
			}
			return &models.RunResponse{
				ID:     "run-1",
				Status: status,
			}, nil
		}

		updateCount := 0
		onUpdate := func(run *models.RunResponse) {
			updateCount++
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, onUpdate)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.StatusDone, result.Status)
		assert.Equal(t, 3, callCount)
		assert.Equal(t, 3, updateCount)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		config := &PollConfig{
			Interval:     100 * time.Millisecond,
			MaxDuration:  10 * time.Second,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			return &models.RunResponse{
				ID:     "run-1",
				Status: models.StatusProcessing,
			}, nil
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after a short delay
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()

		result, err := poller.Poll(ctx, pollFunc, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "polling timeout")
		assert.Nil(t, result)
	})

	t.Run("handles timeout", func(t *testing.T) {
		config := &PollConfig{
			Interval:     50 * time.Millisecond,
			MaxDuration:  100 * time.Millisecond,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			return &models.RunResponse{
				ID:     "run-1",
				Status: models.StatusProcessing,
			}, nil
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "polling timeout")
		assert.Nil(t, result)
	})

	t.Run("handles poll function errors", func(t *testing.T) {
		config := &PollConfig{
			Interval:     50 * time.Millisecond,
			MaxDuration:  1 * time.Second,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		expectedErr := errors.New("network error")
		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			return nil, expectedErr
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, nil)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("nil onUpdate callback is safe", func(t *testing.T) {
		config := &PollConfig{
			Interval:     50 * time.Millisecond,
			MaxDuration:  1 * time.Second,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			return &models.RunResponse{
				ID:     "run-1",
				Status: models.StatusDone,
			}, nil
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     models.RunStatus
		isTerminal bool
	}{
		{"Done status", models.StatusDone, true},
		{"Failed status", models.StatusFailed, true},
		{"Queued status", models.StatusQueued, false},
		{"Initializing status", models.StatusInitializing, false},
		{"Processing status", models.StatusProcessing, false},
		{"PostProcess status", models.StatusPostProcess, false},
		{"Unknown status", "UNKNOWN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTerminalStatus(tt.status)
			assert.Equal(t, tt.isTerminal, result)
		})
	}
}

func TestShowPollingProgress(t *testing.T) {
	// This function writes to stdout, so we'll just test it doesn't panic
	t.Run("handles normal input", func(t *testing.T) {
		startTime := time.Now().Add(-5 * time.Second)

		// Should not panic
		ShowPollingProgress(startTime, "PROCESSING", "Working on task...")
	})

	t.Run("handles empty message", func(t *testing.T) {
		startTime := time.Now()

		// Should not panic
		ShowPollingProgress(startTime, "QUEUED", "")
	})

	t.Run("handles future start time", func(t *testing.T) {
		startTime := time.Now().Add(1 * time.Hour)

		// Should not panic
		ShowPollingProgress(startTime, "DONE", "Complete")
	})
}

func TestClearLine(t *testing.T) {
	// This function writes to stdout, so we'll just test it doesn't panic
	t.Run("executes without error", func(t *testing.T) {
		// Should not panic
		ClearLine()
	})
}

func TestPollerIntegration(t *testing.T) {
	t.Run("complete polling lifecycle", func(t *testing.T) {
		config := &PollConfig{
			Interval:     25 * time.Millisecond,
			MaxDuration:  500 * time.Millisecond,
			ShowProgress: false,
			Debug:        true,
		}
		poller := NewPoller(config)

		// Simulate a run that goes through multiple states
		states := []models.RunStatus{
			models.StatusQueued,
			models.StatusInitializing,
			models.StatusProcessing,
			models.StatusPostProcess,
			models.StatusDone,
		}

		stateIndex := 0
		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			if stateIndex >= len(states) {
				stateIndex = len(states) - 1
			}
			status := states[stateIndex]
			stateIndex++

			return &models.RunResponse{
				ID:     "run-integration",
				Status: status,
			}, nil
		}

		var updates []*models.RunResponse
		onUpdate := func(run *models.RunResponse) {
			updates = append(updates, run)
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, onUpdate)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.StatusDone, result.Status)
		assert.Equal(t, 5, len(updates))

		// Verify we received updates for each state
		for i, update := range updates {
			assert.Equal(t, states[i], update.Status)
		}
	})

	t.Run("rapid status changes", func(t *testing.T) {
		config := &PollConfig{
			Interval:     10 * time.Millisecond,
			MaxDuration:  200 * time.Millisecond,
			ShowProgress: false,
		}
		poller := NewPoller(config)

		// Simulate very rapid state changes
		callCount := 0
		pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
			callCount++
			// Transition to done after 5 calls
			status := models.StatusProcessing
			if callCount >= 5 {
				status = models.StatusDone
			}

			return &models.RunResponse{
				ID:     "run-rapid",
				Status: status,
			}, nil
		}

		ctx := context.Background()
		result, err := poller.Poll(ctx, pollFunc, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.StatusDone, result.Status)
		assert.GreaterOrEqual(t, callCount, 5)
	})
}

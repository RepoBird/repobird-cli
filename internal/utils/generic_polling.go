package utils

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GenericPollConfig holds configuration for polling operations
type GenericPollConfig struct {
	Interval      time.Duration
	MaxInterval   time.Duration
	BackoffFactor float64
	Timeout       time.Duration
	ShowProgress  bool
	Debug         bool
}

// PollFunc is a generic function that fetches the current state
type GenericPollFunc[T any] func(ctx context.Context) (T, error)

// CompletionFunc checks if the result is in a terminal state
type CompletionFunc[T any] func(T) bool

// UpdateCallback is called with each poll result
type UpdateCallback[T any] func(T)

// GenericPoller handles polling operations with exponential backoff
type GenericPoller[T any] struct {
	config    *GenericPollConfig
	startTime time.Time
}

// NewGenericPoller creates a new generic poller
func NewGenericPoller[T any](config *GenericPollConfig) *GenericPoller[T] {
	if config == nil {
		config = &GenericPollConfig{
			Interval:      5 * time.Second,
			MaxInterval:   30 * time.Second,
			BackoffFactor: 1.5,
			Timeout:       45 * time.Minute,
			ShowProgress:  true,
			Debug:         false,
		}
	}
	return &GenericPoller[T]{
		config:    config,
		startTime: time.Now(),
	}
}

// PollUntilComplete polls until the completion function returns true
func (p *GenericPoller[T]) PollUntilComplete(
	ctx context.Context,
	pollFunc GenericPollFunc[T],
	isComplete CompletionFunc[T],
	onUpdate UpdateCallback[T],
) (T, error) {
	var zero T

	// Set up signal handling for graceful interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create context with timeout
	pollCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	currentInterval := p.config.Interval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	// Initial check
	result, err := pollFunc(pollCtx)
	if err != nil {
		return zero, err
	}

	if onUpdate != nil {
		onUpdate(result)
	}

	// Check if already complete
	if isComplete(result) {
		return result, nil
	}

	// Polling loop
	for {
		select {
		case <-pollCtx.Done():
			return zero, fmt.Errorf("polling timeout after %v", FormatDuration(time.Since(p.startTime)))

		case <-sigChan:
			if p.config.Debug {
				fmt.Println("\nPolling interrupted by user")
			}
			return result, fmt.Errorf("polling interrupted")

		case <-ticker.C:
			result, err = pollFunc(pollCtx)
			if err != nil {
				// Log error but continue polling for transient errors
				if p.config.Debug {
					fmt.Printf("Poll error (will retry): %v\n", err)
				}

				// Implement exponential backoff on errors
				if p.config.BackoffFactor > 1 {
					currentInterval = time.Duration(float64(currentInterval) * p.config.BackoffFactor)
					if currentInterval > p.config.MaxInterval {
						currentInterval = p.config.MaxInterval
					}
					ticker.Reset(currentInterval)
				}
				continue
			}

			// Reset interval on success
			if currentInterval != p.config.Interval {
				currentInterval = p.config.Interval
				ticker.Reset(currentInterval)
			}

			if onUpdate != nil {
				onUpdate(result)
			}

			if isComplete(result) {
				return result, nil
			}

			// Show progress if enabled
			if p.config.ShowProgress {
				elapsed := time.Since(p.startTime)
				fmt.Printf("\rPolling... [%s elapsed]", FormatDuration(elapsed))
			}
		}
	}
}

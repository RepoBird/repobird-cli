// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

const (
	DefaultPollInterval = 5 * time.Second
	MinPollInterval     = 1 * time.Second
	MaxPollInterval     = 30 * time.Second
)

type PollConfig struct {
	Interval     time.Duration
	MaxDuration  time.Duration
	ShowProgress bool
	Debug        bool
}

func DefaultPollConfig() *PollConfig {
	return &PollConfig{
		Interval:     DefaultPollInterval,
		MaxDuration:  45 * time.Minute,
		ShowProgress: true,
		Debug:        false,
	}
}

type PollFunc func(ctx context.Context) (*models.RunResponse, error)

type Poller struct {
	config    *PollConfig
	startTime time.Time
}

func NewPoller(config *PollConfig) *Poller {
	if config == nil {
		config = DefaultPollConfig()
	}
	return &Poller{
		config:    config,
		startTime: time.Now(),
	}
}

func (p *Poller) Poll(ctx context.Context, fn PollFunc, onUpdate func(*models.RunResponse)) (*models.RunResponse, error) {
	// Set up signal handling for graceful interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create context with timeout
	pollCtx, cancel := context.WithTimeout(ctx, p.config.MaxDuration)
	defer cancel()

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	// Initial check
	run, err := fn(pollCtx)
	if err != nil {
		return nil, err
	}

	if onUpdate != nil {
		onUpdate(run)
	}

	// Check if already in terminal state
	if IsTerminalStatus(run.Status) {
		return run, nil
	}

	// Polling loop
	for {
		select {
		case <-pollCtx.Done():
			return nil, fmt.Errorf("polling timeout after %v", FormatDuration(time.Since(p.startTime)))

		case <-sigChan:
			if p.config.Debug {
				fmt.Println("\nPolling interrupted by user")
			}
			return run, fmt.Errorf("polling interrupted")

		case <-ticker.C:
			run, err = fn(pollCtx)
			if err != nil {
				// Log error but continue polling for transient errors
				if p.config.Debug {
					fmt.Printf("Poll error (will retry): %v\n", err)
				}
				continue
			}

			if onUpdate != nil {
				onUpdate(run)
			}

			if IsTerminalStatus(run.Status) {
				return run, nil
			}

			// Show progress if enabled
			if p.config.ShowProgress {
				elapsed := time.Since(p.startTime)
				fmt.Printf("\rPolling... [%s elapsed] Status: %s",
					FormatDuration(elapsed), run.Status)
			}
		}
	}
}

func IsTerminalStatus(status models.RunStatus) bool {
	switch status {
	case models.StatusDone, models.StatusFailed:
		return true
	case "CANCELLED", "CANCELED": // Handle both spellings
		return true
	default:
		return false
	}
}

func ShowPollingProgress(startTime time.Time, status string, message string) {
	elapsed := time.Since(startTime)
	if message != "" {
		fmt.Printf("\r[%s] %s - %s", FormatDuration(elapsed), status, message)
	} else {
		fmt.Printf("\r[%s] %s", FormatDuration(elapsed), status)
	}
}

func ClearLine() {
	fmt.Print("\r\033[K")
}

// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/repobird/repobird-cli/internal/errors"
)

type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64
}

func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.2,
	}
}

type Client struct {
	config *Config
	debug  bool
}

func NewClient(config *Config, debug bool) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{
		config: config,
		debug:  debug,
	}
}

func (c *Client) DoWithRetry(ctx context.Context, fn func() error) error {
	return c.DoWithRetryAndResult(ctx, func() (interface{}, error) {
		return nil, fn()
	})
}

func (c *Client) DoWithRetryAndResult(ctx context.Context, fn func() (interface{}, error)) error {
	delay := c.config.InitialDelay
	var lastErr error

	for attempt := 1; attempt <= c.config.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !errors.IsRetryable(err) {
			if c.debug {
				fmt.Printf("Error is not retryable: %v\n", err)
			}
			return fmt.Errorf("permanent error: %w", err)
		}

		// Check if this was the last attempt
		if attempt == c.config.MaxAttempts {
			if c.debug {
				fmt.Printf("Giving up after %d attempts\n", c.config.MaxAttempts)
			}
			return fmt.Errorf("giving up after %d attempts: %w", c.config.MaxAttempts, lastErr)
		}

		// Calculate delay with jitter
		jitter := time.Duration(rand.Float64() * c.config.Jitter * float64(delay))
		actualDelay := delay + jitter

		if c.debug {
			fmt.Printf("Attempt %d/%d failed: %v. Retrying in %v...\n",
				attempt, c.config.MaxAttempts, err, actualDelay)
		}

		// Wait before next attempt
		select {
		case <-time.After(actualDelay):
			// Calculate next delay
			delay = time.Duration(float64(delay) * c.config.Multiplier)
			if delay > c.config.MaxDelay {
				delay = c.config.MaxDelay
			}
		case <-ctx.Done():
			return ctx.Err()
		}

		// Store result if needed (for future use)
		_ = result
	}

	return lastErr
}

type CircuitBreaker struct {
	maxFailures      int
	resetTimeout     time.Duration
	halfOpenRequests int

	failures     int
	lastFailTime time.Time
	state        CircuitState
	successCount int
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:      maxFailures,
		resetTimeout:     resetTimeout,
		halfOpenRequests: 3,
		state:            StateClosed,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	// Check if circuit should be reset
	if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.successCount = 0
	}

	// Check circuit state
	switch cb.state {
	case StateOpen:
		return fmt.Errorf("circuit breaker is open")

	case StateHalfOpen:
		err := fn()
		if err != nil {
			cb.state = StateOpen
			cb.lastFailTime = time.Now()
			cb.failures = cb.maxFailures
			return err
		}

		cb.successCount++
		if cb.successCount >= cb.halfOpenRequests {
			cb.state = StateClosed
			cb.failures = 0
		}
		return nil

	case StateClosed:
		err := fn()
		if err != nil {
			cb.failures++
			cb.lastFailTime = time.Now()

			if cb.failures >= cb.maxFailures {
				cb.state = StateOpen
			}
			return err
		}

		// Reset failures on success
		if cb.failures > 0 {
			cb.failures = int(math.Max(0, float64(cb.failures-1)))
		}
		return nil

	default:
		return fn()
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.resetTimeout {
		return StateHalfOpen
	}
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.state = StateClosed
	cb.failures = 0
	cb.successCount = 0
}

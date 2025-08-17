// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	repobirdErrors "github.com/repobird/repobird-cli/internal/errors"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts = 3, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay = 1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay = 30s, got %v", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("expected Multiplier = 2.0, got %f", config.Multiplier)
	}
	if config.Jitter != 0.2 {
		t.Errorf("expected Jitter = 0.2, got %f", config.Jitter)
	}
}

func TestClient_DoWithRetry_Success(t *testing.T) {
	config := &Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	client := NewClient(config, false)

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 2 {
			return &repobirdErrors.NetworkError{
				Err: errors.New("temporary network error"),
			}
		}
		return nil
	}

	ctx := context.Background()
	err := client.DoWithRetry(ctx, fn)

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_DoWithRetry_NonRetryableError(t *testing.T) {
	config := &Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	client := NewClient(config, false)

	attempts := 0
	nonRetryableError := &repobirdErrors.AuthError{
		Message: "Invalid API key",
		Reason:  "invalid_key",
	}

	fn := func() error {
		attempts++
		return nonRetryableError
	}

	ctx := context.Background()
	err := client.DoWithRetry(ctx, fn)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
	if !errors.Is(err, nonRetryableError) {
		t.Errorf("expected error to contain original error")
	}
}

func TestClient_DoWithRetry_MaxAttemptsExceeded(t *testing.T) {
	config := &Config{
		MaxAttempts:  2,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	client := NewClient(config, false)

	attempts := 0
	retryableError := &repobirdErrors.NetworkError{
		Err: errors.New("persistent network error"),
	}

	fn := func() error {
		attempts++
		return retryableError
	}

	ctx := context.Background()
	err := client.DoWithRetry(ctx, fn)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_DoWithRetry_ContextCancellation(t *testing.T) {
	config := &Config{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}

	client := NewClient(config, false)

	attempts := 0
	fn := func() error {
		attempts++
		return &repobirdErrors.NetworkError{
			Err: errors.New("network error"),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.DoWithRetry(ctx, fn)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestCircuitBreaker_Call(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Test closed state - should allow calls
	if cb.State() != StateClosed {
		t.Errorf("expected StateClosed, got %v", cb.State())
	}

	// First failure
	err := cb.Call(func() error {
		return errors.New("failure")
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if cb.State() != StateClosed {
		t.Errorf("expected StateClosed after first failure, got %v", cb.State())
	}

	// Second failure should open circuit
	err = cb.Call(func() error {
		return errors.New("failure")
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if cb.State() != StateOpen {
		t.Errorf("expected StateOpen after second failure, got %v", cb.State())
	}

	// Circuit should reject calls when open
	err = cb.Call(func() error {
		return nil
	})
	if err == nil {
		t.Error("expected circuit breaker error, got nil")
	}
	if err.Error() != "circuit breaker is open" {
		t.Errorf("expected 'circuit breaker is open', got %v", err.Error())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	// Trigger circuit breaker
	cb.Call(func() error {
		return errors.New("failure")
	})

	if cb.State() != StateOpen {
		t.Errorf("expected StateOpen, got %v", cb.State())
	}

	// Reset circuit breaker
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected StateClosed after reset, got %v", cb.State())
	}

	// Should allow calls again
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected nil error after reset, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)

	// Trigger circuit breaker
	cb.Call(func() error {
		return errors.New("failure")
	})

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should be half-open now
	if cb.State() != StateHalfOpen {
		t.Errorf("expected StateHalfOpen, got %v", cb.State())
	}

	// Successful call should close circuit
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Multiple successes should close the circuit
	for i := 0; i < 3; i++ {
		cb.Call(func() error {
			return nil
		})
	}

	if cb.State() != StateClosed {
		t.Errorf("expected StateClosed after successful calls, got %v", cb.State())
	}
}

func TestNewClient(t *testing.T) {
	// Test with nil config
	client := NewClient(nil, false)
	if client.config == nil {
		t.Error("expected default config, got nil")
	}
	if client.config.MaxAttempts != 3 {
		t.Errorf("expected default MaxAttempts = 3, got %d", client.config.MaxAttempts)
	}

	// Test with custom config
	customConfig := &Config{
		MaxAttempts: 5,
	}
	client = NewClient(customConfig, true)
	if client.config.MaxAttempts != 5 {
		t.Errorf("expected custom MaxAttempts = 5, got %d", client.config.MaxAttempts)
	}
	if !client.debug {
		t.Error("expected debug = true")
	}
}

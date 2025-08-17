// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0ms",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "less than one second",
			duration: 999 * time.Millisecond,
			expected: "999ms",
		},
		{
			name:     "exactly one second",
			duration: 1 * time.Second,
			expected: "1.0s",
		},
		{
			name:     "seconds with decimal",
			duration: 1500 * time.Millisecond,
			expected: "1.5s",
		},
		{
			name:     "multiple seconds",
			duration: 45 * time.Second,
			expected: "45.0s",
		},
		{
			name:     "less than one minute",
			duration: 59 * time.Second,
			expected: "59.0s",
		},
		{
			name:     "exactly one minute",
			duration: 1 * time.Minute,
			expected: "1.0m",
		},
		{
			name:     "minutes with decimal",
			duration: 90 * time.Second,
			expected: "1.5m",
		},
		{
			name:     "multiple minutes",
			duration: 30 * time.Minute,
			expected: "30.0m",
		},
		{
			name:     "less than one hour",
			duration: 59 * time.Minute,
			expected: "59.0m",
		},
		{
			name:     "exactly one hour",
			duration: 1 * time.Hour,
			expected: "1.0h",
		},
		{
			name:     "hours with decimal",
			duration: 90 * time.Minute,
			expected: "1.5h",
		},
		{
			name:     "multiple hours",
			duration: 24 * time.Hour,
			expected: "24.0h",
		},
		{
			name:     "complex duration",
			duration: 2*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "2.5h",
		},
		{
			name:     "negative duration",
			duration: -5 * time.Second,
			expected: "-5000ms",
		},
		{
			name:     "negative milliseconds",
			duration: -100 * time.Millisecond,
			expected: "-100ms",
		},
		{
			name:     "negative hours",
			duration: -2 * time.Hour,
			expected: "-7200000ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateProgress(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		expected float64
	}{
		{
			name:     "zero total returns zero",
			current:  5,
			total:    0,
			expected: 0,
		},
		{
			name:     "zero current",
			current:  0,
			total:    100,
			expected: 0,
		},
		{
			name:     "half progress",
			current:  50,
			total:    100,
			expected: 50.0,
		},
		{
			name:     "full progress",
			current:  100,
			total:    100,
			expected: 100.0,
		},
		{
			name:     "quarter progress",
			current:  25,
			total:    100,
			expected: 25.0,
		},
		{
			name:     "three quarters",
			current:  75,
			total:    100,
			expected: 75.0,
		},
		{
			name:     "decimal percentage",
			current:  1,
			total:    3,
			expected: 33.333333333333336,
		},
		{
			name:     "over 100 percent",
			current:  150,
			total:    100,
			expected: 150.0,
		},
		{
			name:     "negative current",
			current:  -10,
			total:    100,
			expected: -10.0,
		},
		{
			name:     "negative total",
			current:  10,
			total:    -100,
			expected: -10.0,
		},
		{
			name:     "both negative",
			current:  -50,
			total:    -100,
			expected: 50.0,
		},
		{
			name:     "small numbers",
			current:  1,
			total:    10,
			expected: 10.0,
		},
		{
			name:     "large numbers",
			current:  1000000,
			total:    10000000,
			expected: 10.0,
		},
		{
			name:     "exact third",
			current:  33,
			total:    99,
			expected: 33.333333333333336,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateProgress(tt.current, tt.total)
			assert.InDelta(t, tt.expected, result, 0.0000001)
		})
	}
}

func TestCalculateProgressPrecision(t *testing.T) {
	t.Run("floating point precision", func(t *testing.T) {
		// Test that the function handles floating point arithmetic correctly
		result := CalculateProgress(1, 7)
		assert.InDelta(t, 14.285714, result, 0.000001)
	})

	t.Run("very small percentage", func(t *testing.T) {
		result := CalculateProgress(1, 1000000)
		assert.InDelta(t, 0.0001, result, 0.0000001)
	})

	t.Run("very large percentage", func(t *testing.T) {
		result := CalculateProgress(1000000, 1)
		assert.Equal(t, float64(100000000), result)
	})
}

func TestFormatDurationEdgeCases(t *testing.T) {
	t.Run("nanoseconds", func(t *testing.T) {
		// Less than a millisecond should still show 0ms
		result := FormatDuration(500 * time.Nanosecond)
		assert.Equal(t, "0ms", result)
	})

	t.Run("microseconds", func(t *testing.T) {
		// Less than a millisecond should still show 0ms
		result := FormatDuration(500 * time.Microsecond)
		assert.Equal(t, "0ms", result)
	})

	t.Run("exactly 999 microseconds", func(t *testing.T) {
		result := FormatDuration(999 * time.Microsecond)
		assert.Equal(t, "0ms", result)
	})

	t.Run("exactly 1000 microseconds", func(t *testing.T) {
		result := FormatDuration(1000 * time.Microsecond)
		assert.Equal(t, "1ms", result)
	})

	t.Run("maximum duration", func(t *testing.T) {
		// Test with a very large duration
		result := FormatDuration(365 * 24 * time.Hour)
		assert.Equal(t, "8760.0h", result)
	})
}

func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		100 * time.Millisecond,
		5 * time.Second,
		2 * time.Minute,
		1 * time.Hour,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatDuration(durations[i%len(durations)])
	}
}

func BenchmarkCalculateProgress(b *testing.B) {
	pairs := [][2]int{
		{0, 100},
		{50, 100},
		{100, 100},
		{33, 99},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pair := pairs[i%len(pairs)]
		CalculateProgress(pair[0], pair[1])
	}
}

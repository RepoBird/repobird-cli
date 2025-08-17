// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"strings"
)

// SuggestFieldName suggests the most similar field name for an unknown field
func SuggestFieldName(input string, validFields []string) string {
	if input == "" || len(validFields) == 0 {
		return ""
	}

	input = strings.ToLower(input)
	bestMatch := ""
	minDistance := len(input) + 1

	// Only suggest if the distance is reasonable (≤ 2 for most cases)
	threshold := 2
	if len(input) <= 3 {
		threshold = 1 // Stricter for very short fields
	}

	for _, field := range validFields {
		field = strings.ToLower(field)
		distance := levenshteinDistance(input, field)

		// Skip exact matches - we only suggest for similar but not identical fields
		if distance == 0 {
			continue
		}

		// Consider it a good match if:
		// 1. Distance is within threshold
		// 2. It's the best match so far
		if distance <= threshold && distance < minDistance {
			// Extra check: if the input is just the plural of a valid field, prioritize it
			if strings.HasSuffix(input, "s") && field == input[:len(input)-1] {
				return field
			}
			// Or if a valid field is just the plural of input
			if strings.HasSuffix(field, "s") && input == field[:len(field)-1] {
				return field
			}

			bestMatch = field
			minDistance = distance
		}
	}

	// Only return if we found a reasonable match
	if minDistance <= threshold {
		return bestMatch
	}

	return ""
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create a matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

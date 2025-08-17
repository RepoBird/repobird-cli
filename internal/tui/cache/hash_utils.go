// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"crypto/sha256"
	"fmt"
)

// CalculateStringHash calculates SHA256 hash of a string
func CalculateStringHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CalculateFileHash calculates SHA256 hash of file contents
func CalculateFileHash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

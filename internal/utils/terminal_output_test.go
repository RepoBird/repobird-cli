// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteLiveUpdateUsesPlainLinesOutsideTTY(t *testing.T) {
	var output bytes.Buffer

	writeLiveUpdate(&output, false, "Polling... %s", "PROCESSING")
	clearLiveOutput(&output, false)

	require.Equal(t, "Polling... PROCESSING\n", output.String())
	require.NotContains(t, output.String(), "\r")
	require.NotContains(t, output.String(), "\033[")
}

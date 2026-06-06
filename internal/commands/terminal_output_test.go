// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintLiveUpdateUsesPlainLinesOutsideTTY(t *testing.T) {
	var output bytes.Buffer

	printLiveUpdate(&output, false, "Status: %s", "PROCESSING")
	clearLiveOutput(&output, false)

	require.Equal(t, "Status: PROCESSING\n", output.String())
	require.NotContains(t, output.String(), "\r")
	require.NotContains(t, output.String(), "\033[")
}

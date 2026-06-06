// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadAPIKeyInteractiveNonTerminalPromptsOnce(t *testing.T) {
	stdin := pipeWithInput(t, "rpb_test_key\n")
	defer stdin.Close()

	var output bytes.Buffer
	apiKey, err := readAPIKeyInteractive(stdin, &output)

	require.NoError(t, err)
	require.Equal(t, "rpb_test_key", apiKey)
	require.Equal(t, 1, strings.Count(output.String(), "Enter your API key: "))
	require.NotContains(t, output.String(), "\r")
	require.NotContains(t, output.String(), "\033[")
}

func pipeWithInput(t *testing.T, input string) *os.File {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	_, err = writer.WriteString(input)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return reader
}

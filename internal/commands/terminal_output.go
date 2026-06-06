// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func stdoutIsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func clearLiveOutput(output io.Writer, isTerminal bool) {
	if isTerminal {
		fmt.Fprint(output, "\r\033[K")
	}
}

func clearPreviousLiveLines(output io.Writer, isTerminal bool, lineCount int) {
	if !isTerminal {
		return
	}
	for i := 0; i < lineCount; i++ {
		fmt.Fprint(output, "\033[A\033[2K")
	}
}

func printLiveUpdate(output io.Writer, isTerminal bool, format string, args ...interface{}) {
	if isTerminal {
		fmt.Fprintf(output, "\r"+format, args...)
		return
	}
	fmt.Fprintf(output, format+"\n", args...)
}

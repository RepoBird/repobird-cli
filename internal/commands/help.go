// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func coloredHelp(cmd *cobra.Command, _ []string) {
	_ = writeColoredHelp(cmd.OutOrStdout(), cmd)
}

func coloredUsage(cmd *cobra.Command) error {
	return writeColoredUsage(cmd.OutOrStderr(), cmd)
}

func writeColoredHelp(w io.Writer, cmd *cobra.Command) error {
	description := strings.TrimRight(cmd.Long, " \t\r\n")
	if description == "" {
		description = strings.TrimRight(cmd.Short, " \t\r\n")
	}
	if description != "" {
		fmt.Fprintln(w, description)
		fmt.Fprintln(w)
	}
	if cmd.Runnable() || cmd.HasSubCommands() {
		return writeColoredUsage(w, cmd)
	}
	return nil
}

func writeColoredUsage(w io.Writer, cmd *cobra.Command) error {
	styler := styleFor(w)
	fmt.Fprint(w, styler.Heading("Usage:"))
	if cmd.Runnable() {
		fmt.Fprintf(w, "\n  %s", cmd.UseLine())
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(w, "\n  %s [command]", cmd.CommandPath())
	}
	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(w, "\n\n%s\n  %s", styler.Heading("Aliases:"), cmd.NameAndAliases())
	}
	if cmd.HasExample() {
		fmt.Fprintf(w, "\n\n%s\n%s", styler.Heading("Examples:"), cmd.Example)
	}
	if cmd.HasAvailableSubCommands() {
		writeCommandList(w, cmd, styler.Heading("Available Commands:"))
	}
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(w, "\n\n%s\n%s", styler.Heading("Flags:"), strings.TrimRight(cmd.LocalFlags().FlagUsages(), " \t\r\n"))
	}
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(w, "\n\n%s\n%s", styler.Heading("Global Flags:"), strings.TrimRight(cmd.InheritedFlags().FlagUsages(), " \t\r\n"))
	}
	if cmd.HasHelpSubCommands() {
		fmt.Fprintf(w, "\n\n%s", styler.Heading("Additional help topics:"))
		for _, subcmd := range cmd.Commands() {
			if subcmd.IsAdditionalHelpTopicCommand() {
				fmt.Fprintf(w, "\n  %s %s", padRight(subcmd.CommandPath(), subcmd.CommandPathPadding()), subcmd.Short)
			}
		}
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(w, "\n\n%s Use \"%s [command] --help\" for more information about a command.", styler.Info("Tip:"), cmd.CommandPath())
	}
	fmt.Fprintln(w)
	return nil
}

type commandLabeler interface {
	Label(string) string
}

func writeCommandList(w io.Writer, cmd *cobra.Command, heading string) {
	styler := styleFor(w)
	fmt.Fprintf(w, "\n\n%s", heading)
	for _, subcmd := range cmd.Commands() {
		if subcmd.IsAvailableCommand() || subcmd.Name() == "help" {
			fmt.Fprintf(w, "\n  %s %s", padRight(styler.Label(subcmd.Name()), subcmd.NamePadding()+ansiPaddingDelta(styler, subcmd.Name())), subcmd.Short)
		}
	}
}

func padRight(text string, width int) string {
	padding := width - len(text)
	if padding > 0 {
		return text + strings.Repeat(" ", padding)
	}
	return text
}

func ansiPaddingDelta(styler commandLabeler, text string) int {
	return len(styler.Label(text)) - len(text)
}

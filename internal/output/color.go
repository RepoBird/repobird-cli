// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

const (
	ColorAuto   = "auto"
	ColorAlways = "always"
	ColorNever  = "never"
)

type Styler struct {
	enabled  bool
	renderer *lipgloss.Renderer
}

func NewStyler(out io.Writer, mode string) Styler {
	mode = NormalizeColorMode(mode)
	enabled := colorEnabled(out, mode)
	styler := Styler{enabled: enabled}
	if enabled {
		styler.renderer = lipgloss.NewRenderer(out)
		styler.renderer.SetColorProfile(termenv.TrueColor)
	}
	return styler
}

func NormalizeColorMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ColorAlways:
		return ColorAlways
	case ColorNever, "off", "false", "disabled", "none":
		return ColorNever
	default:
		return ColorAuto
	}
}

func ModeFromEnv(configMode string) string {
	if os.Getenv("NO_COLOR") != "" {
		return ColorNever
	}
	if envMode := os.Getenv("REPOBIRD_COLOR"); envMode != "" {
		return NormalizeColorMode(envMode)
	}
	return NormalizeColorMode(configMode)
}

func (s Styler) Success(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true))
}

func (s Styler) Warning(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true))
}

func (s Styler) Error(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true))
}

func (s Styler) Info(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true))
}

func (s Styler) Heading(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true))
}

func (s Styler) Label(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true))
}

func (s Styler) Muted(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
}

func (s Styler) URL(text string) string {
	return s.render(text, lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Underline(true))
}

func (s Styler) Status(status string) string {
	style := lipgloss.NewStyle().Bold(true)
	switch strings.ToUpper(status) {
	case "DONE", "COMPLETED", "SUCCESS", "SUCCEEDED":
		style = style.Foreground(lipgloss.Color("10"))
	case "FAILED", "ERROR", "CANCELLED", "CANCELED":
		style = style.Foreground(lipgloss.Color("9"))
	case "PROCESSING", "RUNNING", "POST_PROCESS", "IN_PROGRESS":
		style = style.Foreground(lipgloss.Color("14"))
	case "QUEUED", "CREATED", "INITIALIZING", "PENDING":
		style = style.Foreground(lipgloss.Color("11"))
	default:
		style = style.Foreground(lipgloss.Color("12"))
	}
	return s.render(status, style)
}

func (s Styler) render(text string, style lipgloss.Style) string {
	if !s.enabled {
		return text
	}
	if s.renderer != nil {
		style = style.Renderer(s.renderer)
	}
	return style.Render(text)
}

func colorEnabled(out io.Writer, mode string) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		file, ok := out.(*os.File)
		return ok && term.IsTerminal(int(file.Fd()))
	}
}

func hasANSI(text string) bool {
	return strings.Contains(text, "\x1b[")
}

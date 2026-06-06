// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"io"
	"os"

	configpkg "github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/output"
)

func styleFor(out io.Writer) output.Styler {
	mode := output.ColorAuto
	if cfg != nil && cfg.Config != nil {
		mode = cfg.Color
	} else if loaded, err := configpkg.LoadSecureConfig(); err == nil && loaded.Config != nil {
		mode = loaded.Color
	}
	return output.NewStyler(out, output.ModeFromEnv(mode))
}

func stdoutStyle() output.Styler {
	return styleFor(os.Stdout)
}

func stderrStyle() output.Styler {
	return styleFor(os.Stderr)
}

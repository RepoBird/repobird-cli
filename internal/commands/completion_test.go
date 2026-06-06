// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCompletionBashAppendsProfileBlock(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(home, ".bashrc")
	require.NoError(t, os.WriteFile(bashrc, []byte("# existing\n"), 0o644))

	var out bytes.Buffer
	err := installCompletion(rootCmd, "bash", completionInstallOptions{
		HomeDir: home,
		Alias:   "rb",
		Out:     &out,
	})

	require.NoError(t, err)
	content := readFile(t, bashrc)
	assert.Contains(t, content, "source <(repobird completion bash)")
	assert.Contains(t, content, "complete -o default -F __start_repobird rb")
	assert.Contains(t, out.String(), "Updated "+bashrc)
}

func TestInstallCompletionBashIsIdempotent(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer
	opts := completionInstallOptions{HomeDir: home, Alias: "rb", Out: &out}

	require.NoError(t, installCompletion(rootCmd, "bash", opts))
	require.NoError(t, installCompletion(rootCmd, "bash", opts))

	content := readFile(t, filepath.Join(home, ".bashrc"))
	assert.Equal(t, 1, strings.Count(content, "source <(repobird completion bash)"))
	assert.Contains(t, out.String(), "Completion setup already present")
}

func TestInstallCompletionZshWritesStaticCompletionAndProfileBlock(t *testing.T) {
	home := t.TempDir()

	var out bytes.Buffer
	err := installCompletion(rootCmd, "zsh", completionInstallOptions{
		HomeDir: home,
		Alias:   "rb",
		Out:     &out,
	})

	require.NoError(t, err)
	completion := readFile(t, filepath.Join(home, ".config", "zsh", "completions", "_repobird"))
	assert.Contains(t, completion, "compdef _repobird repobird")
	assert.Contains(t, completion, "compdef _repobird rb")

	zshrc := readFile(t, filepath.Join(home, ".zshrc"))
	assert.Contains(t, zshrc, "fpath=(~/.config/zsh/completions $fpath)")
	assert.Contains(t, zshrc, "autoload -U compinit")
	assert.Contains(t, zshrc, "compinit")
}

func TestInstallCompletionFishWritesRepobirdAndAliasCompletions(t *testing.T) {
	home := t.TempDir()

	err := installCompletion(rootCmd, "fish", completionInstallOptions{
		HomeDir: home,
		Alias:   "rb",
		Out:     ioDiscard{},
	})

	require.NoError(t, err)
	repobirdCompletion := readFile(t, filepath.Join(home, ".config", "fish", "completions", "repobird.fish"))
	aliasCompletion := readFile(t, filepath.Join(home, ".config", "fish", "completions", "rb.fish"))
	assert.Contains(t, repobirdCompletion, "complete -c repobird")
	assert.Contains(t, aliasCompletion, "complete -c rb")
}

func TestInstallCompletionDryRunDoesNotWriteFiles(t *testing.T) {
	home := t.TempDir()
	var out bytes.Buffer

	err := installCompletion(rootCmd, "fish", completionInstallOptions{
		HomeDir: home,
		Alias:   "rb",
		DryRun:  true,
		Out:     &out,
	})

	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(home, ".config", "fish", "completions", "repobird.fish"))
	assert.Contains(t, out.String(), "Would write")
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

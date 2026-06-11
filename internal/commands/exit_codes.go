// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	stderrors "errors"
	"strings"

	"github.com/repobird/repobird-cli/internal/errors"
)

const (
	ExitCodeSuccess   = 0
	ExitCodeGeneric   = 1
	ExitCodeAuth      = 2
	ExitCodeQuota     = 3
	ExitCodeRunFailed = 4
	ExitCodeTimeout   = 5
)

type exitError struct {
	code int
	err  error
}

func newExitError(code int, message string) error {
	return &exitError{code: code, err: stderrors.New(message)}
}

func wrapExitError(code int, err error) error {
	if err == nil {
		return nil
	}
	return &exitError{code: code, err: err}
}

func (e *exitError) Error() string {
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	return e.err
}

func (e *exitError) ExitCode() int {
	return e.code
}

func exitCodeForError(err error) int {
	if err == nil {
		return ExitCodeSuccess
	}

	var coded interface{ ExitCode() int }
	if stderrors.As(err, &coded) {
		return coded.ExitCode()
	}

	if errors.IsQuotaExceeded(err) || containsAny(err.Error(), "quota exceeded", "no runs remaining", "insufficient credits", "no credits") {
		return ExitCodeQuota
	}
	if errors.IsAuthError(err) || containsAny(err.Error(), "api key not configured", "authentication failed", "unauthorized", "invalid api key") {
		return ExitCodeAuth
	}

	return ExitCodeGeneric
}

func containsAny(value string, needles ...string) bool {
	lower := strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

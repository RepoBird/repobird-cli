// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


// This file shows the fix needed for the help_view.go status line
// Line 462 in help_view.go should be changed from:
// shortHelp := "[↑↓/jk]scroll [Ctrl+u/d]halfpage [g/G]top/bottom [y]copy [?/q/ESC]back"
// To:
// shortHelp := "[↑↓/jk]scroll [Ctrl+u/d]halfpage [g/G]top/bottom [y/Y]copy [q/h/b/?]back"

package components

// The actual fix would be in help_view.go around line 462:
/*
	// Status line
	// FIXED: Updated status line to show correct keys and remove confusing ESC
	shortHelp := "[↑↓/jk]scroll [Ctrl+u/d]halfpage [g/G]top/bottom [y/Y]copy [q/h/b/?]back"
*/

module github.com/repobird/repobird-cli

go 1.24.5

require (
	github.com/charmbracelet/bubbles v0.21.0      // Pre-built UI components for Bubble Tea (text inputs, spinners, progress bars)
	github.com/charmbracelet/bubbletea v1.3.6     // Terminal UI framework based on Elm architecture - powers the TUI mode
	github.com/charmbracelet/lipgloss v1.1.0      // Terminal styling library for colors, borders, and layout in TUI
	github.com/spf13/cobra v1.9.1                 // CLI framework for building commands and subcommands structure
	github.com/spf13/viper v1.20.1                // Configuration management - handles config files and environment variables
	github.com/stretchr/testify v1.10.0           // Testing toolkit with assertions and mocking capabilities
	github.com/zalando/go-keyring v0.2.6          // Secure credential storage using OS keychain (macOS, Windows, Linux)
	golang.org/x/term v0.34.0                     // Terminal handling utilities for raw mode and terminal size detection
)

require (
	// Indirect dependencies (automatically included by Go modules)
	al.essio.dev/pkg/shellescape v1.5.1 // indirect - Shell command escaping for safe command execution
	github.com/atotto/clipboard v0.1.4 // indirect - Cross-platform clipboard access (used by Bubble Tea for copy/paste)
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect - OSC52 terminal clipboard protocol support
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect - Terminal color capability detection
	github.com/charmbracelet/x/ansi v0.9.3 // indirect - ANSI escape sequence utilities for Bubble Tea
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect - Cell buffer management for terminal rendering
	github.com/charmbracelet/x/term v0.2.1 // indirect - Terminal utilities for Bubble Tea framework
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect - Markdown to man page conversion (Cobra dependency)
	github.com/danieljoos/wincred v1.2.2 // indirect - Windows credential manager (go-keyring dependency)
	github.com/davecgh/go-spew v1.1.1 // indirect - Deep pretty printer for debugging (testify dependency)
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect - Windows console input handling
	github.com/fsnotify/fsnotify v1.9.0 // indirect - File system notifications (Viper config watching)
	github.com/gdamore/encoding v1.0.1 // indirect - Character encoding support for terminals
	github.com/gdamore/tcell/v2 v2.6.0 // indirect - Terminal cell-based UI library (alternative to termbox)
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect - Struct to map decoding (Viper dependency)
	github.com/godbus/dbus/v5 v5.1.0 // indirect - D-Bus protocol for Linux keyring access
	github.com/inconshreveable/mousetrap v1.1.0 // indirect - Windows mouse event detection (Cobra dependency)
	github.com/ktr0731/go-ansisgr v0.1.0 // indirect - ANSI SGR (Select Graphic Rendition) parser
	github.com/ktr0731/go-fuzzyfinder v0.9.0 // indirect - Interactive fuzzy finder for repository selection fallback
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect - Color manipulation and conversion library
	github.com/mattn/go-isatty v0.0.20 // indirect - Terminal/TTY detection
	github.com/mattn/go-localereader v0.0.1 // indirect - Locale-aware input reader for terminals
	github.com/mattn/go-runewidth v0.0.16 // indirect - Unicode character width calculation for terminal display
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect - ANSI sequence parser and writer
	github.com/muesli/cancelreader v0.2.2 // indirect - Cancelable reader for terminal input
	github.com/muesli/termenv v0.16.0 // indirect - Terminal environment detection and styling
	github.com/nsf/termbox-go v1.1.1 // indirect - Terminal UI library (alternative backend)
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect - TOML configuration format support (Viper dependency)
	github.com/pkg/errors v0.9.1 // indirect - Error handling with stack traces
	github.com/pmezard/go-difflib v1.0.0 // indirect - Diff algorithm implementation (testify dependency)
	github.com/rivo/uniseg v0.4.7 // indirect - Unicode text segmentation for proper character handling
	github.com/russross/blackfriday/v2 v2.1.0 // indirect - Markdown processor (Cobra help generation)
	github.com/sagikazarmark/locafero v0.10.0 // indirect - File system abstraction layer (Viper dependency)
	github.com/sahilm/fuzzy v0.1.1 // indirect - Fuzzy string matching for FZF search functionality
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect - Structured concurrency utilities
	github.com/spf13/afero v1.14.0 // indirect - File system abstraction (Viper dependency)
	github.com/spf13/cast v1.9.2 // indirect - Type casting utilities (Viper dependency)
	github.com/spf13/pflag v1.0.7 // indirect - POSIX-style command line flags (Cobra dependency)
	github.com/stretchr/objx v0.5.2 // indirect - Object manipulation for testing (testify dependency)
	github.com/subosito/gotenv v1.6.0 // indirect - .env file parser (Viper dependency)
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect - Terminfo database parser for terminal capabilities
	golang.design/x/clipboard v0.7.1 // indirect - Cross-platform clipboard implementation with image support
	golang.org/x/exp/shiny v0.0.0-20250808145144-a408d31f581a // indirect - Experimental UI libraries (clipboard dependency)
	golang.org/x/image v0.30.0 // indirect - Image manipulation (clipboard image support)
	golang.org/x/mobile v0.0.0-20250808145247-395d808d53cd // indirect - Mobile platform support (clipboard dependency)
	golang.org/x/sync v0.16.0 // indirect - Synchronization primitives and concurrent patterns
	golang.org/x/sys v0.35.0 // indirect - Low-level OS interface for system calls
	golang.org/x/text v0.28.0 // indirect - Text processing, encoding, and Unicode support
	gopkg.in/yaml.v3 v3.0.1 // indirect - YAML parsing and serialization (config and test files)
)

# Phase 3: TUI Implementation (Week 4-5)

## Overview
Build an interactive Terminal User Interface (TUI) with vim keybindings for efficient navigation and real-time monitoring of RepoBird agent runs.

## Tasks

### Bubbletea Framework Setup
- [ ] Add Bubbletea dependencies
- [ ] Create TUI application structure
- [ ] Implement main update/view loop
- [ ] Set up message passing system
- [ ] Configure terminal initialization
- [ ] Add graceful shutdown handling

### Core TUI Components
- [ ] Create base component interfaces
- [ ] Implement list component with scrolling
- [ ] Build table component for data display
- [ ] Add input field component
- [ ] Create modal/dialog system
- [ ] Implement status bar component
- [x] Add loading/spinner indicators

### Run List View
- [ ] Display paginated list of runs
- [ ] Show run status with color coding (QUEUED=yellow, PROCESSING=blue, DONE=green, FAILED=red)
- [ ] Map status enums to icons (✓ DONE, ⟳ PROCESSING, ⏳ QUEUED, ✗ FAILED)
- [ ] Implement sorting (createdAt, status, repository)
- [ ] Add filtering capabilities (status, repository, date range)
- [ ] Support run selection with Enter key
- [ ] Show run metadata (repo, source/target branches, duration)
- [ ] Add auto-refresh with 5-second interval for active runs
- [ ] Display remaining runs counter (pro/plan) in status bar
- [x] Implement background preloading of run details for instant access

### Run Details View
- [ ] Display comprehensive run information
- [ ] Show real-time log streaming
- [ ] Add log search functionality
- [ ] Implement log level filtering
- [ ] Support log export to file
- [ ] Display run configuration
- [ ] Show error messages prominently
- [ ] Add copy-to-clipboard support

### New Run Creation View
- [ ] Build file selector/browser
- [ ] Add inline text editor
- [ ] Support template selection
- [ ] Implement repository picker
- [ ] Add branch selection
- [ ] Show configuration options
- [ ] Validate input before submission
- [ ] Display dry-run results

### Vim Keybindings Implementation
- [ ] Set up key mapping system
- [ ] Implement navigation keys (h,j,k,l)
- [ ] Add page movement (Ctrl+d, Ctrl+u, g, G)
- [ ] Implement search (/, ?, n, N)
- [ ] Add command mode (:)
- [ ] Support visual selection mode
- [ ] Implement quick actions (d-delete, r-refresh)
- [ ] Add help system (?)
- [ ] Maintain arrow key fallbacks

### Status Polling (No WebSocket in v1)
- [ ] Implement 5-second polling for active runs
- [ ] Poll only runs with status: QUEUED, INITIALIZING, PROCESSING, POST_PROCESS
- [ ] Stop polling when status becomes DONE or FAILED
- [ ] Update UI components reactively on poll
- [ ] Add visual indicator for polling (spinner/dots)
- [ ] Implement exponential backoff on errors
- [ ] Show last update timestamp
- [ ] Allow manual refresh with 'r' key

### Styling with Lipgloss
- [ ] Define color scheme (dark/light modes)
- [ ] Create consistent styling system
- [ ] Add syntax highlighting for logs
- [ ] Implement responsive layouts
- [ ] Support terminal color detection
- [ ] Add ASCII art logo/branding
- [ ] Create smooth animations

## TUI Views Design

### Main View Layout
```
┌─────────────────────────────────────────────────────┐
│ RepoBird CLI v1.0.0        Pro: 25/30 runs remaining│
├─────────────────────────────────────────────────────┤
│ Runs (25 total)                     Press ? for help│
├─────────────────────────────────────────────────────┤
│ ID     Status         Repository      Time    Branch │
│ 12345  ✓ DONE        acme/webapp     2m ago  main   │
│ 12346  ⟳ PROCESSING  acme/backend    5m ago  dev    │
│ 12347  ⏳ QUEUED     acme/mobile     10m ago fix-123│
│ 12348  ✗ FAILED     acme/api        1h ago  main   │
│ ...                                                  │
├─────────────────────────────────────────────────────┤
│ [n]ew [Enter]view [r]efresh [/]search [:] [q]uit    │
└─────────────────────────────────────────────────────┘
```

### Run Details Layout
```
┌─────────────────────────────────────────────────────┐
│ Run #12345                              Status: DONE│
├─────────────────────────────────────────────────────┤
│ Title: Fix login authentication bug                 │
│ Repository: acme/webapp                             │
│ Source: main → Target: fix/login-bug                │
│ Issue: #123                                          │
│ PR: https://github.com/acme/webapp/pull/456         │
│ Created: 2024-01-15 14:30:00                        │
│ Duration: 5m 23s (agent: 4m 50s)                    │
├─────────────────────────────────────────────────────┤
│ Logs:                           [Polling every 5s]  │
│ Status: QUEUED → INITIALIZING → PROCESSING → DONE   │
│                                                      │
│ Files Modified:                                     │
│ - src/auth/login.js                                 │
│ - src/auth/validate.js                              │
│ - tests/auth.test.js                                │
├─────────────────────────────────────────────────────┤
│ [b]ack [d]iff [l]ogs [p]lan [/]search [q]uit        │
└─────────────────────────────────────────────────────┘
```

## Keybinding Reference

### Navigation Mode
```
Movement:
  j/↓     Move down
  k/↑     Move up
  h/←     Go back
  l/→     Go forward/select
  g       Go to top
  G       Go to bottom
  Ctrl+d  Page down
  Ctrl+u  Page up

Search:
  /       Search forward
  ?       Search backward
  n       Next match
  N       Previous match
  *       Search for word under cursor

Actions:
  Enter   Select/view details
  Esc     Cancel/back
  r       Refresh
  n       New run
  d       Delete/cancel run
  s       View status
  q       Quit
  ?       Show help
```

### Command Mode
```
:q      Quit application
:w      Save (in editor)
:run    Execute new run
:help   Show help
:set    Change settings
:!cmd   Execute shell command
```

## Testing Requirements

### Unit Tests
- [ ] Component rendering tests
- [ ] Key event handling tests
- [ ] State management tests
- [ ] View transition tests

### Integration Tests
- [ ] Full TUI workflow tests
- [ ] Real-time update handling
- [ ] Terminal compatibility tests
- [ ] Performance under load

### Manual Testing
- [ ] Test on different terminal emulators
- [ ] Verify color scheme compatibility
- [ ] Test with various terminal sizes
- [ ] Validate vim keybinding accuracy

## Deliverables

1. Fully functional TUI application
2. Vim-style keybindings with help system
3. Real-time status updates
4. Multiple views (list, details, create)
5. Responsive and accessible design
6. Comprehensive keyboard navigation

## Success Criteria

- [ ] TUI launches in < 100ms
- [ ] Smooth scrolling at 60 FPS
- [ ] All vim keybindings work correctly
- [ ] Real-time updates display within 1s
- [ ] Works on all major terminal emulators
- [ ] Handles 1000+ runs without lag
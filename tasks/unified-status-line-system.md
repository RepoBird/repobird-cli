# Unified Global Status Line System - Implementation Tasks

## Overview

This document outlines the tasks needed to implement a unified global status line system across all RepoBird CLI TUI views. The current implementation has inconsistencies across different views, with some views not showing status lines during loading states and others handling messages differently.

## Current Status Line Implementation Analysis

### Existing StatusLine Component
- **Location**: `/internal/tui/components/statusline.go`
- **Features**: Universal StatusLine struct with left/right/help content support
- **Helper Functions**: DashboardStatusLine, RunListStatusLine, CreateRunStatusLine, DetailsStatusLine
- **Styling**: Consistent lipgloss styling with background color `235` and foreground `252`

### Current View-Specific Implementations

#### 1. Dashboard View (`/internal/tui/views/dashboard.go`)
**Current Issues:**
- Uses custom `renderStatusLine()` method instead of StatusLine component
- Does NOT show status line during initial loading state
- Has complex temporary message handling with `statusMessage` and `statusMessageTime`
- Uses notification line above status line for some feedback
- Inconsistent with other views' approach

**Current Status Line Logic:**
```go
// Custom renderStatusLine method
func (m DashboardView) renderStatusLine() string {
    // Complex logic for temporary messages
    if !m.statusMessageTime.IsZero() && time.Since(m.statusMessageTime) < 3*time.Second {
        // Show temporary message in green
    }
    // Regular status line with help text
}
```

#### 2. Create Run View (`/internal/tui/views/create.go`)
**Current Implementation:**
- Uses `components.CreateRunStatusLine()` helper
- Shows status line consistently
- Has different message handling approach
- Uses notification system for feedback

#### 3. Run List View (`/internal/tui/views/list.go`) 
**Current Implementation:**
- Uses `components.RunListStatusLine()` helper
- Shows status line consistently
- Standard message handling

#### 4. Details View (`/internal/tui/views/details.go`)
**Current Implementation:**
- Uses `components.DetailsStatusLine()` helper  
- Shows status line consistently
- Standard message handling

#### 5. Loading States
**Problem**: Dashboard view doesn't show status line during initial loading, while other views maintain status line visibility.

### Status Line Content Analysis

#### Left Content Patterns:
- Dashboard: `[DASH]` with layout name
- Create: `Create Run: {step}`
- List: `Runs: {count} total`
- Details: `Run Details: {runID}`

#### Right Content Patterns:  
- Dashboard: Data freshness timestamps
- Create: Usually empty
- List: Usually empty
- Details: Run status

#### Help Content Patterns:
- All views: Context-appropriate keyboard shortcuts
- Dashboard: `n:new s:status y:copy o:open ?:help q:quit`
- Create: Form-specific shortcuts
- List: Navigation shortcuts
- Details: Detail-specific shortcuts

#### Temporary Messages:
- Dashboard: Custom 3-second colored messages
- Other views: Notification line approach

## Required Tasks for Unified System

### Phase 1: Core StatusLine Enhancement

#### Task 1.1: Enhance StatusLine Component
**File**: `/internal/tui/components/statusline.go`
**Changes Needed:**
- Add support for temporary colored messages
- Add `SetTemporaryMessage(message, color, duration)` method
- Add message expiration checking in `Render()` method
- Add color theme support for different message types (success, error, info)

```go
type StatusLine struct {
    // ... existing fields
    tempMessage     string
    tempMessageTime time.Time
    tempMessageDuration time.Duration
    tempMessageColor lipgloss.Color
}

func (s *StatusLine) SetTemporaryMessage(message string, color lipgloss.Color, duration time.Duration) *StatusLine {
    s.tempMessage = message
    s.tempMessageTime = time.Now()
    s.tempMessageDuration = duration
    s.tempMessageColor = color
    return s
}
```

#### Task 1.2: Add Message Type Constants
**File**: `/internal/tui/components/statusline.go`
**Changes Needed:**
- Define color constants for different message types
- Success (green), Error (red), Info (blue), Warning (yellow)

### Phase 2: Global StatusLine Manager

#### Task 2.1: Create StatusLine Manager
**File**: `/internal/tui/components/statusline_manager.go` (new file)
**Purpose**: Centralized status line state management
**Features Needed:**
- Global status line instance
- Message queue for temporary messages  
- View registration system
- Consistent update mechanism

```go
type StatusLineManager struct {
    statusLine *StatusLine
    width      int
    viewType   string
    messageQueue []TemporaryMessage
}

type TemporaryMessage struct {
    Text     string
    Color    lipgloss.Color
    Duration time.Duration
    StartTime time.Time
}
```

#### Task 2.2: Implement View Registration
**Purpose**: Allow each view to register with the status line manager
**Methods Needed:**
- `RegisterView(viewType string, statusLine *StatusLine)`
- `UpdateView(viewType string, left, right, help string)`
- `ShowTemporaryMessage(message string, messageType MessageType)`

### Phase 3: Update Dashboard View

#### Task 3.1: Remove Custom Status Line Logic
**File**: `/internal/tui/views/dashboard.go`
**Changes:**
- Remove custom `renderStatusLine()` method
- Remove `statusMessage`, `statusMessageTime` fields
- Remove `clearStatusMessageMsg` type and timer
- Remove notification line rendering for status messages

#### Task 3.2: Integrate with StatusLine Component
**File**: `/internal/tui/views/dashboard.go`
**Changes:**
- Use StatusLine component consistently
- Show status line during loading states
- Use temporary message system for URL opening/copying feedback
- Update `View()` method to always render status line

```go
// Replace custom renderStatusLine with:
func (m DashboardView) renderStatusLine() string {
    // Determine current help text based on selection
    shortHelp := m.getContextualHelp()
    
    return components.DashboardStatusLine(
        m.width,
        "DASH",
        m.getDataFreshness(),
        shortHelp,
    ).SetWidth(m.width).
      SetTemporaryMessage(m.tempMessage, m.tempColor, 3*time.Second).
      Render()
}
```

#### Task 3.3: Fix Loading State Status Line
**File**: `/internal/tui/views/dashboard.go`
**Changes:**
- Show status line even when `m.loading` is true
- Display loading indicator in status line during initial load
- Maintain consistent status line height

### Phase 4: Standardize Message Handling

#### Task 4.1: Update URL Opening Feedback
**Files**: All view files using URL opening
**Changes:**
- Replace notification line messages with temporary status messages
- Use consistent green color for success messages
- Maintain 3-second duration

#### Task 4.2: Update Copy Feedback  
**Files**: All view files using copy functionality
**Changes:**
- Replace notification line messages with temporary status messages
- Use consistent color scheme
- Standardize message text

#### Task 4.3: Update Error Messages
**Files**: All view files
**Changes:**
- Use red color for error messages
- Consistent error message formatting
- Proper duration for error visibility

### Phase 5: View-Specific Updates

#### Task 5.1: Update Create Run View
**File**: `/internal/tui/views/create.go`
**Changes:**
- Ensure consistent use of StatusLine component
- Update error handling to use temporary messages
- Standardize clipboard feedback

#### Task 5.2: Update Run List View
**File**: `/internal/tui/views/list.go`
**Changes:**
- Verify consistent StatusLine usage
- Update any custom message handling

#### Task 5.3: Update Details View
**File**: `/internal/tui/views/details.go`
**Changes:**
- Verify consistent StatusLine usage
- Update any custom message handling

### Phase 6: Testing and Validation

#### Task 6.1: Test Loading States
**Verification Needed:**
- All views show status line during loading
- Loading indicators are visible and appropriate
- No flickering or layout shifts

#### Task 6.2: Test Message Systems
**Verification Needed:**
- Temporary messages work correctly across all views
- No double status lines appear
- Message colors are consistent
- Message timing (3 seconds) works properly

#### Task 6.3: Test View Transitions
**Verification Needed:**
- Status line remains consistent when switching views
- No status line disappearing during transitions
- View indicators ([DASH], [DETAILS], etc.) work correctly

### Phase 7: Documentation and Cleanup

#### Task 7.1: Update Documentation
**Files to Update:**
- `/docs/development-guide.md` - Add status line component usage
- `/CLAUDE.md` - Update TUI implementation section
- Code comments in StatusLine component

#### Task 7.2: Remove Dead Code
**Files to Clean:**
- Remove any unused notification line code
- Remove duplicate status line logic
- Clean up imports

## Implementation Priority

### High Priority (Phase 1-3)
These tasks fix the immediate issues:
1. Enhanced StatusLine component with temporary messages
2. Dashboard view integration (fixes loading state issue)
3. Remove double status line problems

### Medium Priority (Phase 4-5)  
These tasks ensure consistency:
4. Standardize message handling across views
5. Update remaining views for consistency

### Low Priority (Phase 6-7)
These tasks ensure quality:
6. Comprehensive testing
7. Documentation updates

## Success Criteria

✅ **Single Status Line**: Only one status line visible at all times
✅ **Always Visible**: Status line shows in all views and states (including loading)
✅ **Consistent Styling**: Same colors, fonts, and layout across all views
✅ **Temporary Messages**: 3-second colored messages for feedback (URL opening, copying, errors)
✅ **View Indicators**: Clear view identification ([DASH], [DETAILS], etc.)
✅ **Contextual Help**: Appropriate keyboard shortcuts for current view/selection
✅ **No Flickering**: Smooth transitions and updates
✅ **Proper Layout**: Status line doesn't interfere with content area

## Key Behavioral Requirements

1. **Loading States**: Status line must remain visible with loading indicators
2. **Message Colors**: Green (success), Red (error), Blue (info), Yellow (warning)
3. **Message Duration**: 3 seconds for temporary messages
4. **Help Text**: Dynamic based on current selection/context
5. **View Persistence**: Status line state persists during view operations
6. **Keyboard Navigation**: Status line doesn't interfere with view navigation

## Technical Notes

- Use existing StatusLine component as foundation
- Maintain lipgloss styling consistency
- Preserve all current keyboard shortcuts and functionality
- Ensure backward compatibility with existing status line helpers
- Test thoroughly in different terminal sizes and conditions
- Consider performance impact of frequent status line updates

## Expected Files Modified

1. `/internal/tui/components/statusline.go` - Enhanced component
2. `/internal/tui/components/statusline_manager.go` - New manager (optional)
3. `/internal/tui/views/dashboard.go` - Major refactoring
4. `/internal/tui/views/create.go` - Minor updates
5. `/internal/tui/views/list.go` - Minor updates
6. `/internal/tui/views/details.go` - Minor updates
7. Documentation files - Updates for new patterns

This unified system will ensure consistent, always-visible status lines across all RepoBird CLI TUI views while preserving all existing functionality and improving user experience.
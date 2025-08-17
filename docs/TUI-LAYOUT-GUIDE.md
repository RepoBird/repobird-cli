# TUI Layout Guide - Lipgloss Best Practices

## Overview
This guide documents lessons learned and best practices for creating Terminal User Interface (TUI) layouts using the Bubble Tea framework and Lipgloss styling library. These patterns are derived from actual implementations in the RepoBird CLI codebase and ensure consistent, properly-aligned layouts without border cutoffs or spacing issues.

## Key Concepts

### 1. Border Rendering Overhead
**Critical Insight**: Lipgloss borders add pixels to the dimensions you specify.

```go
// WRONG: This will cause right-side cutoff
totalWidth := terminalWidth
leftBox := lipgloss.NewStyle().Width(totalWidth/2).Border(lipgloss.RoundedBorder())
rightBox := lipgloss.NewStyle().Width(totalWidth/2).Border(lipgloss.RoundedBorder())

// CORRECT: Account for border expansion
// Each box with border renders 2 chars wider than set width (1 left + 1 right border)
totalWidth := terminalWidth - 4  // Subtract 2 per box
leftBox := lipgloss.NewStyle().Width(totalWidth/2).Border(lipgloss.RoundedBorder())
rightBox := lipgloss.NewStyle().Width(totalWidth/2).Border(lipgloss.RoundedBorder())
```

### 2. Height Calculations for Full-Screen Views

Different views in the codebase use different height calculations based on their needs:

#### Standard View with Status Bar (create.go pattern)
```go
// Simple: just status bar
availableHeight := v.height - 1  // Reserve 1 line for status bar
```

#### Dashboard with Title and Status (dashboard.go pattern)
```go
// With title and status line
availableHeight := d.height - 3  // 2 for title + spacing, 1 for status
```

#### Dynamic Title Height (dashboard.go loading state)
```go
// When title height varies
titleHeight := lipgloss.Height(title)
statusLineHeight := 1
availableHeight := d.height - titleHeight - statusLineHeight
```

#### Top Border Visibility Fix
```go
// When top border gets cut off, reserve extra space
availableHeight := terminalHeight - 3  // 1 for status, 2 for top margin
boxHeight := availableHeight

// Then add margin when rendering:
contentWithMargin := lipgloss.NewStyle().
    MarginTop(2).  // Push content down to show top border
    Render(content)
```

### 3. Content Area vs Box Dimensions

Understanding the relationship between box size and usable content area:

```go
// Box dimensions include borders and padding
boxWidth := 50
boxHeight := 20

// Actual content area is smaller:
// - Borders take 2 from width and height
// - Padding(1) takes another 2 from each dimension
contentWidth := boxWidth - 4   // If using Border() and Padding(1)
contentHeight := boxHeight - 4

// When building content to fit:
content := buildContent(contentWidth, contentHeight)
box := lipgloss.NewStyle().
    Width(boxWidth).
    Height(boxHeight).
    Border(lipgloss.RoundedBorder()).
    Padding(1).
    Render(content)
```

### 4. Multi-Pane Layouts

For split-pane layouts (like file browsers with preview):

#### Triple Column Layout (dashboard.go Miller columns)
```go
// Each box renders 2 pixels wider, so subtract 2 per column
totalWidth := d.width - 6  // 3 columns * 2 border chars each
leftWidth := totalWidth / 3
centerWidth := totalWidth / 3
rightWidth := totalWidth - leftWidth - centerWidth  // Use remaining
```

#### Two-Pane Split (config_file_selector.go)
```go
// Account for border expansion
totalWidth := terminalWidth - 4  // 2 boxes * 2 border chars each

// Dynamic split with bounds
leftWidth := int(float64(totalWidth) * 0.4)  // 40% for list
if leftWidth < 30 {
    leftWidth = 30  // Minimum readable width
}
if leftWidth > 50 {
    leftWidth = 50  // Maximum for readability
}
rightWidth := totalWidth - leftWidth  // Remaining for preview

// Join WITHOUT gap for maximum space usage
splitView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
// NOT: JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane) - wastes space
```

### 5. Preventing Border Cutoffs

Top border cutoff is a common issue. Solutions:

```go
// Method 1: Add top margin to the content
content := lipgloss.NewStyle().
    MarginTop(1).  // or 2 if needed
    Render(boxes)

// Method 2: Reserve space in height calculation
availableHeight := terminalHeight - 3  // Extra space for top
boxHeight := availableHeight

// Method 3: For nested components, ensure parent accounts for child borders
panelHeight := availableHeight
contentHeight := panelHeight - 4  // Account for border (2) and padding (2)
```

## Common Patterns

### Panel Width Calculations (create.go pattern)

The create view uses a consistent pattern for panel sizing:

```go
// Width with some margin for cleaner look
panelWidth := v.width - 2
if panelWidth < 60 {
    panelWidth = 60  // Minimum width
}

// Content area accounting for border and padding
contentWidth := panelWidth - 4  // Border (2) + Padding (2)
contentHeight := panelHeight - 4

// Panel style with MaxHeight instead of Height for flexibility
panelStyle := lipgloss.NewStyle().
    Width(panelWidth).
    MaxHeight(panelHeight).  // Use MaxHeight for dynamic content
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("63")).
    Padding(1)
```

### Full-Screen Layout with Status Bar

```go
func (v *View) View() string {
    // Calculate dimensions
    availableHeight := v.height - 1  // Status bar
    
    // Build main content
    content := v.renderContent(v.width, availableHeight)
    
    // Build status bar
    statusBar := lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Background(lipgloss.Color("235")).
        Width(v.width).
        Padding(0, 1).
        Height(1).
        Render(statusText)
    
    // Combine
    return lipgloss.JoinVertical(
        lipgloss.Left,
        content,
        statusBar,
    )
}
```

### Responsive Split Panes

```go
func renderSplitPane(width, height int) string {
    // Account for border overhead
    totalWidth := width - 4  // 2 boxes * 2 border chars each
    
    // Dynamic split based on terminal size
    leftWidth := int(float64(totalWidth) * 0.4)
    if leftWidth < 30 {
        leftWidth = 30  // Minimum readable width
    }
    if leftWidth > 50 {
        leftWidth = 50  // Maximum for readability
    }
    rightWidth := totalWidth - leftWidth
    
    // Build panes with proper height
    boxHeight := height - 1  // Leave room for status/help text
    
    leftBox := styleWithBorder.
        Width(leftWidth).
        Height(boxHeight).
        Render(leftContent)
        
    rightBox := styleWithBorder.
        Width(rightWidth).
        Height(boxHeight).
        Render(rightContent)
    
    return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
}
```

### Line Truncation in Preview Panes

When displaying file content or preview text, always truncate long lines:

```go
func renderPreview(content string, maxWidth int) []string {
    lines := strings.Split(content, "\n")
    var result []string
    
    for _, line := range lines {
        // Replace tabs for consistent display
        line = strings.ReplaceAll(line, "\t", "    ")
        
        // Rune-safe truncation
        runes := []rune(line)
        if len(runes) > maxWidth {
            line = string(runes[:maxWidth-3]) + "..."
        }
        
        result = append(result, line)
    }
    
    return result
}
```

## Debugging Layout Issues

### Common Problems and Solutions

1. **Right border cut off**: Not accounting for border width overhead
   - Solution: Subtract 2 * number of boxes from total width

2. **Top border cut off**: Content starting at row 0 without margin
   - Solution: Add `MarginTop(1)` or `MarginTop(2)` to content

3. **Boxes don't align**: Inconsistent width calculations
   - Solution: Ensure all boxes account for same border/padding overhead

4. **Content overflows box**: Not accounting for border and padding
   - Solution: Content dimensions = box dimensions - 4 (for border + padding)

5. **Gaps between panes too large**: Using space separator in JoinHorizontal
   - Solution: Remove separator or use empty string

### Centering Content (Place Functions)

Lipgloss provides Place functions for centering:

```go
// Center a box on screen (dashboard.go status info)
centeredBox := lipgloss.Place(
    d.width, 
    d.height-1,  // Leave room for status line
    lipgloss.Center, 
    lipgloss.Center, 
    boxedContent,
)

// Constrain overflowing content (dashboard.go columns)
if finalWidth > d.width {
    columns = lipgloss.PlaceHorizontal(d.width, lipgloss.Left, columns)
}
```

### Dimension Calculation Checklist

When creating a new TUI component:

- [ ] Account for status bar height (usually -1 from terminal height)
- [ ] Account for title if present (check actual height with `lipgloss.Height()`)
- [ ] Account for border width overhead (2 chars per box with border)
- [ ] Account for padding in content calculations (2 per dimension with `Padding(1)`)
- [ ] Add top margin if borders are cut off (use `MarginTop(1)` or `MarginTop(2)`)
- [ ] Use `MaxHeight` instead of `Height` for flexible content
- [ ] Test with minimum terminal size (80x24)
- [ ] Test with very large terminal sizes
- [ ] Ensure responsive width calculations have min/max bounds
- [ ] Join panes without gaps unless spacing is intentional

## Example: Config File Selector Implementation

Here's a complete example showing all these principles:

```go
func (cfs *ConfigFileSelector) View() string {
    // 1. Calculate available space
    availableHeight := cfs.height - 3  // Status bar + top margin
    boxHeight := availableHeight
    
    // 2. Account for border overhead in width
    totalWidth := cfs.width - 4  // 2 boxes with borders
    
    // 3. Split width with bounds
    listWidth := int(float64(totalWidth) * 0.4)
    if listWidth < 30 {
        listWidth = 30
    }
    if listWidth > 50 {
        listWidth = 50
    }
    previewWidth := totalWidth - listWidth
    
    // 4. Calculate content dimensions
    contentHeight := boxHeight - 2  // Border overhead
    listContentHeight := contentHeight - 3  // Header + filter + spacing
    previewContentHeight := contentHeight - 2  // Header + spacing
    
    // 5. Build content (with truncation for preview)
    // ... build list and preview content ...
    
    // 6. Create boxes
    listBox := borderStyle.
        Width(listWidth).
        Height(boxHeight).
        Render(listContent)
        
    previewBox := borderStyle.
        Width(previewWidth).
        Height(boxHeight).
        Render(previewContent)
    
    // 7. Join without gap
    splitView := lipgloss.JoinHorizontal(lipgloss.Top, listBox, previewBox)
    
    // 8. Add margin to prevent top cutoff
    contentWithMargin := lipgloss.NewStyle().
        MarginTop(2).
        Render(splitView)
    
    // 9. Add status bar
    statusBar := lipgloss.NewStyle().
        Width(cfs.width).
        Background(lipgloss.Color("235")).
        Padding(0, 1).
        Render(statusText)
    
    // 10. Combine everything
    return lipgloss.JoinVertical(
        lipgloss.Left,
        contentWithMargin,
        statusBar,
    )
}
```

## Viewport Scrolling with Scrollbar

### Using Bubbles Viewport for Scrollable Content

The viewport component from Bubbles provides smooth scrolling for content that exceeds the terminal height:

```go
import "github.com/charmbracelet/bubbles/viewport"

// Initialize viewport
vp := viewport.New(width, height)
vp.SetContent(yourMultiLineContent)

// Handle keyboard events for scrolling
switch msg.String() {
case "j", "down":
    vp.ScrollDown(1)
case "k", "up":
    vp.ScrollUp(1)
case "ctrl+d":
    vp.HalfPageDown()
case "ctrl+u":
    vp.HalfPageUp()
case "g", "home":
    vp.GotoTop()
case "G", "end":
    vp.GotoBottom()
}
```

### Adding a Custom Scrollbar

Implementing a scrollbar outside the main content box provides visual feedback for scroll position:

#### Key Implementation Points

1. **Render scrollbar as separate component**: Don't try to embed it within the viewport content
2. **Use NormalBorder for proper connection**: RoundedBorder corners (`╭╮╰╯`) don't connect to vertical bars (`│`)
3. **Match heights exactly**: Scrollbar height should match the main content box height

#### Complete Scrollbar Implementation

```go
// Build scrollbar lines matching the box height
func buildScrollbarLines(totalHeight int, viewport viewport.Model, contentLines []string) []string {
    // Use total height without subtracting for borders
    innerHeight := totalHeight
    
    if innerHeight <= 0 {
        return []string{}
    }
    
    // Calculate thumb size and position
    totalLines := len(contentLines)
    visibleLines := viewport.Height
    
    thumbSize := max(1, (visibleLines * innerHeight) / totalLines)
    if thumbSize > innerHeight {
        thumbSize = innerHeight
    }
    
    percentScrolled := viewport.ScrollPercent()
    maxThumbPos := innerHeight - thumbSize
    thumbPos := int(float64(maxThumbPos) * percentScrolled)
    
    // Build scrollbar characters
    var lines []string
    for i := 0; i < innerHeight; i++ {
        if i >= thumbPos && i < thumbPos+thumbSize {
            // Thumb - highlighted
            lines = append(lines, lipgloss.NewStyle().
                Foreground(lipgloss.Color("63")).
                Render("█"))
        } else {
            // Track - dimmed
            lines = append(lines, lipgloss.NewStyle().
                Foreground(lipgloss.Color("238")).
                Render("│"))
        }
    }
    
    return lines
}

// Render with scrollbar
func renderWithScrollbar(viewportContent string, scrollbarNeeded bool) string {
    // Main content box
    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("63")).
        Padding(0, 1).
        Width(width - 4).  // Leave room for scrollbar
        Height(height - 3)
    
    boxedContent := boxStyle.Render(viewportContent)
    
    if !scrollbarNeeded {
        return boxedContent
    }
    
    // Build scrollbar
    boxHeight := height - 3
    scrollbarLines := buildScrollbarLines(boxHeight, viewport, contentLines)
    scrollbarContent := strings.Join(scrollbarLines, "\n")
    
    // Wrap scrollbar in border (use NormalBorder for proper connection)
    scrollbarStyle := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).  // Critical: Use NormalBorder
        BorderForeground(lipgloss.Color("238")).
        BorderTop(true).
        BorderBottom(true).
        BorderLeft(false).
        BorderRight(false)
    
    scrollbarBox := scrollbarStyle.Render(scrollbarContent)
    
    // Join horizontally
    return lipgloss.JoinHorizontal(lipgloss.Top, boxedContent, scrollbarBox)
}
```

### Border Type Compatibility

**Important**: Border corners must visually connect to your content characters:

| Border Type | Corner Chars | Compatible With | Use Case |
|------------|--------------|-----------------|----------|
| RoundedBorder | `╭╮╰╯` | Spaces only | Standalone boxes |
| NormalBorder | `┌┐└┘` | `│─` chars | Scrollbars, connected elements |
| DoubleBorder | `╔╗╚╝` | `║═` chars | Heavy emphasis |
| ThickBorder | `┏┓┗┛` | `┃━` chars | Bold styling |

### Debugging Tips

1. **Use debug logging**: Write to `/tmp/repobird_debug.log` using `debug.LogToFilef()`
2. **Snapshot rendered output**: Add a debug key to copy the entire rendered view to clipboard
3. **Check height calculations**: Log viewport.Height, box height, and scrollbar line count
4. **Verify character alignment**: Ensure border type matches content characters

### Common Scrollbar Issues and Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| Gap between border and scrollbar | Using RoundedBorder with vertical bars | Use NormalBorder instead |
| Scrollbar too short | Subtracting border height incorrectly | Use full height without subtraction |
| Scrollbar misaligned | lipgloss.JoinHorizontal handling | Ensure equal heights for both components |
| Thumb size incorrect | Wrong total lines calculation | Use actual content line count |

## References

- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
- [Bubble Tea Framework](https://github.com/charmbracelet/bubbletea)
- [Bubbles Viewport Component](https://github.com/charmbracelet/bubbles/tree/master/viewport)
- Dashboard implementation: `/internal/tui/views/dashboard.go`
- Create view layouts: `/internal/tui/views/create.go`
- Help view with scrollbar: `/internal/tui/components/help_view.go`
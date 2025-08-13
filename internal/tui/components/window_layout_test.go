package components

import (
	"testing"
)

func TestNewWindowLayout(t *testing.T) {
	tests := []struct {
		name           string
		terminalWidth  int
		terminalHeight int
		wantBoxWidth   int
		wantBoxHeight  int
	}{
		{
			name:           "Standard terminal",
			terminalWidth:  80,
			terminalHeight: 24,
			wantBoxWidth:   78, // 80 - 2 (border margin)
			wantBoxHeight:  21, // 24 - 1 (statusline) - 2 (top margin)
		},
		{
			name:           "Large terminal",
			terminalWidth:  120,
			terminalHeight: 40,
			wantBoxWidth:   118, // 120 - 2
			wantBoxHeight:  37,  // 40 - 1 - 2
		},
		{
			name:           "Small terminal",
			terminalWidth:  40,
			terminalHeight: 10,
			wantBoxWidth:   38, // 40 - 2
			wantBoxHeight:  7,  // 10 - 1 - 2
		},
		{
			name:           "Minimum viable terminal",
			terminalWidth:  20,
			terminalHeight: 5,
			wantBoxWidth:   18, // 20 - 2
			wantBoxHeight:  3,  // 5 - 1 - 2 - 1 = 1, but enforced minimum is 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := NewWindowLayout(tt.terminalWidth, tt.terminalHeight)

			// Test terminal dimensions are stored correctly
			if layout.terminalWidth != tt.terminalWidth {
				t.Errorf("terminalWidth = %d, want %d", layout.terminalWidth, tt.terminalWidth)
			}
			if layout.terminalHeight != tt.terminalHeight {
				t.Errorf("terminalHeight = %d, want %d", layout.terminalHeight, tt.terminalHeight)
			}

			// Test calculated box dimensions
			boxWidth, boxHeight := layout.GetBoxDimensions()
			if boxWidth != tt.wantBoxWidth {
				t.Errorf("GetBoxDimensions() width = %d, want %d", boxWidth, tt.wantBoxWidth)
			}
			if boxHeight != tt.wantBoxHeight {
				t.Errorf("GetBoxDimensions() height = %d, want %d", boxHeight, tt.wantBoxHeight)
			}
		})
	}
}

func TestWindowLayout_GetContentDimensions(t *testing.T) {
	tests := []struct {
		name              string
		terminalWidth     int
		terminalHeight    int
		wantContentWidth  int
		wantContentHeight int
	}{
		{
			name:              "Standard terminal",
			terminalWidth:     80,
			terminalHeight:    24,
			wantContentWidth:  74, // 78 - 4 (border + padding)
			wantContentHeight: 18, // 21 - 3 (title + borders/padding)
		},
		{
			name:              "Large terminal",
			terminalWidth:     120,
			terminalHeight:    40,
			wantContentWidth:  114, // 118 - 4
			wantContentHeight: 34,  // 37 - 3
		},
		{
			name:              "Small terminal with calculated content",
			terminalWidth:     20,
			terminalHeight:    5,
			wantContentWidth:  14, // 18 (box) - 4 (border+padding) = 14
			wantContentHeight: 1,  // Minimum enforced (1 - 3 = -2, so minimum 1)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := NewWindowLayout(tt.terminalWidth, tt.terminalHeight)

			contentWidth, contentHeight := layout.GetContentDimensions()
			if contentWidth != tt.wantContentWidth {
				t.Errorf("GetContentDimensions() width = %d, want %d", contentWidth, tt.wantContentWidth)
			}
			if contentHeight != tt.wantContentHeight {
				t.Errorf("GetContentDimensions() height = %d, want %d", contentHeight, tt.wantContentHeight)
			}
		})
	}
}

func TestWindowLayout_GetViewportDimensions(t *testing.T) {
	// Viewport dimensions should be the same as content dimensions
	layout := NewWindowLayout(80, 24)
	
	contentWidth, contentHeight := layout.GetContentDimensions()
	viewportWidth, viewportHeight := layout.GetViewportDimensions()

	if viewportWidth != contentWidth {
		t.Errorf("GetViewportDimensions() width = %d, want %d (same as content)", viewportWidth, contentWidth)
	}
	if viewportHeight != contentHeight {
		t.Errorf("GetViewportDimensions() height = %d, want %d (same as content)", viewportHeight, contentHeight)
	}
}

func TestWindowLayout_Update(t *testing.T) {
	layout := NewWindowLayout(80, 24)

	// Get initial dimensions
	initialBoxWidth, initialBoxHeight := layout.GetBoxDimensions()

	// Update to new dimensions
	newWidth, newHeight := 100, 30
	layout.Update(newWidth, newHeight)

	// Check that dimensions are updated
	if layout.terminalWidth != newWidth {
		t.Errorf("After Update, terminalWidth = %d, want %d", layout.terminalWidth, newWidth)
	}
	if layout.terminalHeight != newHeight {
		t.Errorf("After Update, terminalHeight = %d, want %d", layout.terminalHeight, newHeight)
	}

	// Check that calculated dimensions changed
	newBoxWidth, newBoxHeight := layout.GetBoxDimensions()
	if newBoxWidth == initialBoxWidth {
		t.Errorf("Box width did not update after Update()")
	}
	if newBoxHeight == initialBoxHeight {
		t.Errorf("Box height did not update after Update()")
	}

	// Verify new calculations are correct
	expectedBoxWidth := newWidth - 2  // border margin
	expectedBoxHeight := newHeight - 3 // statusline + top margin
	if newBoxWidth != expectedBoxWidth {
		t.Errorf("After Update, box width = %d, want %d", newBoxWidth, expectedBoxWidth)
	}
	if newBoxHeight != expectedBoxHeight {
		t.Errorf("After Update, box height = %d, want %d", newBoxHeight, expectedBoxHeight)
	}
}

func TestWindowLayout_IsValidDimensions(t *testing.T) {
	tests := []struct {
		name           string
		terminalWidth  int
		terminalHeight int
		wantValid      bool
	}{
		{
			name:           "Standard terminal is valid",
			terminalWidth:  80,
			terminalHeight: 24,
			wantValid:      true,
		},
		{
			name:           "Minimum valid terminal",
			terminalWidth:  20,
			terminalHeight: 5,
			wantValid:      true,
		},
		{
			name:           "Too narrow",
			terminalWidth:  19,
			terminalHeight: 24,
			wantValid:      false,
		},
		{
			name:           "Too short",
			terminalWidth:  80,
			terminalHeight: 4,
			wantValid:      false,
		},
		{
			name:           "Both too small",
			terminalWidth:  10,
			terminalHeight: 3,
			wantValid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := NewWindowLayout(tt.terminalWidth, tt.terminalHeight)
			if got := layout.IsValidDimensions(); got != tt.wantValid {
				t.Errorf("IsValidDimensions() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestWindowLayout_GetMinimalView(t *testing.T) {
	tests := []struct {
		name           string
		terminalWidth  int
		message        string
		expectedOutput string
	}{
		{
			name:           "Normal message",
			terminalWidth:  80,
			message:        "Loading...",
			expectedOutput: "Loading...",
		},
		{
			name:           "Message too long for terminal",
			terminalWidth:  10,
			message:        "This is a very long message",
			expectedOutput: "This is ",
		},
		{
			name:           "Very small terminal",
			terminalWidth:  5,
			message:        "Loading",
			expectedOutput: "Loa",
		},
		{
			name:           "Empty message",
			terminalWidth:  80,
			message:        "",
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := NewWindowLayout(tt.terminalWidth, 24)
			got := layout.GetMinimalView(tt.message)
			if got != tt.expectedOutput {
				t.Errorf("GetMinimalView() = %q, want %q", got, tt.expectedOutput)
			}
		})
	}
}

func TestWindowLayout_CreateStandardBox(t *testing.T) {
	layout := NewWindowLayout(80, 24)
	boxStyle := layout.CreateStandardBox()

	// Test that the box has the correct dimensions
	boxWidth, boxHeight := layout.GetBoxDimensions()
	
	// Note: We can't directly compare lipgloss styles, but we can verify 
	// the style has the expected dimensions
	if boxStyle.GetWidth() == 0 && boxStyle.GetHeight() == 0 {
		t.Error("CreateStandardBox() returned style with zero dimensions")
	}

	// Test that the method is consistent
	boxStyle2 := layout.CreateStandardBox()
	if boxStyle.GetWidth() != boxStyle2.GetWidth() {
		t.Error("CreateStandardBox() returns inconsistent styles")
	}

	// Verify dimensions match what we expect
	if boxStyle.GetWidth() != boxWidth {
		t.Errorf("Box style width = %d, want %d", boxStyle.GetWidth(), boxWidth)
	}
	if boxStyle.GetHeight() != boxHeight {
		t.Errorf("Box style height = %d, want %d", boxStyle.GetHeight(), boxHeight)
	}
}

func TestWindowLayout_CreateTitleStyle(t *testing.T) {
	layout := NewWindowLayout(80, 24)
	titleStyle := layout.CreateTitleStyle()

	// Verify style was created with non-zero dimensions
	if titleStyle.GetWidth() == 0 {
		t.Error("CreateTitleStyle() returned style with zero width")
	}

	// Test that the method is consistent
	titleStyle2 := layout.CreateTitleStyle()
	if titleStyle.GetWidth() != titleStyle2.GetWidth() {
		t.Error("CreateTitleStyle() returns inconsistent styles")
	}

	// Title should be box width - 2 (for border)
	boxWidth, _ := layout.GetBoxDimensions()
	expectedTitleWidth := boxWidth - 2
	if titleStyle.GetWidth() != expectedTitleWidth {
		t.Errorf("Title style width = %d, want %d", titleStyle.GetWidth(), expectedTitleWidth)
	}
}

func TestWindowLayout_CreateContentStyle(t *testing.T) {
	layout := NewWindowLayout(80, 24)
	contentStyle := layout.CreateContentStyle()

	// Verify style was created with non-zero dimensions
	if contentStyle.GetWidth() == 0 && contentStyle.GetHeight() == 0 {
		t.Error("CreateContentStyle() returned style with zero dimensions")
	}

	// Test that the method is consistent
	contentStyle2 := layout.CreateContentStyle()
	if contentStyle.GetWidth() != contentStyle2.GetWidth() {
		t.Error("CreateContentStyle() returns inconsistent styles")
	}

	// Content should match the content height from layout
	_, contentHeight := layout.GetContentDimensions()
	if contentStyle.GetHeight() != contentHeight {
		t.Errorf("Content style height = %d, want %d", contentStyle.GetHeight(), contentHeight)
	}
}

func TestWindowLayout_MinimumDimensions(t *testing.T) {
	// Test that minimum dimensions are enforced
	layout := NewWindowLayout(5, 2) // Very small terminal

	boxWidth, boxHeight := layout.GetBoxDimensions()
	if boxWidth < 10 {
		t.Errorf("Box width %d should be at least 10 (minimum enforced)", boxWidth)
	}
	if boxHeight < 3 {
		t.Errorf("Box height %d should be at least 3 (minimum enforced)", boxHeight)
	}

	contentWidth, contentHeight := layout.GetContentDimensions()
	if contentWidth < 5 {
		t.Errorf("Content width %d should be at least 5 (minimum enforced)", contentWidth)
	}
	if contentHeight < 1 {
		t.Errorf("Content height %d should be at least 1 (minimum enforced)", contentHeight)
	}
}

func TestWindowLayout_GetLayoutForType(t *testing.T) {
	layout := NewWindowLayout(80, 24)

	tests := []struct {
		name       string
		layoutType LayoutType
	}{
		{"Standard layout", LayoutStandard},
		{"Dashboard layout", LayoutDashboard},
		{"Split layout", LayoutSplit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultLayout := layout.GetLayoutForType(tt.layoutType)
			
			// For now, all layout types return the same layout
			// This test ensures the method works and can be extended
			if resultLayout == nil {
				t.Errorf("GetLayoutForType(%v) returned nil", tt.layoutType)
			}
			
			// Should return the same layout object for now
			if resultLayout != layout {
				t.Errorf("GetLayoutForType(%v) should return same layout for now", tt.layoutType)
			}
		})
	}
}

func TestWindowLayout_BorderCalculations(t *testing.T) {
	// Test that the layout correctly accounts for lipgloss border expansion
	terminalWidth := 80
	layout := NewWindowLayout(terminalWidth, 24)

	boxWidth, _ := layout.GetBoxDimensions()
	
	// Box should be 2 pixels narrower than terminal (lipgloss expansion)
	expectedBoxWidth := terminalWidth - 2
	if boxWidth != expectedBoxWidth {
		t.Errorf("Box width = %d, want %d (terminal %d - 2 for lipgloss expansion)", 
			boxWidth, expectedBoxWidth, terminalWidth)
	}

	// Content should be 4 pixels narrower than box (border + padding)
	contentWidth, _ := layout.GetContentDimensions()
	expectedContentWidth := boxWidth - 4
	if contentWidth != expectedContentWidth {
		t.Errorf("Content width = %d, want %d (box %d - 4 for border+padding)", 
			contentWidth, expectedContentWidth, boxWidth)
	}
}

func TestWindowLayout_HeightCalculations(t *testing.T) {
	// Test that height calculations account for statusline and margins
	terminalHeight := 24
	layout := NewWindowLayout(80, terminalHeight)

	boxHeight := layout.boxHeight
	
	// Box height should account for statusline (1) + top margin (2)
	expectedBoxHeight := terminalHeight - 1 - 2
	if boxHeight != expectedBoxHeight {
		t.Errorf("Box height = %d, want %d (terminal %d - 3 for statusline+margins)", 
			boxHeight, expectedBoxHeight, terminalHeight)
	}

	// Content height should be box height minus title and padding
	_, contentHeight := layout.GetContentDimensions()
	expectedContentHeight := boxHeight - 3
	if contentHeight != expectedContentHeight {
		t.Errorf("Content height = %d, want %d (box %d - 3 for title+padding)", 
			contentHeight, expectedContentHeight, boxHeight)
	}
}
package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestColorPalette(t *testing.T) {
	tests := []struct {
		name  string
		color lipgloss.AdaptiveColor
	}{
		{"ColorPrimary", ColorPrimary},
		{"ColorSuccess", ColorSuccess},
		{"ColorWarning", ColorWarning},
		{"ColorError", ColorError},
		{"ColorInfo", ColorInfo},
		{"ColorInProgress", ColorInProgress},
		{"ColorComplete", ColorComplete},
		{"ColorFailed", ColorFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that the color has both light and dark variants
			if tt.color.Light == "" {
				t.Errorf("%s: Light color variant is empty", tt.name)
			}
			if tt.color.Dark == "" {
				t.Errorf("%s: Dark color variant is empty", tt.name)
			}
		})
	}
}

func TestTextStyles(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
		text  string
	}{
		{"StyleHeading", StyleHeading, "Test Heading"},
		{"StyleNormal", StyleNormal, "Normal text"},
		{"StyleStatus", StyleStatus, "Status message"},
		{"StyleError", StyleError, "Error message"},
		{"StyleWarning", StyleWarning, "Warning message"},
		{"StyleSuccess", StyleSuccess, "Success message"},
		{"StyleSubtle", StyleSubtle, "Subtle text"},
		{"StyleHighlight", StyleHighlight, "Highlighted text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that the style can be rendered
			rendered := tt.style.Render(tt.text)
			if rendered == "" {
				t.Errorf("%s: Rendered output is empty", tt.name)
			}
		})
	}
}

func TestBorderStyles(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"StyleBox", StyleBox},
		{"StyleBoxSuccess", StyleBoxSuccess},
		{"StyleBoxError", StyleBoxError},
		{"StyleBoxWarning", StyleBoxWarning},
		{"StyleBoxInfo", StyleBoxInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that the box style can be rendered
			rendered := tt.style.Render("Test content")
			if rendered == "" {
				t.Errorf("%s: Rendered output is empty", tt.name)
			}
			// Verify the style has a border
			if !tt.style.GetBorderTop() && !tt.style.GetBorderBottom() &&
				!tt.style.GetBorderLeft() && !tt.style.GetBorderRight() {
				t.Errorf("%s: No border defined", tt.name)
			}
		})
	}
}

func TestProgressBarStyles(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"StyleProgressBar", StyleProgressBar},
		{"StyleProgressBarComplete", StyleProgressBarComplete},
		{"StyleProgressBarError", StyleProgressBarError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that the progress bar style can be rendered
			rendered := tt.style.Render("████████░░")
			if rendered == "" {
				t.Errorf("%s: Rendered output is empty", tt.name)
			}
		})
	}
}

func TestIcons(t *testing.T) {
	icons := map[string]string{
		"IconCheckmark": IconCheckmark,
		"IconCross":     IconCross,
		"IconWarning":   IconWarning,
		"IconInfo":      IconInfo,
		"IconSpinner":   IconSpinner,
		"IconArrow":     IconArrow,
	}

	for name, icon := range icons {
		t.Run(name, func(t *testing.T) {
			if icon == "" {
				t.Errorf("%s is empty", name)
			}
		})
	}
}

func TestASCIIIcons(t *testing.T) {
	icons := map[string]string{
		"IconCheckmarkASCII": IconCheckmarkASCII,
		"IconCrossASCII":     IconCrossASCII,
		"IconWarningASCII":   IconWarningASCII,
		"IconInfoASCII":      IconInfoASCII,
		"IconSpinnerASCII":   IconSpinnerASCII,
		"IconArrowASCII":     IconArrowASCII,
	}

	for name, icon := range icons {
		t.Run(name, func(t *testing.T) {
			if icon == "" {
				t.Errorf("%s is empty", name)
			}
		})
	}
}

func TestSemanticColorMapping(t *testing.T) {
	// Verify semantic mappings are correct
	if ColorInProgress != ColorInfo {
		t.Error("ColorInProgress should map to ColorInfo (blue)")
	}
	if ColorComplete != ColorSuccess {
		t.Error("ColorComplete should map to ColorSuccess (green)")
	}
	if ColorFailed != ColorError {
		t.Error("ColorFailed should map to ColorError (red)")
	}
}

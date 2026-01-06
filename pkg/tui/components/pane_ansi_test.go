package components_test

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/tui/components"
	"github.com/charmbracelet/x/ansi"
)

func TestPane_View_TruncationIsANSIAndWidthAware(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Test", ShortcutKey: "1"})
	pane.SetSize(20, 4)

	// Pane content width is pane width minus borders/padding: 20 - 4 = 16.
	// Provide a styled string with wide runes that exceeds 16 cells so truncation occurs.
	content := "\x1b[31m你好你好你好你好你\x1b[0m" // 9 wide runes => 18 cells
	view := pane.View(content)

	// Ensure ANSI sequences aren't broken such that stripping still leaves ESC bytes behind.
	if strings.Contains(ansi.Strip(view), "\x1b") {
		t.Fatalf("expected stripped output to contain no ESC bytes")
	}

	// Every line should render to the pane width in terminal cells.
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		if w := ansi.StringWidth(line); w != 20 {
			t.Fatalf("expected line %d to have display width 20, got %d: %q", i, w, ansi.Strip(line))
		}
	}

	// Truncation uses an ellipsis when width allows.
	if !strings.Contains(ansi.Strip(view), "...") {
		t.Fatalf("expected output to contain ellipsis when truncating")
	}
}

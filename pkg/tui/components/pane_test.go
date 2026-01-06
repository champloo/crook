package components_test

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/tui/components"
)

func TestPane_NewPane(t *testing.T) {
	config := components.PaneConfig{
		Title:       "Nodes",
		ShortcutKey: "1",
	}

	pane := components.NewPane(config)

	if pane == nil {
		t.Fatal("NewPane returned nil")
	}

	if pane.GetTitle() != "Nodes" {
		t.Errorf("expected title 'Nodes', got %q", pane.GetTitle())
	}

	if pane.GetShortcutKey() != "1" {
		t.Errorf("expected shortcut key '1', got %q", pane.GetShortcutKey())
	}

	if pane.IsActive() {
		t.Error("new pane should not be active by default")
	}

	if pane.GetBadge() != "" {
		t.Errorf("new pane should have empty badge, got %q", pane.GetBadge())
	}
}

func TestPane_SetActive(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Test"})

	// Initially inactive
	if pane.IsActive() {
		t.Error("pane should be inactive initially")
	}

	// Set active
	pane.SetActive(true)
	if !pane.IsActive() {
		t.Error("pane should be active after SetActive(true)")
	}

	// Set inactive
	pane.SetActive(false)
	if pane.IsActive() {
		t.Error("pane should be inactive after SetActive(false)")
	}
}

func TestPane_SetBadge(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Nodes"})

	// Set simple count badge
	pane.SetBadge("6")
	if pane.GetBadge() != "6" {
		t.Errorf("expected badge '6', got %q", pane.GetBadge())
	}

	// Set filtered/total badge
	pane.SetBadge("12/15")
	if pane.GetBadge() != "12/15" {
		t.Errorf("expected badge '12/15', got %q", pane.GetBadge())
	}

	// Clear badge
	pane.SetBadge("")
	if pane.GetBadge() != "" {
		t.Errorf("expected empty badge, got %q", pane.GetBadge())
	}
}

func TestPane_SetSize(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Test"})

	pane.SetSize(100, 20)
	width, height := pane.GetSize()

	if width != 100 {
		t.Errorf("expected width 100, got %d", width)
	}
	if height != 20 {
		t.Errorf("expected height 20, got %d", height)
	}
}

func TestPane_SetTitle(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Deployments"})

	if pane.GetTitle() != "Deployments" {
		t.Errorf("expected title 'Deployments', got %q", pane.GetTitle())
	}

	pane.SetTitle("Pods")

	if pane.GetTitle() != "Pods" {
		t.Errorf("expected title 'Pods' after SetTitle, got %q", pane.GetTitle())
	}
}

func TestPane_View_Active(t *testing.T) {
	config := components.PaneConfig{
		Title:       "Nodes",
		ShortcutKey: "1",
	}
	pane := components.NewPane(config)
	pane.SetActive(true)
	pane.SetBadge("6")
	pane.SetSize(40, 10)

	view := pane.View("Test content")

	// Should contain title with shortcut and badge in new format: [1] Nodes (6)
	if !strings.Contains(view, "[1] Nodes") {
		t.Error("view should contain '[1] Nodes'")
	}
	if !strings.Contains(view, "(6)") {
		t.Error("view should contain badge '(6)'")
	}
	if !strings.Contains(view, "Test content") {
		t.Error("view should contain 'Test content'")
	}
}

func TestPane_View_Inactive(t *testing.T) {
	config := components.PaneConfig{
		Title:       "Deployments",
		ShortcutKey: "2",
	}
	pane := components.NewPane(config)
	pane.SetActive(false)
	pane.SetBadge("12")
	pane.SetSize(40, 10)

	view := pane.View("Deployment list")

	// Should contain title with shortcut and badge in new format: [2] Deployments (12)
	if !strings.Contains(view, "[2] Deployments") {
		t.Error("view should contain '[2] Deployments'")
	}
	if !strings.Contains(view, "(12)") {
		t.Error("view should contain badge '(12)'")
	}
	if !strings.Contains(view, "Deployment list") {
		t.Error("view should contain 'Deployment list'")
	}
}

func TestPane_View_WithoutShortcutKey(t *testing.T) {
	config := components.PaneConfig{
		Title: "OSDs",
		// No shortcut key
	}
	pane := components.NewPane(config)
	pane.SetSize(40, 10)

	view := pane.View("OSD list")

	// Should contain title without shortcut prefix
	if !strings.Contains(view, "OSDs") {
		t.Error("view should contain 'OSDs'")
	}
	// Should NOT contain bracket prefix pattern when no shortcut key
	if strings.Contains(view, "[") && strings.Contains(view, "] OSDs") {
		t.Error("view should not contain '[X] OSDs' when no shortcut key")
	}
}

func TestPane_View_WithoutBadge(t *testing.T) {
	config := components.PaneConfig{
		Title:       "Nodes",
		ShortcutKey: "1",
	}
	pane := components.NewPane(config)
	pane.SetSize(40, 10)
	// No badge set

	view := pane.View("Content")

	// Should contain title in new format: [1] Nodes
	if !strings.Contains(view, "[1] Nodes") {
		t.Error("view should contain '[1] Nodes'")
	}
	// Should NOT contain parentheses (no badge)
	// Note: The border uses box-drawing characters, but badge would show as (X)
}

func TestPane_View_ContentClipping(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Test"})
	pane.SetSize(20, 5)

	// Content with many lines
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8"

	view := pane.View(content)

	// View should be rendered (we can't easily check exact clipping due to borders)
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestPane_View_EmptyContent(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Empty"})
	pane.SetSize(30, 5)

	view := pane.View("")

	// Should still render with title and border
	if !strings.Contains(view, "Empty") {
		t.Error("view should contain title 'Empty'")
	}
}

func TestPane_View_MinimalSize(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Small"})
	pane.SetSize(10, 3) // Very small

	view := pane.View("Content")

	// Should not panic, should produce some output
	if view == "" {
		t.Error("view should not be empty even with minimal size")
	}
}

func TestPane_View_ZeroSize(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Zero", ShortcutKey: "1"})
	pane.SetBadge("5")
	pane.SetSize(0, 0) // Zero size - should not panic

	// Should not panic
	view := pane.View("Content")

	// Should produce some output (with minimum safeguards applied)
	if view == "" {
		t.Error("view should not be empty even with zero size")
	}
}

func TestPane_View_NegativeSize(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{Title: "Negative", ShortcutKey: "2"})
	pane.SetSize(-10, -5) // Negative size - should not panic

	// Should not panic
	view := pane.View("Content")

	// Should produce some output (with minimum safeguards applied)
	if view == "" {
		t.Error("view should not be empty even with negative size")
	}
}

func TestPane_FilteredTotalBadge(t *testing.T) {
	pane := components.NewPane(components.PaneConfig{
		Title:       "Deployments",
		ShortcutKey: "2",
	})
	pane.SetBadge("5/10")
	pane.SetSize(50, 10)

	view := pane.View("Filtered content")

	if !strings.Contains(view, "(5/10)") {
		t.Error("view should contain filtered/total badge '(5/10)'")
	}
}

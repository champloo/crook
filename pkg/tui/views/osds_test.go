package views

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/k8s"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewOSDsView(t *testing.T) {
	v := NewOSDsView()

	if v == nil {
		t.Fatal("NewOSDsView() returned nil")
	}

	if v.cursor != 0 {
		t.Errorf("cursor = %d, want 0", v.cursor)
	}

	if len(v.osds) != 0 {
		t.Errorf("osds len = %d, want 0", len(v.osds))
	}

	if v.nooutSet {
		t.Error("nooutSet should be false by default")
	}
}

func TestOSDsView_SetOSDs(t *testing.T) {
	v := NewOSDsView()

	osds := []k8s.OSDInfo{
		{ID: 0, Name: "osd.0", Status: "up", InOut: "in", Hostname: "worker-1"},
		{ID: 1, Name: "osd.1", Status: "up", InOut: "in", Hostname: "worker-1"},
		{ID: 2, Name: "osd.2", Status: "down", InOut: "out", Hostname: "worker-2"},
	}

	v.SetOSDs(osds)

	if len(v.osds) != 3 {
		t.Errorf("osds len = %d, want 3", len(v.osds))
	}

	if v.Count() != 3 {
		t.Errorf("Count() = %d, want 3", v.Count())
	}
}

func TestOSDsView_CursorNavigation(t *testing.T) {
	v := NewOSDsView()
	v.SetSize(100, 50)

	osds := []k8s.OSDInfo{
		{ID: 0, Name: "osd.0"},
		{ID: 1, Name: "osd.1"},
		{ID: 2, Name: "osd.2"},
	}
	v.SetOSDs(osds)

	// Test j/down key
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if v.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", v.cursor)
	}

	// Test k/up key
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if v.cursor != 0 {
		t.Errorf("cursor after 'k' = %d, want 0", v.cursor)
	}

	// Test G (go to end)
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if v.cursor != 2 {
		t.Errorf("cursor after 'G' = %d, want 2", v.cursor)
	}

	// Test g (go to start)
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if v.cursor != 0 {
		t.Errorf("cursor after 'g' = %d, want 0", v.cursor)
	}
}

func TestOSDsView_Enter(t *testing.T) {
	v := NewOSDsView()

	osds := []k8s.OSDInfo{
		{ID: 0, Name: "osd.0", Status: "up"},
		{ID: 1, Name: "osd.1", Status: "up"},
	}
	v.SetOSDs(osds)

	// Press enter on first OSD
	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("enter key should return a command")
	}

	msg := cmd()
	selectedMsg, ok := msg.(OSDSelectedMsg)
	if !ok {
		t.Fatalf("expected OSDSelectedMsg, got %T", msg)
	}

	if selectedMsg.OSD.Name != "osd.0" {
		t.Errorf("selected OSD = %s, want osd.0", selectedMsg.OSD.Name)
	}
}

func TestOSDsView_View(t *testing.T) {
	v := NewOSDsView()
	v.SetSize(120, 30)

	osds := []k8s.OSDInfo{
		{
			ID:             0,
			Name:           "osd.0",
			Hostname:       "worker-1",
			Status:         "up",
			InOut:          "in",
			Weight:         0.873,
			DeviceClass:    "ssd",
			DeploymentName: "rook-ceph-osd-0",
		},
		{
			ID:             1,
			Name:           "osd.1",
			Hostname:       "worker-1",
			Status:         "down",
			InOut:          "out",
			Weight:         0.873,
			DeviceClass:    "ssd",
			DeploymentName: "rook-ceph-osd-1",
		},
	}
	v.SetOSDs(osds)

	output := v.View()

	// Check header is present
	if !strings.Contains(output, "OSD") {
		t.Error("output should contain OSD header")
	}
	if !strings.Contains(output, "HOST") {
		t.Error("output should contain HOST header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("output should contain STATUS header")
	}

	// Check OSD names are present
	if !strings.Contains(output, "osd.0") {
		t.Error("output should contain osd.0")
	}
	if !strings.Contains(output, "osd.1") {
		t.Error("output should contain osd.1")
	}

	// Check hostname is present
	if !strings.Contains(output, "worker-1") {
		t.Error("output should contain worker-1")
	}
}

func TestOSDsView_NooutBanner(t *testing.T) {
	v := NewOSDsView()
	v.SetSize(120, 30)

	osds := []k8s.OSDInfo{
		{ID: 0, Name: "osd.0", Status: "up"},
	}
	v.SetOSDs(osds)

	// Without noout flag
	output := v.View()
	if strings.Contains(output, "noout flag is SET") {
		t.Error("output should not contain noout warning when flag is not set")
	}

	// With noout flag
	v.SetNooutFlag(true)
	output = v.View()
	if !strings.Contains(output, "noout flag is SET") {
		t.Error("output should contain noout warning when flag is set")
	}
}

func TestOSDsView_EmptyView(t *testing.T) {
	v := NewOSDsView()
	v.SetSize(100, 30)

	output := v.View()

	if !strings.Contains(output, "No OSDs found") {
		t.Errorf("empty view should show 'No OSDs found', got: %s", output)
	}
}

func TestOSDsView_GetSelectedOSD(t *testing.T) {
	v := NewOSDsView()

	// Empty view
	if v.GetSelectedOSD() != nil {
		t.Error("GetSelectedOSD() on empty view should return nil")
	}

	osds := []k8s.OSDInfo{
		{ID: 0, Name: "osd.0"},
		{ID: 1, Name: "osd.1"},
	}
	v.SetOSDs(osds)

	// First OSD selected
	selected := v.GetSelectedOSD()
	if selected == nil {
		t.Fatal("GetSelectedOSD() should not be nil")
	}
	if selected.Name != "osd.0" {
		t.Errorf("selected OSD = %s, want osd.0", selected.Name)
	}

	// Move to second OSD
	v.SetCursor(1)
	selected = v.GetSelectedOSD()
	if selected.Name != "osd.1" {
		t.Errorf("selected OSD = %s, want osd.1", selected.Name)
	}
}

func TestOSDsView_CountDownOut(t *testing.T) {
	v := NewOSDsView()

	osds := []k8s.OSDInfo{
		{ID: 0, Name: "osd.0", Status: "up", InOut: "in"},
		{ID: 1, Name: "osd.1", Status: "down", InOut: "in"},
		{ID: 2, Name: "osd.2", Status: "up", InOut: "out"},
		{ID: 3, Name: "osd.3", Status: "down", InOut: "out"},
	}
	v.SetOSDs(osds)

	if v.CountDown() != 2 {
		t.Errorf("CountDown() = %d, want 2", v.CountDown())
	}

	if v.CountOut() != 2 {
		t.Errorf("CountOut() = %d, want 2", v.CountOut())
	}
}

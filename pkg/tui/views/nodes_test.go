package views

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/k8s"

	tea "charm.land/bubbletea/v2"
)

func TestNewNodesView(t *testing.T) {
	v := NewNodesView()

	if v == nil {
		t.Fatal("NewNodesView() returned nil")
	}

	if v.cursor != 0 {
		t.Errorf("cursor = %d, want 0", v.cursor)
	}

	if len(v.nodes) != 0 {
		t.Errorf("nodes len = %d, want 0", len(v.nodes))
	}
}

func TestNodesView_SetNodes(t *testing.T) {
	v := NewNodesView()

	nodes := []k8s.NodeInfo{
		{Name: "node-1", Status: "Ready", CephPodCount: 3},
		{Name: "node-2", Status: "Ready", CephPodCount: 2},
		{Name: "node-3", Status: "NotReady", CephPodCount: 0},
	}

	v.SetNodes(nodes)

	if len(v.nodes) != 3 {
		t.Errorf("nodes len = %d, want 3", len(v.nodes))
	}

	if v.Count() != 3 {
		t.Errorf("Count() = %d, want 3", v.Count())
	}
}

func TestNodesView_CursorNavigation(t *testing.T) {
	v := NewNodesView()
	v.SetSize(100, 50)

	nodes := []k8s.NodeInfo{
		{Name: "node-1"},
		{Name: "node-2"},
		{Name: "node-3"},
	}
	v.SetNodes(nodes)

	// Test j/down key
	v.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if v.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", v.cursor)
	}

	// Test k/up key
	v.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if v.cursor != 0 {
		t.Errorf("cursor after 'k' = %d, want 0", v.cursor)
	}

	// Test G (go to end)
	v.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	if v.cursor != 2 {
		t.Errorf("cursor after 'G' = %d, want 2", v.cursor)
	}

	// Test g (go to start)
	v.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if v.cursor != 0 {
		t.Errorf("cursor after 'g' = %d, want 0", v.cursor)
	}

	// Test cursor doesn't go below 0
	v.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if v.cursor != 0 {
		t.Errorf("cursor after 'k' at start = %d, want 0", v.cursor)
	}

	// Test cursor doesn't go above max
	v.cursor = 2
	v.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if v.cursor != 2 {
		t.Errorf("cursor after 'j' at end = %d, want 2", v.cursor)
	}
}

func TestNodesView_Enter(t *testing.T) {
	v := NewNodesView()

	nodes := []k8s.NodeInfo{
		{Name: "node-1", Status: "Ready"},
		{Name: "node-2", Status: "Ready"},
	}
	v.SetNodes(nodes)

	// Press enter on first node
	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("enter key should return a command")
	}

	msg := cmd()
	selectedMsg, ok := msg.(NodeSelectedMsg)
	if !ok {
		t.Fatalf("expected NodeSelectedMsg, got %T", msg)
	}

	if selectedMsg.Node.Name != "node-1" {
		t.Errorf("selected node = %s, want node-1", selectedMsg.Node.Name)
	}
}

func TestNodesView_View(t *testing.T) {
	v := NewNodesView()
	v.SetSize(120, 30)

	nodes := []k8s.NodeInfo{
		{
			Name:         "worker-1",
			Status:       "Ready",
			Roles:        []string{"worker"},
			Cordoned:     false,
			CephPodCount: 3,
			Age:          "1d",
		},
		{
			Name:         "control-plane-1",
			Status:       "Ready",
			Roles:        []string{"control-plane"},
			Cordoned:     true,
			CephPodCount: 1,
			Age:          "2d",
		},
	}
	v.SetNodes(nodes)

	output := v.Render()

	// Check header is present
	if !strings.Contains(output, "NAME") {
		t.Error("output should contain NAME header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("output should contain STATUS header")
	}
	if !strings.Contains(output, "CEPH PODS") {
		t.Error("output should contain CEPH PODS header")
	}

	// Check node names are present
	if !strings.Contains(output, "worker-1") {
		t.Error("output should contain worker-1")
	}
	if !strings.Contains(output, "control-plane-1") {
		t.Error("output should contain control-plane-1")
	}

	// Check cordoned node shows Cordoned status
	if !strings.Contains(output, "Cordoned") {
		t.Error("output should contain 'Cordoned' for cordoned node")
	}
}

func TestNodesView_View_TinyHeightLimitsOutput(t *testing.T) {
	v := NewNodesView()
	v.SetSize(120, 3)
	v.SetNodes([]k8s.NodeInfo{
		{Name: "node-1", Status: "Ready"},
		{Name: "node-2", Status: "Ready"},
		{Name: "node-3", Status: "Ready"},
	})

	output := v.Render()
	if !strings.Contains(output, "(1/3)") {
		t.Fatalf("expected scroll indicator for tiny height, got: %q", output)
	}
	if strings.Contains(output, "node-3") {
		t.Fatalf("expected tiny height to not render all rows, got: %q", output)
	}
}

func TestNodesView_View_TruncatesRolesWithEllipsis(t *testing.T) {
	v := NewNodesView()
	v.SetSize(100, 30) // width >= 100 includes Roles column
	v.SetNodes([]k8s.NodeInfo{
		{Name: "node-1", Status: "Ready", Roles: []string{"control-plane", "very-long-role-name"}},
	})

	output := v.Render()
	if !strings.Contains(output, "...") {
		t.Fatalf("expected roles to be truncated with ellipsis, got: %q", output)
	}
}

func TestNodesView_View_DisplaysIPAddress(t *testing.T) {
	v := NewNodesView()
	v.SetSize(100, 30) // width >= 66 includes IP column
	v.SetNodes([]k8s.NodeInfo{
		{Name: "node-1", IP: "192.168.1.10", Status: "Ready"},
		{Name: "node-2", IP: "", Status: "Ready"}, // no IP case
	})

	output := v.Render()
	if !strings.Contains(output, "IP") {
		t.Fatalf("expected IP header, got: %q", output)
	}
	if !strings.Contains(output, "192.168.1.10") {
		t.Fatalf("expected IP address to be displayed, got: %q", output)
	}
	if !strings.Contains(output, "-") {
		t.Fatalf("expected '-' for node with no IP, got: %q", output)
	}
}

func TestNodesView_View_HidesIPOnNarrowWidth(t *testing.T) {
	v := NewNodesView()
	v.SetSize(50, 30) // narrow width hides IP column
	v.SetNodes([]k8s.NodeInfo{
		{Name: "node-1", IP: "192.168.1.10", Status: "Ready"},
	})

	// IP should not appear in header when width is too narrow
	layout := v.columnLayout()
	if layout.showIP {
		t.Fatalf("expected showIP to be false at narrow width, layout: %+v", layout)
	}
}

func TestNodesView_EmptyView(t *testing.T) {
	v := NewNodesView()
	v.SetSize(100, 30)

	output := v.Render()

	if !strings.Contains(output, "No nodes found") {
		t.Errorf("empty view should show 'No nodes found', got: %s", output)
	}
}

func TestNodesView_GetSelectedNode(t *testing.T) {
	v := NewNodesView()

	// Empty view
	if v.GetSelectedNode() != nil {
		t.Error("GetSelectedNode() on empty view should return nil")
	}

	nodes := []k8s.NodeInfo{
		{Name: "node-1"},
		{Name: "node-2"},
	}
	v.SetNodes(nodes)

	// First node selected
	selected := v.GetSelectedNode()
	if selected == nil {
		t.Fatal("GetSelectedNode() should not be nil")
	}
	if selected.Name != "node-1" {
		t.Errorf("selected node = %s, want node-1", selected.Name)
	}

	// Move to second node
	v.SetCursor(1)
	selected = v.GetSelectedNode()
	if selected.Name != "node-2" {
		t.Errorf("selected node = %s, want node-2", selected.Name)
	}
}

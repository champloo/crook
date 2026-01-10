package views

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/k8s"

	tea "charm.land/bubbletea/v2"
)

func TestNewDeploymentsView(t *testing.T) {
	v := NewDeploymentsView()

	if v == nil {
		t.Fatal("NewDeploymentsView() returned nil")
	}

	if v.cursor != 0 {
		t.Errorf("cursor = %d, want 0", v.cursor)
	}

	if len(v.deployments) != 0 {
		t.Errorf("deployments len = %d, want 0", len(v.deployments))
	}

	if !v.groupByType {
		t.Error("groupByType should be true by default")
	}
}

func TestDeploymentsView_SetDeployments(t *testing.T) {
	v := NewDeploymentsView()

	deployments := []k8s.DeploymentInfo{
		{Name: "rook-ceph-osd-0", Type: "osd", ReadyReplicas: 1, DesiredReplicas: 1, Status: "Ready"},
		{Name: "rook-ceph-osd-1", Type: "osd", ReadyReplicas: 1, DesiredReplicas: 1, Status: "Ready"},
		{Name: "rook-ceph-mon-a", Type: "mon", ReadyReplicas: 1, DesiredReplicas: 1, Status: "Ready"},
	}

	v.SetDeployments(deployments)

	if len(v.deployments) != 3 {
		t.Errorf("deployments len = %d, want 3", len(v.deployments))
	}

	if v.Count() != 3 {
		t.Errorf("Count() = %d, want 3", v.Count())
	}
}

func TestDeploymentsView_CursorNavigation(t *testing.T) {
	v := NewDeploymentsView()
	v.SetSize(100, 50)

	deployments := []k8s.DeploymentInfo{
		{Name: "dep-1", Type: "osd"},
		{Name: "dep-2", Type: "osd"},
		{Name: "dep-3", Type: "mon"},
	}
	v.SetDeployments(deployments)

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
}

func TestDeploymentsView_Enter(t *testing.T) {
	v := NewDeploymentsView()

	deployments := []k8s.DeploymentInfo{
		{Name: "rook-ceph-osd-0", Type: "osd", Status: "Ready"},
		{Name: "rook-ceph-mon-a", Type: "mon", Status: "Ready"},
	}
	v.SetDeployments(deployments)

	// Press enter on first deployment
	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("enter key should return a command")
	}

	msg := cmd()
	selectedMsg, ok := msg.(DeploymentSelectedMsg)
	if !ok {
		t.Fatalf("expected DeploymentSelectedMsg, got %T", msg)
	}

	// First item should be osd (sorted by type)
	if selectedMsg.Deployment.Type != "osd" {
		t.Errorf("selected deployment type = %s, want osd", selectedMsg.Deployment.Type)
	}
}

func TestDeploymentsView_View(t *testing.T) {
	v := NewDeploymentsView()
	v.SetSize(120, 30)

	deployments := []k8s.DeploymentInfo{
		{
			Name:            "rook-ceph-osd-0",
			Namespace:       "rook-ceph",
			Type:            "osd",
			ReadyReplicas:   1,
			DesiredReplicas: 1,
			NodeName:        "worker-1",
			Status:          "Ready",
			Age:             "1d",
		},
		{
			Name:            "rook-ceph-mon-a",
			Namespace:       "rook-ceph",
			Type:            "mon",
			ReadyReplicas:   1,
			DesiredReplicas: 1,
			NodeName:        "worker-2",
			Status:          "Ready",
			Age:             "2d",
		},
		{
			Name:            "rook-ceph-crashcollector",
			Namespace:       "rook-ceph",
			Type:            "crashcollector",
			ReadyReplicas:   0,
			DesiredReplicas: 0,
			Status:          "Scaled Down",
			Age:             "3d",
		},
	}
	v.SetDeployments(deployments)

	output := v.Render()

	// Check header is present
	if !strings.Contains(output, "NAME") {
		t.Error("output should contain NAME header")
	}
	if !strings.Contains(output, "READY") {
		t.Error("output should contain READY header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("output should contain STATUS header")
	}

	// Check group headers when groupByType is true
	if !strings.Contains(output, "OSD") {
		t.Error("output should contain OSD group header")
	}
	if !strings.Contains(output, "MON") {
		t.Error("output should contain MON group header")
	}

	// Check deployment names are present
	if !strings.Contains(output, "rook-ceph-osd-0") {
		t.Error("output should contain rook-ceph-osd-0")
	}
	if !strings.Contains(output, "rook-ceph-mon-a") {
		t.Error("output should contain rook-ceph-mon-a")
	}
}

func TestDeploymentsView_EmptyView(t *testing.T) {
	v := NewDeploymentsView()
	v.SetSize(100, 30)

	output := v.Render()

	if !strings.Contains(output, "No deployments found") {
		t.Errorf("empty view should show 'No deployments found', got: %s", output)
	}
}

func TestDeploymentsView_GetSelectedDeployment(t *testing.T) {
	v := NewDeploymentsView()

	// Empty view
	if v.GetSelectedDeployment() != nil {
		t.Error("GetSelectedDeployment() on empty view should return nil")
	}

	deployments := []k8s.DeploymentInfo{
		{Name: "dep-1", Type: "osd"},
		{Name: "dep-2", Type: "osd"},
	}
	v.SetDeployments(deployments)

	// First deployment selected
	selected := v.GetSelectedDeployment()
	if selected == nil {
		t.Fatal("GetSelectedDeployment() should not be nil")
	}

	// Move to second deployment
	v.SetCursor(1)
	selected = v.GetSelectedDeployment()
	if selected == nil {
		t.Fatal("GetSelectedDeployment() should not be nil after SetCursor")
	}
}

func TestDeploymentsView_Grouping(t *testing.T) {
	v := NewDeploymentsView()
	v.SetSize(120, 50)

	deployments := []k8s.DeploymentInfo{
		{Name: "rook-ceph-mon-a", Type: "mon"},
		{Name: "rook-ceph-osd-0", Type: "osd"},
		{Name: "rook-ceph-osd-1", Type: "osd"},
		{Name: "rook-ceph-crashcollector", Type: "crashcollector"},
	}
	v.SetDeployments(deployments)

	// With grouping enabled, OSDs should come first
	output := v.Render()
	osdPos := strings.Index(output, "OSD")
	monPos := strings.Index(output, "MON")

	if osdPos == -1 || monPos == -1 {
		t.Error("output should contain both OSD and MON group headers")
	}

	// OSDs should appear before MONs based on type order
	if osdPos > monPos {
		t.Error("OSD group should appear before MON group")
	}

	// Disable grouping
	v.SetGroupByType(false)
	if v.groupByType {
		t.Error("groupByType should be false after SetGroupByType(false)")
	}
}

func TestTypeOrder(t *testing.T) {
	tests := []struct {
		typ   string
		order int
	}{
		{"osd", 0},
		{"mon", 1},
		{"mgr", 2},
		{"crashcollector", 6},
		{"unknown", 99},
		{"", 99},
	}

	for _, tt := range tests {
		got := typeOrder(tt.typ)
		if got != tt.order {
			t.Errorf("typeOrder(%q) = %d, want %d", tt.typ, got, tt.order)
		}
	}
}

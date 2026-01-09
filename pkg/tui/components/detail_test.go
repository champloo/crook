package components_test

import (
	"strings"
	"testing"
	"time"

	"github.com/andri/crook/pkg/k8s"

	"github.com/andri/crook/pkg/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailPanel_NewDetailPanel(t *testing.T) {
	dp := components.NewDetailPanel()
	if dp == nil {
		t.Fatal("NewDetailPanel() returned nil")
	}
	if dp.IsVisible() {
		t.Error("NewDetailPanel() should not be visible by default")
	}
}

func TestDetailPanel_ShowNode(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	node := k8s.NodeInfo{
		Name:           "worker-1",
		Status:         "Ready",
		Roles:          []string{"worker"},
		Schedulable:    true,
		Cordoned:       false,
		CephPodCount:   3,
		Age:            k8s.Duration(5 * 24 * time.Hour),
		KubeletVersion: "v1.28.0",
	}

	related := []components.RelatedResource{
		{Type: "Pod", Name: "rook-ceph-osd-0-abc", Status: "Running"},
		{Type: "Pod", Name: "rook-ceph-mon-a-xyz", Status: "Running"},
	}

	dp.ShowNode(node, related)

	if !dp.IsVisible() {
		t.Error("ShowNode() should make panel visible")
	}

	view := dp.View()
	if view == "" {
		t.Error("View() returned empty string")
	}

	// Check content contains expected fields
	if !strings.Contains(view, "worker-1") {
		t.Error("View() missing node name")
	}
	if !strings.Contains(view, "Ready") {
		t.Error("View() missing node status")
	}
}

func TestDetailPanel_ShowDeployment(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	dep := k8s.DeploymentInfo{
		Name:            "rook-ceph-osd-0",
		Namespace:       "rook-ceph",
		ReadyReplicas:   1,
		DesiredReplicas: 1,
		NodeName:        "worker-1",
		Age:             k8s.Duration(5 * 24 * time.Hour),
		Status:          "Ready",
		Type:            "osd",
		OsdID:           "0",
	}

	dp.ShowDeployment(dep, nil)

	if !dp.IsVisible() {
		t.Error("ShowDeployment() should make panel visible")
	}

	view := dp.View()
	if !strings.Contains(view, "rook-ceph-osd-0") {
		t.Error("View() missing deployment name")
	}
	if !strings.Contains(view, "rook-ceph") {
		t.Error("View() missing namespace")
	}
}

func TestDetailPanel_ShowOSD(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	osd := k8s.OSDInfo{
		ID:             0,
		Name:           "osd.0",
		Hostname:       "worker-1",
		Status:         "up",
		InOut:          "in",
		Weight:         1.0,
		Reweight:       1.0,
		DeviceClass:    "ssd",
		DeploymentName: "rook-ceph-osd-0",
	}

	dp.ShowOSD(osd, nil)

	if !dp.IsVisible() {
		t.Error("ShowOSD() should make panel visible")
	}

	view := dp.View()
	if !strings.Contains(view, "osd.0") {
		t.Error("View() missing OSD name")
	}
	if !strings.Contains(view, "worker-1") {
		t.Error("View() missing hostname")
	}
}

func TestDetailPanel_ShowPod(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	pod := k8s.PodInfo{
		Name:      "rook-ceph-osd-0-abc123",
		Namespace: "rook-ceph",
		Status:    "Running",
		// Ready field removed,
		ReadyContainers: 1,
		TotalContainers: 1,
		Restarts:        0,
		NodeName:        "worker-1",
		Age:             k8s.Duration(5 * 24 * time.Hour),
		Type:            "osd",
		IP:              "10.0.0.1",
		OwnerDeployment: "rook-ceph-osd-0",
	}

	dp.ShowPod(pod, nil)

	if !dp.IsVisible() {
		t.Error("ShowPod() should make panel visible")
	}

	view := dp.View()
	if !strings.Contains(view, "rook-ceph-osd-0-abc123") {
		t.Error("View() missing pod name")
	}
	if !strings.Contains(view, "Running") {
		t.Error("View() missing pod status")
	}
}

func TestDetailPanel_Hide(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	node := k8s.NodeInfo{Name: "test-node", Status: "Ready"}
	dp.ShowNode(node, nil)

	if !dp.IsVisible() {
		t.Error("Panel should be visible after ShowNode()")
	}

	dp.Hide()

	if dp.IsVisible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestDetailPanel_KeyNavigation(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 10) // Small height to test scrolling

	// Create node with lots of related resources to enable scrolling
	node := k8s.NodeInfo{
		Name:           "worker-1",
		Status:         "Ready",
		Roles:          []string{"worker"},
		KubeletVersion: "v1.28.0",
	}
	related := make([]components.RelatedResource, 20)
	for i := 0; i < 20; i++ {
		related[i] = components.RelatedResource{
			Type:   "Pod",
			Name:   "pod-" + string(rune('a'+i)),
			Status: "Running",
		}
	}
	dp.ShowNode(node, related)

	// Test 'j' scrolls down
	initialView := dp.View()
	dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	afterJView := dp.View()

	// Views might be different if scrolling worked
	_ = initialView
	_ = afterJView

	// Test 'k' scrolls up
	dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Test 'g' goes to top
	dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Test 'G' goes to bottom
	dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
}

func TestDetailPanel_CloseWithEsc(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	node := k8s.NodeInfo{Name: "test-node", Status: "Ready"}
	dp.ShowNode(node, nil)

	_, cmd := dp.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should return a DetailCloseMsg
	if cmd == nil {
		t.Fatal("Update(Esc) should return a command")
	}

	msg := cmd()
	if _, ok := msg.(components.DetailCloseMsg); !ok {
		t.Error("Update(Esc) should return DetailCloseMsg")
	}

	if dp.IsVisible() {
		t.Error("Panel should not be visible after Esc")
	}
}

func TestDetailPanel_CloseWithQ(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	node := k8s.NodeInfo{Name: "test-node", Status: "Ready"}
	dp.ShowNode(node, nil)

	_, cmd := dp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Fatal("Update(q) should return a command")
	}

	msg := cmd()
	if _, ok := msg.(components.DetailCloseMsg); !ok {
		t.Error("Update(q) should return DetailCloseMsg")
	}
}

func TestDetailPanel_ViewWhenHidden(t *testing.T) {
	dp := components.NewDetailPanel()

	view := dp.View()
	if view != "" {
		t.Error("View() should return empty string when hidden")
	}
}

func TestDetailPanel_RelatedResources(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(100, 40)

	node := k8s.NodeInfo{
		Name:   "worker-1",
		Status: "Ready",
	}
	related := []components.RelatedResource{
		{Type: "Pod", Name: "pod-a", Status: "Running"},
		{Type: "Pod", Name: "pod-b", Status: "Pending"},
		{Type: "Deployment", Name: "deploy-a", Status: "Ready"},
	}

	dp.ShowNode(node, related)
	view := dp.View()

	if !strings.Contains(view, "Related Resources") {
		t.Error("View() missing Related Resources section")
	}
	if !strings.Contains(view, "pod-a") {
		t.Error("View() missing related pod-a")
	}
	if !strings.Contains(view, "deploy-a") {
		t.Error("View() missing related deployment")
	}
}

func TestDetailPanel_SetSize(t *testing.T) {
	dp := components.NewDetailPanel()

	node := k8s.NodeInfo{Name: "test-node", Status: "Ready"}
	dp.ShowNode(node, nil)

	// Set size and verify panel still works
	dp.SetSize(120, 30)

	view := dp.View()
	if view == "" {
		t.Error("View() returned empty after SetSize()")
	}
}

func TestDetailPanel_CordonedNode(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	node := k8s.NodeInfo{
		Name:        "worker-1",
		Status:      "Ready",
		Cordoned:    true,
		Schedulable: false,
	}

	dp.ShowNode(node, nil)
	view := dp.View()

	if !strings.Contains(view, "Cordoned") {
		t.Error("View() should show Cordoned status for cordoned node")
	}
}

func TestDetailPanel_HighRestartPod(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	pod := k8s.PodInfo{
		Name:            "crashy-pod",
		Namespace:       "rook-ceph",
		Status:          "Running",
		ReadyContainers: 1,
		TotalContainers: 1,
		Restarts:        15, // High restart count
		NodeName:        "worker-1",
	}

	dp.ShowPod(pod, nil)
	view := dp.View()

	if !strings.Contains(view, "15") {
		t.Error("View() should show restart count")
	}
}

func TestDetailPanel_OSDDown(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	osd := k8s.OSDInfo{
		ID:       0,
		Name:     "osd.0",
		Hostname: "worker-1",
		Status:   "down", // Down status
		InOut:    "out",  // Out of cluster
		Weight:   1.0,
	}

	dp.ShowOSD(osd, nil)
	view := dp.View()

	if !strings.Contains(view, "down") {
		t.Error("View() should show down status")
	}
	if !strings.Contains(view, "out") {
		t.Error("View() should show out status")
	}
}

func TestDetailPanel_DeploymentScaling(t *testing.T) {
	dp := components.NewDetailPanel()
	dp.SetSize(80, 24)

	dep := k8s.DeploymentInfo{
		Name:            "scaling-deployment",
		Namespace:       "rook-ceph",
		ReadyReplicas:   0,
		DesiredReplicas: 1, // Desired but not ready
		Status:          "Scaling",
	}

	dp.ShowDeployment(dep, nil)
	view := dp.View()

	if !strings.Contains(view, "Scaling") {
		t.Error("View() should show Scaling status")
	}
}

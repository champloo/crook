package views

import (
	"strings"
	"testing"
	"time"
)

func TestNewPodsView(t *testing.T) {
	v := NewPodsView()
	if v == nil {
		t.Fatal("NewPodsView returned nil")
	}
	if v.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", v.cursor)
	}
	if len(v.pods) != 0 {
		t.Errorf("expected empty pods, got %d", len(v.pods))
	}
}

func TestPodsView_SetPods(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "rook-ceph-osd-0-abc", Status: "Running", NodeName: "node-1"},
		{Name: "rook-ceph-mon-a-xyz", Status: "Running", NodeName: "node-2"},
	}

	v.SetPods(pods)

	if v.Count() != 2 {
		t.Errorf("expected 2 pods, got %d", v.Count())
	}
	if v.TotalCount() != 2 {
		t.Errorf("expected TotalCount=2, got %d", v.TotalCount())
	}
}

func TestPodsView_SetFilter(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "rook-ceph-osd-0-abc", Status: "Running", NodeName: "node-1"},
		{Name: "rook-ceph-mon-a-xyz", Status: "Running", NodeName: "node-2"},
		{Name: "rook-ceph-osd-1-def", Status: "Pending", NodeName: "node-1"},
	}

	v.SetPods(pods)
	v.SetFilter("osd")

	if v.Count() != 2 {
		t.Errorf("expected 2 pods matching 'osd', got %d", v.Count())
	}
	if v.TotalCount() != 3 {
		t.Errorf("expected TotalCount=3, got %d", v.TotalCount())
	}
}

func TestPodsView_SetNodeFilter(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "rook-ceph-osd-0-abc", Status: "Running", NodeName: "node-1"},
		{Name: "rook-ceph-mon-a-xyz", Status: "Running", NodeName: "node-2"},
		{Name: "rook-ceph-osd-1-def", Status: "Pending", NodeName: "node-1"},
	}

	v.SetPods(pods)
	v.SetNodeFilter("node-1")

	if v.Count() != 2 {
		t.Errorf("expected 2 pods on node-1, got %d", v.Count())
	}
}

func TestPodsView_CombinedFilters(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "rook-ceph-osd-0-abc", Status: "Running", NodeName: "node-1"},
		{Name: "rook-ceph-mon-a-xyz", Status: "Running", NodeName: "node-2"},
		{Name: "rook-ceph-osd-1-def", Status: "Pending", NodeName: "node-1"},
		{Name: "rook-ceph-mon-b-ghi", Status: "Running", NodeName: "node-1"},
	}

	v.SetPods(pods)
	v.SetNodeFilter("node-1")
	v.SetFilter("osd")

	// Should filter to only OSD pods on node-1
	if v.Count() != 2 {
		t.Errorf("expected 2 osd pods on node-1, got %d", v.Count())
	}
}

func TestPodsView_View_Empty(t *testing.T) {
	v := NewPodsView()
	view := v.View()

	if !strings.Contains(view, "No pods found") {
		t.Errorf("expected 'No pods found' in empty view, got: %s", view)
	}
}

func TestPodsView_View_WithPods(t *testing.T) {
	v := NewPodsView()
	v.SetSize(120, 30)

	pods := []PodInfo{
		{
			Name:            "rook-ceph-osd-0-abc123",
			Namespace:       "rook-ceph",
			Status:          "Running",
			ReadyContainers: 2,
			TotalContainers: 2,
			Restarts:        0,
			NodeName:        "worker-01",
			Age:             24 * time.Hour,
		},
	}

	v.SetPods(pods)
	view := v.View()

	// Check header
	if !strings.Contains(view, "NAME") {
		t.Errorf("expected 'NAME' header in view, got: %s", view)
	}
	if !strings.Contains(view, "STATUS") {
		t.Errorf("expected 'STATUS' header in view, got: %s", view)
	}
	if !strings.Contains(view, "READY") {
		t.Errorf("expected 'READY' header in view, got: %s", view)
	}
	if !strings.Contains(view, "RESTARTS") {
		t.Errorf("expected 'RESTARTS' header in view, got: %s", view)
	}

	// Check pod data
	if !strings.Contains(view, "rook-ceph-osd-0") {
		t.Errorf("expected pod name in view, got: %s", view)
	}
	if !strings.Contains(view, "Running") {
		t.Errorf("expected 'Running' status in view, got: %s", view)
	}
	if !strings.Contains(view, "2/2") {
		t.Errorf("expected '2/2' ready count in view, got: %s", view)
	}
}

func TestPodsView_Cursor(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "pod-1", Status: "Running"},
		{Name: "pod-2", Status: "Running"},
		{Name: "pod-3", Status: "Running"},
	}

	v.SetPods(pods)

	if v.GetCursor() != 0 {
		t.Errorf("expected initial cursor=0, got %d", v.GetCursor())
	}

	v.SetCursor(1)
	if v.GetCursor() != 1 {
		t.Errorf("expected cursor=1, got %d", v.GetCursor())
	}

	// Should not go beyond bounds
	v.SetCursor(10)
	if v.GetCursor() != 1 {
		t.Errorf("expected cursor to stay at 1 when setting out of bounds, got %d", v.GetCursor())
	}
}

func TestPodsView_GetSelectedPod(t *testing.T) {
	v := NewPodsView()

	// No pods
	if v.GetSelectedPod() != nil {
		t.Error("expected nil for empty view")
	}

	pods := []PodInfo{
		{Name: "pod-1", Status: "Running"},
		{Name: "pod-2", Status: "Pending"},
	}

	v.SetPods(pods)
	v.SetCursor(1)

	selected := v.GetSelectedPod()
	if selected == nil {
		t.Fatal("expected selected pod, got nil")
	}
	if selected.Name != "pod-2" {
		t.Errorf("expected pod-2, got %s", selected.Name)
	}
}

func TestPodsView_CountByStatus(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "pod-1", Status: "Running"},
		{Name: "pod-2", Status: "Running"},
		{Name: "pod-3", Status: "Pending"},
		{Name: "pod-4", Status: "Failed"},
	}

	v.SetPods(pods)

	if v.CountByStatus("Running") != 2 {
		t.Errorf("expected 2 Running pods, got %d", v.CountByStatus("Running"))
	}
	if v.CountByStatus("Pending") != 1 {
		t.Errorf("expected 1 Pending pod, got %d", v.CountByStatus("Pending"))
	}
	if v.CountByStatus("Failed") != 1 {
		t.Errorf("expected 1 Failed pod, got %d", v.CountByStatus("Failed"))
	}
}

func TestPodsView_CountHighRestarts(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "pod-1", Restarts: 0},
		{Name: "pod-2", Restarts: 3},
		{Name: "pod-3", Restarts: 6},  // High
		{Name: "pod-4", Restarts: 10}, // High
	}

	v.SetPods(pods)

	if v.CountHighRestarts() != 2 {
		t.Errorf("expected 2 pods with high restarts, got %d", v.CountHighRestarts())
	}
}

func TestPodsView_FilterCaseInsensitive(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "rook-ceph-OSD-0", Status: "Running"},
		{Name: "rook-ceph-mon-a", Status: "Running"},
	}

	v.SetPods(pods)
	v.SetFilter("osd") // lowercase filter should match uppercase OSD

	if v.Count() != 1 {
		t.Errorf("expected 1 pod matching case-insensitive 'osd', got %d", v.Count())
	}
}

func TestPodsView_FilterByStatus(t *testing.T) {
	v := NewPodsView()

	pods := []PodInfo{
		{Name: "pod-1", Status: "Running"},
		{Name: "pod-2", Status: "Pending"},
		{Name: "pod-3", Status: "CrashLoopBackOff"},
	}

	v.SetPods(pods)
	v.SetFilter("pending")

	if v.Count() != 1 {
		t.Errorf("expected 1 pod matching status 'pending', got %d", v.Count())
	}
}

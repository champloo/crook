package views_test

import (
	"testing"
	"time"

	"github.com/andri/crook/pkg/tui/views"
)

func TestDeploymentsPodsView_NewDeploymentsPodsView(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	if v == nil {
		t.Fatal("NewDeploymentsPodsView returned nil")
	}

	// Should default to showing deployments
	if v.IsShowingPods() {
		t.Error("new view should show deployments by default, not pods")
	}

	if v.GetTitle() != "Deployments" {
		t.Errorf("expected title 'Deployments', got %q", v.GetTitle())
	}
}

func TestDeploymentsPodsView_ShowDeployments(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	// Switch to pods first
	v.ShowPods()
	if !v.IsShowingPods() {
		t.Error("should be showing pods after ShowPods()")
	}

	// Switch back to deployments
	v.ShowDeployments()
	if v.IsShowingPods() {
		t.Error("should not be showing pods after ShowDeployments()")
	}

	if v.GetTitle() != "Deployments" {
		t.Errorf("expected title 'Deployments', got %q", v.GetTitle())
	}
}

func TestDeploymentsPodsView_ShowPods(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	// Initially showing deployments
	if v.IsShowingPods() {
		t.Error("should initially show deployments")
	}

	// Switch to pods
	v.ShowPods()
	if !v.IsShowingPods() {
		t.Error("should be showing pods after ShowPods()")
	}

	if v.GetTitle() != "Pods" {
		t.Errorf("expected title 'Pods', got %q", v.GetTitle())
	}
}

func TestDeploymentsPodsView_Toggle_ResetsCursor(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	// Add some deployments and set cursor
	deployments := []views.DeploymentInfo{
		{Name: "dep1", Namespace: "ns", Status: "Ready"},
		{Name: "dep2", Namespace: "ns", Status: "Ready"},
		{Name: "dep3", Namespace: "ns", Status: "Ready"},
	}
	v.SetDeployments(deployments)
	v.SetCursor(2)

	if v.GetCursor() != 2 {
		t.Errorf("expected cursor 2, got %d", v.GetCursor())
	}

	// Add some pods and set cursor
	pods := []views.PodInfo{
		{Name: "pod1", Namespace: "ns", Status: "Running"},
		{Name: "pod2", Namespace: "ns", Status: "Running"},
	}
	v.SetPods(pods)

	// Switch to pods - cursor should reset
	v.ShowPods()
	if v.GetCursor() != 0 {
		t.Errorf("cursor should reset to 0 after toggling to pods, got %d", v.GetCursor())
	}

	// Set cursor in pods view
	v.SetCursor(1)
	if v.GetCursor() != 1 {
		t.Errorf("expected cursor 1 in pods view, got %d", v.GetCursor())
	}

	// Switch back to deployments - cursor should reset
	v.ShowDeployments()
	if v.GetCursor() != 0 {
		t.Errorf("cursor should reset to 0 after toggling to deployments, got %d", v.GetCursor())
	}
}

func TestDeploymentsPodsView_GetTitle(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	tests := []struct {
		name      string
		showPods  bool
		wantTitle string
	}{
		{"deployments view", false, "Deployments"},
		{"pods view", true, "Pods"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.showPods {
				v.ShowPods()
			} else {
				v.ShowDeployments()
			}

			if v.GetTitle() != tt.wantTitle {
				t.Errorf("GetTitle() = %q, want %q", v.GetTitle(), tt.wantTitle)
			}
		})
	}
}

func TestDeploymentsPodsView_Count_DelegatesToActiveView(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	deployments := []views.DeploymentInfo{
		{Name: "dep1", Namespace: "ns", Status: "Ready"},
		{Name: "dep2", Namespace: "ns", Status: "Ready"},
		{Name: "dep3", Namespace: "ns", Status: "Ready"},
	}
	v.SetDeployments(deployments)

	pods := []views.PodInfo{
		{Name: "pod1", Namespace: "ns", Status: "Running"},
		{Name: "pod2", Namespace: "ns", Status: "Running"},
	}
	v.SetPods(pods)

	// Check deployments count
	if v.Count() != 3 {
		t.Errorf("deployments Count() = %d, want 3", v.Count())
	}

	// Switch to pods and check count
	v.ShowPods()
	if v.Count() != 2 {
		t.Errorf("pods Count() = %d, want 2", v.Count())
	}
}

func TestDeploymentsPodsView_SetSize_ApplesToBothViews(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	v.SetSize(100, 50)

	// Can't directly check internal view sizes, but we can verify View() works
	v.SetDeployments([]views.DeploymentInfo{})
	v.SetPods([]views.PodInfo{})

	// Should not panic
	_ = v.View()
	v.ShowPods()
	_ = v.View()
}

func TestDeploymentsPodsView_SetNodeFilter(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	pods := []views.PodInfo{
		{Name: "pod1", Namespace: "ns", NodeName: "node1", Status: "Running"},
		{Name: "pod2", Namespace: "ns", NodeName: "node2", Status: "Running"},
		{Name: "pod3", Namespace: "ns", NodeName: "node1", Status: "Running"},
	}
	v.SetPods(pods)

	v.SetNodeFilter("node1")

	if v.GetNodeFilter() != "node1" {
		t.Errorf("GetNodeFilter() = %q, want 'node1'", v.GetNodeFilter())
	}

	v.ShowPods()

	// Should only show pods on node1
	if v.Count() != 2 {
		t.Errorf("filtered Count() = %d, want 2 (pods on node1)", v.Count())
	}
}

func TestDeploymentsPodsView_View(t *testing.T) {
	v := views.NewDeploymentsPodsView()
	v.SetSize(100, 20)

	deployments := []views.DeploymentInfo{
		{Name: "test-deployment", Namespace: "ns", Status: "Ready", Age: time.Hour},
	}
	v.SetDeployments(deployments)

	pods := []views.PodInfo{
		{Name: "test-pod", Namespace: "ns", Status: "Running", Age: time.Hour},
	}
	v.SetPods(pods)

	// View deployments
	deployView := v.View()
	if deployView == "" {
		t.Error("deployments View() returned empty string")
	}

	// Switch and view pods
	v.ShowPods()
	podsView := v.View()
	if podsView == "" {
		t.Error("pods View() returned empty string")
	}

	// Views should be different
	if deployView == podsView {
		t.Error("deployments and pods views should be different")
	}
}

func TestDeploymentsPodsView_DeploymentsCount(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	deployments := []views.DeploymentInfo{
		{Name: "dep1", Namespace: "ns", Status: "Ready"},
		{Name: "dep2", Namespace: "ns", Status: "Ready"},
	}
	v.SetDeployments(deployments)

	// DeploymentsCount should work regardless of which view is active
	if v.DeploymentsCount() != 2 {
		t.Errorf("DeploymentsCount() = %d, want 2", v.DeploymentsCount())
	}

	v.ShowPods()

	if v.DeploymentsCount() != 2 {
		t.Errorf("DeploymentsCount() after ShowPods() = %d, want 2", v.DeploymentsCount())
	}
}

func TestDeploymentsPodsView_PodsCount(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	pods := []views.PodInfo{
		{Name: "pod1", Namespace: "ns", Status: "Running"},
		{Name: "pod2", Namespace: "ns", Status: "Running"},
		{Name: "pod3", Namespace: "ns", Status: "Running"},
	}
	v.SetPods(pods)

	// PodsCount should work regardless of which view is active
	if v.PodsCount() != 3 {
		t.Errorf("PodsCount() = %d, want 3", v.PodsCount())
	}

	v.ShowDeployments()

	if v.PodsCount() != 3 {
		t.Errorf("PodsCount() after ShowDeployments() = %d, want 3", v.PodsCount())
	}
}

func TestDeploymentsPodsView_GetSelectedDeployment(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	deployments := []views.DeploymentInfo{
		{Name: "dep1", Namespace: "ns", Status: "Ready"},
		{Name: "dep2", Namespace: "ns", Status: "Ready"},
	}
	v.SetDeployments(deployments)
	v.SetCursor(1)

	selected := v.GetSelectedDeployment()
	if selected == nil {
		t.Fatal("GetSelectedDeployment() returned nil")
	}
	if selected.Name != "dep2" {
		t.Errorf("selected deployment name = %q, want 'dep2'", selected.Name)
	}

	// Should return nil when showing pods
	v.ShowPods()
	if v.GetSelectedDeployment() != nil {
		t.Error("GetSelectedDeployment() should return nil when showing pods")
	}
}

func TestDeploymentsPodsView_GetSelectedPod(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	pods := []views.PodInfo{
		{Name: "pod1", Namespace: "ns", Status: "Running"},
		{Name: "pod2", Namespace: "ns", Status: "Running"},
	}
	v.SetPods(pods)

	// Should return nil when showing deployments
	if v.GetSelectedPod() != nil {
		t.Error("GetSelectedPod() should return nil when showing deployments")
	}

	v.ShowPods()
	v.SetCursor(1)

	selected := v.GetSelectedPod()
	if selected == nil {
		t.Fatal("GetSelectedPod() returned nil")
	}
	if selected.Name != "pod2" {
		t.Errorf("selected pod name = %q, want 'pod2'", selected.Name)
	}
}

func TestDeploymentsPodsView_GetViews(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	if v.GetDeploymentsView() == nil {
		t.Error("GetDeploymentsView() returned nil")
	}

	if v.GetPodsView() == nil {
		t.Error("GetPodsView() returned nil")
	}
}

func TestDeploymentsPodsView_Toggle_NoOpWhenAlreadyOnView(t *testing.T) {
	v := views.NewDeploymentsPodsView()

	deployments := []views.DeploymentInfo{
		{Name: "dep1", Namespace: "ns", Status: "Ready"},
		{Name: "dep2", Namespace: "ns", Status: "Ready"},
	}
	v.SetDeployments(deployments)
	v.SetCursor(1)

	// Calling ShowDeployments when already on deployments should not reset cursor
	v.ShowDeployments()
	if v.GetCursor() != 1 {
		t.Errorf("cursor should remain at 1 when calling ShowDeployments() on deployments view, got %d", v.GetCursor())
	}

	// Switch to pods
	v.ShowPods()
	pods := []views.PodInfo{
		{Name: "pod1", Namespace: "ns", Status: "Running"},
		{Name: "pod2", Namespace: "ns", Status: "Running"},
	}
	v.SetPods(pods)
	v.SetCursor(1)

	// Calling ShowPods when already on pods should not reset cursor
	v.ShowPods()
	if v.GetCursor() != 1 {
		t.Errorf("cursor should remain at 1 when calling ShowPods() on pods view, got %d", v.GetCursor())
	}
}

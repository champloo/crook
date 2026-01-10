package views

import (
	"github.com/andri/crook/pkg/k8s"
	tea "charm.land/bubbletea/v2"
)

// DeploymentsPodsView is a composite view that can toggle between showing
// deployments or pods. It wraps the existing DeploymentsView and PodsView,
// forwarding all operations to the currently active sub-view.
type DeploymentsPodsView struct {
	deploymentsView *DeploymentsView
	podsView        *PodsView
	showPods        bool // false = deployments (default), true = pods
}

// NewDeploymentsPodsView creates a new composite view containing both
// DeploymentsView and PodsView.
func NewDeploymentsPodsView() *DeploymentsPodsView {
	return &DeploymentsPodsView{
		deploymentsView: NewDeploymentsView(),
		podsView:        NewPodsView(),
		showPods:        false,
	}
}

// ShowDeployments switches to the deployments view.
// This is triggered by the '[' key when this pane is active.
func (v *DeploymentsPodsView) ShowDeployments() {
	if v.showPods {
		v.showPods = false
		// Reset cursor when toggling views
		v.deploymentsView.SetCursor(0)
	}
}

// ShowPods switches to the pods view.
// This is triggered by the ']' key when this pane is active.
func (v *DeploymentsPodsView) ShowPods() {
	if !v.showPods {
		v.showPods = true
		// Reset cursor when toggling views
		v.podsView.SetCursor(0)
	}
}

// IsShowingPods returns true if currently showing pods, false for deployments.
func (v *DeploymentsPodsView) IsShowingPods() bool {
	return v.showPods
}

// GetTitle returns the appropriate title based on current view.
// Returns "Deployments" or "Pods".
func (v *DeploymentsPodsView) GetTitle() string {
	if v.showPods {
		return "Pods"
	}
	return "Deployments"
}

// View returns the rendered content from the active sub-view.
func (v *DeploymentsPodsView) View() tea.View {
	return tea.NewView(v.Render())
}

// Render returns the string representation for composition.
func (v *DeploymentsPodsView) Render() string {
	if v.showPods {
		return v.podsView.Render()
	}
	return v.deploymentsView.Render()
}

// SetSize forwards the size to both sub-views.
func (v *DeploymentsPodsView) SetSize(width, height int) {
	v.deploymentsView.SetSize(width, height)
	v.podsView.SetSize(width, height)
}

// GetCursor returns the cursor position from the active sub-view.
func (v *DeploymentsPodsView) GetCursor() int {
	if v.showPods {
		return v.podsView.GetCursor()
	}
	return v.deploymentsView.GetCursor()
}

// SetCursor sets the cursor position on the active sub-view.
func (v *DeploymentsPodsView) SetCursor(pos int) {
	if v.showPods {
		v.podsView.SetCursor(pos)
	} else {
		v.deploymentsView.SetCursor(pos)
	}
}

// Count returns the count from the active sub-view.
func (v *DeploymentsPodsView) Count() int {
	if v.showPods {
		return v.podsView.Count()
	}
	return v.deploymentsView.Count()
}

// SetDeployments updates the deployments in the deployments sub-view.
func (v *DeploymentsPodsView) SetDeployments(deployments []k8s.DeploymentInfo) {
	v.deploymentsView.SetDeployments(deployments)
}

// SetPods updates the pods in the pods sub-view.
func (v *DeploymentsPodsView) SetPods(pods []k8s.PodInfo) {
	v.podsView.SetPods(pods)
}

// SetNodeFilter sets the node filter on the pods sub-view.
func (v *DeploymentsPodsView) SetNodeFilter(nodeFilter string) {
	v.podsView.SetNodeFilter(nodeFilter)
}

// GetNodeFilter returns the node filter from the pods sub-view.
func (v *DeploymentsPodsView) GetNodeFilter() string {
	return v.podsView.GetNodeFilter()
}

// DeploymentsCount returns the count from deployments view (useful for badges).
func (v *DeploymentsPodsView) DeploymentsCount() int {
	return v.deploymentsView.Count()
}

// PodsCount returns the count from pods view (useful for badges).
func (v *DeploymentsPodsView) PodsCount() int {
	return v.podsView.Count()
}

// GetSelectedDeployment returns the selected deployment if showing deployments.
func (v *DeploymentsPodsView) GetSelectedDeployment() *k8s.DeploymentInfo {
	if v.showPods {
		return nil
	}
	return v.deploymentsView.GetSelectedDeployment()
}

// GetSelectedPod returns the selected pod if showing pods.
func (v *DeploymentsPodsView) GetSelectedPod() *k8s.PodInfo {
	if !v.showPods {
		return nil
	}
	return v.podsView.GetSelectedPod()
}

// SetGroupByType enables or disables grouping by type in deployments view.
func (v *DeploymentsPodsView) SetGroupByType(group bool) {
	v.deploymentsView.SetGroupByType(group)
}

// GetDeploymentsView returns the underlying DeploymentsView.
// This is useful for direct access when needed.
func (v *DeploymentsPodsView) GetDeploymentsView() *DeploymentsView {
	return v.deploymentsView
}

// GetPodsView returns the underlying PodsView.
// This is useful for direct access when needed.
func (v *DeploymentsPodsView) GetPodsView() *PodsView {
	return v.podsView
}

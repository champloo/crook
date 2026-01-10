// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
)

// ResourceType represents the type of resource being displayed
type ResourceType string

const (
	ResourceTypeNode       ResourceType = "node"
	ResourceTypeDeployment ResourceType = "deployment"
	ResourceTypeOSD        ResourceType = "osd"
	ResourceTypePod        ResourceType = "pod"
)

// RelatedResource represents a related resource shown in the detail panel
type RelatedResource struct {
	Type   string
	Name   string
	Status string
}

// DetailPanel displays detailed information about a selected resource
type DetailPanel struct {
	// resourceType is the type of resource being displayed
	resourceType ResourceType

	// resource holds the resource data (type depends on resourceType)
	resource interface{}

	// relatedResources lists resources related to the selected resource
	relatedResources []RelatedResource

	// scrollOffset is the current scroll position
	scrollOffset int

	// maxScroll is the maximum scroll offset
	maxScroll int

	// width is the panel width
	width int

	// height is the panel height
	height int

	// visible indicates if the panel is visible
	visible bool

	// content is the rendered content (cached)
	content []string
}

// NewDetailPanel creates a new detail panel
func NewDetailPanel() *DetailPanel {
	return &DetailPanel{
		relatedResources: make([]RelatedResource, 0),
		content:          make([]string, 0),
	}
}

// DetailCloseMsg is sent when the detail panel is closed
type DetailCloseMsg struct{}

// Init implements tea.Model
func (d *DetailPanel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (d *DetailPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			d.visible = false
			return d, func() tea.Msg { return DetailCloseMsg{} }

		case "j", "down":
			if d.scrollOffset < d.maxScroll {
				d.scrollOffset++
			}

		case "k", "up":
			if d.scrollOffset > 0 {
				d.scrollOffset--
			}

		case "g":
			d.scrollOffset = 0

		case "G":
			d.scrollOffset = d.maxScroll
		}

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.renderContent()
	}

	return d, nil
}

// View implements tea.Model
func (d *DetailPanel) View() tea.View {
	return tea.NewView(d.Render())
}

// Render returns the string representation for composition
func (d *DetailPanel) Render() string {
	if !d.visible {
		return ""
	}

	return d.renderPanel()
}

// ShowNode displays node details
func (d *DetailPanel) ShowNode(node k8s.NodeInfo, related []RelatedResource) {
	d.resourceType = ResourceTypeNode
	d.resource = node
	d.relatedResources = related
	d.scrollOffset = 0
	d.visible = true
	d.renderContent()
}

// ShowDeployment displays deployment details
func (d *DetailPanel) ShowDeployment(dep k8s.DeploymentInfo, related []RelatedResource) {
	d.resourceType = ResourceTypeDeployment
	d.resource = dep
	d.relatedResources = related
	d.scrollOffset = 0
	d.visible = true
	d.renderContent()
}

// ShowOSD displays OSD details
func (d *DetailPanel) ShowOSD(osd k8s.OSDInfo, related []RelatedResource) {
	d.resourceType = ResourceTypeOSD
	d.resource = osd
	d.relatedResources = related
	d.scrollOffset = 0
	d.visible = true
	d.renderContent()
}

// ShowPod displays pod details
func (d *DetailPanel) ShowPod(pod k8s.PodInfo, related []RelatedResource) {
	d.resourceType = ResourceTypePod
	d.resource = pod
	d.relatedResources = related
	d.scrollOffset = 0
	d.visible = true
	d.renderContent()
}

// Hide hides the detail panel
func (d *DetailPanel) Hide() {
	d.visible = false
}

// IsVisible returns whether the panel is visible
func (d *DetailPanel) IsVisible() bool {
	return d.visible
}

// SetSize sets the panel dimensions
func (d *DetailPanel) SetSize(width, height int) {
	d.width = width
	d.height = height
	if d.visible {
		d.renderContent()
	}
}

// renderContent renders the content based on resource type
func (d *DetailPanel) renderContent() {
	d.content = make([]string, 0)

	switch d.resourceType {
	case ResourceTypeNode:
		d.renderNodeContent()
	case ResourceTypeDeployment:
		d.renderDeploymentContent()
	case ResourceTypeOSD:
		d.renderOSDContent()
	case ResourceTypePod:
		d.renderPodContent()
	}

	// Calculate max scroll
	visibleLines := d.height - 6 // Account for border, title, and footer
	if visibleLines < 1 {
		visibleLines = 1
	}
	d.maxScroll = len(d.content) - visibleLines
	if d.maxScroll < 0 {
		d.maxScroll = 0
	}
}

// renderNodeContent renders node detail content
func (d *DetailPanel) renderNodeContent() {
	node, ok := d.resource.(k8s.NodeInfo)
	if !ok {
		d.content = append(d.content, "Error: invalid node data")
		return
	}

	d.addSection("Node Information")
	d.addField("Name", node.Name)
	d.addField("Status", d.colorStatus(node.Status, "Ready"))
	d.addField("Kubelet Version", node.KubeletVersion)

	if len(node.Roles) > 0 {
		d.addField("Roles", strings.Join(node.Roles, ", "))
	} else {
		d.addField("Roles", "<none>")
	}

	if node.Cordoned {
		d.addField("Schedulable", styles.StyleWarning.Render("Cordoned (Unschedulable)"))
	} else {
		d.addField("Schedulable", styles.StyleSuccess.Render("Ready"))
	}

	d.addField("Ceph Pods", fmt.Sprintf("%d", node.CephPodCount))
	d.addField("Age", node.Age)

	d.content = append(d.content, "")
	d.addRelatedResources()
}

// renderDeploymentContent renders deployment detail content
func (d *DetailPanel) renderDeploymentContent() {
	dep, ok := d.resource.(k8s.DeploymentInfo)
	if !ok {
		d.content = append(d.content, "Error: invalid deployment data")
		return
	}

	d.addSection("Deployment Information")
	d.addField("Name", dep.Name)
	d.addField("Namespace", dep.Namespace)
	d.addField("Type", dep.Type)
	d.addField("Status", d.colorDeploymentStatus(dep.Status))

	// Replica status
	d.content = append(d.content, "")
	d.addSection("Replicas")
	d.addField("Desired", fmt.Sprintf("%d", dep.DesiredReplicas))
	d.addField("Ready", d.colorReplicaCount(dep.ReadyReplicas, dep.DesiredReplicas))

	if dep.OsdID != "" {
		d.content = append(d.content, "")
		d.addSection("Ceph OSD")
		d.addField("OSD ID", dep.OsdID)
	}

	if dep.NodeName != "" {
		d.content = append(d.content, "")
		d.addSection("Placement")
		d.addField("Node", dep.NodeName)
	}

	d.addField("Age", dep.Age)

	d.content = append(d.content, "")
	d.addRelatedResources()
}

// renderOSDContent renders OSD detail content
func (d *DetailPanel) renderOSDContent() {
	osd, ok := d.resource.(k8s.OSDInfo)
	if !ok {
		d.content = append(d.content, "Error: invalid OSD data")
		return
	}

	d.addSection("OSD Information")
	d.addField("OSD ID", fmt.Sprintf("%d", osd.ID))
	d.addField("Name", osd.Name)
	d.addField("Hostname", osd.Hostname)

	// Status
	d.content = append(d.content, "")
	d.addSection("Status")
	statusColor := styles.StyleSuccess
	if osd.Status != "up" {
		statusColor = styles.StyleError
	}
	d.addField("State", statusColor.Render(osd.Status))

	inOutColor := styles.StyleSuccess
	if osd.InOut != "in" {
		inOutColor = styles.StyleError
	}
	d.addField("In/Out", inOutColor.Render(osd.InOut))

	// Weight information
	d.content = append(d.content, "")
	d.addSection("CRUSH")
	d.addField("Device Class", osd.DeviceClass)
	d.addField("Weight", fmt.Sprintf("%.4f", osd.Weight))
	d.addField("Reweight", fmt.Sprintf("%.4f", osd.Reweight))

	if osd.PGCount > 0 {
		d.addField("Primary PGs", fmt.Sprintf("%d", osd.PGCount))
	}

	// K8s deployment mapping
	if osd.DeploymentName != "" {
		d.content = append(d.content, "")
		d.addSection("Kubernetes")
		d.addField("Deployment", osd.DeploymentName)
	}

	d.content = append(d.content, "")
	d.addRelatedResources()
}

// renderPodContent renders pod detail content
func (d *DetailPanel) renderPodContent() {
	pod, ok := d.resource.(k8s.PodInfo)
	if !ok {
		d.content = append(d.content, "Error: invalid pod data")
		return
	}

	d.addSection("Pod Information")
	d.addField("Name", pod.Name)
	d.addField("Namespace", pod.Namespace)
	d.addField("Type", pod.Type)
	d.addField("Status", d.colorPodStatus(pod.Status))

	// Container status
	d.content = append(d.content, "")
	d.addSection("Containers")
	d.addField("Ready", d.colorContainerCount(pod.ReadyContainers, pod.TotalContainers))
	d.addField("Total", fmt.Sprintf("%d", pod.TotalContainers))

	restartColor := styles.StyleNormal
	if pod.Restarts > 10 {
		restartColor = styles.StyleError
	} else if pod.Restarts > 5 {
		restartColor = styles.StyleWarning
	}
	d.addField("Restarts", restartColor.Render(fmt.Sprintf("%d", pod.Restarts)))

	// Placement
	d.content = append(d.content, "")
	d.addSection("Placement")
	d.addField("Node", pod.NodeName)
	if pod.IP != "" {
		d.addField("IP", pod.IP)
	}

	// Owner
	if pod.OwnerDeployment != "" {
		d.content = append(d.content, "")
		d.addSection("Owner")
		d.addField("Deployment", pod.OwnerDeployment)
	}

	d.addField("Age", pod.Age)

	d.content = append(d.content, "")
	d.addRelatedResources()
}

// addSection adds a section header
func (d *DetailPanel) addSection(title string) {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)
	d.content = append(d.content, headerStyle.Render("── "+title+" ──"))
}

// addField adds a key-value field
func (d *DetailPanel) addField(key, value string) {
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.ColorSubtle).
		Width(16)
	line := keyStyle.Render(key+":") + " " + value
	d.content = append(d.content, line)
}

// addRelatedResources adds the related resources section
func (d *DetailPanel) addRelatedResources() {
	if len(d.relatedResources) == 0 {
		return
	}

	d.addSection("Related Resources")
	for _, r := range d.relatedResources {
		statusStyle := styles.StyleNormal
		switch r.Status {
		case "Ready", "Running", "up", "in":
			statusStyle = styles.StyleSuccess
		case "NotReady", "Failed", "down", "out":
			statusStyle = styles.StyleError
		case "Pending", "Scaling":
			statusStyle = styles.StyleWarning
		}

		typeStyle := lipgloss.NewStyle().
			Foreground(styles.ColorSubtle).
			Width(12)

		line := typeStyle.Render(r.Type+":") + " " + r.Name
		if r.Status != "" {
			line += " " + statusStyle.Render("["+r.Status+"]")
		}
		d.content = append(d.content, line)
	}
}

// colorStatus returns a colored status string
func (d *DetailPanel) colorStatus(status, expected string) string {
	if status == expected {
		return styles.StyleSuccess.Render(status)
	}
	return styles.StyleError.Render(status)
}

// colorDeploymentStatus returns a colored deployment status
func (d *DetailPanel) colorDeploymentStatus(status string) string {
	switch status {
	case "Ready":
		return styles.StyleSuccess.Render(status)
	case "Scaling":
		return styles.StyleWarning.Render(status)
	case "Unavailable":
		return styles.StyleError.Render(status)
	default:
		return status
	}
}

// colorPodStatus returns a colored pod status
func (d *DetailPanel) colorPodStatus(status string) string {
	switch status {
	case "Running", "Succeeded":
		return styles.StyleSuccess.Render(status)
	case "Pending", "Unknown":
		return styles.StyleWarning.Render(status)
	case "Failed", "Error", "CrashLoopBackOff":
		return styles.StyleError.Render(status)
	default:
		return status
	}
}

// colorReplicaCount returns a colored replica count
func (d *DetailPanel) colorReplicaCount(ready, desired int32) string {
	str := fmt.Sprintf("%d/%d", ready, desired)
	if ready == 0 && desired > 0 {
		return styles.StyleError.Render(str)
	}
	if ready < desired {
		return styles.StyleWarning.Render(str)
	}
	return styles.StyleSuccess.Render(str)
}

// colorContainerCount returns a colored container count
func (d *DetailPanel) colorContainerCount(ready, total int) string {
	str := fmt.Sprintf("%d/%d", ready, total)
	if ready == 0 && total > 0 {
		return styles.StyleError.Render(str)
	}
	if ready < total {
		return styles.StyleWarning.Render(str)
	}
	return styles.StyleSuccess.Render(str)
}

// renderPanel renders the complete panel with borders
func (d *DetailPanel) renderPanel() string {
	// Calculate content area
	contentWidth := d.width - 4   // Border padding
	contentHeight := d.height - 4 // Border and title padding

	if contentWidth < 20 {
		contentWidth = 20
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	var b strings.Builder

	// Title
	title := d.getTitle()
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorHighlight)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Top border
	b.WriteString(styles.StyleSubtle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")

	// Calculate visible content with scroll
	startLine := d.scrollOffset
	endLine := startLine + contentHeight
	if endLine > len(d.content) {
		endLine = len(d.content)
	}

	// Render visible content
	for i := startLine; i < endLine; i++ {
		line := d.content[i]
		// Truncate if too wide (using display width for proper Unicode handling)
		line = format.TruncateWithEllipsis(line, contentWidth)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Pad remaining lines
	for i := endLine - startLine; i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Bottom border
	b.WriteString(styles.StyleSubtle.Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")

	// Footer with hints and scroll indicator
	footer := "j/k: scroll  g/G: top/bottom  q/Esc: close"
	if d.maxScroll > 0 {
		footer = fmt.Sprintf("(%d/%d) ", d.scrollOffset+1, d.maxScroll+1) + footer
	}
	b.WriteString(styles.StyleSubtle.Render(footer))

	return b.String()
}

// getTitle returns the title for the current resource type
func (d *DetailPanel) getTitle() string {
	switch d.resourceType {
	case ResourceTypeNode:
		if node, ok := d.resource.(k8s.NodeInfo); ok {
			return "Node: " + node.Name
		}
	case ResourceTypeDeployment:
		if dep, ok := d.resource.(k8s.DeploymentInfo); ok {
			return "Deployment: " + dep.Name
		}
	case ResourceTypeOSD:
		if osd, ok := d.resource.(k8s.OSDInfo); ok {
			return "OSD: " + osd.Name
		}
	case ResourceTypePod:
		if pod, ok := d.resource.(k8s.PodInfo); ok {
			return "Pod: " + pod.Name
		}
	}
	return "Details"
}

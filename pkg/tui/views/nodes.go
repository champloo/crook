// Package views provides view implementations for the ls command TUI.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NodesView displays cluster nodes with Ceph workload information
type NodesView struct {
	// nodes is the list of nodes to display
	nodes []k8s.NodeInfo

	// cursor is the currently selected row
	cursor int

	// width is the terminal width
	width int

	// height is the terminal height
	height int
}

type nodesColumnLayout struct {
	name     int
	status   int
	roles    int
	schedule int
	cephPods int
	age      int

	showRoles bool
	showAge   bool
}

// NewNodesView creates a new nodes view
func NewNodesView() *NodesView {
	return &NodesView{
		nodes: make([]k8s.NodeInfo, 0),
	}
}

// NodeSelectedMsg is sent when a node is selected
type NodeSelectedMsg struct {
	Node k8s.NodeInfo
}

// Init implements tea.Model
func (v *NodesView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (v *NodesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if v.cursor < len(v.nodes)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.nodes) > 0 {
				v.cursor = len(v.nodes) - 1
			}
		case "enter":
			if v.cursor >= 0 && v.cursor < len(v.nodes) {
				return v, func() tea.Msg {
					return NodeSelectedMsg{Node: v.nodes[v.cursor]}
				}
			}
		}
	}
	return v, nil
}

// View implements tea.Model
func (v *NodesView) View() string {
	if len(v.nodes) == 0 {
		return styles.StyleSubtle.Render("No nodes found")
	}

	var b strings.Builder

	// Header
	header := v.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Separator
	b.WriteString(styles.StyleSubtle.Render(strings.Repeat("─", v.getTableWidth())))
	b.WriteString("\n")

	// Calculate visible rows based on height
	visibleRows := max(1, v.height-4) // Account for header, separator, and padding

	// Calculate scroll offset
	startIdx := 0
	if v.cursor >= visibleRows {
		startIdx = v.cursor - visibleRows + 1
	}
	endIdx := startIdx + visibleRows
	if endIdx > len(v.nodes) {
		endIdx = len(v.nodes)
	}

	// Rows
	for i := startIdx; i < endIdx; i++ {
		node := v.nodes[i]
		row := v.renderRow(node, i == v.cursor)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(v.nodes) > visibleRows {
		scrollInfo := styles.StyleSubtle.Render(fmt.Sprintf("(%d/%d)", v.cursor+1, len(v.nodes)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

func truncateEllipsis(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if format.DisplayWidth(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	return format.Truncate(s, width-3) + "..."
}

func (v *NodesView) columnLayout() nodesColumnLayout {
	switch {
	case v.width >= 100:
		return nodesColumnLayout{
			name:      30,
			status:    10,
			roles:     20,
			schedule:  12,
			cephPods:  10,
			age:       10,
			showRoles: true,
			showAge:   true,
		}
	case v.width >= 82:
		return nodesColumnLayout{
			name:      24,
			status:    9,
			roles:     14,
			schedule:  11,
			cephPods:  8,
			age:       8,
			showRoles: true,
			showAge:   true,
		}
	case v.width >= 66:
		return nodesColumnLayout{
			name:      24,
			status:    9,
			schedule:  11,
			cephPods:  8,
			age:       8,
			showRoles: false,
			showAge:   true,
		}
	default:
		return nodesColumnLayout{
			name:      max(12, v.width-(8+10+6+6)-4),
			status:    8,
			schedule:  10,
			cephPods:  6,
			age:       6,
			showRoles: false,
			showAge:   true,
		}
	}
}

// renderHeader renders the table header
func (v *NodesView) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)

	layout := v.columnLayout()
	cols := []string{
		format.PadRight("NAME", layout.name),
		format.PadRight("STATUS", layout.status),
	}
	if layout.showRoles {
		cols = append(cols, format.PadRight("ROLES", layout.roles))
	}
	cols = append(cols, format.PadRight("SCHEDULE", layout.schedule))

	cephTitle := "CEPH"
	if layout.cephPods >= 9 {
		cephTitle = "CEPH PODS"
	}
	cols = append(cols, format.PadRight(cephTitle, layout.cephPods))

	if layout.showAge {
		cols = append(cols, format.PadRight("AGE", layout.age))
	}

	return headerStyle.Render(strings.Join(cols, " "))
}

// renderRow renders a single node row
func (v *NodesView) renderRow(node k8s.NodeInfo, selected bool) string {
	layout := v.columnLayout()

	// Determine styles based on node state
	var nameStyle, statusStyle, scheduleStyle lipgloss.Style

	if selected {
		nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorHighlight).
			Background(styles.ColorPrimary)
	} else {
		nameStyle = styles.StyleNormal
	}

	// Status style
	switch node.Status {
	case "Ready":
		statusStyle = styles.StyleSuccess
	case "NotReady":
		statusStyle = styles.StyleError
	default:
		statusStyle = styles.StyleWarning
	}

	// Schedule style
	if node.Cordoned {
		scheduleStyle = styles.StyleWarning
	} else {
		scheduleStyle = styles.StyleNormal
	}

	// Build row
	scheduleText := "Ready"
	if node.Cordoned {
		scheduleText = "Cordoned"
	}

	rolesText := strings.Join(node.Roles, ",")
	if rolesText == "" {
		rolesText = "<none>"
	}

	if layout.showRoles {
		rolesText = truncateEllipsis(rolesText, layout.roles)
	}

	cols := []string{
		nameStyle.Render(format.PadRight(node.Name, layout.name)),
		statusStyle.Render(format.PadRight(node.Status, layout.status)),
	}
	if layout.showRoles {
		cols = append(cols, styles.StyleNormal.Render(format.PadRight(rolesText, layout.roles)))
	}
	cols = append(cols,
		scheduleStyle.Render(format.PadRight(scheduleText, layout.schedule)),
		v.renderCephPodCount(node.CephPodCount, selected, layout.cephPods),
	)
	if layout.showAge {
		cols = append(cols, styles.StyleSubtle.Render(format.PadRight(formatAge(node.Age.Duration()), layout.age)))
	}

	return strings.Join(cols, " ")
}

// renderCephPodCount renders the Ceph pod count with appropriate styling
func (v *NodesView) renderCephPodCount(count int, selected bool, width int) string {
	countStr := fmt.Sprintf("%d", count)

	var style lipgloss.Style
	if selected {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorHighlight)
	} else if count > 0 {
		// Subtle highlight for nodes with Ceph pods
		style = styles.StyleStatus
	} else {
		style = styles.StyleSubtle
	}

	return style.Render(format.PadRight(countStr, width))
}

// getTableWidth returns the total table width
func (v *NodesView) getTableWidth() int {
	layout := v.columnLayout()
	width := layout.name + layout.status + layout.schedule + layout.cephPods
	cols := 4
	if layout.showRoles {
		width += layout.roles
		cols++
	}
	if layout.showAge {
		width += layout.age
		cols++
	}
	return width + max(0, cols-1)
}

// SetNodes updates the nodes list
func (v *NodesView) SetNodes(nodes []k8s.NodeInfo) {
	v.nodes = nodes
	// Reset cursor if out of bounds
	if v.cursor >= len(v.nodes) {
		v.cursor = len(v.nodes) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
}

// SetSize sets the view dimensions
func (v *NodesView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetCursor returns the current cursor position
func (v *NodesView) GetCursor() int {
	return v.cursor
}

// SetCursor sets the cursor position
func (v *NodesView) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(v.nodes) {
		v.cursor = cursor
	}
}

// SetCursorByName selects the row with the matching node name.
func (v *NodesView) SetCursorByName(name string) bool {
	for i, node := range v.nodes {
		if node.Name == name {
			v.cursor = i
			return true
		}
	}
	return false
}

// Count returns the number of nodes
func (v *NodesView) Count() int {
	return len(v.nodes)
}

// GetSelectedNode returns the currently selected node
func (v *NodesView) GetSelectedNode() *k8s.NodeInfo {
	if v.cursor >= 0 && v.cursor < len(v.nodes) {
		return &v.nodes[v.cursor]
	}
	return nil
}

// formatAge formats a duration as a human-readable age string
func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return fmt.Sprintf("%dd", days)
	}
	if days < 365 {
		return fmt.Sprintf("%dmo", days/30)
	}
	return fmt.Sprintf("%dy", days/365)
}

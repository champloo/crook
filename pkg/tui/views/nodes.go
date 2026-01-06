// Package views provides view implementations for the ls command TUI.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NodesView displays cluster nodes with Ceph workload information
type NodesView struct {
	// nodes is the list of nodes to display
	nodes []NodeInfo

	// cursor is the currently selected row
	cursor int

	// filter is the current filter string
	filter string

	// filtered is the filtered nodes list
	filtered []NodeInfo

	// width is the terminal width
	width int

	// height is the terminal height
	height int
}

// NewNodesView creates a new nodes view
func NewNodesView() *NodesView {
	return &NodesView{
		nodes:    make([]NodeInfo, 0),
		filtered: make([]NodeInfo, 0),
	}
}

// NodeSelectedMsg is sent when a node is selected
type NodeSelectedMsg struct {
	Node NodeInfo
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
			if v.cursor < len(v.filtered)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.filtered) > 0 {
				v.cursor = len(v.filtered) - 1
			}
		case "enter":
			if v.cursor >= 0 && v.cursor < len(v.filtered) {
				return v, func() tea.Msg {
					return NodeSelectedMsg{Node: v.filtered[v.cursor]}
				}
			}
		}
	}
	return v, nil
}

// View implements tea.Model
func (v *NodesView) View() string {
	if len(v.filtered) == 0 {
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
	visibleRows := v.height - 4 // Account for header, separator, and padding
	if visibleRows < 1 {
		visibleRows = len(v.filtered)
	}

	// Calculate scroll offset
	startIdx := 0
	if v.cursor >= visibleRows {
		startIdx = v.cursor - visibleRows + 1
	}
	endIdx := startIdx + visibleRows
	if endIdx > len(v.filtered) {
		endIdx = len(v.filtered)
	}

	// Rows
	for i := startIdx; i < endIdx; i++ {
		node := v.filtered[i]
		row := v.renderRow(node, i == v.cursor)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(v.filtered) > visibleRows {
		scrollInfo := styles.StyleSubtle.Render(fmt.Sprintf("(%d/%d)", v.cursor+1, len(v.filtered)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

// renderHeader renders the table header
func (v *NodesView) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)

	cols := []string{
		format.PadRight("NAME", 30),
		format.PadRight("STATUS", 10),
		format.PadRight("ROLES", 20),
		format.PadRight("SCHEDULE", 12),
		format.PadRight("CEPH PODS", 10),
		format.PadRight("AGE", 10),
	}

	return headerStyle.Render(strings.Join(cols, " "))
}

// renderRow renders a single node row
func (v *NodesView) renderRow(node NodeInfo, selected bool) string {
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

	// Truncate roles if too long
	if len(rolesText) > 18 {
		rolesText = rolesText[:15] + "..."
	}

	cols := []string{
		nameStyle.Render(format.PadRight(node.Name, 30)),
		statusStyle.Render(format.PadRight(node.Status, 10)),
		styles.StyleNormal.Render(format.PadRight(rolesText, 20)),
		scheduleStyle.Render(format.PadRight(scheduleText, 12)),
		v.renderCephPodCount(node.CephPodCount, selected),
		styles.StyleSubtle.Render(format.PadRight(formatAge(node.Age), 10)),
	}

	return strings.Join(cols, " ")
}

// renderCephPodCount renders the Ceph pod count with appropriate styling
func (v *NodesView) renderCephPodCount(count int, selected bool) string {
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

	return style.Render(format.PadRight(countStr, 10))
}

// getTableWidth returns the total table width
func (v *NodesView) getTableWidth() int {
	return 30 + 10 + 20 + 12 + 10 + 10 + 5 // column widths + spacing
}

// SetNodes updates the nodes list
func (v *NodesView) SetNodes(nodes []NodeInfo) {
	v.nodes = nodes
	v.applyFilter()
}

// SetFilter sets the filter string and applies it
func (v *NodesView) SetFilter(filter string) {
	v.filter = filter
	v.applyFilter()
}

// applyFilter filters nodes based on the current filter
func (v *NodesView) applyFilter() {
	if v.filter == "" {
		v.filtered = v.nodes
	} else {
		filterLower := strings.ToLower(v.filter)
		v.filtered = make([]NodeInfo, 0)
		for _, node := range v.nodes {
			if strings.Contains(strings.ToLower(node.Name), filterLower) {
				v.filtered = append(v.filtered, node)
			}
		}
	}

	// Reset cursor if out of bounds
	if v.cursor >= len(v.filtered) {
		v.cursor = len(v.filtered) - 1
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
	if cursor >= 0 && cursor < len(v.filtered) {
		v.cursor = cursor
	}
}

// Count returns the number of nodes (filtered)
func (v *NodesView) Count() int {
	return len(v.filtered)
}

// TotalCount returns the total number of nodes (unfiltered)
func (v *NodesView) TotalCount() int {
	return len(v.nodes)
}

// GetSelectedNode returns the currently selected node
func (v *NodesView) GetSelectedNode() *NodeInfo {
	if v.cursor >= 0 && v.cursor < len(v.filtered) {
		return &v.filtered[v.cursor]
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

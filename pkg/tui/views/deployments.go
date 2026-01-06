package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Column widths for the deployments table
const (
	iconPrefixWidth   = 2  // Space for warning icon + padding
	nameColWidth      = 33 // Deployment name
	namespaceColWidth = 15 // Namespace
	readyColWidth     = 8  // Ready status (X/Y)
	nodeColWidth      = 20 // Node name
	ageColWidth       = 8  // Age
	statusColWidth    = 12 // Status text
)

// DeploymentsView displays Rook-Ceph deployments with node mapping
type DeploymentsView struct {
	// deployments is the list of deployments to display
	deployments []DeploymentInfo

	// cursor is the currently selected row
	cursor int

	// filter is the current filter string
	filter string

	// filtered is the filtered deployments list
	filtered []DeploymentInfo

	// groupByType controls whether to group deployments by type
	groupByType bool

	// width is the terminal width
	width int

	// height is the terminal height
	height int
}

// NewDeploymentsView creates a new deployments view
func NewDeploymentsView() *DeploymentsView {
	return &DeploymentsView{
		deployments: make([]DeploymentInfo, 0),
		filtered:    make([]DeploymentInfo, 0),
		groupByType: true,
	}
}

// DeploymentSelectedMsg is sent when a deployment is selected
type DeploymentSelectedMsg struct {
	Deployment DeploymentInfo
}

// Init implements tea.Model
func (v *DeploymentsView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (v *DeploymentsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
					return DeploymentSelectedMsg{Deployment: v.filtered[v.cursor]}
				}
			}
		}
	}
	return v, nil
}

// View implements tea.Model
func (v *DeploymentsView) View() string {
	if len(v.filtered) == 0 {
		return styles.StyleSubtle.Render("No deployments found")
	}

	var b strings.Builder

	// Header
	header := v.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Separator
	b.WriteString(styles.StyleSubtle.Render(strings.Repeat("─", v.getTableWidth())))
	b.WriteString("\n")

	// Calculate visible rows
	visibleRows := v.height - 4
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

	// Group and render
	if v.groupByType {
		b.WriteString(v.renderGrouped(startIdx, endIdx))
	} else {
		for i := startIdx; i < endIdx; i++ {
			b.WriteString(v.renderRow(v.filtered[i], i == v.cursor))
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(v.filtered) > visibleRows {
		scrollInfo := styles.StyleSubtle.Render(fmt.Sprintf("(%d/%d)", v.cursor+1, len(v.filtered)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

// renderHeader renders the table header
func (v *DeploymentsView) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)

	// Reserve space for icon prefix (icon + space)
	cols := []string{
		format.PadRight("", iconPrefixWidth),
		format.PadRight("NAME", nameColWidth),
		format.PadRight("NAMESPACE", namespaceColWidth),
		format.PadRight("READY", readyColWidth),
		format.PadRight("NODE", nodeColWidth),
		format.PadRight("AGE", ageColWidth),
		format.PadRight("STATUS", statusColWidth),
	}

	return headerStyle.Render(strings.Join(cols, " "))
}

// renderGrouped renders deployments grouped by type
func (v *DeploymentsView) renderGrouped(startIdx, endIdx int) string {
	var b strings.Builder

	// Group deployments by type
	groups := make(map[string][]int)
	typeOrder := []string{"osd", "mon", "mgr", "mds", "rgw", "exporter", "crashcollector", "other"}

	for i, d := range v.filtered {
		typ := d.Type
		if typ == "" {
			typ = "other"
		}
		groups[typ] = append(groups[typ], i)
	}

	// Track current position for cursor highlighting
	displayIdx := 0

	for _, typ := range typeOrder {
		indices, ok := groups[typ]
		if !ok || len(indices) == 0 {
			continue
		}

		// Check if any items in this group are visible
		groupVisible := false
		for _, idx := range indices {
			if idx >= startIdx && idx < endIdx {
				groupVisible = true
				break
			}
		}

		if !groupVisible {
			displayIdx += len(indices)
			continue
		}

		// Group header with subtle background
		headerText := fmt.Sprintf(" %s (%d) ", strings.ToUpper(typ), len(indices))
		// Pad to full table width for consistent background
		tableWidth := v.getTableWidth()
		padding := tableWidth - lipgloss.Width(headerText)
		if padding > 0 {
			headerText += strings.Repeat(" ", padding)
		}
		b.WriteString(styles.StyleGroupHeader.Render(headerText))
		b.WriteString("\n")

		// Render items in this group
		for _, idx := range indices {
			if idx >= startIdx && idx < endIdx {
				b.WriteString(v.renderRow(v.filtered[idx], idx == v.cursor))
				b.WriteString("\n")
			}
			displayIdx++
		}
	}

	return b.String()
}

// renderRow renders a single deployment row
func (v *DeploymentsView) renderRow(dep DeploymentInfo, selected bool) string {
	var nameStyle, statusStyle lipgloss.Style

	if selected {
		nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorHighlight).
			Background(styles.ColorPrimary)
	} else {
		nameStyle = styles.StyleNormal
	}

	// Status style
	switch dep.Status {
	case "Ready":
		statusStyle = styles.StyleSuccess
	case "Scaling":
		statusStyle = styles.StyleWarning
	case "Unavailable":
		statusStyle = styles.StyleError
	default:
		statusStyle = styles.StyleNormal
	}

	// Ready column format: X/Y
	readyStr := fmt.Sprintf("%d/%d", dep.ReadyReplicas, dep.DesiredReplicas)
	var readyStyle lipgloss.Style
	if dep.ReadyReplicas == 0 && dep.DesiredReplicas > 0 {
		readyStyle = styles.StyleError
	} else if dep.ReadyReplicas < dep.DesiredReplicas {
		readyStyle = styles.StyleWarning
	} else {
		readyStyle = styles.StyleSuccess
	}

	// Icon prefix column - always reserve space, show warning icon if scaled down
	iconPrefix := "  " // Default: empty space (2 chars to match icon width)
	if dep.DesiredReplicas == 0 {
		iconPrefix = styles.IconWarning + " "
	}

	// Truncate node name if needed
	nodeName := dep.NodeName
	if nodeName == "" {
		nodeName = "<none>"
	}
	if len(nodeName) > 18 {
		nodeName = nodeName[:15] + "..."
	}

	cols := []string{
		styles.StyleWarning.Render(iconPrefix),
		nameStyle.Render(format.PadRight(dep.Name, nameColWidth)),
		styles.StyleSubtle.Render(format.PadRight(dep.Namespace, namespaceColWidth)),
		readyStyle.Render(format.PadRight(readyStr, readyColWidth)),
		styles.StyleNormal.Render(format.PadRight(nodeName, nodeColWidth)),
		styles.StyleSubtle.Render(format.PadRight(formatAge(dep.Age), ageColWidth)),
		statusStyle.Render(format.PadRight(dep.Status, statusColWidth)),
	}

	return strings.Join(cols, " ")
}

// getTableWidth returns the total table width
func (v *DeploymentsView) getTableWidth() int {
	// icon + name + namespace + ready + node + age + status + spacing between columns
	return iconPrefixWidth + nameColWidth + namespaceColWidth + readyColWidth + nodeColWidth + ageColWidth + statusColWidth + 6
}

// SetDeployments updates the deployments list
func (v *DeploymentsView) SetDeployments(deployments []DeploymentInfo) {
	v.deployments = deployments
	v.applyFilter()
}

// SetFilter sets the filter string and applies it
func (v *DeploymentsView) SetFilter(filter string) {
	v.filter = filter
	v.applyFilter()
}

// applyFilter filters deployments based on the current filter
func (v *DeploymentsView) applyFilter() {
	if v.filter == "" {
		v.filtered = v.deployments
	} else {
		filterLower := strings.ToLower(v.filter)
		v.filtered = make([]DeploymentInfo, 0)
		for _, dep := range v.deployments {
			if strings.Contains(strings.ToLower(dep.Name), filterLower) ||
				strings.Contains(strings.ToLower(dep.NodeName), filterLower) ||
				strings.Contains(strings.ToLower(dep.Type), filterLower) {
				v.filtered = append(v.filtered, dep)
			}
		}
	}

	// Sort by type then name for consistent grouping
	if v.groupByType {
		sort.Slice(v.filtered, func(i, j int) bool {
			if v.filtered[i].Type != v.filtered[j].Type {
				return typeOrder(v.filtered[i].Type) < typeOrder(v.filtered[j].Type)
			}
			return v.filtered[i].Name < v.filtered[j].Name
		})
	}

	// Reset cursor if out of bounds
	if v.cursor >= len(v.filtered) {
		v.cursor = len(v.filtered) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
}

// typeOrder returns a sort order for deployment types
func typeOrder(t string) int {
	order := map[string]int{
		"osd":            0,
		"mon":            1,
		"mgr":            2,
		"mds":            3,
		"rgw":            4,
		"exporter":       5,
		"crashcollector": 6,
	}
	if o, ok := order[t]; ok {
		return o
	}
	return 99
}

// SetSize sets the view dimensions
func (v *DeploymentsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetCursor returns the current cursor position
func (v *DeploymentsView) GetCursor() int {
	return v.cursor
}

// SetCursor sets the cursor position
func (v *DeploymentsView) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(v.filtered) {
		v.cursor = cursor
	}
}

// Count returns the number of deployments (filtered)
func (v *DeploymentsView) Count() int {
	return len(v.filtered)
}

// TotalCount returns the total number of deployments (unfiltered)
func (v *DeploymentsView) TotalCount() int {
	return len(v.deployments)
}

// GetSelectedDeployment returns the currently selected deployment
func (v *DeploymentsView) GetSelectedDeployment() *DeploymentInfo {
	if v.cursor >= 0 && v.cursor < len(v.filtered) {
		return &v.filtered[v.cursor]
	}
	return nil
}

// SetGroupByType enables or disables grouping by type
func (v *DeploymentsView) SetGroupByType(group bool) {
	v.groupByType = group
	v.applyFilter() // Re-sort
}

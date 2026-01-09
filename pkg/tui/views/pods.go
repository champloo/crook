package views

import (
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PodsView displays Rook-Ceph pods with ownership information
type PodsView struct {
	// pods is the list of pods to display (may be filtered by node)
	pods []k8s.PodInfo

	// allPods stores all pods before node filtering
	allPods []k8s.PodInfo

	// cursor is the currently selected row
	cursor int

	// nodeFilter filters pods to a specific node
	nodeFilter string

	// width is the terminal width
	width int

	// height is the terminal height
	height int
}

// NewPodsView creates a new pods view
func NewPodsView() *PodsView {
	return &PodsView{
		pods:    make([]k8s.PodInfo, 0),
		allPods: make([]k8s.PodInfo, 0),
	}
}

// PodSelectedMsg is sent when a pod is selected
type PodSelectedMsg struct {
	Pod k8s.PodInfo
}

// Init implements tea.Model
func (v *PodsView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (v *PodsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if v.cursor < len(v.pods)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.pods) > 0 {
				v.cursor = len(v.pods) - 1
			}
		case "enter":
			if v.cursor >= 0 && v.cursor < len(v.pods) {
				return v, func() tea.Msg {
					return PodSelectedMsg{Pod: v.pods[v.cursor]}
				}
			}
		}
	}
	return v, nil
}

// View implements tea.Model
func (v *PodsView) View() string {
	if len(v.pods) == 0 {
		return styles.StyleSubtle.Render("No pods found")
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
		visibleRows = len(v.pods)
	}

	// Calculate scroll offset
	startIdx := 0
	if v.cursor >= visibleRows {
		startIdx = v.cursor - visibleRows + 1
	}
	endIdx := startIdx + visibleRows
	if endIdx > len(v.pods) {
		endIdx = len(v.pods)
	}

	// Rows
	for i := startIdx; i < endIdx; i++ {
		pod := v.pods[i]
		row := v.renderRow(pod, i == v.cursor)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(v.pods) > visibleRows {
		scrollInfo := styles.StyleSubtle.Render(fmt.Sprintf("(%d/%d)", v.cursor+1, len(v.pods)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

// renderHeader renders the table header
func (v *PodsView) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)

	cols := []string{
		format.PadRight("NAME", 40),
		format.PadRight("NAMESPACE", 15),
		format.PadRight("NODE", 20),
		format.PadRight("STATUS", 12),
		format.PadRight("READY", 8),
		format.PadRight("RESTARTS", 10),
		format.PadRight("AGE", 8),
	}

	return headerStyle.Render(strings.Join(cols, " "))
}

// renderRow renders a single pod row
func (v *PodsView) renderRow(pod k8s.PodInfo, selected bool) string {
	var nameStyle, statusStyle, readyStyle, restartStyle lipgloss.Style

	if selected {
		nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorHighlight).
			Background(styles.ColorPrimary)
	} else {
		nameStyle = styles.StyleNormal
	}

	// Status style based on pod phase
	switch pod.Status {
	case "Running":
		statusStyle = styles.StyleSuccess
	case "Pending":
		statusStyle = styles.StyleWarning
	case "Failed", "Error", "CrashLoopBackOff":
		statusStyle = styles.StyleError
	case "Succeeded":
		statusStyle = styles.StyleSuccess
	case "Unknown":
		statusStyle = styles.StyleWarning
	default:
		statusStyle = styles.StyleNormal
	}

	// Ready column format: X/Y
	readyStr := fmt.Sprintf("%d/%d", pod.ReadyContainers, pod.TotalContainers)
	if pod.TotalContainers == 0 {
		readyStyle = styles.StyleSubtle
	} else if pod.ReadyContainers == 0 {
		readyStyle = styles.StyleError
	} else if pod.ReadyContainers < pod.TotalContainers {
		readyStyle = styles.StyleWarning
	} else {
		readyStyle = styles.StyleSuccess
	}

	// Restart count styling (highlight if high)
	restartStr := fmt.Sprintf("%d", pod.Restarts)
	if pod.Restarts > 10 {
		restartStyle = styles.StyleError
	} else if pod.Restarts > 5 {
		restartStyle = styles.StyleWarning
	} else {
		restartStyle = styles.StyleSubtle
	}

	// Warning indicators
	hasWarning := pod.Status != "Running" || pod.Restarts > 5

	// Truncate name if needed (using display width for proper Unicode handling)
	nameDisplay := format.TruncateWithEllipsis(pod.Name, 38)

	// Truncate node name if needed (using display width for proper Unicode handling)
	nodeName := pod.NodeName
	if nodeName == "" {
		nodeName = "<none>"
	}
	nodeName = format.TruncateWithEllipsis(nodeName, 18)

	cols := []string{
		nameStyle.Render(format.PadRight(nameDisplay, 40)),
		styles.StyleSubtle.Render(format.PadRight(pod.Namespace, 15)),
		v.renderWithWarning(format.PadRight(nodeName, 20), hasWarning, selected),
		statusStyle.Render(format.PadRight(pod.Status, 12)),
		readyStyle.Render(format.PadRight(readyStr, 8)),
		restartStyle.Render(format.PadRight(restartStr, 10)),
		styles.StyleSubtle.Render(format.PadRight(pod.Age, 8)),
	}

	return strings.Join(cols, " ")
}

// renderWithWarning renders text with warning style if needed
func (v *PodsView) renderWithWarning(s string, warning, selected bool) string {
	if selected {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorHighlight).
			Render(s)
	}
	if warning {
		return styles.StyleWarning.Render(s)
	}
	return styles.StyleNormal.Render(s)
}

// getTableWidth returns the total table width
func (v *PodsView) getTableWidth() int {
	return 40 + 15 + 20 + 12 + 8 + 10 + 8 + 6 // column widths + spacing
}

// SetPods updates the pods list
func (v *PodsView) SetPods(pods []k8s.PodInfo) {
	v.allPods = pods
	v.applyNodeFilter()
}

// SetNodeFilter sets the node filter for filtering pods by node
func (v *PodsView) SetNodeFilter(nodeFilter string) {
	v.nodeFilter = nodeFilter
	v.applyNodeFilter()
}

// applyNodeFilter filters pods based on the node filter
func (v *PodsView) applyNodeFilter() {
	if v.nodeFilter == "" {
		v.pods = v.allPods
	} else {
		v.pods = make([]k8s.PodInfo, 0, len(v.allPods))
		for _, pod := range v.allPods {
			if pod.NodeName == v.nodeFilter {
				v.pods = append(v.pods, pod)
			}
		}
	}

	// Reset cursor if out of bounds
	if v.cursor >= len(v.pods) {
		v.cursor = len(v.pods) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
}

// SetSize sets the view dimensions
func (v *PodsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetCursor returns the current cursor position
func (v *PodsView) GetCursor() int {
	return v.cursor
}

// SetCursor sets the cursor position
func (v *PodsView) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(v.pods) {
		v.cursor = cursor
	}
}

// Count returns the number of pods (may be filtered by node)
func (v *PodsView) Count() int {
	return len(v.pods)
}

// TotalCount returns the total number of pods (before node filtering)
func (v *PodsView) TotalCount() int {
	return len(v.allPods)
}

// GetSelectedPod returns the currently selected pod
func (v *PodsView) GetSelectedPod() *k8s.PodInfo {
	if v.cursor >= 0 && v.cursor < len(v.pods) {
		return &v.pods[v.cursor]
	}
	return nil
}

// GetNodeFilter returns the current node filter
func (v *PodsView) GetNodeFilter() string {
	return v.nodeFilter
}

// CountByStatus returns the number of pods by status
func (v *PodsView) CountByStatus(status string) int {
	count := 0
	for _, pod := range v.pods {
		if pod.Status == status {
			count++
		}
	}
	return count
}

// CountHighRestarts returns the number of pods with high restart counts (>5)
func (v *PodsView) CountHighRestarts() int {
	count := 0
	for _, pod := range v.pods {
		if pod.Restarts > 5 {
			count++
		}
	}
	return count
}

package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/format"
	"golang.org/x/term"
)

// TableWriter writes data as a formatted ASCII table
type TableWriter struct {
	w      io.Writer
	color  bool
	width  int
	indent string
}

// NewTableWriter creates a new table writer
func NewTableWriter(w io.Writer) *TableWriter {
	tw := &TableWriter{
		w:      w,
		color:  isTerminal(w),
		width:  80,
		indent: "",
	}

	// Try to get terminal width
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			tw.width = width
		}
	}

	return tw
}

// isTerminal checks if the writer is a terminal
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// Write writes the data as a formatted table
func (tw *TableWriter) Write(data *Data) error {
	// Cluster health header
	if data.ClusterHealth != nil {
		tw.writeClusterHealth(data.ClusterHealth)
		_, _ = fmt.Fprintln(tw.w)
	}

	// Nodes
	if len(data.Nodes) > 0 {
		tw.writeSectionHeader("NODES", len(data.Nodes))
		tw.writeNodesTable(data.Nodes)
		_, _ = fmt.Fprintln(tw.w)
	}

	// Deployments
	if len(data.Deployments) > 0 {
		tw.writeSectionHeader("DEPLOYMENTS", len(data.Deployments))
		tw.writeDeploymentsTable(data.Deployments)
		_, _ = fmt.Fprintln(tw.w)
	}

	// OSDs
	if len(data.OSDs) > 0 {
		tw.writeSectionHeader("OSDS", len(data.OSDs))
		tw.writeOSDsTable(data.OSDs)
		_, _ = fmt.Fprintln(tw.w)
	}

	// Pods
	if len(data.Pods) > 0 {
		tw.writeSectionHeader("PODS", len(data.Pods))
		tw.writePodsTable(data.Pods)
		_, _ = fmt.Fprintln(tw.w)
	}

	return nil
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// writeClusterHealth writes the cluster health summary
func (tw *TableWriter) writeClusterHealth(health *ClusterHealth) {
	// Health status
	healthColor := colorGreen
	switch health.Status {
	case "HEALTH_ERR":
		healthColor = colorRed
	case "HEALTH_WARN":
		healthColor = colorYellow
	}

	_, _ = fmt.Fprintf(tw.w, "Ceph: %s\n", tw.colorize(health.Status, healthColor))

	// OSD stats
	osdUpColor := colorGreen
	if health.OSDsUp < health.OSDs {
		osdUpColor = colorYellow
	}
	osdInColor := colorGreen
	if health.OSDsIn < health.OSDs {
		osdInColor = colorYellow
	}

	_, _ = fmt.Fprintf(tw.w, "OSDs: %s up, %s in\n",
		tw.colorize(fmt.Sprintf("%d/%d", health.OSDsUp, health.OSDs), osdUpColor),
		tw.colorize(fmt.Sprintf("%d/%d", health.OSDsIn, health.OSDs), osdInColor))

	// Monitor stats
	monColor := colorGreen
	if health.MonsInQuorum < health.MonsTotal {
		monColor = colorYellow
	}
	if health.MonsInQuorum <= health.MonsTotal/2 {
		monColor = colorRed
	}

	_, _ = fmt.Fprintf(tw.w, "MONs: %s in quorum\n",
		tw.colorize(fmt.Sprintf("%d/%d", health.MonsInQuorum, health.MonsTotal), monColor))

	// Noout flag
	if health.NooutSet {
		_, _ = fmt.Fprintf(tw.w, "Flags: %s\n", tw.colorize("noout SET", colorYellow))
	}

	// Storage usage
	if health.TotalBytes > 0 {
		percent := float64(health.UsedBytes) / float64(health.TotalBytes) * 100
		usageColor := colorGreen
		if percent >= 85 {
			usageColor = colorRed
		} else if percent >= 70 {
			usageColor = colorYellow
		}

		_, _ = fmt.Fprintf(tw.w, "Storage: %s / %s (%s)\n",
			format.FormatBytes(health.UsedBytes),
			format.FormatBytes(health.TotalBytes),
			tw.colorize(format.FormatPercent(percent), usageColor))
	}
}

// writeSectionHeader writes a section header
func (tw *TableWriter) writeSectionHeader(title string, count int) {
	header := fmt.Sprintf("=== %s (%d) ===", title, count)
	_, _ = fmt.Fprintln(tw.w, tw.colorize(header, colorBold+colorCyan))
}

// writeNodesTable writes the nodes table
func (tw *TableWriter) writeNodesTable(nodes []k8s.NodeInfo) {
	// Column widths
	cols := []column{
		{header: "NAME", width: 30},
		{header: "STATUS", width: 10},
		{header: "ROLES", width: 20},
		{header: "SCHEDULE", width: 12},
		{header: "CEPH PODS", width: 10},
		{header: "AGE", width: 8},
	}

	tw.writeTableHeader(cols)
	tw.writeTableSeparator(cols)

	for _, node := range nodes {
		rolesStr := "<none>"
		if len(node.Roles) > 0 {
			rolesStr = strings.Join(node.Roles, ",")
		}
		if len(rolesStr) > 18 {
			rolesStr = rolesStr[:15] + "..."
		}

		scheduleStr := "Ready"
		scheduleColor := ""
		if node.Cordoned {
			scheduleStr = "Cordoned"
			scheduleColor = colorYellow
		}

		statusColor := colorGreen
		if node.Status == "NotReady" {
			statusColor = colorRed
		} else if node.Status != "Ready" {
			statusColor = colorYellow
		}

		row := []cell{
			{value: node.Name},
			{value: node.Status, color: statusColor},
			{value: rolesStr},
			{value: scheduleStr, color: scheduleColor},
			{value: fmt.Sprintf("%d", node.CephPodCount)},
			{value: node.Age.String()},
		}
		tw.writeTableRow(cols, row)
	}
}

// writeDeploymentsTable writes the deployments table
func (tw *TableWriter) writeDeploymentsTable(deployments []k8s.DeploymentInfo) {
	cols := []column{
		{header: "NAME", width: 35},
		{header: "NAMESPACE", width: 15},
		{header: "READY", width: 8},
		{header: "NODE", width: 20},
		{header: "AGE", width: 8},
		{header: "STATUS", width: 12},
	}

	tw.writeTableHeader(cols)
	tw.writeTableSeparator(cols)

	for _, dep := range deployments {
		readyStr := fmt.Sprintf("%d/%d", dep.ReadyReplicas, dep.DesiredReplicas)
		readyColor := colorGreen
		if dep.ReadyReplicas == 0 && dep.DesiredReplicas > 0 {
			readyColor = colorRed
		} else if dep.ReadyReplicas < dep.DesiredReplicas {
			readyColor = colorYellow
		}

		statusColor := colorGreen
		switch dep.Status {
		case "Scaling":
			statusColor = colorYellow
		case "Unavailable":
			statusColor = colorRed
		}

		nodeName := dep.NodeName
		if nodeName == "" {
			nodeName = "<none>"
		}
		if len(nodeName) > 18 {
			nodeName = nodeName[:15] + "..."
		}

		row := []cell{
			{value: dep.Name},
			{value: dep.Namespace},
			{value: readyStr, color: readyColor},
			{value: nodeName},
			{value: dep.Age.String()},
			{value: dep.Status, color: statusColor},
		}
		tw.writeTableRow(cols, row)
	}
}

// writeOSDsTable writes the OSDs table
func (tw *TableWriter) writeOSDsTable(osds []k8s.OSDInfo) {
	cols := []column{
		{header: "OSD", width: 10},
		{header: "HOST", width: 20},
		{header: "STATUS", width: 8},
		{header: "IN/OUT", width: 8},
		{header: "WEIGHT", width: 10},
		{header: "CLASS", width: 8},
		{header: "DEPLOYMENT", width: 30},
	}

	tw.writeTableHeader(cols)
	tw.writeTableSeparator(cols)

	for _, osd := range osds {
		statusColor := colorGreen
		if osd.Status != "up" {
			statusColor = colorRed
		}

		inOutColor := colorGreen
		if osd.InOut != "in" {
			inOutColor = colorRed
		}

		weightStr := fmt.Sprintf("%.3f", osd.Weight)

		deploymentName := osd.DeploymentName
		if len(deploymentName) > 28 {
			deploymentName = deploymentName[:25] + "..."
		}

		row := []cell{
			{value: osd.Name},
			{value: osd.Hostname},
			{value: osd.Status, color: statusColor},
			{value: osd.InOut, color: inOutColor},
			{value: weightStr},
			{value: osd.DeviceClass},
			{value: deploymentName},
		}
		tw.writeTableRow(cols, row)
	}
}

// writePodsTable writes the pods table
func (tw *TableWriter) writePodsTable(pods []k8s.PodInfo) {
	cols := []column{
		{header: "NAME", width: 40},
		{header: "NAMESPACE", width: 15},
		{header: "NODE", width: 20},
		{header: "STATUS", width: 12},
		{header: "READY", width: 8},
		{header: "RESTARTS", width: 10},
		{header: "AGE", width: 8},
	}

	tw.writeTableHeader(cols)
	tw.writeTableSeparator(cols)

	for _, pod := range pods {
		statusColor := colorGreen
		switch pod.Status {
		case "Pending":
			statusColor = colorYellow
		case "Failed", "Error", "CrashLoopBackOff":
			statusColor = colorRed
		case "Unknown":
			statusColor = colorYellow
		}

		readyStr := fmt.Sprintf("%d/%d", pod.ReadyContainers, pod.TotalContainers)
		readyColor := colorGreen
		if pod.ReadyContainers == 0 && pod.TotalContainers > 0 {
			readyColor = colorRed
		} else if pod.ReadyContainers < pod.TotalContainers {
			readyColor = colorYellow
		}

		restartColor := ""
		if pod.Restarts > 10 {
			restartColor = colorRed
		} else if pod.Restarts > 5 {
			restartColor = colorYellow
		}

		podName := pod.Name
		if len(podName) > 38 {
			podName = podName[:35] + "..."
		}

		nodeName := pod.NodeName
		if len(nodeName) > 18 {
			nodeName = nodeName[:15] + "..."
		}

		row := []cell{
			{value: podName},
			{value: pod.Namespace},
			{value: nodeName},
			{value: pod.Status, color: statusColor},
			{value: readyStr, color: readyColor},
			{value: fmt.Sprintf("%d", pod.Restarts), color: restartColor},
			{value: pod.Age.String()},
		}
		tw.writeTableRow(cols, row)
	}
}

// column defines a table column
type column struct {
	header string
	width  int
}

// cell defines a table cell
type cell struct {
	value string
	color string
}

// writeTableHeader writes the table header row
func (tw *TableWriter) writeTableHeader(cols []column) {
	parts := make([]string, len(cols))
	for i, col := range cols {
		parts[i] = tw.padRight(col.header, col.width)
	}
	_, _ = fmt.Fprintln(tw.w, tw.colorize(strings.Join(parts, " "), colorBold))
}

// writeTableSeparator writes a separator line
func (tw *TableWriter) writeTableSeparator(cols []column) {
	totalWidth := 0
	for _, col := range cols {
		totalWidth += col.width + 1
	}
	_, _ = fmt.Fprintln(tw.w, strings.Repeat("-", totalWidth-1))
}

// writeTableRow writes a table row
func (tw *TableWriter) writeTableRow(cols []column, cells []cell) {
	parts := make([]string, len(cols))
	for i, col := range cols {
		value := ""
		color := ""
		if i < len(cells) {
			value = cells[i].value
			color = cells[i].color
		}
		padded := tw.padRight(value, col.width)
		if color != "" {
			padded = tw.colorize(padded, color)
		}
		parts[i] = padded
	}
	_, _ = fmt.Fprintln(tw.w, strings.Join(parts, " "))
}

// padRight pads a string to the specified width
func (tw *TableWriter) padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// colorize adds ANSI color codes if color is enabled
func (tw *TableWriter) colorize(s, color string) string {
	if !tw.color || color == "" {
		return s
	}
	return color + s + colorReset
}

// RenderTable renders data to a table and writes to the given writer
func RenderTable(w io.Writer, data *Data) error {
	tw := NewTableWriter(w)
	return tw.Write(data)
}

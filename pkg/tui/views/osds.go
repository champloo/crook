package views

import (
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// OSDsView displays Ceph OSD status from ceph osd tree
type OSDsView struct {
	// osds is the list of OSDs to display
	osds []k8s.OSDInfo

	// cursor is the currently selected row
	cursor int

	// nooutSet indicates if the cluster noout flag is set
	nooutSet bool

	// width is the terminal width
	width int

	// height is the terminal height
	height int
}

// NewOSDsView creates a new OSDs view
func NewOSDsView() *OSDsView {
	return &OSDsView{
		osds: make([]k8s.OSDInfo, 0),
	}
}

// OSDSelectedMsg is sent when an OSD is selected
type OSDSelectedMsg struct {
	OSD k8s.OSDInfo
}

// Init implements tea.Model
func (v *OSDsView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (v *OSDsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if v.cursor < len(v.osds)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.osds) > 0 {
				v.cursor = len(v.osds) - 1
			}
		case "enter":
			if v.cursor >= 0 && v.cursor < len(v.osds) {
				return v, func() tea.Msg {
					return OSDSelectedMsg{OSD: v.osds[v.cursor]}
				}
			}
		}
	}
	return v, nil
}

// View implements tea.Model
func (v *OSDsView) View() tea.View {
	return tea.NewView(v.Render())
}

func (v *OSDsView) Render() string {
	if len(v.osds) == 0 {
		return styles.StyleSubtle.Render("No OSDs found")
	}

	var b strings.Builder

	// noout flag warning banner
	if v.nooutSet {
		banner := styles.StyleWarning.Render(styles.IconWarning + " noout flag is SET - OSDs will not be marked out")
		b.WriteString(banner)
		b.WriteString("\n\n")
	}

	// Header
	header := v.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Separator
	b.WriteString(styles.StyleSubtle.Render(strings.Repeat("─", v.getTableWidth())))
	b.WriteString("\n")

	// Calculate visible rows
	visibleRows := v.height - 6 // Account for header, separator, banner, and padding
	if visibleRows < 1 {
		visibleRows = len(v.osds)
	}

	// Calculate scroll offset
	startIdx := 0
	if v.cursor >= visibleRows {
		startIdx = v.cursor - visibleRows + 1
	}
	endIdx := startIdx + visibleRows
	if endIdx > len(v.osds) {
		endIdx = len(v.osds)
	}

	// Rows
	for i := startIdx; i < endIdx; i++ {
		osd := v.osds[i]
		row := v.renderRow(osd, i == v.cursor)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(v.osds) > visibleRows {
		scrollInfo := styles.StyleSubtle.Render(fmt.Sprintf("(%d/%d)", v.cursor+1, len(v.osds)))
		b.WriteString(scrollInfo)
	}

	return b.String()
}

// renderHeader renders the table header
func (v *OSDsView) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary)

	cols := []string{
		format.PadRight("OSD", 10),
		format.PadRight("HOST", 20),
		format.PadRight("STATUS", 8),
		format.PadRight("IN/OUT", 8),
		format.PadRight("WEIGHT", 10),
		format.PadRight("CLASS", 8),
		format.PadRight("DEPLOYMENT", 30),
	}

	return headerStyle.Render(strings.Join(cols, " "))
}

// renderRow renders a single OSD row
func (v *OSDsView) renderRow(osd k8s.OSDInfo, selected bool) string {
	var nameStyle, statusStyle, inOutStyle lipgloss.Style

	if selected {
		nameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorHighlight).
			Background(styles.ColorPrimary)
	} else {
		nameStyle = styles.StyleNormal
	}

	// Status style
	if osd.Status == "up" {
		statusStyle = styles.StyleSuccess
	} else {
		statusStyle = styles.StyleError
	}

	// In/Out style
	if osd.InOut == "in" {
		inOutStyle = styles.StyleSuccess
	} else {
		inOutStyle = styles.StyleError
	}

	// Highlight entire row if OSD is down or out
	rowWarning := osd.Status == "down" || osd.InOut == "out"

	// Weight formatting
	weightStr := fmt.Sprintf("%.3f", osd.Weight)

	// Deployment name with truncation (using display width for proper Unicode handling)
	deploymentName := osd.DeploymentName
	if deploymentName == "" {
		deploymentName = "<none>"
	}
	deploymentName = format.TruncateWithEllipsis(deploymentName, 28)

	// Build columns
	cols := []string{
		nameStyle.Render(format.PadRight(osd.Name, 10)),
		v.renderWithWarning(format.PadRight(osd.Hostname, 20), rowWarning, selected),
		statusStyle.Render(format.PadRight(osd.Status, 8)),
		inOutStyle.Render(format.PadRight(osd.InOut, 8)),
		styles.StyleSubtle.Render(format.PadRight(weightStr, 10)),
		styles.StyleSubtle.Render(format.PadRight(osd.DeviceClass, 8)),
		v.renderWithWarning(format.PadRight(deploymentName, 30), rowWarning, selected),
	}

	return strings.Join(cols, " ")
}

// renderWithWarning renders text with warning style if needed
func (v *OSDsView) renderWithWarning(s string, warning, selected bool) string {
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
func (v *OSDsView) getTableWidth() int {
	return 10 + 20 + 8 + 8 + 10 + 8 + 30 + 6 // column widths + spacing
}

// SetOSDs updates the OSDs list
func (v *OSDsView) SetOSDs(osds []k8s.OSDInfo) {
	v.osds = osds
	// Reset cursor if out of bounds
	if v.cursor >= len(v.osds) {
		v.cursor = len(v.osds) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
}

// SetNooutFlag sets the noout flag status
func (v *OSDsView) SetNooutFlag(set bool) {
	v.nooutSet = set
}

// SetSize sets the view dimensions
func (v *OSDsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetCursor returns the current cursor position
func (v *OSDsView) GetCursor() int {
	return v.cursor
}

// SetCursor sets the cursor position
func (v *OSDsView) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(v.osds) {
		v.cursor = cursor
	}
}

// Count returns the number of OSDs
func (v *OSDsView) Count() int {
	return len(v.osds)
}

// GetSelectedOSD returns the currently selected OSD
func (v *OSDsView) GetSelectedOSD() *k8s.OSDInfo {
	if v.cursor >= 0 && v.cursor < len(v.osds) {
		return &v.osds[v.cursor]
	}
	return nil
}

// IsNooutSet returns whether the noout flag is set
func (v *OSDsView) IsNooutSet() bool {
	return v.nooutSet
}

// CountDown returns the number of OSDs with status "down"
func (v *OSDsView) CountDown() int {
	count := 0
	for _, osd := range v.osds {
		if osd.Status == "down" {
			count++
		}
	}
	return count
}

// CountOut returns the number of OSDs with status "out"
func (v *OSDsView) CountOut() int {
	count := 0
	for _, osd := range v.osds {
		if osd.InOut == "out" {
			count++
		}
	}
	return count
}

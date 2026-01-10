// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ClusterHeaderData holds all the data displayed in the cluster header
type ClusterHeaderData struct {
	// Health status (HEALTH_OK, HEALTH_WARN, HEALTH_ERR)
	Health string

	// HealthMessages contains warning/error messages if not healthy
	HealthMessages []string

	// OSD statistics
	OSDs   int // Total OSDs
	OSDsUp int // OSDs that are up
	OSDsIn int // OSDs that are in

	// Monitor statistics
	MonsTotal    int // Total monitors
	MonsInQuorum int // Monitors in quorum

	// Flags
	NooutSet bool

	// Storage usage
	UsedBytes  int64
	TotalBytes int64

	// Last update time
	LastUpdate time.Time
}

// ClusterHeader displays a summary of Ceph cluster health
type ClusterHeader struct {
	// data holds the current cluster data
	data *ClusterHeaderData

	// width is the terminal width for responsive rendering
	width int

	// loading indicates if data is being fetched
	loading bool

	// error holds any error from data fetching
	err error
}

// NewClusterHeader creates a new cluster header component
func NewClusterHeader() *ClusterHeader {
	return &ClusterHeader{
		loading: true,
	}
}

// HeaderUpdateMsg is sent when header data is updated
type HeaderUpdateMsg struct {
	Data  *ClusterHeaderData
	Error error
}

// Init implements tea.Model
func (h *ClusterHeader) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (h *ClusterHeader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case HeaderUpdateMsg:
		h.loading = false
		h.err = msg.Error
		if msg.Data != nil {
			h.data = msg.Data
		}
	case tea.WindowSizeMsg:
		h.width = msg.Width
	}
	return h, nil
}

// View implements tea.Model
func (h *ClusterHeader) View() tea.View {
	return tea.NewView(h.Render())
}

// Render returns the string representation for composition
func (h *ClusterHeader) Render() string {
	if h.loading {
		return h.renderLoading()
	}

	if h.err != nil {
		return h.renderError()
	}

	if h.data == nil {
		return h.renderLoading()
	}

	// Use compact view for narrow terminals
	if h.width > 0 && h.width < 80 {
		return h.renderCompact()
	}

	return h.renderFull()
}

// renderLoading renders a loading placeholder
func (h *ClusterHeader) renderLoading() string {
	return styles.StyleSubtle.Render("Ceph: loading...")
}

// renderError renders an error message
func (h *ClusterHeader) renderError() string {
	return styles.StyleError.Render(fmt.Sprintf("Ceph: error - %v", h.err))
}

// renderFull renders the full two-line header
func (h *ClusterHeader) renderFull() string {
	var b strings.Builder

	// Row 1: Health, OSDs, MONs, noout flag
	b.WriteString(h.renderHealthBadge())
	b.WriteString("  ")
	b.WriteString(h.renderOSDStats())
	b.WriteString("  ")
	b.WriteString(h.renderMonStats())
	b.WriteString("  ")
	b.WriteString(h.renderNooutFlag())
	b.WriteString("\n")

	// Row 2: Storage usage and last updated
	b.WriteString(h.renderStorageUsage())
	b.WriteString("  ")
	b.WriteString(h.renderLastUpdated())

	return b.String()
}

// renderCompact renders a single-line compact header
func (h *ClusterHeader) renderCompact() string {
	var b strings.Builder

	b.WriteString(h.renderHealthBadge())
	b.WriteString(" ")
	b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("OSDs:%d/%d", h.data.OSDsUp, h.data.OSDs)))
	b.WriteString(" ")
	b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("MONs:%d/%d in quorum", h.data.MonsInQuorum, h.data.MonsTotal)))

	if h.data.NooutSet {
		b.WriteString(" ")
		b.WriteString(styles.StyleWarning.Render(styles.IconWarning + "noout"))
	}

	return b.String()
}

// renderHealthBadge renders the health status badge
func (h *ClusterHeader) renderHealthBadge() string {
	health := strings.ToUpper(h.data.Health)
	label := "Ceph: "

	var style lipgloss.Style
	switch health {
	case "HEALTH_OK":
		style = styles.StyleSuccess
	case "HEALTH_WARN":
		style = styles.StyleWarning
	case "HEALTH_ERR":
		style = styles.StyleError
	default:
		style = styles.StyleSubtle
	}

	badge := style.Render("[" + health + "]")
	return label + badge
}

// renderOSDStats renders OSD statistics
func (h *ClusterHeader) renderOSDStats() string {
	upColor := styles.StyleSuccess
	if h.data.OSDsUp < h.data.OSDs {
		upColor = styles.StyleWarning
	}

	inColor := styles.StyleSuccess
	if h.data.OSDsIn < h.data.OSDs {
		inColor = styles.StyleWarning
	}

	return fmt.Sprintf("OSDs: %s up, %s in",
		upColor.Render(fmt.Sprintf("%d/%d", h.data.OSDsUp, h.data.OSDs)),
		inColor.Render(fmt.Sprintf("%d/%d", h.data.OSDsIn, h.data.OSDs)),
	)
}

// renderMonStats renders monitor statistics
func (h *ClusterHeader) renderMonStats() string {
	color := styles.StyleSuccess
	if h.data.MonsInQuorum < h.data.MonsTotal {
		color = styles.StyleWarning
	}
	if h.data.MonsInQuorum <= h.data.MonsTotal/2 {
		color = styles.StyleError
	}

	return fmt.Sprintf("MONs: %s in quorum",
		color.Render(fmt.Sprintf("%d/%d", h.data.MonsInQuorum, h.data.MonsTotal)),
	)
}

// renderNooutFlag renders the noout flag status
func (h *ClusterHeader) renderNooutFlag() string {
	if h.data.NooutSet {
		return styles.StyleWarning.Render(styles.IconWarning + " noout: " + styles.IconCheckmark)
	}
	return styles.StyleSubtle.Render("noout: " + styles.IconCross)
}

// renderStorageUsage renders storage usage information
func (h *ClusterHeader) renderStorageUsage() string {
	if h.data.TotalBytes == 0 {
		return styles.StyleSubtle.Render("Storage: N/A")
	}

	percent := float64(h.data.UsedBytes) / float64(h.data.TotalBytes) * 100

	// Color based on usage
	var color lipgloss.Style
	switch {
	case percent >= 85:
		color = styles.StyleError
	case percent >= 70:
		color = styles.StyleWarning
	default:
		color = styles.StyleNormal
	}

	usageStr := fmt.Sprintf("%s / %s (%s)",
		format.FormatBytes(h.data.UsedBytes),
		format.FormatBytes(h.data.TotalBytes),
		format.FormatPercent(percent),
	)

	return "Storage: " + color.Render(usageStr)
}

// renderLastUpdated renders the last update timestamp
func (h *ClusterHeader) renderLastUpdated() string {
	if h.data.LastUpdate.IsZero() {
		return styles.StyleSubtle.Render("Last updated: never")
	}

	elapsed := time.Since(h.data.LastUpdate)
	var timeStr string

	switch {
	case elapsed < time.Minute:
		timeStr = fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	case elapsed < time.Hour:
		timeStr = fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	default:
		timeStr = fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	}

	return styles.StyleSubtle.Render("Last updated: " + timeStr)
}

// SetData updates the header data
func (h *ClusterHeader) SetData(data *ClusterHeaderData) {
	h.data = data
	h.loading = false
	h.err = nil
}

// SetError sets an error state
func (h *ClusterHeader) SetError(err error) {
	h.err = err
	h.loading = false
}

// SetLoading sets the loading state
func (h *ClusterHeader) SetLoading(loading bool) {
	h.loading = loading
}

// SetWidth sets the terminal width for responsive rendering
func (h *ClusterHeader) SetWidth(width int) {
	h.width = width
}

// GetData returns the current header data
func (h *ClusterHeader) GetData() *ClusterHeaderData {
	return h.data
}

// IsLoading returns true if the header is in loading state
func (h *ClusterHeader) IsLoading() bool {
	return h.loading
}

// HasError returns true if there's an error
func (h *ClusterHeader) HasError() bool {
	return h.err != nil
}

// GetError returns the current error
func (h *ClusterHeader) GetError() error {
	return h.err
}

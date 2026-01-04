package components

import (
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableColumn defines a column in the table
type TableColumn struct {
	// Title is the column header
	Title string

	// Width is the column width in characters (0 = auto)
	Width int

	// Align specifies text alignment
	Align lipgloss.Position

	// Style is applied to all cells in this column
	Style lipgloss.Style
}

// TableRow represents a row of data
type TableRow struct {
	// Cells contains the cell values
	Cells []string

	// Style overrides the default row style
	Style lipgloss.Style

	// Highlighted marks this row for visual emphasis
	Highlighted bool
}

// Table is a simple table display component
type Table struct {
	// Columns defines the table structure
	Columns []TableColumn

	// Rows contains the table data
	Rows []TableRow

	// Title is displayed above the table (optional)
	Title string

	// ShowHeader determines if column headers are displayed
	ShowHeader bool

	// ShowBorders adds borders around cells
	ShowBorders bool

	// Width is the total table width (0 = auto)
	Width int

	// MaxRows limits displayed rows (0 = no limit)
	MaxRows int

	// styles
	headerStyle    lipgloss.Style
	cellStyle      lipgloss.Style
	highlightStyle lipgloss.Style
	borderStyle    lipgloss.Style
}

// NewTable creates a new table with the given columns
func NewTable(columns []TableColumn) *Table {
	return &Table{
		Columns:     columns,
		Rows:        make([]TableRow, 0),
		ShowHeader:  true,
		ShowBorders: false,
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorPrimary),
		cellStyle: lipgloss.NewStyle().
			Foreground(styles.ColorHighlight),
		highlightStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorInfo),
		borderStyle: lipgloss.NewStyle().
			Foreground(styles.ColorBorder),
	}
}

// NewSimpleTable creates a table with auto-width columns
func NewSimpleTable(titles ...string) *Table {
	columns := make([]TableColumn, len(titles))
	for i, title := range titles {
		columns[i] = TableColumn{
			Title: title,
			Width: 0, // auto
			Align: lipgloss.Left,
		}
	}
	return NewTable(columns)
}

// Init implements tea.Model
func (t *Table) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Future: handle keyboard navigation for selectable tables
	return t, nil
}

// View implements tea.Model
func (t *Table) View() string {
	if len(t.Columns) == 0 {
		return ""
	}

	// Calculate column widths
	widths := t.calculateWidths()

	var lines []string

	// Add title if present
	if t.Title != "" {
		lines = append(lines, styles.StyleHeading.Render(t.Title))
		lines = append(lines, "")
	}

	// Add header row
	if t.ShowHeader {
		headerLine := t.renderRow(t.getHeaderCells(), widths, t.headerStyle, false)
		lines = append(lines, headerLine)

		if t.ShowBorders {
			separator := t.renderSeparator(widths)
			lines = append(lines, separator)
		}
	}

	// Add data rows
	rowCount := len(t.Rows)
	if t.MaxRows > 0 && rowCount > t.MaxRows {
		rowCount = t.MaxRows
	}

	for i := 0; i < rowCount; i++ {
		row := t.Rows[i]
		style := t.cellStyle
		if row.Highlighted {
			style = t.highlightStyle
		}
		if row.Style.Value() != "" {
			style = row.Style
		}

		rowLine := t.renderRow(row.Cells, widths, style, row.Highlighted)
		lines = append(lines, rowLine)
	}

	// Show truncation indicator if rows were limited
	if t.MaxRows > 0 && len(t.Rows) > t.MaxRows {
		remaining := len(t.Rows) - t.MaxRows
		truncMsg := styles.StyleSubtle.Render(fmt.Sprintf("... and %d more rows", remaining))
		lines = append(lines, truncMsg)
	}

	return strings.Join(lines, "\n")
}

// getHeaderCells returns the column titles as cells
func (t *Table) getHeaderCells() []string {
	cells := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		cells[i] = col.Title
	}
	return cells
}

// calculateWidths determines the width for each column
func (t *Table) calculateWidths() []int {
	widths := make([]int, len(t.Columns))

	// Start with specified widths or header widths
	for i, col := range t.Columns {
		if col.Width > 0 {
			widths[i] = col.Width
		} else {
			widths[i] = len(col.Title)
		}
	}

	// Expand to fit content only for auto-width columns (Width == 0)
	for _, row := range t.Rows {
		for i, cell := range row.Cells {
			if i < len(widths) && t.Columns[i].Width == 0 && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Apply total width constraint if specified
	if t.Width > 0 {
		totalWidth := 0
		for _, w := range widths {
			totalWidth += w
		}
		// Add space for separators
		totalWidth += (len(widths) - 1) * 3 // " | "

		if totalWidth > t.Width {
			// Proportionally reduce column widths
			ratio := float64(t.Width) / float64(totalWidth)
			for i := range widths {
				widths[i] = int(float64(widths[i]) * ratio)
				if widths[i] < 3 {
					widths[i] = 3 // minimum width
				}
			}
		}
	}

	return widths
}

// renderRow renders a single row with the given widths
func (t *Table) renderRow(cells []string, widths []int, style lipgloss.Style, _ bool) string {
	paddedCells := make([]string, len(t.Columns))

	for i := range t.Columns {
		var cell string
		if i < len(cells) {
			cell = cells[i]
		}

		// Truncate if needed
		width := widths[i]
		if len(cell) > width {
			if width > 3 {
				cell = cell[:width-3] + "..."
			} else {
				cell = cell[:width]
			}
		}

		// Pad to width
		padding := width - len(cell)
		align := t.Columns[i].Align

		// lipgloss.Position: Left=0.0, Top=0.0, Center=0.5, Right=1.0, Bottom=1.0
		switch align { //nolint:exhaustive // only 3 meaningful alignment values (Left, Center, Right) - Top/Bottom are identical to Left/Right
		case lipgloss.Right:
			cell = strings.Repeat(" ", padding) + cell
		case lipgloss.Center:
			leftPad := padding / 2
			rightPad := padding - leftPad
			cell = strings.Repeat(" ", leftPad) + cell + strings.Repeat(" ", rightPad)
		default:
			// Left (0.0) alignment - also covers Top which has same value
			cell = cell + strings.Repeat(" ", padding)
		}

		// Apply column-specific style if set
		if t.Columns[i].Style.Value() != "" {
			paddedCells[i] = t.Columns[i].Style.Render(cell)
		} else {
			paddedCells[i] = style.Render(cell)
		}
	}

	separator := "  "
	if t.ShowBorders {
		separator = t.borderStyle.Render(" │ ")
	}

	return strings.Join(paddedCells, separator)
}

// renderSeparator renders a horizontal separator line
func (t *Table) renderSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("─", w)
	}
	return t.borderStyle.Render(strings.Join(parts, "─┼─"))
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, TableRow{Cells: cells})
}

// AddHighlightedRow adds a highlighted row to the table
func (t *Table) AddHighlightedRow(cells ...string) {
	t.Rows = append(t.Rows, TableRow{
		Cells:       cells,
		Highlighted: true,
	})
}

// AddStyledRow adds a row with custom style
func (t *Table) AddStyledRow(style lipgloss.Style, cells ...string) {
	t.Rows = append(t.Rows, TableRow{
		Cells: cells,
		Style: style,
	})
}

// SetRows replaces all rows
func (t *Table) SetRows(rows []TableRow) {
	t.Rows = rows
}

// ClearRows removes all rows
func (t *Table) ClearRows() {
	t.Rows = make([]TableRow, 0)
}

// SetWidth sets the total table width
func (t *Table) SetWidth(width int) {
	t.Width = width
}

// SetMaxRows limits the number of displayed rows
func (t *Table) SetMaxRows(max int) {
	t.MaxRows = max
}

// SetTitle sets the table title
func (t *Table) SetTitle(title string) {
	t.Title = title
}

// WithBorders enables borders (for chaining)
func (t *Table) WithBorders() *Table {
	t.ShowBorders = true
	return t
}

// WithoutHeader hides the header row (for chaining)
func (t *Table) WithoutHeader() *Table {
	t.ShowHeader = false
	return t
}

// RowCount returns the number of rows
func (t *Table) RowCount() int {
	return len(t.Rows)
}

// KeyValueTable is a specialized table for key-value pairs
type KeyValueTable struct {
	items []keyValueItem
	width int
}

type keyValueItem struct {
	key       string
	value     string
	valueType StatusType
}

// NewKeyValueTable creates a new key-value table
func NewKeyValueTable() *KeyValueTable {
	return &KeyValueTable{
		items: make([]keyValueItem, 0),
		width: 40,
	}
}

// Add adds a key-value pair
func (kv *KeyValueTable) Add(key, value string) {
	kv.items = append(kv.items, keyValueItem{
		key:       key,
		value:     value,
		valueType: StatusTypeInfo,
	})
}

// AddWithType adds a key-value pair with status type for coloring
func (kv *KeyValueTable) AddWithType(key, value string, statusType StatusType) {
	kv.items = append(kv.items, keyValueItem{
		key:       key,
		value:     value,
		valueType: statusType,
	})
}

// Init implements tea.Model
func (kv *KeyValueTable) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (kv *KeyValueTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return kv, nil
}

// View implements tea.Model
func (kv *KeyValueTable) View() string {
	if len(kv.items) == 0 {
		return ""
	}

	// Find max key length
	maxKeyLen := 0
	for _, item := range kv.items {
		if len(item.key) > maxKeyLen {
			maxKeyLen = len(item.key)
		}
	}

	var lines []string
	keyStyle := styles.StyleSubtle
	for _, item := range kv.items {
		// Pad key to align values
		paddedKey := item.key + strings.Repeat(" ", maxKeyLen-len(item.key))

		// Get value style based on type
		valueStyle := kv.getValueStyle(item.valueType)

		line := fmt.Sprintf("%s  %s",
			keyStyle.Render(paddedKey+":"),
			valueStyle.Render(item.value))
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// getValueStyle returns the appropriate style for the status type
func (kv *KeyValueTable) getValueStyle(statusType StatusType) lipgloss.Style {
	switch statusType {
	case StatusTypeSuccess:
		return styles.StyleSuccess
	case StatusTypeWarning:
		return styles.StyleWarning
	case StatusTypeError:
		return styles.StyleError
	case StatusTypeInfo, StatusTypePending, StatusTypeRunning:
		return styles.StyleNormal
	}
	return styles.StyleNormal
}

// SetWidth sets the table width
func (kv *KeyValueTable) SetWidth(width int) {
	kv.width = width
}

// Clear removes all items
func (kv *KeyValueTable) Clear() {
	kv.items = make([]keyValueItem, 0)
}

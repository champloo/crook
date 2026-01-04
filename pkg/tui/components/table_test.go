package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewTable(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name", Width: 20},
		{Title: "Status", Width: 10},
	}

	table := NewTable(columns)

	if len(table.Columns) != 2 {
		t.Errorf("Columns count = %d, want 2", len(table.Columns))
	}

	if !table.ShowHeader {
		t.Error("ShowHeader should be true by default")
	}

	if table.ShowBorders {
		t.Error("ShowBorders should be false by default")
	}
}

func TestNewSimpleTable(t *testing.T) {
	table := NewSimpleTable("Name", "Age", "City")

	if len(table.Columns) != 3 {
		t.Errorf("Columns count = %d, want 3", len(table.Columns))
	}

	if table.Columns[0].Title != "Name" {
		t.Errorf("Column[0].Title = %q, want %q", table.Columns[0].Title, "Name")
	}

	if table.Columns[0].Width != 0 {
		t.Error("Simple table columns should have auto width (0)")
	}
}

func TestTable_AddRow(t *testing.T) {
	table := NewSimpleTable("A", "B")

	table.AddRow("1", "2")
	table.AddRow("3", "4")

	if table.RowCount() != 2 {
		t.Errorf("RowCount() = %d, want 2", table.RowCount())
	}

	if table.Rows[0].Cells[0] != "1" {
		t.Error("First row first cell should be '1'")
	}
}

func TestTable_AddHighlightedRow(t *testing.T) {
	table := NewSimpleTable("A", "B")

	table.AddHighlightedRow("1", "2")

	if !table.Rows[0].Highlighted {
		t.Error("Row should be highlighted")
	}
}

func TestTable_AddStyledRow(t *testing.T) {
	table := NewSimpleTable("A", "B")
	style := lipgloss.NewStyle().Bold(true)

	table.AddStyledRow(style, "1", "2")

	// Just verify the row was added with a style set
	if len(table.Rows) != 1 {
		t.Error("Row should be added")
	}
}

func TestTable_SetRows(t *testing.T) {
	table := NewSimpleTable("A", "B")
	table.AddRow("1", "2")

	newRows := []TableRow{
		{Cells: []string{"x", "y"}},
	}
	table.SetRows(newRows)

	if table.RowCount() != 1 {
		t.Errorf("RowCount() = %d, want 1", table.RowCount())
	}

	if table.Rows[0].Cells[0] != "x" {
		t.Error("Rows should be replaced")
	}
}

func TestTable_ClearRows(t *testing.T) {
	table := NewSimpleTable("A", "B")
	table.AddRow("1", "2")

	table.ClearRows()

	if table.RowCount() != 0 {
		t.Errorf("RowCount() = %d, want 0 after clear", table.RowCount())
	}
}

func TestTable_View_Empty(t *testing.T) {
	table := NewTable([]TableColumn{})

	view := table.View()

	if view != "" {
		t.Error("Empty table should return empty view")
	}
}

func TestTable_View_WithHeader(t *testing.T) {
	table := NewSimpleTable("Name", "Value")
	table.AddRow("foo", "bar")

	view := table.View()

	if !strings.Contains(view, "Name") {
		t.Error("View should contain header 'Name'")
	}

	if !strings.Contains(view, "Value") {
		t.Error("View should contain header 'Value'")
	}

	if !strings.Contains(view, "foo") {
		t.Error("View should contain data 'foo'")
	}
}

func TestTable_View_WithoutHeader(t *testing.T) {
	table := NewSimpleTable("Name", "Value").WithoutHeader()
	table.AddRow("foo", "bar")

	view := table.View()

	// The header text might still appear if it's in the data
	// But should not be styled as header
	if !strings.Contains(view, "foo") {
		t.Error("View should contain data")
	}
}

func TestTable_View_WithTitle(t *testing.T) {
	table := NewSimpleTable("A", "B")
	table.SetTitle("My Table")
	table.AddRow("1", "2")

	view := table.View()

	if !strings.Contains(view, "My Table") {
		t.Error("View should contain title")
	}
}

func TestTable_View_WithMaxRows(t *testing.T) {
	table := NewSimpleTable("A")
	table.SetMaxRows(2)

	table.AddRow("1")
	table.AddRow("2")
	table.AddRow("3")
	table.AddRow("4")

	view := table.View()

	if !strings.Contains(view, "1") {
		t.Error("View should contain first row")
	}

	if !strings.Contains(view, "2") {
		t.Error("View should contain second row")
	}

	// Should show truncation message
	if !strings.Contains(view, "2 more rows") {
		t.Error("View should show truncation indicator")
	}
}

func TestTable_View_WithBorders(t *testing.T) {
	table := NewSimpleTable("A", "B").WithBorders()
	table.AddRow("1", "2")

	view := table.View()

	if !strings.Contains(view, "│") {
		t.Error("View should contain border character")
	}
}

func TestTable_CalculateWidths_Fixed(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name", Width: 10},
		{Title: "Value", Width: 20},
	}
	table := NewTable(columns)

	widths := table.calculateWidths()

	if widths[0] != 10 {
		t.Errorf("Width[0] = %d, want 10", widths[0])
	}

	if widths[1] != 20 {
		t.Errorf("Width[1] = %d, want 20", widths[1])
	}
}

func TestTable_CalculateWidths_Auto(t *testing.T) {
	table := NewSimpleTable("A", "B")
	table.AddRow("Hello", "World")

	widths := table.calculateWidths()

	// Should expand to fit content
	if widths[0] < 5 {
		t.Errorf("Width[0] = %d, should be at least 5 for 'Hello'", widths[0])
	}
}

func TestTable_CalculateWidths_TotalConstraint(t *testing.T) {
	columns := []TableColumn{
		{Title: "A", Width: 50},
		{Title: "B", Width: 50},
	}
	table := NewTable(columns)
	table.SetWidth(50) // Constraint to 50 chars total

	widths := table.calculateWidths()

	// Widths should be reduced
	total := widths[0] + widths[1]
	// Account for separator
	if total > 50 {
		t.Errorf("Total width %d exceeds constraint 50", total)
	}
}

func TestTable_RenderRow_Truncation(t *testing.T) {
	columns := []TableColumn{
		{Title: "Name", Width: 8}, // Width 8 to allow truncation with "..."
	}
	table := NewTable(columns)

	// The renderRow is internal, test through View
	table.AddRow("VeryLongName")

	view := table.View()

	// Should be truncated - original text shouldn't fully appear
	if strings.Contains(view, "VeryLongName") {
		t.Error("Long cell should be truncated")
	}
}

func TestTable_Chaining(t *testing.T) {
	table := NewSimpleTable("A", "B").WithBorders().WithoutHeader()

	if !table.ShowBorders {
		t.Error("WithBorders should enable borders")
	}

	if table.ShowHeader {
		t.Error("WithoutHeader should disable header")
	}
}

func TestKeyValueTable(t *testing.T) {
	kv := NewKeyValueTable()

	kv.Add("Name", "John")
	kv.Add("Age", "30")

	view := kv.View()

	if !strings.Contains(view, "Name") {
		t.Error("View should contain 'Name'")
	}

	if !strings.Contains(view, "John") {
		t.Error("View should contain 'John'")
	}

	if !strings.Contains(view, "Age") {
		t.Error("View should contain 'Age'")
	}
}

func TestKeyValueTable_WithType(t *testing.T) {
	kv := NewKeyValueTable()

	kv.AddWithType("Status", "OK", StatusTypeSuccess)
	kv.AddWithType("Error", "Failed", StatusTypeError)

	view := kv.View()

	// Just verify the view contains the values
	// (styling is hard to test without ANSI parsing)
	if !strings.Contains(view, "OK") {
		t.Error("View should contain 'OK'")
	}

	if !strings.Contains(view, "Failed") {
		t.Error("View should contain 'Failed'")
	}
}

func TestKeyValueTable_Empty(t *testing.T) {
	kv := NewKeyValueTable()

	view := kv.View()

	if view != "" {
		t.Error("Empty key-value table should return empty string")
	}
}

func TestKeyValueTable_Clear(t *testing.T) {
	kv := NewKeyValueTable()
	kv.Add("Key", "Value")
	kv.Clear()

	if len(kv.items) != 0 {
		t.Error("Clear should remove all items")
	}
}

func TestKeyValueTable_SetWidth(t *testing.T) {
	kv := NewKeyValueTable()
	kv.SetWidth(60)

	if kv.width != 60 {
		t.Errorf("width = %d, want 60", kv.width)
	}
}

func TestTable_Init(t *testing.T) {
	table := NewSimpleTable("A", "B")
	cmd := table.Init()

	if cmd != nil {
		t.Error("Table Init should return nil")
	}
}

func TestTable_Update(t *testing.T) {
	table := NewSimpleTable("A", "B")
	model, cmd := table.Update(nil)

	if model != table {
		t.Error("Update should return same table")
	}

	if cmd != nil {
		t.Error("Update should return nil cmd")
	}
}

func TestKeyValueTable_Init(t *testing.T) {
	kv := NewKeyValueTable()
	cmd := kv.Init()

	if cmd != nil {
		t.Error("KeyValueTable Init should return nil")
	}
}

func TestKeyValueTable_Update(t *testing.T) {
	kv := NewKeyValueTable()
	model, cmd := kv.Update(nil)

	if model != kv {
		t.Error("Update should return same table")
	}

	if cmd != nil {
		t.Error("Update should return nil cmd")
	}
}

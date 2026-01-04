package output

import (
	"fmt"
	"io"
)

// Render renders data to the specified format and writes to the given writer
func Render(w io.Writer, data *Data, format Format) error {
	switch format {
	case FormatTable:
		return RenderTable(w, data)
	case FormatJSON:
		return RenderJSON(w, data)
	case FormatYAML:
		return RenderYAML(w, data)
	case FormatTUI:
		return fmt.Errorf("TUI format should not be rendered through this function")
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

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
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

package output

import (
	"encoding/json"
	"io"
)

// RenderJSON renders data as JSON and writes to the given writer
func RenderJSON(w io.Writer, data *Data) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

// RenderYAML renders data as YAML and writes to the given writer
func RenderYAML(w io.Writer, data *Data) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	err := encoder.Encode(data)
	if err != nil {
		return err
	}
	return encoder.Close()
}

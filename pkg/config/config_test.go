package config

import (
	"strings"
	"testing"
)

func TestConfigStringIncludesSections(t *testing.T) {
	cfg := DefaultConfig()
	output := cfg.String()

	for _, section := range []string{"kubernetes:", "state:", "deployment-filters:", "ui:", "timeouts:", "logging:"} {
		if !strings.Contains(output, section) {
			t.Fatalf("expected output to include %q", section)
		}
	}
}

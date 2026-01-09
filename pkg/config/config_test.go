package config_test

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/config"
)

func TestConfigStringIncludesSections(t *testing.T) {
	cfg := config.DefaultConfig()
	output := cfg.String()

	// Note: kubernetes section is excluded from YAML output (CLI-only settings)
	for _, section := range []string{"ui:", "timeouts:", "logging:"} {
		if !strings.Contains(output, section) {
			t.Fatalf("expected output to include %q", section)
		}
	}

	// kubernetes should NOT appear in YAML output
	if strings.Contains(output, "kubernetes:") {
		t.Fatal("kubernetes section should not appear in YAML output")
	}
}

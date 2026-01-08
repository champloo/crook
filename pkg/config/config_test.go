package config_test

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/config"
)

func TestConfigStringIncludesSections(t *testing.T) {
	cfg := config.DefaultConfig()
	output := cfg.String()

	for _, section := range []string{"kubernetes:", "ui:", "timeouts:", "logging:"} {
		if !strings.Contains(output, section) {
			t.Fatalf("expected output to include %q", section)
		}
	}
}

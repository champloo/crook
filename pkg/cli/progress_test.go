package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andri/crook/pkg/cli"
	"github.com/andri/crook/pkg/maintenance"
)

func TestProgressWriter_OnDownProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress maintenance.DownPhaseProgress
		wantIcon string
	}{
		{
			name: "pre-flight stage",
			progress: maintenance.DownPhaseProgress{
				Stage:       "pre-flight",
				Description: "Running pre-flight checks",
			},
			wantIcon: "\u2192", // arrow
		},
		{
			name: "complete stage",
			progress: maintenance.DownPhaseProgress{
				Stage:       "complete",
				Description: "Down phase completed",
			},
			wantIcon: "\u2713", // checkmark
		},
		{
			name: "error stage",
			progress: maintenance.DownPhaseProgress{
				Stage:       "error",
				Description: "Something failed",
			},
			wantIcon: "\u2717", // X mark
		},
		{
			name: "scale-down stage",
			progress: maintenance.DownPhaseProgress{
				Stage:       "scale-down",
				Description: "Scaling down deployment",
				Deployment:  "rook-ceph-osd-0",
			},
			wantIcon: "\u2192", // arrow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			pw := cli.NewProgressWriter(buf)

			pw.OnDownProgress(tt.progress)

			output := buf.String()
			if !strings.Contains(output, tt.wantIcon) {
				t.Errorf("expected icon %q in output, got: %s", tt.wantIcon, output)
			}
			if !strings.Contains(output, tt.progress.Description) {
				t.Errorf("expected description in output, got: %s", output)
			}
		})
	}
}

func TestProgressWriter_OnUpProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress maintenance.UpPhaseProgress
		wantIcon string
	}{
		{
			name: "pre-flight stage",
			progress: maintenance.UpPhaseProgress{
				Stage:       "pre-flight",
				Description: "Running pre-flight checks",
			},
			wantIcon: "\u2192", // arrow
		},
		{
			name: "complete stage",
			progress: maintenance.UpPhaseProgress{
				Stage:       "complete",
				Description: "Up phase completed",
			},
			wantIcon: "\u2713", // checkmark
		},
		{
			name: "quorum stage",
			progress: maintenance.UpPhaseProgress{
				Stage:       "quorum",
				Description: "Waiting for quorum",
			},
			wantIcon: "\u2192", // arrow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			pw := cli.NewProgressWriter(buf)

			pw.OnUpProgress(tt.progress)

			output := buf.String()
			if !strings.Contains(output, tt.wantIcon) {
				t.Errorf("expected icon %q in output, got: %s", tt.wantIcon, output)
			}
			if !strings.Contains(output, tt.progress.Description) {
				t.Errorf("expected description in output, got: %s", output)
			}
		})
	}
}

func TestProgressWriter_PrintSummary(t *testing.T) {
	buf := &bytes.Buffer{}
	pw := cli.NewProgressWriter(buf)

	pw.PrintSummary("worker-1", 3, []string{"osd-0", "osd-1", "mon-a"})

	output := buf.String()
	if !strings.Contains(output, "worker-1") {
		t.Errorf("expected node name in output, got: %s", output)
	}
	if !strings.Contains(output, "3") {
		t.Errorf("expected deployment count in output, got: %s", output)
	}
	if !strings.Contains(output, "osd-0") {
		t.Errorf("expected deployment name in output, got: %s", output)
	}
}

func TestProgressWriter_PrintSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	pw := cli.NewProgressWriter(buf)

	pw.PrintSuccess("Operation completed")

	output := buf.String()
	if !strings.Contains(output, "\u2713") {
		t.Errorf("expected checkmark in output, got: %s", output)
	}
	if !strings.Contains(output, "Operation completed") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestProgressWriter_PrintError(t *testing.T) {
	buf := &bytes.Buffer{}
	pw := cli.NewProgressWriter(buf)

	pw.PrintError("Something failed")

	output := buf.String()
	if !strings.Contains(output, "\u2717") {
		t.Errorf("expected X mark in output, got: %s", output)
	}
	if !strings.Contains(output, "Something failed") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestProgressWriter_NilWriter(t *testing.T) {
	// Test that nil writer defaults to stdout without panicking
	pw := cli.NewProgressWriter(nil)
	if pw == nil {
		t.Error("expected non-nil ProgressWriter")
	}
}

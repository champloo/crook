package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andri/crook/pkg/maintenance"
)

// ProgressWriter outputs progress updates to the terminal.
type ProgressWriter struct {
	w io.Writer
}

// NewProgressWriter creates a new ProgressWriter.
// If w is nil, os.Stdout is used.
func NewProgressWriter(w io.Writer) *ProgressWriter {
	if w == nil {
		w = os.Stdout
	}
	return &ProgressWriter{w: w}
}

// OnDownProgress handles progress updates from the down phase.
func (pw *ProgressWriter) OnDownProgress(p maintenance.DownPhaseProgress) {
	pw.printProgress(p.Stage, p.Description)
}

// OnUpProgress handles progress updates from the up phase.
func (pw *ProgressWriter) OnUpProgress(p maintenance.UpPhaseProgress) {
	pw.printProgress(p.Stage, p.Description)
}

// printProgress prints a progress message with appropriate formatting.
func (pw *ProgressWriter) printProgress(stage, description string) {
	var prefix string
	switch stage {
	case "complete":
		prefix = "\u2713" // checkmark
	case "error":
		prefix = "\u2717" // X mark
	default:
		prefix = "\u2192" // right arrow
	}

	// Clean up description for display
	description = strings.TrimSpace(description)
	if description == "" {
		description = stage
	}

	_, _ = fmt.Fprintf(pw.w, "%s %s\n", prefix, description)
}

// PrintSummary prints a summary of deployments that will be affected.
func (pw *ProgressWriter) PrintSummary(nodeName string, deploymentCount int, deploymentNames []string) {
	_, _ = fmt.Fprintf(pw.w, "Target node: %s\n", nodeName)
	_, _ = fmt.Fprintf(pw.w, "Deployments to process: %d\n", deploymentCount)

	if len(deploymentNames) > 0 {
		_, _ = fmt.Fprintln(pw.w, "Deployments:")
		for _, name := range deploymentNames {
			_, _ = fmt.Fprintf(pw.w, "  - %s\n", name)
		}
	}
	_, _ = fmt.Fprintln(pw.w)
}

// PrintSuccess prints a success message.
func (pw *ProgressWriter) PrintSuccess(message string) {
	_, _ = fmt.Fprintf(pw.w, "\u2713 %s\n", message)
}

// PrintError prints an error message.
func (pw *ProgressWriter) PrintError(message string) {
	_, _ = fmt.Fprintf(pw.w, "\u2717 %s\n", message)
}

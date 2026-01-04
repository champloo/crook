// Package main is the entry point for the crook CLI application.
package main

import (
	"fmt"
	"os"

	"github.com/andri/crook/cmd/crook/commands"
)

// These variables are set at build time via ldflags
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// Set version info for the commands package
	commands.SetVersionInfo(version, commit, buildDate)

	// Execute the root command
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Package format provides formatting utilities for the TUI.
package format

import (
	"fmt"
)

// Binary unit sizes (IEC standard)
const (
	KiB int64 = 1024
	MiB int64 = 1024 * KiB
	GiB int64 = 1024 * MiB
	TiB int64 = 1024 * GiB
	PiB int64 = 1024 * TiB
	EiB int64 = 1024 * PiB
)

// FormatBytes formats a byte count as a human-readable string using binary units.
// Uses IEC standard units (KiB, MiB, GiB, TiB, PiB).
// Examples:
//
//	FormatBytes(1024) = "1.0 KiB"
//	FormatBytes(1536) = "1.5 KiB"
//	FormatBytes(1073741824) = "1.0 GiB"
//	FormatBytes(1099511627776) = "1.0 TiB"
func FormatBytes(bytes int64) string {
	if bytes < 0 {
		return "-" + FormatBytes(-bytes)
	}

	switch {
	case bytes >= EiB:
		return fmt.Sprintf("%.1f EiB", float64(bytes)/float64(EiB))
	case bytes >= PiB:
		return fmt.Sprintf("%.1f PiB", float64(bytes)/float64(PiB))
	case bytes >= TiB:
		return fmt.Sprintf("%.1f TiB", float64(bytes)/float64(TiB))
	case bytes >= GiB:
		return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(GiB))
	case bytes >= MiB:
		return fmt.Sprintf("%.1f MiB", float64(bytes)/float64(MiB))
	case bytes >= KiB:
		return fmt.Sprintf("%.1f KiB", float64(bytes)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatPercent formats a percentage value with one decimal place.
func FormatPercent(percent float64) string {
	return fmt.Sprintf("%.1f%%", percent)
}

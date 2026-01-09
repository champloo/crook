package k8s

import (
	"encoding/json"
	"time"
)

// Duration wraps time.Duration to provide human-readable JSON serialization.
// JSON output format: "5d", "2h", "30m", "45s"
type Duration time.Duration

// MarshalJSON implements json.Marshaler for human-readable duration output
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(formatDuration(time.Duration(d)))
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try as raw nanoseconds (int64)
		var ns int64
		if nsErr := json.Unmarshal(data, &ns); nsErr != nil {
			return nsErr
		}
		*d = Duration(ns)
		return nil
	}
	// Parse duration string - if standard parsing fails, try our custom format
	dur, parseErr := time.ParseDuration(s)
	if parseErr != nil {
		// Try parsing our custom format (e.g., "5d")
		// parseDurationString returns 0 on failure which is acceptable
		*d = Duration(parseDurationString(s))
		return nil //nolint:nilerr // intentionally ignoring parseErr, falling back to custom parser
	}
	*d = Duration(dur)
	return nil
}

// Duration returns the underlying time.Duration value
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// formatDuration formats a duration as a human-readable age string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return formatInt(int(d.Seconds())) + "s"
	}
	if d < time.Hour {
		return formatInt(int(d.Minutes())) + "m"
	}
	if d < 24*time.Hour {
		return formatInt(int(d.Hours())) + "h"
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return formatInt(days) + "d"
	}
	if days < 365 {
		return formatInt(days/30) + "mo"
	}
	return formatInt(days/365) + "y"
}

// formatInt converts an int to a string
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	// Build digits in reverse
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// parseDurationString parses our custom duration format (e.g., "5d", "2h")
func parseDurationString(s string) time.Duration {
	if len(s) < 2 {
		return 0
	}

	// Parse number
	numStr := s[:len(s)-1]
	suffix := s[len(s)-1]

	var num int
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}

	switch suffix {
	case 's':
		return time.Duration(num) * time.Second
	case 'm':
		return time.Duration(num) * time.Minute
	case 'h':
		return time.Duration(num) * time.Hour
	case 'd':
		return time.Duration(num) * 24 * time.Hour
	case 'y':
		return time.Duration(num) * 365 * 24 * time.Hour
	}

	return 0
}

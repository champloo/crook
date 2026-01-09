package k8s

import (
	"encoding/json"
	"time"

	"k8s.io/apimachinery/pkg/util/duration"
)

// Duration wraps time.Duration to provide human-readable JSON serialization.
// Uses Kubernetes-style formatting: "5d", "36h", "5m", "30s"
type Duration time.Duration

// MarshalJSON implements json.Marshaler for human-readable duration output
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(duration.HumanDuration(time.Duration(d)))
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

// String returns a human-readable representation of the duration (Kubernetes style)
func (d Duration) String() string {
	return duration.HumanDuration(time.Duration(d))
}

// parseDurationString parses duration format (e.g., "5d", "2h")
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

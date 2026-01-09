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

// Duration returns the underlying time.Duration value
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// String returns a human-readable representation of the duration (Kubernetes style)
func (d Duration) String() string {
	return duration.HumanDuration(time.Duration(d))
}

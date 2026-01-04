package k8s

import (
	"testing"
)

func TestParseFlagsString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *CephFlags
	}{
		{
			name:  "empty string",
			input: "",
			expected: &CephFlags{
				NoOut:       false,
				NoIn:        false,
				NoDown:      false,
				NoUp:        false,
				NoRebalance: false,
				NoRecover:   false,
				NoScrub:     false,
				NoDeepScrub: false,
				NoBackfill:  false,
				Pause:       false,
			},
		},
		{
			name:  "noout flag only",
			input: "sortbitwise,recovery_deletes,purged_snapdirs,pglog_hardlimit,noout",
			expected: &CephFlags{
				NoOut:       true,
				NoIn:        false,
				NoDown:      false,
				NoUp:        false,
				NoRebalance: false,
				NoRecover:   false,
				NoScrub:     false,
				NoDeepScrub: false,
				NoBackfill:  false,
				Pause:       false,
			},
		},
		{
			name:  "multiple maintenance flags",
			input: "noout,nodown,sortbitwise,recovery_deletes",
			expected: &CephFlags{
				NoOut:       true,
				NoIn:        false,
				NoDown:      true,
				NoUp:        false,
				NoRebalance: false,
				NoRecover:   false,
				NoScrub:     false,
				NoDeepScrub: false,
				NoBackfill:  false,
				Pause:       false,
			},
		},
		{
			name:  "all known flags",
			input: "noout,noin,nodown,noup,norebalance,norecover,noscrub,nodeep-scrub,nobackfill,pause",
			expected: &CephFlags{
				NoOut:       true,
				NoIn:        true,
				NoDown:      true,
				NoUp:        true,
				NoRebalance: true,
				NoRecover:   true,
				NoScrub:     true,
				NoDeepScrub: true,
				NoBackfill:  true,
				Pause:       true,
			},
		},
		{
			name:  "scrub flags only",
			input: "sortbitwise,noscrub,nodeep-scrub,pglog_hardlimit",
			expected: &CephFlags{
				NoOut:       false,
				NoIn:        false,
				NoDown:      false,
				NoUp:        false,
				NoRebalance: false,
				NoRecover:   false,
				NoScrub:     true,
				NoDeepScrub: true,
				NoBackfill:  false,
				Pause:       false,
			},
		},
		{
			name:  "recovery flags only",
			input: "norebalance,norecover,nobackfill",
			expected: &CephFlags{
				NoOut:       false,
				NoIn:        false,
				NoDown:      false,
				NoUp:        false,
				NoRebalance: true,
				NoRecover:   true,
				NoScrub:     false,
				NoDeepScrub: false,
				NoBackfill:  true,
				Pause:       false,
			},
		},
		{
			name:  "with whitespace",
			input: "noout, nodown ,  norebalance",
			expected: &CephFlags{
				NoOut:       true,
				NoIn:        false,
				NoDown:      true,
				NoUp:        false,
				NoRebalance: true,
				NoRecover:   false,
				NoScrub:     false,
				NoDeepScrub: false,
				NoBackfill:  false,
				Pause:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFlagsString(tt.input)

			if result.NoOut != tt.expected.NoOut {
				t.Errorf("NoOut = %v, want %v", result.NoOut, tt.expected.NoOut)
			}
			if result.NoIn != tt.expected.NoIn {
				t.Errorf("NoIn = %v, want %v", result.NoIn, tt.expected.NoIn)
			}
			if result.NoDown != tt.expected.NoDown {
				t.Errorf("NoDown = %v, want %v", result.NoDown, tt.expected.NoDown)
			}
			if result.NoUp != tt.expected.NoUp {
				t.Errorf("NoUp = %v, want %v", result.NoUp, tt.expected.NoUp)
			}
			if result.NoRebalance != tt.expected.NoRebalance {
				t.Errorf("NoRebalance = %v, want %v", result.NoRebalance, tt.expected.NoRebalance)
			}
			if result.NoRecover != tt.expected.NoRecover {
				t.Errorf("NoRecover = %v, want %v", result.NoRecover, tt.expected.NoRecover)
			}
			if result.NoScrub != tt.expected.NoScrub {
				t.Errorf("NoScrub = %v, want %v", result.NoScrub, tt.expected.NoScrub)
			}
			if result.NoDeepScrub != tt.expected.NoDeepScrub {
				t.Errorf("NoDeepScrub = %v, want %v", result.NoDeepScrub, tt.expected.NoDeepScrub)
			}
			if result.NoBackfill != tt.expected.NoBackfill {
				t.Errorf("NoBackfill = %v, want %v", result.NoBackfill, tt.expected.NoBackfill)
			}
			if result.Pause != tt.expected.Pause {
				t.Errorf("Pause = %v, want %v", result.Pause, tt.expected.Pause)
			}
		})
	}
}

func TestParseCephFlags(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantNoOut  bool
		wantNoDown bool
		wantErr    bool
	}{
		{
			name:       "valid JSON with noout",
			input:      `{"flags": "noout,sortbitwise,recovery_deletes,purged_snapdirs,pglog_hardlimit"}`,
			wantNoOut:  true,
			wantNoDown: false,
			wantErr:    false,
		},
		{
			name:       "valid JSON without noout",
			input:      `{"flags": "sortbitwise,recovery_deletes,purged_snapdirs"}`,
			wantNoOut:  false,
			wantNoDown: false,
			wantErr:    false,
		},
		{
			name:       "valid JSON with multiple flags",
			input:      `{"flags": "noout,nodown,norebalance"}`,
			wantNoOut:  true,
			wantNoDown: true,
			wantErr:    false,
		},
		{
			name:       "empty flags",
			input:      `{"flags": ""}`,
			wantNoOut:  false,
			wantNoDown: false,
			wantErr:    false,
		},
		{
			name:    "invalid JSON",
			input:   `not valid json`,
			wantErr: true,
		},
		{
			name:       "real ceph osd dump excerpt",
			input:      `{"epoch": 123, "fsid": "abc-123", "flags": "noout,nodown,sortbitwise,recovery_deletes,purged_snapdirs,pglog_hardlimit", "flags_num": 123456}`,
			wantNoOut:  true,
			wantNoDown: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCephFlags(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.NoOut != tt.wantNoOut {
				t.Errorf("NoOut = %v, want %v", result.NoOut, tt.wantNoOut)
			}
			if result.NoDown != tt.wantNoDown {
				t.Errorf("NoDown = %v, want %v", result.NoDown, tt.wantNoDown)
			}
		})
	}
}

func TestCephFlags_HasMaintenanceFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    CephFlags
		expected bool
	}{
		{
			name:     "no flags",
			flags:    CephFlags{},
			expected: false,
		},
		{
			name:     "noout only",
			flags:    CephFlags{NoOut: true},
			expected: true,
		},
		{
			name:     "noin only",
			flags:    CephFlags{NoIn: true},
			expected: true,
		},
		{
			name:     "nodown only",
			flags:    CephFlags{NoDown: true},
			expected: true,
		},
		{
			name:     "noup only",
			flags:    CephFlags{NoUp: true},
			expected: true,
		},
		{
			name:     "all maintenance flags",
			flags:    CephFlags{NoOut: true, NoIn: true, NoDown: true, NoUp: true},
			expected: true,
		},
		{
			name:     "only scrub flags",
			flags:    CephFlags{NoScrub: true, NoDeepScrub: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.flags.HasMaintenanceFlags(); got != tt.expected {
				t.Errorf("HasMaintenanceFlags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCephFlags_HasScrubFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    CephFlags
		expected bool
	}{
		{
			name:     "no flags",
			flags:    CephFlags{},
			expected: false,
		},
		{
			name:     "noscrub only",
			flags:    CephFlags{NoScrub: true},
			expected: true,
		},
		{
			name:     "nodeep-scrub only",
			flags:    CephFlags{NoDeepScrub: true},
			expected: true,
		},
		{
			name:     "both scrub flags",
			flags:    CephFlags{NoScrub: true, NoDeepScrub: true},
			expected: true,
		},
		{
			name:     "only maintenance flags",
			flags:    CephFlags{NoOut: true, NoDown: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.flags.HasScrubFlags(); got != tt.expected {
				t.Errorf("HasScrubFlags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCephFlags_HasRecoveryFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    CephFlags
		expected bool
	}{
		{
			name:     "no flags",
			flags:    CephFlags{},
			expected: false,
		},
		{
			name:     "norebalance only",
			flags:    CephFlags{NoRebalance: true},
			expected: true,
		},
		{
			name:     "norecover only",
			flags:    CephFlags{NoRecover: true},
			expected: true,
		},
		{
			name:     "nobackfill only",
			flags:    CephFlags{NoBackfill: true},
			expected: true,
		},
		{
			name:     "all recovery flags",
			flags:    CephFlags{NoRebalance: true, NoRecover: true, NoBackfill: true},
			expected: true,
		},
		{
			name:     "only maintenance flags",
			flags:    CephFlags{NoOut: true, NoDown: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.flags.HasRecoveryFlags(); got != tt.expected {
				t.Errorf("HasRecoveryFlags() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCephFlags_ActiveFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    CephFlags
		expected []string
	}{
		{
			name:     "no flags",
			flags:    CephFlags{},
			expected: nil,
		},
		{
			name:     "single flag",
			flags:    CephFlags{NoOut: true},
			expected: []string{"noout"},
		},
		{
			name:     "multiple flags",
			flags:    CephFlags{NoOut: true, NoDown: true, NoScrub: true},
			expected: []string{"noout", "nodown", "noscrub"},
		},
		{
			name: "all flags",
			flags: CephFlags{
				NoOut:       true,
				NoIn:        true,
				NoDown:      true,
				NoUp:        true,
				NoRebalance: true,
				NoRecover:   true,
				NoScrub:     true,
				NoDeepScrub: true,
				NoBackfill:  true,
				Pause:       true,
			},
			expected: []string{"noout", "noin", "nodown", "noup", "norebalance", "norecover", "noscrub", "nodeep-scrub", "nobackfill", "pause"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.flags.ActiveFlags()

			if len(result) != len(tt.expected) {
				t.Errorf("ActiveFlags() returned %d flags, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %v, want: %v", result, tt.expected)
				return
			}

			for i, flag := range result {
				if flag != tt.expected[i] {
					t.Errorf("ActiveFlags()[%d] = %q, want %q", i, flag, tt.expected[i])
				}
			}
		})
	}
}

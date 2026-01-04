package format

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		// Zero and small values
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},

		// KiB
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{10240, "10.0 KiB"},
		{1048575, "1024.0 KiB"}, // Just under 1 MiB

		// MiB
		{1048576, "1.0 MiB"},
		{1572864, "1.5 MiB"},
		{10485760, "10.0 MiB"},
		{104857600, "100.0 MiB"},
		{524288000, "500.0 MiB"},

		// GiB
		{1073741824, "1.0 GiB"},
		{1610612736, "1.5 GiB"},
		{10737418240, "10.0 GiB"},
		{107374182400, "100.0 GiB"},
		{536870912000, "500.0 GiB"},

		// TiB
		{1099511627776, "1.0 TiB"},
		{1649267441664, "1.5 TiB"},
		{10995116277760, "10.0 TiB"},
		{4398046511104, "4.0 TiB"},

		// PiB
		{1125899906842624, "1.0 PiB"},
		{2251799813685248, "2.0 PiB"},

		// EiB
		{1152921504606846976, "1.0 EiB"},

		// Negative values
		{-1024, "-1.0 KiB"},
		{-1073741824, "-1.0 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesShort(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0K"},
		{1048576, "1.0M"},
		{1073741824, "1.0G"},
		{1099511627776, "1.0T"},
		{1125899906842624, "1.0P"},
		{1152921504606846976, "1.0E"},
		{-1024, "-1.0K"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytesShort(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytesShort(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesPrecise(t *testing.T) {
	tests := []struct {
		bytes     int64
		precision int
		expected  string
	}{
		{0, 2, "0 B"},
		{1536, 2, "1.50 KiB"},
		{1536, 0, "2 KiB"},
		{1073741824, 2, "1.00 GiB"},
		{1610612736, 2, "1.50 GiB"},
		{1610612736, 3, "1.500 GiB"},
		{-1536, 2, "-1.50 KiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytesPrecise(tt.bytes, tt.precision)
			if result != tt.expected {
				t.Errorf("FormatBytesPrecise(%d, %d) = %q, want %q",
					tt.bytes, tt.precision, result, tt.expected)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{0.0, "0.0%"},
		{50.0, "50.0%"},
		{99.9, "99.9%"},
		{100.0, "100.0%"},
		{33.33, "33.3%"},
		{66.666, "66.7%"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatPercent(tt.percent)
			if result != tt.expected {
				t.Errorf("FormatPercent(%f) = %q, want %q", tt.percent, result, tt.expected)
			}
		})
	}
}

func TestFormatStorageUsage(t *testing.T) {
	tests := []struct {
		usedBytes  int64
		totalBytes int64
		expected   string
	}{
		{0, 1073741824, "0 B / 1.0 GiB (0.0%)"},
		{536870912, 1073741824, "512.0 MiB / 1.0 GiB (50.0%)"},
		{1073741824, 1073741824, "1.0 GiB / 1.0 GiB (100.0%)"},
		{1319413953433, 4398046511104, "1.2 TiB / 4.0 TiB (30.0%)"},
		{0, 0, "0 B / 0 B (0.0%)"}, // Edge case: zero total
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatStorageUsage(tt.usedBytes, tt.totalBytes)
			if result != tt.expected {
				t.Errorf("FormatStorageUsage(%d, %d) = %q, want %q",
					tt.usedBytes, tt.totalBytes, result, tt.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify the constants are correct
	if KiB != 1024 {
		t.Errorf("KiB = %d, want 1024", KiB)
	}
	if MiB != 1024*1024 {
		t.Errorf("MiB = %d, want %d", MiB, 1024*1024)
	}
	if GiB != 1024*1024*1024 {
		t.Errorf("GiB = %d, want %d", GiB, 1024*1024*1024)
	}
	if TiB != 1024*1024*1024*1024 {
		t.Errorf("TiB = %d, want %d", TiB, 1024*1024*1024*1024)
	}
	if PiB != 1024*1024*1024*1024*1024 {
		t.Errorf("PiB = %d, want %d", PiB, 1024*1024*1024*1024*1024)
	}
	if EiB != 1024*1024*1024*1024*1024*1024 {
		t.Errorf("EiB = %d, want %d", EiB, 1024*1024*1024*1024*1024*1024)
	}
}

package k8s

import (
	"fmt"
	"math"
	"testing"
)

func TestParseStorageUsage(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantTotalBytes  int64
		wantUsedBytes   int64
		wantAvailBytes  int64
		wantUsedPercent float64
		wantPoolCount   int
		wantErr         bool
	}{
		{
			name: "valid JSON with stats and pools",
			input: `{
				"stats": {
					"total_bytes": 4398046511104,
					"total_used_bytes": 1319413953433,
					"total_avail_bytes": 3078632557671,
					"total_used_raw_bytes": 2638827906866
				},
				"pools": [
					{
						"name": "replicapool",
						"id": 1,
						"stats": {
							"stored": 123456789,
							"objects": 1234,
							"percent_used": 0.03,
							"max_avail": 1000000000
						}
					},
					{
						"name": "device_health_metrics",
						"id": 2,
						"stats": {
							"stored": 987654,
							"objects": 56,
							"percent_used": 0.001,
							"max_avail": 500000000
						}
					}
				]
			}`,
			wantTotalBytes:  4398046511104,
			wantUsedBytes:   1319413953433,
			wantAvailBytes:  3078632557671,
			wantUsedPercent: 30.0, // approximately 30%
			wantPoolCount:   2,
			wantErr:         false,
		},
		{
			name: "empty cluster",
			input: `{
				"stats": {
					"total_bytes": 1000000000000,
					"total_used_bytes": 0,
					"total_avail_bytes": 1000000000000
				},
				"pools": []
			}`,
			wantTotalBytes:  1000000000000,
			wantUsedBytes:   0,
			wantAvailBytes:  1000000000000,
			wantUsedPercent: 0.0,
			wantPoolCount:   0,
			wantErr:         false,
		},
		{
			name: "near full cluster",
			input: `{
				"stats": {
					"total_bytes": 1000000000000,
					"total_used_bytes": 900000000000,
					"total_avail_bytes": 100000000000
				},
				"pools": []
			}`,
			wantTotalBytes:  1000000000000,
			wantUsedBytes:   900000000000,
			wantAvailBytes:  100000000000,
			wantUsedPercent: 90.0,
			wantPoolCount:   0,
			wantErr:         false,
		},
		{
			name: "zero total bytes",
			input: `{
				"stats": {
					"total_bytes": 0,
					"total_used_bytes": 0,
					"total_avail_bytes": 0
				},
				"pools": []
			}`,
			wantTotalBytes:  0,
			wantUsedBytes:   0,
			wantAvailBytes:  0,
			wantUsedPercent: 0.0,
			wantPoolCount:   0,
			wantErr:         false,
		},
		{
			name:    "invalid JSON",
			input:   `not valid json`,
			wantErr: true,
		},
		{
			// Real output captured from test cluster running Ceph Tentacle (v20).
			// Confirms percent_used is a fraction (1.52e-05 = 0.0015%).
			name:            "real cluster output - Ceph Tentacle",
			input:           `{"stats":{"total_bytes":32212254720,"total_avail_bytes":32102449152,"total_used_bytes":109805568,"total_used_raw_bytes":109805568,"total_used_raw_ratio":0.0034088133834302425,"num_osds":3,"num_per_pool_osds":3,"num_per_pool_omap_osds":1},"stats_by_class":{"hdd":{"total_bytes":32212254720,"total_avail_bytes":32102449152,"total_used_bytes":109805568,"total_used_raw_bytes":109805568,"total_used_raw_ratio":0.0034088133834302425}},"pools":[{"name":".mgr","id":1,"stats":{"stored":459280,"objects":2,"kb_used":452,"bytes_used":462848,"percent_used":1.5197146240097936e-05,"max_avail":30455781376}}]}`,
			wantTotalBytes:  32212254720,
			wantUsedBytes:   109805568,
			wantAvailBytes:  32102449152,
			wantUsedPercent: 0.34, // ~0.34% cluster usage
			wantPoolCount:   1,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseStorageUsage(tt.input)

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

			if result.TotalBytes != tt.wantTotalBytes {
				t.Errorf("TotalBytes = %d, want %d", result.TotalBytes, tt.wantTotalBytes)
			}

			if result.UsedBytes != tt.wantUsedBytes {
				t.Errorf("UsedBytes = %d, want %d", result.UsedBytes, tt.wantUsedBytes)
			}

			if result.AvailableBytes != tt.wantAvailBytes {
				t.Errorf("AvailableBytes = %d, want %d", result.AvailableBytes, tt.wantAvailBytes)
			}

			// Check percentage with tolerance for floating point
			if math.Abs(result.UsedPercent-tt.wantUsedPercent) > 1.0 {
				t.Errorf("UsedPercent = %.2f, want approximately %.2f", result.UsedPercent, tt.wantUsedPercent)
			}

			if len(result.Pools) != tt.wantPoolCount {
				t.Errorf("Pool count = %d, want %d", len(result.Pools), tt.wantPoolCount)
			}
		})
	}
}

// TestParseStorageUsage_PoolDetails verifies pool percent_used conversion.
// Ceph JSON API returns percent_used as a fraction (0.0-1.0), not a percentage.
// Source: PGMap.cc dump_object_stat_sum() calculates used = used_bytes/(used_bytes+avail)
// Verified in Quincy, Reef, and Tentacle branches at github.com/ceph/ceph/blob/main/src/mon/PGMap.cc
func TestParseStorageUsage_PoolDetails(t *testing.T) {
	input := `{
		"stats": {
			"total_bytes": 4398046511104,
			"total_used_bytes": 1319413953433,
			"total_avail_bytes": 3078632557671
		},
		"pools": [
			{
				"name": "replicapool",
				"id": 1,
				"stats": {
					"stored": 123456789,
					"objects": 1234,
					"percent_used": 0.03,
					"max_avail": 1000000000
				}
			}
		]
	}`

	result, err := parseStorageUsage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(result.Pools))
	}

	pool := result.Pools[0]

	if pool.Name != "replicapool" {
		t.Errorf("Pool.Name = %q, want %q", pool.Name, "replicapool")
	}

	if pool.ID != 1 {
		t.Errorf("Pool.ID = %d, want %d", pool.ID, 1)
	}

	if pool.StoredBytes != 123456789 {
		t.Errorf("Pool.StoredBytes = %d, want %d", pool.StoredBytes, 123456789)
	}

	if pool.Objects != 1234 {
		t.Errorf("Pool.Objects = %d, want %d", pool.Objects, 1234)
	}

	// percent_used is 0.03 (fraction), converted to 3.0 (percentage)
	if math.Abs(pool.UsedPercent-3.0) > 0.1 {
		t.Errorf("Pool.UsedPercent = %.2f, want approximately 3.0", pool.UsedPercent)
	}

	if pool.MaxAvail != 1000000000 {
		t.Errorf("Pool.MaxAvail = %d, want %d", pool.MaxAvail, 1000000000)
	}
}

// TestParseStorageUsage_RealClusterPoolPercent verifies pool percent_used from real cluster.
// Real output from test cluster: percent_used = 1.5197146240097936e-05 (fraction)
// Expected display: 0.0015197% (after *100 conversion)
func TestParseStorageUsage_RealClusterPoolPercent(t *testing.T) {
	// Actual output from `ceph df --format json` on test cluster running Ceph Tentacle (v20)
	input := `{"stats":{"total_bytes":32212254720,"total_avail_bytes":32102449152,"total_used_bytes":109805568,"total_used_raw_bytes":109805568,"total_used_raw_ratio":0.0034088133834302425,"num_osds":3,"num_per_pool_osds":3,"num_per_pool_omap_osds":1},"stats_by_class":{"hdd":{"total_bytes":32212254720,"total_avail_bytes":32102449152,"total_used_bytes":109805568,"total_used_raw_bytes":109805568,"total_used_raw_ratio":0.0034088133834302425}},"pools":[{"name":".mgr","id":1,"stats":{"stored":459280,"objects":2,"kb_used":452,"bytes_used":462848,"percent_used":1.5197146240097936e-05,"max_avail":30455781376}}]}`

	result, err := parseStorageUsage(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(result.Pools))
	}

	pool := result.Pools[0]

	// Verify pool metadata
	if pool.Name != ".mgr" {
		t.Errorf("Pool.Name = %q, want %q", pool.Name, ".mgr")
	}

	// The critical test: percent_used = 1.5197146240097936e-05 (fraction)
	// After *100 conversion: 0.0015197146240097936 (percentage)
	// This proves we're NOT getting 100x inflation (which would be 0.15%)
	expectedPercent := 1.5197146240097936e-05 * 100 // = 0.0015197...
	if math.Abs(pool.UsedPercent-expectedPercent) > 0.0001 {
		t.Errorf("Pool.UsedPercent = %.10f, want %.10f (fraction*100)", pool.UsedPercent, expectedPercent)
	}

	// Sanity check: if there was a 100x bug, we'd see ~0.15% instead of ~0.0015%
	if pool.UsedPercent > 0.01 {
		t.Errorf("Pool.UsedPercent = %.6f%%, suspiciously high - possible 100x inflation bug", pool.UsedPercent)
	}
}

// TestParseStorageUsage_PoolPercentBoundary verifies pool percent_used boundary cases.
// Ensures correct conversion at 0% and 100% (fraction 0.0 and 1.0).
func TestParseStorageUsage_PoolPercentBoundary(t *testing.T) {
	tests := []struct {
		name               string
		percentUsedInput   float64 // Fraction as returned by Ceph JSON
		wantPercentDisplay float64 // Expected percentage for display
	}{
		{"empty pool", 0.0, 0.0},
		{"nearly empty", 0.001, 0.1},
		{"half full", 0.5, 50.0},
		{"nearly full", 0.9999, 99.99},
		{"completely full", 1.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf(`{
				"stats": {
					"total_bytes": 1000000000000,
					"total_used_bytes": 500000000000,
					"total_avail_bytes": 500000000000
				},
				"pools": [
					{
						"name": "testpool",
						"id": 1,
						"stats": {
							"stored": 100,
							"objects": 1,
							"percent_used": %v,
							"max_avail": 100
						}
					}
				]
			}`, tt.percentUsedInput)

			result, err := parseStorageUsage(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Pools) != 1 {
				t.Fatalf("expected 1 pool, got %d", len(result.Pools))
			}

			if math.Abs(result.Pools[0].UsedPercent-tt.wantPercentDisplay) > 0.01 {
				t.Errorf("Pool.UsedPercent = %.4f, want %.4f", result.Pools[0].UsedPercent, tt.wantPercentDisplay)
			}
		})
	}
}

func TestStorageUsage_IsNearFull(t *testing.T) {
	tests := []struct {
		name        string
		usedPercent float64
		expected    bool
	}{
		{"empty", 0.0, false},
		{"half full", 50.0, false},
		{"84 percent", 84.0, false},
		{"85 percent", 85.0, true},
		{"90 percent", 90.0, true},
		{"95 percent", 95.0, true},
		{"100 percent", 100.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StorageUsage{UsedPercent: tt.usedPercent}
			if got := s.IsNearFull(); got != tt.expected {
				t.Errorf("IsNearFull() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStorageUsage_IsFull(t *testing.T) {
	tests := []struct {
		name        string
		usedPercent float64
		expected    bool
	}{
		{"empty", 0.0, false},
		{"half full", 50.0, false},
		{"85 percent", 85.0, false},
		{"94 percent", 94.0, false},
		{"95 percent", 95.0, true},
		{"100 percent", 100.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StorageUsage{UsedPercent: tt.usedPercent}
			if got := s.IsFull(); got != tt.expected {
				t.Errorf("IsFull() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStorageUsage_GetPoolByName(t *testing.T) {
	s := &StorageUsage{
		Pools: []PoolUsage{
			{Name: "pool1", ID: 1, StoredBytes: 100},
			{Name: "pool2", ID: 2, StoredBytes: 200},
			{Name: "pool3", ID: 3, StoredBytes: 300},
		},
	}

	tests := []struct {
		name     string
		poolName string
		wantNil  bool
		wantID   int
	}{
		{"existing pool", "pool1", false, 1},
		{"another existing pool", "pool2", false, 2},
		{"non-existent pool", "pool4", true, 0},
		{"empty name", "", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.GetPoolByName(tt.poolName)

			if tt.wantNil {
				if result != nil {
					t.Errorf("GetPoolByName(%q) = %v, want nil", tt.poolName, result)
				}
				return
			}

			if result == nil {
				t.Errorf("GetPoolByName(%q) = nil, want pool with ID %d", tt.poolName, tt.wantID)
				return
			}

			if result.ID != tt.wantID {
				t.Errorf("GetPoolByName(%q).ID = %d, want %d", tt.poolName, result.ID, tt.wantID)
			}
		})
	}
}

func TestStorageUsage_GetPoolByName_Empty(t *testing.T) {
	s := &StorageUsage{Pools: nil}

	result := s.GetPoolByName("anypool")
	if result != nil {
		t.Errorf("GetPoolByName on empty pools = %v, want nil", result)
	}
}

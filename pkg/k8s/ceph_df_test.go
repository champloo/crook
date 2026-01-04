package k8s

import (
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

	// percent_used is 0.03 (3%), converted to 3.0
	if math.Abs(pool.UsedPercent-3.0) > 0.1 {
		t.Errorf("Pool.UsedPercent = %.2f, want approximately 3.0", pool.UsedPercent)
	}

	if pool.MaxAvail != 1000000000 {
		t.Errorf("Pool.MaxAvail = %d, want %d", pool.MaxAvail, 1000000000)
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

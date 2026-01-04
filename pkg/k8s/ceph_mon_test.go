package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMonitorStatus_Healthy(t *testing.T) {
	// Load fixture
	fixturePath := filepath.Join("..", "..", "test", "fixtures", "ceph_quorum_status.json")
	data, err := os.ReadFile(fixturePath) //nolint:gosec // G304: test fixture path is hardcoded
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	status, err := parseMonitorStatus(string(data))
	if err != nil {
		t.Fatalf("failed to parse monitor status: %v", err)
	}

	// Verify total count
	if status.TotalCount != 3 {
		t.Errorf("expected TotalCount=3, got %d", status.TotalCount)
	}

	// Verify in quorum count
	if status.InQuorum != 3 {
		t.Errorf("expected InQuorum=3, got %d", status.InQuorum)
	}

	// Verify quorum names
	expectedQuorumNames := []string{"a", "b", "c"}
	if len(status.QuorumNames) != len(expectedQuorumNames) {
		t.Errorf("expected %d quorum names, got %d", len(expectedQuorumNames), len(status.QuorumNames))
	}
	for i, name := range expectedQuorumNames {
		if status.QuorumNames[i] != name {
			t.Errorf("expected quorum name %q at index %d, got %q", name, i, status.QuorumNames[i])
		}
	}

	// Verify leader
	if status.Leader != "a" {
		t.Errorf("expected leader=a, got %s", status.Leader)
	}

	// Verify election epoch
	if status.ElectionEpoch != 12 {
		t.Errorf("expected election_epoch=12, got %d", status.ElectionEpoch)
	}

	// Verify no monitors out of quorum
	if len(status.OutOfQuorum) != 0 {
		t.Errorf("expected no monitors out of quorum, got %v", status.OutOfQuorum)
	}

	// Verify IsHealthy
	if !status.IsHealthy() {
		t.Error("expected IsHealthy()=true for healthy cluster")
	}

	// Verify HasQuorum
	if !status.HasQuorum() {
		t.Error("expected HasQuorum()=true for healthy cluster")
	}
}

func TestParseMonitorStatus_Degraded(t *testing.T) {
	// Load fixture
	fixturePath := filepath.Join("..", "..", "test", "fixtures", "ceph_quorum_status_degraded.json")
	data, err := os.ReadFile(fixturePath) //nolint:gosec // G304: test fixture path is hardcoded
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	status, err := parseMonitorStatus(string(data))
	if err != nil {
		t.Fatalf("failed to parse monitor status: %v", err)
	}

	// Verify total count
	if status.TotalCount != 3 {
		t.Errorf("expected TotalCount=3, got %d", status.TotalCount)
	}

	// Verify in quorum count (only 2 monitors in quorum)
	if status.InQuorum != 2 {
		t.Errorf("expected InQuorum=2, got %d", status.InQuorum)
	}

	// Verify quorum names
	expectedQuorumNames := []string{"a", "c"}
	if len(status.QuorumNames) != len(expectedQuorumNames) {
		t.Errorf("expected %d quorum names, got %d", len(expectedQuorumNames), len(status.QuorumNames))
	}

	// Verify out of quorum monitors
	if len(status.OutOfQuorum) != 1 {
		t.Errorf("expected 1 monitor out of quorum, got %d", len(status.OutOfQuorum))
	}
	if len(status.OutOfQuorum) > 0 && status.OutOfQuorum[0] != "b" {
		t.Errorf("expected monitor 'b' out of quorum, got %q", status.OutOfQuorum[0])
	}

	// Verify leader
	if status.Leader != "a" {
		t.Errorf("expected leader=a, got %s", status.Leader)
	}

	// Verify IsHealthy (should be false since not all monitors are in quorum)
	if status.IsHealthy() {
		t.Error("expected IsHealthy()=false for degraded cluster")
	}

	// Verify HasQuorum (should be true since 2/3 have quorum)
	if !status.HasQuorum() {
		t.Error("expected HasQuorum()=true for degraded cluster (still has majority)")
	}
}

func TestMonitorStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		name        string
		status      MonitorStatus
		wantHealthy bool
	}{
		{
			name: "all monitors in quorum",
			status: MonitorStatus{
				TotalCount: 3,
				InQuorum:   3,
			},
			wantHealthy: true,
		},
		{
			name: "one monitor out",
			status: MonitorStatus{
				TotalCount: 3,
				InQuorum:   2,
			},
			wantHealthy: false,
		},
		{
			name: "no monitors",
			status: MonitorStatus{
				TotalCount: 0,
				InQuorum:   0,
			},
			wantHealthy: false,
		},
		{
			name: "single monitor healthy",
			status: MonitorStatus{
				TotalCount: 1,
				InQuorum:   1,
			},
			wantHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsHealthy()
			if got != tt.wantHealthy {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.wantHealthy)
			}
		})
	}
}

func TestMonitorStatus_HasQuorum(t *testing.T) {
	tests := []struct {
		name       string
		status     MonitorStatus
		wantQuorum bool
	}{
		{
			name: "all 3 monitors in quorum",
			status: MonitorStatus{
				TotalCount: 3,
				InQuorum:   3,
			},
			wantQuorum: true,
		},
		{
			name: "2 of 3 monitors in quorum (majority)",
			status: MonitorStatus{
				TotalCount: 3,
				InQuorum:   2,
			},
			wantQuorum: true,
		},
		{
			name: "1 of 3 monitors (no quorum)",
			status: MonitorStatus{
				TotalCount: 3,
				InQuorum:   1,
			},
			wantQuorum: false,
		},
		{
			name: "0 of 3 monitors (no quorum)",
			status: MonitorStatus{
				TotalCount: 3,
				InQuorum:   0,
			},
			wantQuorum: false,
		},
		{
			name: "5 monitor cluster with 3 in quorum",
			status: MonitorStatus{
				TotalCount: 5,
				InQuorum:   3,
			},
			wantQuorum: true,
		},
		{
			name: "5 monitor cluster with 2 in quorum (no quorum)",
			status: MonitorStatus{
				TotalCount: 5,
				InQuorum:   2,
			},
			wantQuorum: false,
		},
		{
			name: "no monitors",
			status: MonitorStatus{
				TotalCount: 0,
				InQuorum:   0,
			},
			wantQuorum: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.HasQuorum()
			if got != tt.wantQuorum {
				t.Errorf("HasQuorum() = %v, want %v", got, tt.wantQuorum)
			}
		})
	}
}

func TestParseMonitorStatus_InvalidJSON(t *testing.T) {
	_, err := parseMonitorStatus("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseMonitorStatus_EmptyJSON(t *testing.T) {
	_, err := parseMonitorStatus("{}")
	if err != nil {
		t.Fatalf("unexpected error for empty JSON: %v", err)
	}
}

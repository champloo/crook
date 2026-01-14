package maintenance

import (
	"strings"
	"testing"
)

func TestOtherNodesMaintenanceInfo_HasWarning(t *testing.T) {
	tests := []struct {
		name     string
		info     *OtherNodesMaintenanceInfo
		expected bool
	}{
		{
			name: "no warnings - empty",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{},
				NoOutFlagSet:       false,
			},
			expected: false,
		},
		{
			name: "warning - noout flag set",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{},
				NoOutFlagSet:       true,
			},
			expected: true,
		},
		{
			name: "warning - cordoned node",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-1", Cordoned: true},
				},
				NoOutFlagSet: false,
			},
			expected: true,
		},
		{
			name: "warning - node with scaled-down deployments",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-1", HasScaledDownDeployments: true},
				},
				NoOutFlagSet: false,
			},
			expected: true,
		},
		{
			name: "warning - multiple indicators",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-1", Cordoned: true, HasScaledDownDeployments: true},
					{NodeName: "worker-2", Cordoned: true},
				},
				NoOutFlagSet: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.HasWarning()
			if got != tt.expected {
				t.Errorf("HasWarning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOtherNodesMaintenanceInfo_WarningMessage(t *testing.T) {
	tests := []struct {
		name           string
		info           *OtherNodesMaintenanceInfo
		expectEmpty    bool
		expectContains []string
	}{
		{
			name: "no warning - returns empty",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{},
				NoOutFlagSet:       false,
			},
			expectEmpty: true,
		},
		{
			name: "noout flag warning",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{},
				NoOutFlagSet:       true,
			},
			expectEmpty:    false,
			expectContains: []string{"WARNING", "noout", "flag"},
		},
		{
			name: "cordoned node warning",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-1", Cordoned: true},
				},
				NoOutFlagSet: false,
			},
			expectEmpty:    false,
			expectContains: []string{"WARNING", "worker-1", "cordoned"},
		},
		{
			name: "scaled-down deployments warning",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-2", HasScaledDownDeployments: true},
				},
				NoOutFlagSet: false,
			},
			expectEmpty:    false,
			expectContains: []string{"WARNING", "worker-2", "scaled-down"},
		},
		{
			name: "cordoned and scaled-down warning",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-3", Cordoned: true, HasScaledDownDeployments: true},
				},
				NoOutFlagSet: false,
			},
			expectEmpty:    false,
			expectContains: []string{"WARNING", "worker-3", "cordoned", "scaled-down"},
		},
		{
			name: "includes risk explanation",
			info: &OtherNodesMaintenanceInfo{
				NodesInMaintenance: []MaintenanceStatus{
					{NodeName: "worker-1", Cordoned: true},
				},
				NoOutFlagSet: false,
			},
			expectEmpty:    false,
			expectContains: []string{"multiple nodes", "Ceph cluster availability", "redundancy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.WarningMessage()

			if tt.expectEmpty {
				if got != "" {
					t.Errorf("WarningMessage() = %q, want empty string", got)
				}
				return
			}

			if got == "" {
				t.Error("WarningMessage() returned empty string, expected content")
				return
			}

			for _, substring := range tt.expectContains {
				if !strings.Contains(got, substring) {
					t.Errorf("WarningMessage() = %q, expected to contain %q", got, substring)
				}
			}
		})
	}
}

func TestMaintenanceStatus_Fields(t *testing.T) {
	status := MaintenanceStatus{
		NodeName:                 "test-node",
		Cordoned:                 true,
		HasScaledDownDeployments: false,
	}

	if status.NodeName != "test-node" {
		t.Errorf("NodeName = %s, want test-node", status.NodeName)
	}
	if !status.Cordoned {
		t.Error("Cordoned = false, want true")
	}
	if status.HasScaledDownDeployments {
		t.Error("HasScaledDownDeployments = true, want false")
	}
}

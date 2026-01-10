package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/output"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    output.Format
		wantErr bool
	}{
		{name: "table", input: "table", want: output.FormatTable, wantErr: false},
		{name: "json", input: "json", want: output.FormatJSON, wantErr: false},
		{name: "invalid", input: "invalid", wantErr: true},
		{name: "yaml", input: "yaml", wantErr: true},
		{name: "tui", input: "tui", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := output.ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFormat(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFormat(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseResourceTypes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []output.ResourceType
		wantErr bool
	}{
		{
			name:  "empty returns all",
			input: "",
			want:  output.AllResourceTypes(),
		},
		{
			name:  "single type",
			input: "nodes",
			want:  []output.ResourceType{output.ResourceNodes},
		},
		{
			name:  "multiple types",
			input: "nodes,osds",
			want:  []output.ResourceType{output.ResourceNodes, output.ResourceOSDs},
		},
		{
			name:  "with spaces",
			input: "nodes, deployments, pods",
			want:  []output.ResourceType{output.ResourceNodes, output.ResourceDeployments, output.ResourcePods},
		},
		{
			name:  "all types",
			input: "nodes,deployments,osds,pods",
			want:  output.AllResourceTypes(),
		},
		{
			name:    "invalid values",
			input:   "nodes,widgets",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := output.ParseResourceTypes(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseResourceTypes(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseResourceTypes(%q) unexpected error: %v", tt.input, err)
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseResourceTypes(%q) returned %d types, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, rt := range got {
				if rt != tt.want[i] {
					t.Errorf("ParseResourceTypes(%q)[%d] = %v, want %v", tt.input, i, rt, tt.want[i])
				}
			}
		})
	}
}

func createTestData() *output.Data {
	return &output.Data{
		ClusterHealth: &output.ClusterHealth{
			Status:       "HEALTH_OK",
			OSDs:         6,
			OSDsUp:       6,
			OSDsIn:       6,
			MonsTotal:    3,
			MonsInQuorum: 3,
			NooutSet:     false,
			UsedBytes:    1073741824,  // 1 GB
			TotalBytes:   10737418240, // 10 GB
		},
		Nodes: []k8s.NodeInfo{
			{
				Name:           "worker-1",
				Status:         "Ready",
				Roles:          []string{"worker"},
				Schedulable:    true,
				Cordoned:       false,
				CephPodCount:   3,
				Age:            "5d",
				KubeletVersion: "v1.28.0",
			},
			{
				Name:           "worker-2",
				Status:         "Ready",
				Roles:          []string{"worker"},
				Schedulable:    false,
				Cordoned:       true,
				CephPodCount:   2,
				Age:            "5d",
				KubeletVersion: "v1.28.0",
			},
		},
		Deployments: []k8s.DeploymentInfo{
			{
				Name:            "rook-ceph-osd-0",
				Namespace:       "rook-ceph",
				ReadyReplicas:   1,
				DesiredReplicas: 1,
				NodeName:        "worker-1",
				Age:             "5d",
				Status:          "Ready",
				Type:            "osd",
				OsdID:           "0",
			},
		},
		OSDs: []k8s.OSDInfo{
			{
				ID:             0,
				Name:           "osd.0",
				Hostname:       "worker-1",
				Status:         "up",
				InOut:          "in",
				Weight:         1.0,
				Reweight:       1.0,
				DeviceClass:    "ssd",
				DeploymentName: "rook-ceph-osd-0",
			},
		},
		Pods: []k8s.PodInfo{
			{
				Name:            "rook-ceph-osd-0-abc123",
				Namespace:       "rook-ceph",
				Status:          "Running",
				ReadyContainers: 1,
				TotalContainers: 1,
				Restarts:        0,
				NodeName:        "worker-1",
				Age:             "5d",
				Type:            "osd",
				IP:              "10.0.0.1",
				OwnerDeployment: "rook-ceph-osd-0",
			},
		},
		FetchedAt: time.Now(),
	}
}

func TestRenderJSON(t *testing.T) {
	data := createTestData()
	var buf bytes.Buffer

	err := output.RenderJSON(&buf, data)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if parseErr := json.Unmarshal(buf.Bytes(), &parsed); parseErr != nil {
		t.Fatalf("RenderJSON() produced invalid JSON: %v", parseErr)
	}

	// Verify structure
	if _, ok := parsed["cluster_health"]; !ok {
		t.Error("RenderJSON() missing cluster_health")
	}
	if _, ok := parsed["nodes"]; !ok {
		t.Error("RenderJSON() missing nodes")
	}
	if _, ok := parsed["deployments"]; !ok {
		t.Error("RenderJSON() missing deployments")
	}
	if _, ok := parsed["osds"]; !ok {
		t.Error("RenderJSON() missing osds")
	}
	if _, ok := parsed["pods"]; !ok {
		t.Error("RenderJSON() missing pods")
	}
}

func TestRenderJSONParseable(t *testing.T) {
	data := createTestData()
	var buf bytes.Buffer

	err := output.RenderJSON(&buf, data)
	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	// Verify JSON is valid and contains expected fields
	var parsed map[string]any
	if parseErr := json.Unmarshal(buf.Bytes(), &parsed); parseErr != nil {
		t.Fatalf("RenderJSON() output not parseable: %v", parseErr)
	}

	health, ok := parsed["cluster_health"].(map[string]any)
	if !ok {
		t.Fatal("missing cluster_health")
	}
	if health["status"] != "HEALTH_OK" {
		t.Errorf("Parsed health status = %v, want %q", health["status"], "HEALTH_OK")
	}

	nodes, ok := parsed["nodes"].([]any)
	if !ok {
		t.Fatal("missing nodes")
	}
	if len(nodes) != 2 {
		t.Errorf("Parsed nodes count = %d, want %d", len(nodes), 2)
	}
}

func TestRenderTable(t *testing.T) {
	data := createTestData()
	var buf bytes.Buffer

	err := output.RenderTable(&buf, data)
	if err != nil {
		t.Fatalf("RenderTable() error: %v", err)
	}

	tableOutput := buf.String()

	// Verify sections are present
	if !strings.Contains(tableOutput, "NODES") {
		t.Error("RenderTable() missing NODES section")
	}
	if !strings.Contains(tableOutput, "DEPLOYMENTS") {
		t.Error("RenderTable() missing DEPLOYMENTS section")
	}
	if !strings.Contains(tableOutput, "OSDS") {
		t.Error("RenderTable() missing OSDS section")
	}
	if !strings.Contains(tableOutput, "PODS") {
		t.Error("RenderTable() missing PODS section")
	}

	// Verify data is present
	if !strings.Contains(tableOutput, "worker-1") {
		t.Error("RenderTable() missing node data")
	}
	if !strings.Contains(tableOutput, "rook-ceph-osd-0") {
		t.Error("RenderTable() missing deployment data")
	}
	if !strings.Contains(tableOutput, "osd.0") {
		t.Error("RenderTable() missing OSD data")
	}
}

func TestRenderTableHealthStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "healthy", status: "HEALTH_OK", want: "HEALTH_OK"},
		{name: "warning", status: "HEALTH_WARN", want: "HEALTH_WARN"},
		{name: "error", status: "HEALTH_ERR", want: "HEALTH_ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &output.Data{
				ClusterHealth: &output.ClusterHealth{
					Status: tt.status,
				},
			}

			var buf bytes.Buffer
			err := output.RenderTable(&buf, data)
			if err != nil {
				t.Fatalf("RenderTable() error: %v", err)
			}

			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("RenderTable() output missing %q", tt.want)
			}
		})
	}
}

func TestRenderTableNoData(t *testing.T) {
	data := &output.Data{
		FetchedAt: time.Now(),
	}

	var buf bytes.Buffer
	err := output.RenderTable(&buf, data)
	if err != nil {
		t.Fatalf("RenderTable() error: %v", err)
	}

	// Should not have section headers when no data
	if strings.Contains(buf.String(), "=== NODES") {
		t.Error("RenderTable() should not show NODES section when no nodes")
	}
}

func TestRender(t *testing.T) {
	data := createTestData()

	tests := []struct {
		name    string
		format  output.Format
		wantErr bool
	}{
		{name: "table", format: output.FormatTable, wantErr: false},
		{name: "json", format: output.FormatJSON, wantErr: false},
		{name: "invalid", format: output.Format("invalid"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := output.Render(&buf, data, tt.format)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Render(%v) expected error, got nil", tt.format)
				}
				return
			}
			if err != nil {
				t.Errorf("Render(%v) unexpected error: %v", tt.format, err)
			}
		})
	}
}

func TestAllResourceTypes(t *testing.T) {
	types := output.AllResourceTypes()

	if len(types) != 4 {
		t.Errorf("AllResourceTypes() returned %d types, want 4", len(types))
	}

	// Verify all expected types are present
	expected := []output.ResourceType{
		output.ResourceNodes,
		output.ResourceDeployments,
		output.ResourceOSDs,
		output.ResourcePods,
	}

	for i, rt := range expected {
		if types[i] != rt {
			t.Errorf("AllResourceTypes()[%d] = %v, want %v", i, types[i], rt)
		}
	}
}

package k8s

import (
	"os"
	"testing"
)

func TestParseOSDTree(t *testing.T) {
	// Read the test fixture
	data, err := os.ReadFile("../../test/fixtures/ceph_osd_tree.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	tree, err := ParseOSDTree(string(data))
	if err != nil {
		t.Fatalf("ParseOSDTree() error = %v", err)
	}

	if len(tree.Nodes) == 0 {
		t.Error("tree.Nodes should not be empty")
	}

	// Count OSD nodes
	osdCount := 0
	for _, node := range tree.Nodes {
		if node.Type == "osd" {
			osdCount++
		}
	}

	if osdCount != 6 {
		t.Errorf("expected 6 OSDs, got %d", osdCount)
	}

	// Check specific OSD
	var osd0 *CephOSDNode
	for i := range tree.Nodes {
		if tree.Nodes[i].Name == "osd.0" {
			osd0 = &tree.Nodes[i]
			break
		}
	}

	if osd0 == nil {
		t.Fatal("osd.0 not found")
	}

	if osd0.ID != 0 {
		t.Errorf("osd.0 ID = %d, want 0", osd0.ID)
	}

	if osd0.Status != "up" {
		t.Errorf("osd.0 Status = %s, want up", osd0.Status)
	}

	if osd0.DeviceClass != "ssd" {
		t.Errorf("osd.0 DeviceClass = %s, want ssd", osd0.DeviceClass)
	}
}

func TestParseOSDTreeWithDown(t *testing.T) {
	// Read the test fixture with down OSDs
	data, err := os.ReadFile("../../test/fixtures/ceph_osd_tree_with_down.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	tree, err := ParseOSDTree(string(data))
	if err != nil {
		t.Fatalf("ParseOSDTree() error = %v", err)
	}

	// Find the down OSD
	var downOSD *CephOSDNode
	for i := range tree.Nodes {
		if tree.Nodes[i].Name == "osd.1" {
			downOSD = &tree.Nodes[i]
			break
		}
	}

	if downOSD == nil {
		t.Fatal("osd.1 not found")
	}

	if downOSD.Status != "down" {
		t.Errorf("osd.1 Status = %s, want down", downOSD.Status)
	}

	if downOSD.Reweight != 0 {
		t.Errorf("osd.1 Reweight = %f, want 0", downOSD.Reweight)
	}
}

func TestBuildHostnameMap(t *testing.T) {
	// Read the test fixture
	data, err := os.ReadFile("../../test/fixtures/ceph_osd_tree.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	tree, err := ParseOSDTree(string(data))
	if err != nil {
		t.Fatalf("ParseOSDTree() error = %v", err)
	}

	hostMap := buildHostnameMap(tree)

	// Check OSD 0 and 1 are on worker-1
	if hostMap[0] != "worker-1" {
		t.Errorf("OSD 0 hostname = %s, want worker-1", hostMap[0])
	}
	if hostMap[1] != "worker-1" {
		t.Errorf("OSD 1 hostname = %s, want worker-1", hostMap[1])
	}

	// Check OSD 2 and 3 are on worker-2
	if hostMap[2] != "worker-2" {
		t.Errorf("OSD 2 hostname = %s, want worker-2", hostMap[2])
	}
	if hostMap[3] != "worker-2" {
		t.Errorf("OSD 3 hostname = %s, want worker-2", hostMap[3])
	}

	// Check OSD 4 and 5 are on worker-3
	if hostMap[4] != "worker-3" {
		t.Errorf("OSD 4 hostname = %s, want worker-3", hostMap[4])
	}
	if hostMap[5] != "worker-3" {
		t.Errorf("OSD 5 hostname = %s, want worker-3", hostMap[5])
	}
}

func TestParseOSDTree_InvalidJSON(t *testing.T) {
	_, err := ParseOSDTree("invalid json")
	if err == nil {
		t.Error("ParseOSDTree() should return error for invalid JSON")
	}
}

func TestParseOSDTree_EmptyNodes(t *testing.T) {
	tree, err := ParseOSDTree(`{"nodes": []}`)
	if err != nil {
		t.Fatalf("ParseOSDTree() error = %v", err)
	}

	if len(tree.Nodes) != 0 {
		t.Errorf("expected empty nodes, got %d", len(tree.Nodes))
	}
}

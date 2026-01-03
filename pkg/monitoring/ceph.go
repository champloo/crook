package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andri/crook/pkg/k8s"
)

// CephHealth represents the health status of a Ceph cluster
type CephHealth struct {
	OverallStatus  string
	HealthMessages []string
	OSDCount       int
	OSDsUp         int
	OSDsIn         int
	MonCount       int
	DataUsed       uint64
	DataTotal      uint64
	DataAvailable  uint64
	PGStates       map[string]int
	LastUpdateTime time.Time
}

// cephStatusJSON represents the full JSON structure from 'ceph status --format json'
type cephStatusJSON struct {
	Health struct {
		Status string `json:"status"`
		Checks map[string]struct {
			Severity string `json:"severity"`
			Summary  struct {
				Message string `json:"message"`
			} `json:"summary"`
		} `json:"checks"`
	} `json:"health"`
	OSDMap struct {
		NumOSDs   int `json:"num_osds"`
		NumUpOSDs int `json:"num_up_osds"`
		NumInOSDs int `json:"num_in_osds"`
	} `json:"osdmap"`
	MonMap struct {
		NumMons int `json:"num_mons"`
	} `json:"monmap"`
	PGMap struct {
		NumPGs       int            `json:"num_pgs"`
		PGsByState   []pgStateEntry `json:"pgs_by_state"`
		BytesUsed    uint64         `json:"bytes_used"`
		BytesTotal   uint64         `json:"bytes_total"`
		BytesAvail   uint64         `json:"bytes_avail"`
		DataBytes    uint64         `json:"data_bytes"`
	} `json:"pgmap"`
}

type pgStateEntry struct {
	StateName string `json:"state_name"`
	Count     int    `json:"count"`
}

// HealthColor returns a color indicator for the Ceph health status
func (ch *CephHealth) HealthColor() string {
	switch ch.OverallStatus {
	case "HEALTH_OK":
		return "green"
	case "HEALTH_WARN":
		return "yellow"
	case "HEALTH_ERR":
		return "red"
	default:
		return "yellow"
	}
}

// IsHealthy returns true if the cluster is healthy
func (ch *CephHealth) IsHealthy() bool {
	return ch.OverallStatus == "HEALTH_OK"
}

// MonitorCephHealth retrieves the current health status of the Ceph cluster
func MonitorCephHealth(ctx context.Context, client *k8s.Client, namespace string) (*CephHealth, error) {
	// Execute 'ceph status --format json'
	output, err := client.ExecuteCephCommand(ctx, namespace, []string{"ceph", "status", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to execute ceph status: %w", err)
	}

	// Parse the JSON output
	var status cephStatusJSON
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse ceph status JSON: %w", err)
	}

	// Extract health messages
	var healthMessages []string
	for _, check := range status.Health.Checks {
		if check.Summary.Message != "" {
			healthMessages = append(healthMessages, check.Summary.Message)
		}
	}

	// Convert PG states to map
	pgStates := make(map[string]int)
	for _, pgState := range status.PGMap.PGsByState {
		pgStates[pgState.StateName] = pgState.Count
	}

	health := &CephHealth{
		OverallStatus:  status.Health.Status,
		HealthMessages: healthMessages,
		OSDCount:       status.OSDMap.NumOSDs,
		OSDsUp:         status.OSDMap.NumUpOSDs,
		OSDsIn:         status.OSDMap.NumInOSDs,
		MonCount:       status.MonMap.NumMons,
		DataUsed:       status.PGMap.BytesUsed,
		DataTotal:      status.PGMap.BytesTotal,
		DataAvailable:  status.PGMap.BytesAvail,
		PGStates:       pgStates,
		LastUpdateTime: time.Now(),
	}

	return health, nil
}

// StartCephHealthMonitoring starts background monitoring of Ceph health with the given refresh interval
func StartCephHealthMonitoring(ctx context.Context, client *k8s.Client, namespace string, interval time.Duration) <-chan *CephHealth {
	updates := make(chan *CephHealth, 1)

	go func() {
		defer close(updates)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Send initial status
		if health, err := MonitorCephHealth(ctx, client, namespace); err == nil {
			select {
			case updates <- health:
			case <-ctx.Done():
				return
			}
		}

		// Send periodic updates
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				health, err := MonitorCephHealth(ctx, client, namespace)
				if err != nil {
					// Continue monitoring even if we get an error
					continue
				}

				select {
				case updates <- health:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return updates
}

// OSDStatus represents the status of an individual OSD
type OSDStatus struct {
	ID             int
	Name           string
	Up             bool
	In             bool
	Weight         float64
	NodeName       string
	DeploymentName string
}

// OSDTreeStatus represents the OSD tree structure with node filtering
type OSDTreeStatus struct {
	OSDs           []OSDStatus
	NoOutSet       bool
	LastUpdateTime time.Time
}

// cephOSDTreeJSON represents the JSON structure from 'ceph osd tree --format json'
type cephOSDTreeJSON struct {
	Nodes []struct {
		ID       int     `json:"id"`
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Status   string  `json:"status"`
		Reweight float64 `json:"reweight"`
		CrushWeight float64 `json:"crush_weight"`
		Children []int   `json:"children,omitempty"`
	} `json:"nodes"`
	Stray []interface{} `json:"stray"`
}

// MonitorOSDStatus retrieves the status of OSDs on a specific node
func MonitorOSDStatus(ctx context.Context, client *k8s.Client, namespace, nodeName string) (*OSDTreeStatus, error) {
	// Execute 'ceph osd tree --format json'
	output, err := client.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "tree", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to execute ceph osd tree: %w", err)
	}

	// Parse the JSON output
	var tree cephOSDTreeJSON
	if err := json.Unmarshal([]byte(output), &tree); err != nil {
		return nil, fmt.Errorf("failed to parse ceph osd tree JSON: %w", err)
	}

	// Check if noout is set
	noOutSet, err := checkNoOutFlag(ctx, client, namespace)
	if err != nil {
		// Don't fail on this - it's not critical
		noOutSet = false
	}

	// Build a map of node ID to node name for hosts
	nodeIDToName := make(map[int]string)
	for _, node := range tree.Nodes {
		if node.Type == "host" {
			nodeIDToName[node.ID] = node.Name
		}
	}

	// Build a map of OSD ID to parent host ID
	osdToHost := make(map[int]int)
	for _, node := range tree.Nodes {
		if node.Type == "host" {
			for _, childID := range node.Children {
				osdToHost[childID] = node.ID
			}
		}
	}

	// Extract OSDs for the target node
	var osds []OSDStatus
	for _, node := range tree.Nodes {
		if node.Type == "osd" {
			hostID, hasHost := osdToHost[node.ID]
			if !hasHost {
				continue
			}

			hostName := nodeIDToName[hostID]
			// Filter by node name if specified
			if nodeName != "" && hostName != nodeName {
				continue
			}

			osd := OSDStatus{
				ID:       node.ID,
				Name:     node.Name,
				Up:       node.Status == "up",
				In:       node.Reweight > 0,
				Weight:   node.Reweight,
				NodeName: hostName,
				DeploymentName: fmt.Sprintf("rook-ceph-osd-%d", node.ID),
			}

			osds = append(osds, osd)
		}
	}

	status := &OSDTreeStatus{
		OSDs:           osds,
		NoOutSet:       noOutSet,
		LastUpdateTime: time.Now(),
	}

	return status, nil
}

// checkNoOutFlag checks if the noout flag is set
func checkNoOutFlag(ctx context.Context, client *k8s.Client, namespace string) (bool, error) {
	output, err := client.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "dump", "--format", "json"})
	if err != nil {
		return false, fmt.Errorf("failed to execute ceph osd dump: %w", err)
	}

	var dump struct {
		Flags string `json:"flags"`
	}
	if err := json.Unmarshal([]byte(output), &dump); err != nil {
		return false, fmt.Errorf("failed to parse ceph osd dump JSON: %w", err)
	}

	// Check if "noout" is in the flags string
	for _, flag := range []string{"noout"} {
		if containsFlag(dump.Flags, flag) {
			return true, nil
		}
	}

	return false, nil
}

// containsFlag checks if a flag string contains a specific flag
func containsFlag(flags, flag string) bool {
	// Flags are comma-separated in the output
	for i := 0; i < len(flags); {
		j := i
		for j < len(flags) && flags[j] != ',' {
			j++
		}
		if flags[i:j] == flag {
			return true
		}
		i = j + 1
	}
	return false
}

// StartOSDMonitoring starts background monitoring of OSDs with the given refresh interval
func StartOSDMonitoring(ctx context.Context, client *k8s.Client, namespace, nodeName string, interval time.Duration) <-chan *OSDTreeStatus {
	updates := make(chan *OSDTreeStatus, 1)

	go func() {
		defer close(updates)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Send initial status
		if status, err := MonitorOSDStatus(ctx, client, namespace, nodeName); err == nil {
			select {
			case updates <- status:
			case <-ctx.Done():
				return
			}
		}

		// Send periodic updates
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status, err := MonitorOSDStatus(ctx, client, namespace, nodeName)
				if err != nil {
					// Continue monitoring even if we get an error
					continue
				}

				select {
				case updates <- status:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return updates
}

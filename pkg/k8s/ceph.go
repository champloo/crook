package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CephStatus represents the parsed output of 'ceph status --format json'
type CephStatus struct {
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
	OSDMap struct {
		NumOSDs   int  `json:"num_osds"`
		NumUpOSDs int  `json:"num_up_osds"`
		NumInOSDs int  `json:"num_in_osds"`
		Full      bool `json:"full"`
		NearFull  bool `json:"nearfull"`
	} `json:"osdmap"`
}

// CephOSDTree represents the parsed output of 'ceph osd tree --format json'
type CephOSDTree struct {
	Nodes []CephOSDNode `json:"nodes"`
}

// CephOSDNode represents a node in the OSD tree
type CephOSDNode struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Reweight    float64 `json:"reweight"`
	CrushWeight float64 `json:"crush_weight"`
	DeviceClass string  `json:"device_class"`
	Children    []int   `json:"children,omitempty"`
}

// ExecuteCephCommand executes a Ceph command via the rook-ceph-tools pod
func (c *Client) ExecuteCephCommand(ctx context.Context, namespace string, command []string) (string, error) {
	// Find the rook-ceph-tools pod
	pod, err := c.findRookCephToolsPod(ctx, namespace)
	if err != nil {
		return "", err
	}

	// Execute the command in the pod
	output, err := c.ExecInPod(ctx, namespace, pod.Name, "", command)
	if err != nil {
		return "", fmt.Errorf("failed to execute ceph command: %w", err)
	}

	return output, nil
}

// ExecuteCephCommand is a package-level function that uses the global client
func ExecuteCephCommand(ctx context.Context, namespace string, command []string) (string, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return "", err
	}
	return client.ExecuteCephCommand(ctx, namespace, command)
}

// findRookCephToolsPod finds a ready rook-ceph-tools pod in the namespace
func (c *Client) findRookCephToolsPod(ctx context.Context, namespace string) (*corev1.Pod, error) {
	// List pods with label selector for rook-ceph-tools
	listOptions := metav1.ListOptions{
		LabelSelector: "app=rook-ceph-tools",
	}

	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list rook-ceph-tools pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf(
			"no rook-ceph-tools pod found in namespace %s. "+
				"Please ensure the rook-ceph-tools deployment is running. "+
				"See https://rook.io/docs/rook/latest/Troubleshooting/ceph-toolbox/",
			namespace,
		)
	}

	// Find a ready pod
	for _, pod := range podList.Items {
		if isPodReady(&pod) {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf(
		"no ready rook-ceph-tools pod found in namespace %s. "+
			"Found %d pod(s) but none are ready",
		namespace,
		len(podList.Items),
	)
}

// isPodReady checks if a pod is in the Ready state
func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}

// SetNoOut sets the Ceph noout flag
func (c *Client) SetNoOut(ctx context.Context, namespace string) error {
	_, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "set", "noout"})
	if err != nil {
		return fmt.Errorf("failed to set noout flag: %w", err)
	}
	return nil
}

// SetNoOut is a package-level function that uses the global client
func SetNoOut(ctx context.Context, namespace string) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.SetNoOut(ctx, namespace)
}

// UnsetNoOut unsets the Ceph noout flag
func (c *Client) UnsetNoOut(ctx context.Context, namespace string) error {
	_, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "unset", "noout"})
	if err != nil {
		return fmt.Errorf("failed to unset noout flag: %w", err)
	}
	return nil
}

// UnsetNoOut is a package-level function that uses the global client
func UnsetNoOut(ctx context.Context, namespace string) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.UnsetNoOut(ctx, namespace)
}

// GetCephStatus gets the Ceph cluster status
func (c *Client) GetCephStatus(ctx context.Context, namespace string) (*CephStatus, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "status", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph status: %w", err)
	}

	var status CephStatus
	if unmarshalErr := json.Unmarshal([]byte(output), &status); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse ceph status JSON: %w", unmarshalErr)
	}

	return &status, nil
}

// GetCephStatus is a package-level function that uses the global client
func GetCephStatus(ctx context.Context, namespace string) (*CephStatus, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetCephStatus(ctx, namespace)
}

// GetOSDTree gets the Ceph OSD tree
func (c *Client) GetOSDTree(ctx context.Context, namespace string) (*CephOSDTree, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "tree", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph osd tree: %w", err)
	}

	var tree CephOSDTree
	if unmarshalErr := json.Unmarshal([]byte(output), &tree); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse ceph osd tree JSON: %w", unmarshalErr)
	}

	return &tree, nil
}

// GetOSDTree is a package-level function that uses the global client
func GetOSDTree(ctx context.Context, namespace string) (*CephOSDTree, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetOSDTree(ctx, namespace)
}

// IsHealthy checks if the Ceph cluster is healthy
func (s *CephStatus) IsHealthy() bool {
	return strings.ToUpper(s.Health.Status) == "HEALTH_OK"
}

// IsWarning checks if the Ceph cluster has warnings
func (s *CephStatus) IsWarning() bool {
	return strings.ToUpper(s.Health.Status) == "HEALTH_WARN"
}

// IsError checks if the Ceph cluster has errors
func (s *CephStatus) IsError() bool {
	return strings.ToUpper(s.Health.Status) == "HEALTH_ERR"
}

// CephFlags represents the state of Ceph cluster flags
type CephFlags struct {
	NoOut       bool `json:"noout"`
	NoIn        bool `json:"noin"`
	NoDown      bool `json:"nodown"`
	NoUp        bool `json:"noup"`
	NoRebalance bool `json:"norebalance"`
	NoRecover   bool `json:"norecover"`
	NoScrub     bool `json:"noscrub"`
	NoDeepScrub bool `json:"nodeep-scrub"`
	NoBackfill  bool `json:"nobackfill"`
	Pause       bool `json:"pause"`
}

// cephOSDDump represents the parsed output of 'ceph osd dump --format json'
type cephOSDDump struct {
	Flags string `json:"flags"`
}

// GetCephFlags gets the current Ceph cluster flags
func (c *Client) GetCephFlags(ctx context.Context, namespace string) (*CephFlags, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "dump", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph osd dump: %w", err)
	}

	return parseCephFlags(output)
}

// GetCephFlags is a package-level function that uses the global client
func GetCephFlags(ctx context.Context, namespace string) (*CephFlags, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetCephFlags(ctx, namespace)
}

// parseCephFlags parses the flags from ceph osd dump output
func parseCephFlags(output string) (*CephFlags, error) {
	var dump cephOSDDump
	if err := json.Unmarshal([]byte(output), &dump); err != nil {
		return nil, fmt.Errorf("failed to parse ceph osd dump JSON: %w", err)
	}

	return parseFlagsString(dump.Flags), nil
}

// parseFlagsString parses a comma-separated flags string into CephFlags
func parseFlagsString(flagsStr string) *CephFlags {
	flags := &CephFlags{}
	if flagsStr == "" {
		return flags
	}

	// Flags come as comma-separated values like "noout,nodown,sortbitwise,..."
	flagsList := strings.Split(flagsStr, ",")
	flagSet := make(map[string]bool, len(flagsList))
	for _, f := range flagsList {
		flagSet[strings.TrimSpace(f)] = true
	}

	// Map known flags
	flags.NoOut = flagSet["noout"]
	flags.NoIn = flagSet["noin"]
	flags.NoDown = flagSet["nodown"]
	flags.NoUp = flagSet["noup"]
	flags.NoRebalance = flagSet["norebalance"]
	flags.NoRecover = flagSet["norecover"]
	flags.NoScrub = flagSet["noscrub"]
	flags.NoDeepScrub = flagSet["nodeep-scrub"]
	flags.NoBackfill = flagSet["nobackfill"]
	flags.Pause = flagSet["pause"]

	return flags
}

// HasMaintenanceFlags returns true if any maintenance-related flags are set
func (f *CephFlags) HasMaintenanceFlags() bool {
	return f.NoOut || f.NoIn || f.NoDown || f.NoUp
}

// HasScrubFlags returns true if any scrub-related flags are set
func (f *CephFlags) HasScrubFlags() bool {
	return f.NoScrub || f.NoDeepScrub
}

// HasRecoveryFlags returns true if any recovery-related flags are set
func (f *CephFlags) HasRecoveryFlags() bool {
	return f.NoRebalance || f.NoRecover || f.NoBackfill
}

// ActiveFlags returns a slice of all currently active flag names
func (f *CephFlags) ActiveFlags() []string {
	var active []string
	if f.NoOut {
		active = append(active, "noout")
	}
	if f.NoIn {
		active = append(active, "noin")
	}
	if f.NoDown {
		active = append(active, "nodown")
	}
	if f.NoUp {
		active = append(active, "noup")
	}
	if f.NoRebalance {
		active = append(active, "norebalance")
	}
	if f.NoRecover {
		active = append(active, "norecover")
	}
	if f.NoScrub {
		active = append(active, "noscrub")
	}
	if f.NoDeepScrub {
		active = append(active, "nodeep-scrub")
	}
	if f.NoBackfill {
		active = append(active, "nobackfill")
	}
	if f.Pause {
		active = append(active, "pause")
	}
	return active
}

// StorageUsage represents Ceph cluster storage statistics
type StorageUsage struct {
	TotalBytes     int64       `json:"total_bytes"`
	UsedBytes      int64       `json:"used_bytes"`
	AvailableBytes int64       `json:"available_bytes"`
	UsedPercent    float64     `json:"used_percent"`
	Pools          []PoolUsage `json:"pools,omitempty"`
}

// PoolUsage represents storage statistics for a single Ceph pool
type PoolUsage struct {
	Name        string  `json:"name"`
	ID          int     `json:"id"`
	StoredBytes int64   `json:"stored_bytes"`
	UsedPercent float64 `json:"used_percent"`
	MaxAvail    int64   `json:"max_avail"`
	Objects     int64   `json:"objects"`
}

// cephDF represents the parsed output of 'ceph df --format json'
type cephDF struct {
	Stats struct {
		TotalBytes        int64 `json:"total_bytes"`
		TotalUsedBytes    int64 `json:"total_used_bytes"`
		TotalAvailBytes   int64 `json:"total_avail_bytes"`
		TotalUsedRawBytes int64 `json:"total_used_raw_bytes"`
	} `json:"stats"`
	Pools []struct {
		Name  string `json:"name"`
		ID    int    `json:"id"`
		Stats struct {
			Stored      int64   `json:"stored"`
			Objects     int64   `json:"objects"`
			PercentUsed float64 `json:"percent_used"`
			MaxAvail    int64   `json:"max_avail"`
		} `json:"stats"`
	} `json:"pools"`
}

// GetStorageUsage gets the Ceph cluster storage usage statistics
func (c *Client) GetStorageUsage(ctx context.Context, namespace string) (*StorageUsage, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "df", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph df: %w", err)
	}

	return parseStorageUsage(output)
}

// GetStorageUsage is a package-level function that uses the global client
func GetStorageUsage(ctx context.Context, namespace string) (*StorageUsage, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetStorageUsage(ctx, namespace)
}

// parseStorageUsage parses the output of 'ceph df --format json'
func parseStorageUsage(output string) (*StorageUsage, error) {
	var df cephDF
	if err := json.Unmarshal([]byte(output), &df); err != nil {
		return nil, fmt.Errorf("failed to parse ceph df JSON: %w", err)
	}

	usage := &StorageUsage{
		TotalBytes:     df.Stats.TotalBytes,
		UsedBytes:      df.Stats.TotalUsedBytes,
		AvailableBytes: df.Stats.TotalAvailBytes,
	}

	// Calculate percentage (avoid division by zero)
	if df.Stats.TotalBytes > 0 {
		usage.UsedPercent = float64(df.Stats.TotalUsedBytes) / float64(df.Stats.TotalBytes) * 100
	}

	// Parse pool statistics
	for _, pool := range df.Pools {
		usage.Pools = append(usage.Pools, PoolUsage{
			Name:        pool.Name,
			ID:          pool.ID,
			StoredBytes: pool.Stats.Stored,
			UsedPercent: pool.Stats.PercentUsed * 100, // Convert from decimal to percentage
			MaxAvail:    pool.Stats.MaxAvail,
			Objects:     pool.Stats.Objects,
		})
	}

	return usage, nil
}

// IsNearFull returns true if storage usage is above 85%
func (s *StorageUsage) IsNearFull() bool {
	return s.UsedPercent >= 85.0
}

// IsFull returns true if storage usage is above 95%
func (s *StorageUsage) IsFull() bool {
	return s.UsedPercent >= 95.0
}

// GetPoolByName returns the pool usage for a named pool, or nil if not found
func (s *StorageUsage) GetPoolByName(name string) *PoolUsage {
	for i := range s.Pools {
		if s.Pools[i].Name == name {
			return &s.Pools[i]
		}
	}
	return nil
}

// OSDInfoForLS holds OSD information for the ls command view
type OSDInfoForLS struct {
	// ID is the numeric OSD ID
	ID int

	// Name is the OSD name ('osd.0' format)
	Name string

	// Hostname is the node hostname from CRUSH tree
	Hostname string

	// Status is the OSD status ('up' or 'down')
	Status string

	// InOut indicates if OSD is 'in' or 'out' of the cluster
	InOut string

	// Weight is the CRUSH weight
	Weight float64

	// Reweight is the OSD reweight value
	Reweight float64

	// DeviceClass is the device class (hdd/ssd/nvme)
	DeviceClass string

	// DeploymentName is the mapped K8s deployment name
	DeploymentName string

	// PGCount is the number of primary PGs (if available)
	PGCount int
}

// GetOSDInfoList returns a list of OSD info with hostname mappings from the CRUSH tree
func (c *Client) GetOSDInfoList(ctx context.Context, namespace string) ([]OSDInfoForLS, error) {
	// Get the OSD tree
	tree, err := c.GetOSDTree(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get osd tree: %w", err)
	}

	// Build hostname map (osd id -> hostname)
	hostMap := buildHostnameMap(tree)

	// Extract OSDs
	result := make([]OSDInfoForLS, 0)
	for _, node := range tree.Nodes {
		if node.Type != "osd" {
			continue
		}

		inOut := "in"
		if node.Reweight == 0 {
			inOut = "out"
		}

		info := OSDInfoForLS{
			ID:             node.ID,
			Name:           node.Name,
			Hostname:       hostMap[node.ID],
			Status:         node.Status,
			InOut:          inOut,
			Weight:         node.CrushWeight,
			Reweight:       node.Reweight,
			DeviceClass:    node.DeviceClass,
			DeploymentName: fmt.Sprintf("rook-ceph-osd-%d", node.ID),
		}
		result = append(result, info)
	}

	return result, nil
}

// GetOSDInfoList is a package-level function that uses the global client
func GetOSDInfoList(ctx context.Context, namespace string) ([]OSDInfoForLS, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetOSDInfoList(ctx, namespace)
}

// buildHostnameMap builds a map of OSD ID to hostname from the CRUSH tree
func buildHostnameMap(tree *CephOSDTree) map[int]string {
	hostMap := make(map[int]string)

	// First pass: find all host nodes and their children
	for _, node := range tree.Nodes {
		if node.Type == "host" {
			for _, childID := range node.Children {
				hostMap[childID] = node.Name
			}
		}
	}

	return hostMap
}

// ParseOSDTree parses the JSON output of 'ceph osd tree --format json'
func ParseOSDTree(output string) (*CephOSDTree, error) {
	var tree CephOSDTree
	if err := json.Unmarshal([]byte(output), &tree); err != nil {
		return nil, fmt.Errorf("failed to parse ceph osd tree JSON: %w", err)
	}
	return &tree, nil
}

// MonitorStatus represents the Ceph monitor quorum status
type MonitorStatus struct {
	// TotalCount is the total number of monitors in the cluster
	TotalCount int `json:"total_count"`

	// InQuorum is the number of monitors currently in quorum
	InQuorum int `json:"in_quorum"`

	// QuorumNames is the list of monitor names in quorum
	QuorumNames []string `json:"quorum_names"`

	// OutOfQuorum is the list of monitor names not in quorum
	OutOfQuorum []string `json:"out_of_quorum"`

	// Leader is the name of the current quorum leader
	Leader string `json:"leader"`

	// ElectionEpoch is the current election epoch
	ElectionEpoch int `json:"election_epoch"`
}

// cephQuorumStatus represents the parsed output of 'ceph quorum_status --format json'
type cephQuorumStatus struct {
	ElectionEpoch    int      `json:"election_epoch"`
	Quorum           []int    `json:"quorum"`
	QuorumNames      []string `json:"quorum_names"`
	QuorumLeaderName string   `json:"quorum_leader_name"`
	QuorumAge        int64    `json:"quorum_age"`
	Monmap           struct {
		Epoch   int `json:"epoch"`
		NumMons int `json:"num_mons"`
		Mons    []struct {
			Rank int    `json:"rank"`
			Name string `json:"name"`
			Addr string `json:"addr"`
		} `json:"mons"`
	} `json:"monmap"`
}

// GetMonitorStatus gets the Ceph monitor quorum status
func (c *Client) GetMonitorStatus(ctx context.Context, namespace string) (*MonitorStatus, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "quorum_status", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph quorum status: %w", err)
	}

	return parseMonitorStatus(output)
}

// GetMonitorStatus is a package-level function that uses the global client
func GetMonitorStatus(ctx context.Context, namespace string) (*MonitorStatus, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetMonitorStatus(ctx, namespace)
}

// parseMonitorStatus parses the output of 'ceph quorum_status --format json'
func parseMonitorStatus(output string) (*MonitorStatus, error) {
	var qs cephQuorumStatus
	if err := json.Unmarshal([]byte(output), &qs); err != nil {
		return nil, fmt.Errorf("failed to parse ceph quorum status JSON: %w", err)
	}

	status := &MonitorStatus{
		TotalCount:    qs.Monmap.NumMons,
		InQuorum:      len(qs.QuorumNames),
		QuorumNames:   qs.QuorumNames,
		Leader:        qs.QuorumLeaderName,
		ElectionEpoch: qs.ElectionEpoch,
	}

	// Determine which monitors are out of quorum
	quorumSet := make(map[string]bool, len(qs.QuorumNames))
	for _, name := range qs.QuorumNames {
		quorumSet[name] = true
	}

	for _, mon := range qs.Monmap.Mons {
		if !quorumSet[mon.Name] {
			status.OutOfQuorum = append(status.OutOfQuorum, mon.Name)
		}
	}

	return status, nil
}

// IsHealthy returns true if all monitors are in quorum
func (m *MonitorStatus) IsHealthy() bool {
	return m.InQuorum == m.TotalCount && m.TotalCount > 0
}

// HasQuorum returns true if the cluster has quorum (majority of monitors)
func (m *MonitorStatus) HasQuorum() bool {
	if m.TotalCount == 0 {
		return false
	}
	return m.InQuorum > m.TotalCount/2
}

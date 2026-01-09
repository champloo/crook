package output

import (
	"context"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
)

// ResourceType represents the type of resource to display
type ResourceType string

const (
	// ResourceNodes represents cluster nodes
	ResourceNodes ResourceType = "nodes"
	// ResourceDeployments represents Rook-Ceph deployments
	ResourceDeployments ResourceType = "deployments"
	// ResourceOSDs represents Ceph OSDs
	ResourceOSDs ResourceType = "osds"
	// ResourcePods represents Rook-Ceph pods
	ResourcePods ResourceType = "pods"
)

// AllResourceTypes returns all supported resource types
func AllResourceTypes() []ResourceType {
	return []ResourceType{ResourceNodes, ResourceDeployments, ResourceOSDs, ResourcePods}
}

// ParseResourceTypes parses a comma-separated list of resource types
func ParseResourceTypes(show string) []ResourceType {
	if show == "" {
		return AllResourceTypes()
	}

	types := make([]ResourceType, 0)
	// Split and validate (validation already done in CLI)
	for _, s := range splitTrim(show, ",") {
		switch s {
		case "nodes":
			types = append(types, ResourceNodes)
		case "deployments":
			types = append(types, ResourceDeployments)
		case "osds":
			types = append(types, ResourceOSDs)
		case "pods":
			types = append(types, ResourcePods)
		}
	}

	if len(types) == 0 {
		return AllResourceTypes()
	}
	return types
}

// splitTrim splits a string and trims whitespace from each part
func splitTrim(s, sep string) []string {
	if s == "" {
		return nil
	}

	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || (len(sep) == 1 && s[i] == sep[0]) {
			part := s[start:i]
			// Trim whitespace manually
			for len(part) > 0 && (part[0] == ' ' || part[0] == '\t') {
				part = part[1:]
			}
			for len(part) > 0 && (part[len(part)-1] == ' ' || part[len(part)-1] == '\t') {
				part = part[:len(part)-1]
			}
			if len(part) > 0 {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}

// ClusterHealth represents the Ceph cluster health status
type ClusterHealth struct {
	// Status is the overall health (HEALTH_OK, HEALTH_WARN, HEALTH_ERR)
	Status string `json:"status" yaml:"status"`
	// OSDs is the total number of OSDs
	OSDs int `json:"osds" yaml:"osds"`
	// OSDsUp is the number of up OSDs
	OSDsUp int `json:"osds_up" yaml:"osds_up"`
	// OSDsIn is the number of in OSDs
	OSDsIn int `json:"osds_in" yaml:"osds_in"`
	// MonsTotal is the total number of monitors
	MonsTotal int `json:"mons_total" yaml:"mons_total"`
	// MonsInQuorum is the number of monitors in quorum
	MonsInQuorum int `json:"mons_in_quorum" yaml:"mons_in_quorum"`
	// NooutSet indicates if the noout flag is set
	NooutSet bool `json:"noout_set" yaml:"noout_set"`
	// UsedBytes is the used storage in bytes
	UsedBytes int64 `json:"used_bytes" yaml:"used_bytes"`
	// TotalBytes is the total storage in bytes
	TotalBytes int64 `json:"total_bytes" yaml:"total_bytes"`
}

// NodeOutput represents a node for output
type NodeOutput struct {
	Name           string   `json:"name" yaml:"name"`
	Status         string   `json:"status" yaml:"status"`
	Roles          []string `json:"roles" yaml:"roles"`
	Schedulable    bool     `json:"schedulable" yaml:"schedulable"`
	Cordoned       bool     `json:"cordoned" yaml:"cordoned"`
	CephPodCount   int      `json:"ceph_pod_count" yaml:"ceph_pod_count"`
	Age            string   `json:"age" yaml:"age"`
	KubeletVersion string   `json:"kubelet_version" yaml:"kubelet_version"`
}

// DeploymentOutput represents a deployment for output
type DeploymentOutput struct {
	Name            string `json:"name" yaml:"name"`
	Namespace       string `json:"namespace" yaml:"namespace"`
	ReadyReplicas   int32  `json:"ready_replicas" yaml:"ready_replicas"`
	DesiredReplicas int32  `json:"desired_replicas" yaml:"desired_replicas"`
	NodeName        string `json:"node_name,omitempty" yaml:"node_name,omitempty"`
	Age             string `json:"age" yaml:"age"`
	Status          string `json:"status" yaml:"status"`
	Type            string `json:"type" yaml:"type"`
	OsdID           string `json:"osd_id,omitempty" yaml:"osd_id,omitempty"`
}

// OSDOutput represents an OSD for output
type OSDOutput struct {
	ID             int     `json:"id" yaml:"id"`
	Name           string  `json:"name" yaml:"name"`
	Hostname       string  `json:"hostname" yaml:"hostname"`
	Status         string  `json:"status" yaml:"status"`
	InOut          string  `json:"in_out" yaml:"in_out"`
	Weight         float64 `json:"weight" yaml:"weight"`
	Reweight       float64 `json:"reweight" yaml:"reweight"`
	DeviceClass    string  `json:"device_class" yaml:"device_class"`
	DeploymentName string  `json:"deployment_name,omitempty" yaml:"deployment_name,omitempty"`
}

// PodOutput represents a pod for output
type PodOutput struct {
	Name            string `json:"name" yaml:"name"`
	Namespace       string `json:"namespace" yaml:"namespace"`
	Status          string `json:"status" yaml:"status"`
	ReadyContainers int    `json:"ready_containers" yaml:"ready_containers"`
	TotalContainers int    `json:"total_containers" yaml:"total_containers"`
	Restarts        int32  `json:"restarts" yaml:"restarts"`
	NodeName        string `json:"node_name" yaml:"node_name"`
	Age             string `json:"age" yaml:"age"`
	Type            string `json:"type" yaml:"type"`
	IP              string `json:"ip,omitempty" yaml:"ip,omitempty"`
	OwnerDeployment string `json:"owner_deployment,omitempty" yaml:"owner_deployment,omitempty"`
}

// Data holds all data for output formatting
type Data struct {
	// ClusterHealth contains Ceph cluster health information
	ClusterHealth *ClusterHealth `json:"cluster_health,omitempty" yaml:"cluster_health,omitempty"`
	// Nodes contains node information
	Nodes []NodeOutput `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	// Deployments contains deployment information
	Deployments []DeploymentOutput `json:"deployments,omitempty" yaml:"deployments,omitempty"`
	// OSDs contains OSD information
	OSDs []OSDOutput `json:"osds,omitempty" yaml:"osds,omitempty"`
	// Pods contains pod information
	Pods []PodOutput `json:"pods,omitempty" yaml:"pods,omitempty"`
	// FetchedAt is when the data was fetched
	FetchedAt time.Time `json:"fetched_at" yaml:"fetched_at"`
}

// FetchOptions configures data fetching
type FetchOptions struct {
	// Client is the Kubernetes client
	Client *k8s.Client
	// Config is the application configuration
	Config config.Config
	// ResourceTypes specifies which resources to fetch
	ResourceTypes []ResourceType
	// NodeFilter optionally filters to a specific node
	NodeFilter string
}

// FetchData fetches all requested data for non-TUI output
func FetchData(ctx context.Context, opts FetchOptions) (*Data, error) {
	data := &Data{
		FetchedAt: time.Now(),
	}

	namespace := opts.Config.Namespace

	// Always fetch cluster health for header (non-fatal - Ceph may be degraded)
	health, err := fetchClusterHealth(ctx, opts.Client, namespace)
	if err != nil {
		// Non-fatal: continue without health data
		data.ClusterHealth = nil
	} else {
		data.ClusterHealth = health
	}

	// Fetch requested resource types
	for _, rt := range opts.ResourceTypes {
		switch rt {
		case ResourceNodes:
			nodes, fetchErr := fetchNodes(ctx, opts.Client, namespace)
			if fetchErr != nil {
				return nil, fetchErr
			}
			data.Nodes = nodes

		case ResourceDeployments:
			deployments, fetchErr := fetchDeployments(ctx, opts.Client, namespace, opts.NodeFilter)
			if fetchErr != nil {
				return nil, fetchErr
			}
			data.Deployments = deployments

		case ResourceOSDs:
			// Non-fatal: OSDs require Ceph commands which may timeout on degraded clusters
			osds, fetchErr := fetchOSDs(ctx, opts.Client, namespace, opts.NodeFilter)
			if fetchErr != nil {
				// Continue without OSD data - cluster may be degraded
				data.OSDs = nil
			} else {
				data.OSDs = osds
			}

		case ResourcePods:
			pods, fetchErr := fetchPods(ctx, opts.Client, namespace, opts.NodeFilter)
			if fetchErr != nil {
				return nil, fetchErr
			}
			data.Pods = pods
		}
	}

	return data, nil
}

// fetchClusterHealth fetches Ceph cluster health data
func fetchClusterHealth(ctx context.Context, client *k8s.Client, namespace string) (*ClusterHealth, error) {
	status, err := client.GetCephStatus(ctx, namespace)
	if err != nil {
		return nil, err
	}

	health := &ClusterHealth{
		Status: status.Health.Status,
		OSDs:   status.OSDMap.NumOSDs,
		OSDsUp: status.OSDMap.NumUpOSDs,
		OSDsIn: status.OSDMap.NumInOSDs,
	}

	// Fetch monitor status
	monStatus, err := client.GetMonitorStatus(ctx, namespace)
	if err == nil {
		health.MonsTotal = monStatus.TotalCount
		health.MonsInQuorum = monStatus.InQuorum
	}

	// Fetch flags
	flags, err := client.GetCephFlags(ctx, namespace)
	if err == nil {
		health.NooutSet = flags.NoOut
	}

	// Fetch storage usage
	storage, err := client.GetStorageUsage(ctx, namespace)
	if err == nil {
		health.UsedBytes = storage.UsedBytes
		health.TotalBytes = storage.TotalBytes
	}

	return health, nil
}

// fetchNodes fetches node data
func fetchNodes(ctx context.Context, client *k8s.Client, namespace string) ([]NodeOutput, error) {
	nodes, err := client.ListNodesWithCephPods(ctx, namespace)
	if err != nil {
		return nil, err
	}

	result := make([]NodeOutput, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, NodeOutput{
			Name:           n.Name,
			Status:         n.Status,
			Roles:          n.Roles,
			Schedulable:    n.Schedulable,
			Cordoned:       n.Cordoned,
			CephPodCount:   n.CephPodCount,
			Age:            formatDuration(n.Age),
			KubeletVersion: n.KubeletVersion,
		})
	}
	return result, nil
}

// fetchDeployments fetches deployment data
func fetchDeployments(ctx context.Context, client *k8s.Client, namespace string, nodeFilter string) ([]DeploymentOutput, error) {
	deployments, err := client.ListCephDeployments(ctx, namespace)
	if err != nil {
		return nil, err
	}

	result := make([]DeploymentOutput, 0)
	for _, d := range deployments {
		// Apply node filter if specified
		if nodeFilter != "" && d.NodeName != nodeFilter {
			continue
		}

		result = append(result, DeploymentOutput{
			Name:            d.Name,
			Namespace:       d.Namespace,
			ReadyReplicas:   d.ReadyReplicas,
			DesiredReplicas: d.DesiredReplicas,
			NodeName:        d.NodeName,
			Age:             formatDuration(d.Age),
			Status:          d.Status,
			Type:            d.Type,
			OsdID:           d.OsdID,
		})
	}
	return result, nil
}

// fetchOSDs fetches OSD data
func fetchOSDs(ctx context.Context, client *k8s.Client, namespace, nodeFilter string) ([]OSDOutput, error) {
	osds, err := client.GetOSDInfoList(ctx, namespace)
	if err != nil {
		return nil, err
	}

	result := make([]OSDOutput, 0)
	for _, o := range osds {
		// Apply node filter if specified
		if nodeFilter != "" && o.Hostname != nodeFilter {
			continue
		}

		result = append(result, OSDOutput{
			ID:             o.ID,
			Name:           o.Name,
			Hostname:       o.Hostname,
			Status:         o.Status,
			InOut:          o.InOut,
			Weight:         o.Weight,
			Reweight:       o.Reweight,
			DeviceClass:    o.DeviceClass,
			DeploymentName: o.DeploymentName,
		})
	}
	return result, nil
}

// fetchPods fetches pod data
func fetchPods(ctx context.Context, client *k8s.Client, namespace string, nodeFilter string) ([]PodOutput, error) {
	pods, err := client.ListCephPods(ctx, namespace, nodeFilter)
	if err != nil {
		return nil, err
	}

	result := make([]PodOutput, 0, len(pods))
	for _, p := range pods {
		result = append(result, PodOutput{
			Name:            p.Name,
			Namespace:       p.Namespace,
			Status:          p.Status,
			ReadyContainers: p.ReadyContainers,
			TotalContainers: p.TotalContainers,
			Restarts:        p.Restarts,
			NodeName:        p.NodeName,
			Age:             formatDuration(p.Age),
			Type:            p.Type,
			IP:              p.IP,
			OwnerDeployment: p.OwnerDeployment,
		})
	}
	return result, nil
}

// formatDuration formats a duration as a human-readable age string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return formatInt(int(d.Seconds())) + "s"
	}
	if d < time.Hour {
		return formatInt(int(d.Minutes())) + "m"
	}
	if d < 24*time.Hour {
		return formatInt(int(d.Hours())) + "h"
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return formatInt(days) + "d"
	}
	if days < 365 {
		return formatInt(days/30) + "mo"
	}
	return formatInt(days/365) + "y"
}

// formatInt converts an int to a string (simple implementation without strconv)
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	// Build digits in reverse
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

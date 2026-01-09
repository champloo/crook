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

// Data holds all data for output formatting
type Data struct {
	// ClusterHealth contains Ceph cluster health information
	ClusterHealth *ClusterHealth `json:"cluster_health,omitempty" yaml:"cluster_health,omitempty"`
	// Nodes contains node information
	Nodes []k8s.NodeInfo `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	// Deployments contains deployment information
	Deployments []k8s.DeploymentInfo `json:"deployments,omitempty" yaml:"deployments,omitempty"`
	// OSDs contains OSD information
	OSDs []k8s.OSDInfo `json:"osds,omitempty" yaml:"osds,omitempty"`
	// Pods contains pod information
	Pods []k8s.PodInfo `json:"pods,omitempty" yaml:"pods,omitempty"`
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
func fetchNodes(ctx context.Context, client *k8s.Client, namespace string) ([]k8s.NodeInfo, error) {
	return client.ListNodesWithCephPods(ctx, namespace)
}

// fetchDeployments fetches deployment data
func fetchDeployments(ctx context.Context, client *k8s.Client, namespace string, nodeFilter string) ([]k8s.DeploymentInfo, error) {
	deployments, err := client.ListCephDeployments(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// No filter - return all
	if nodeFilter == "" {
		return deployments, nil
	}

	// Apply node filter
	result := make([]k8s.DeploymentInfo, 0, len(deployments))
	for _, d := range deployments {
		if d.NodeName == nodeFilter {
			result = append(result, d)
		}
	}
	return result, nil
}

// fetchOSDs fetches OSD data
func fetchOSDs(ctx context.Context, client *k8s.Client, namespace, nodeFilter string) ([]k8s.OSDInfo, error) {
	osds, err := client.GetOSDInfoList(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// No filter - return all
	if nodeFilter == "" {
		return osds, nil
	}

	// Apply node filter
	result := make([]k8s.OSDInfo, 0, len(osds))
	for _, o := range osds {
		if o.Hostname == nodeFilter {
			result = append(result, o)
		}
	}
	return result, nil
}

// fetchPods fetches pod data
func fetchPods(ctx context.Context, client *k8s.Client, namespace string, nodeFilter string) ([]k8s.PodInfo, error) {
	return client.ListCephPods(ctx, namespace, nodeFilter)
}

// Package views provides view implementations for the ls command TUI.
package views

import (
	"time"
)

// NodeInfo represents information about a cluster node
type NodeInfo struct {
	// Name is the node name
	Name string

	// Status is the node status (Ready/NotReady/Unknown)
	Status string

	// Roles are the node roles (control-plane, worker, etc.)
	Roles []string

	// Schedulable indicates if the node accepts new pods
	Schedulable bool

	// Cordoned indicates if the node is cordoned (unschedulable)
	Cordoned bool

	// CephPodCount is the number of Ceph pods on this node
	CephPodCount int

	// Age is the time since the node was created
	Age time.Duration

	// KubeletVersion is the kubelet version
	KubeletVersion string
}

// DeploymentInfo represents information about a Rook-Ceph deployment
type DeploymentInfo struct {
	// Name is the deployment name
	Name string

	// Namespace is the deployment namespace
	Namespace string

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32

	// DesiredReplicas is the desired number of replicas
	DesiredReplicas int32

	// NodeName is the node where the deployment's pod runs
	NodeName string

	// Age is the time since the deployment was created
	Age time.Duration

	// Status is the deployment status (Ready/Scaling/Unavailable)
	Status string

	// Type is the deployment type (osd/mon/exporter/crashcollector)
	Type string

	// OsdID is the OSD ID (from label ceph-osd-id, if applicable)
	OsdID string
}

// OSDInfo represents information about a Ceph OSD
type OSDInfo struct {
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

// PodInfo represents information about a Rook-Ceph pod
type PodInfo struct {
	// Name is the pod name
	Name string

	// Namespace is the pod namespace
	Namespace string

	// Status is the pod status (Running/Pending/Failed/etc.)
	Status string

	// Ready indicates if the pod is ready
	Ready bool

	// ReadyContainers is the number of ready containers
	ReadyContainers int

	// TotalContainers is the total number of containers
	TotalContainers int

	// Restarts is the number of container restarts
	Restarts int32

	// NodeName is the node where the pod runs
	NodeName string

	// Age is the time since the pod was created
	Age time.Duration

	// Type is the pod type (osd/mon/exporter/crashcollector)
	Type string

	// IP is the pod IP address
	IP string

	// OwnerDeployment is the name of the owning deployment (if any)
	OwnerDeployment string
}

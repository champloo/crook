package monitoring

import (
	"fmt"
	"time"
)

// OverallHealthStatus represents the aggregated health status
type OverallHealthStatus string

const (
	HealthStatusHealthy  OverallHealthStatus = "Healthy"
	HealthStatusDegraded OverallHealthStatus = "Degraded"
	HealthStatusCritical OverallHealthStatus = "Critical"
	HealthStatusUnknown  OverallHealthStatus = "Unknown"
)

// HealthSummary represents the aggregated health summary
type HealthSummary struct {
	Status           OverallHealthStatus
	Reasons          []string
	NodeHealth       *NodeHealthInfo
	CephHealth       *CephHealthInfo
	DeploymentHealth *DeploymentHealthInfo
	LastUpdateTime   time.Time
}

// NodeHealthInfo contains health information about the node
type NodeHealthInfo struct {
	Healthy bool
	Ready   bool
	Reason  string
}

// CephHealthInfo contains health information about Ceph
type CephHealthInfo struct {
	Healthy     bool
	Status      string
	Reason      string
	OSDsHealthy bool
	OSDsUp      int
	OSDsTotal   int
}

// DeploymentHealthInfo contains health information about deployments
type DeploymentHealthInfo struct {
	Healthy          bool
	Status           DeploymentHealthStatus
	Reason           string
	TotalCount       int
	HealthyCount     int
	UnavailableCount int
}

// StatusColor returns a color indicator for the overall health status
func (hs *HealthSummary) StatusColor() string {
	switch hs.Status {
	case HealthStatusHealthy:
		return "green"
	case HealthStatusDegraded:
		return "yellow"
	case HealthStatusCritical:
		return "red"
	default:
		return "yellow"
	}
}

// AggregateHealth combines node, Ceph, and deployment health to produce an overall status
func AggregateHealth(nodeStatus *NodeStatus, cephHealth *CephHealth, deploymentsStatus *DeploymentsStatus, osdStatus *OSDTreeStatus) *HealthSummary {
	summary := &HealthSummary{
		Status:         HealthStatusHealthy,
		Reasons:        []string{},
		LastUpdateTime: time.Now(),
	}

	// Evaluate node health
	nodeHealthy := evaluateNodeHealth(nodeStatus, summary)

	// Evaluate Ceph health
	cephHealthy := evaluateCephHealth(cephHealth, osdStatus, summary)

	// Evaluate deployment health
	deploymentsHealthy := evaluateDeploymentsHealth(deploymentsStatus, summary)

	// Determine overall status based on component health
	if !nodeHealthy || !cephHealthy || !deploymentsHealthy {
		// Check for critical conditions
		if hasAnyCriticalCondition(nodeStatus, cephHealth, deploymentsStatus) {
			summary.Status = HealthStatusCritical
		} else {
			summary.Status = HealthStatusDegraded
		}
	}

	return summary
}

// evaluateNodeHealth evaluates node health and updates the summary
func evaluateNodeHealth(nodeStatus *NodeStatus, summary *HealthSummary) bool {
	if nodeStatus == nil {
		summary.NodeHealth = &NodeHealthInfo{
			Healthy: false,
			Ready:   false,
			Reason:  "Node status unavailable",
		}
		summary.Reasons = append(summary.Reasons, "Node status unavailable")
		return false
	}

	healthy := nodeStatus.Ready && !nodeStatus.Unschedulable
	reason := ""

	if !nodeStatus.Ready {
		reason = "Node not ready"
		summary.Reasons = append(summary.Reasons, "Node not ready")
	} else if nodeStatus.Unschedulable {
		reason = "Node is cordoned (unschedulable)"
		summary.Reasons = append(summary.Reasons, "Node is cordoned")
	}

	summary.NodeHealth = &NodeHealthInfo{
		Healthy: healthy,
		Ready:   nodeStatus.Ready,
		Reason:  reason,
	}

	return healthy
}

// evaluateCephHealth evaluates Ceph cluster health and updates the summary
func evaluateCephHealth(cephHealth *CephHealth, osdStatus *OSDTreeStatus, summary *HealthSummary) bool {
	if cephHealth == nil {
		summary.CephHealth = &CephHealthInfo{
			Healthy: false,
			Status:  "Unknown",
			Reason:  "Ceph status unavailable",
		}
		summary.Reasons = append(summary.Reasons, "Ceph status unavailable")
		return false
	}

	healthy := cephHealth.OverallStatus == "HEALTH_OK"
	reason := ""

	switch cephHealth.OverallStatus {
	case "HEALTH_WARN":
		reason = "Ceph cluster has warnings"
		if len(cephHealth.HealthMessages) > 0 {
			reason = fmt.Sprintf("Ceph warnings: %s", cephHealth.HealthMessages[0])
		}
		summary.Reasons = append(summary.Reasons, reason)
	case "HEALTH_ERR":
		reason = "Ceph cluster has errors"
		if len(cephHealth.HealthMessages) > 0 {
			reason = fmt.Sprintf("Ceph errors: %s", cephHealth.HealthMessages[0])
		}
		summary.Reasons = append(summary.Reasons, reason)
	}

	// Check OSD health
	osdsUp := cephHealth.OSDsUp
	osdsTotal := cephHealth.OSDCount
	osdsHealthy := cephHealth.OSDsUp == cephHealth.OSDCount && cephHealth.OSDsIn == cephHealth.OSDCount

	if osdStatus != nil && len(osdStatus.OSDs) > 0 {
		// If we have OSD status for a specific node, use that
		osdsUp = 0
		osdsTotal = len(osdStatus.OSDs)
		for _, osd := range osdStatus.OSDs {
			if osd.Up && osd.In {
				osdsUp++
			}
		}
		osdsHealthy = osdsUp == osdsTotal
		if !osdsHealthy {
			reason := fmt.Sprintf("%d of %d OSDs are down or out on this node", osdsTotal-osdsUp, osdsTotal)
			summary.Reasons = append(summary.Reasons, reason)
		}
	} else if !osdsHealthy {
		// Use cluster-wide OSD stats
		if cephHealth.OSDsUp < cephHealth.OSDCount {
			reason := fmt.Sprintf("%d of %d OSDs are down", cephHealth.OSDCount-cephHealth.OSDsUp, cephHealth.OSDCount)
			summary.Reasons = append(summary.Reasons, reason)
		}
		if cephHealth.OSDsIn < cephHealth.OSDCount {
			reason := fmt.Sprintf("%d of %d OSDs are out", cephHealth.OSDCount-cephHealth.OSDsIn, cephHealth.OSDCount)
			summary.Reasons = append(summary.Reasons, reason)
		}
	}

	summary.CephHealth = &CephHealthInfo{
		Healthy:     healthy && osdsHealthy,
		Status:      cephHealth.OverallStatus,
		Reason:      reason,
		OSDsHealthy: osdsHealthy,
		OSDsUp:      osdsUp,
		OSDsTotal:   osdsTotal,
	}

	return healthy && osdsHealthy
}

// evaluateDeploymentsHealth evaluates deployment health and updates the summary
func evaluateDeploymentsHealth(deploymentsStatus *DeploymentsStatus, summary *HealthSummary) bool {
	if deploymentsStatus == nil {
		summary.DeploymentHealth = &DeploymentHealthInfo{
			Healthy: false,
			Status:  DeploymentUnavailable,
			Reason:  "Deployment status unavailable",
		}
		summary.Reasons = append(summary.Reasons, "Deployment status unavailable")
		return false
	}

	healthy := deploymentsStatus.OverallStatus == DeploymentHealthy
	reason := ""

	healthyCount := 0
	unavailableCount := 0
	for _, deployment := range deploymentsStatus.Deployments {
		switch deployment.Status {
		case DeploymentHealthy:
			healthyCount++
		case DeploymentUnavailable:
			unavailableCount++
		}
	}

	totalCount := len(deploymentsStatus.Deployments)

	switch deploymentsStatus.OverallStatus {
	case DeploymentUnavailable:
		reason = fmt.Sprintf("%d of %d deployments unavailable", unavailableCount, totalCount)
		summary.Reasons = append(summary.Reasons, reason)
	case DeploymentScaling:
		reason = fmt.Sprintf("Deployments are scaling (%d of %d healthy)", healthyCount, totalCount)
		summary.Reasons = append(summary.Reasons, reason)
	case DeploymentProgressing:
		reason = "Deployments are progressing"
		summary.Reasons = append(summary.Reasons, reason)
	}

	summary.DeploymentHealth = &DeploymentHealthInfo{
		Healthy:          healthy,
		Status:           deploymentsStatus.OverallStatus,
		Reason:           reason,
		TotalCount:       totalCount,
		HealthyCount:     healthyCount,
		UnavailableCount: unavailableCount,
	}

	return healthy
}

// hasAnyCriticalCondition checks if any component has a critical condition
func hasAnyCriticalCondition(nodeStatus *NodeStatus, cephHealth *CephHealth, deploymentsStatus *DeploymentsStatus) bool {
	// Node not ready is critical
	if nodeStatus != nil && !nodeStatus.Ready {
		return true
	}

	// Ceph errors are critical
	if cephHealth != nil && cephHealth.OverallStatus == "HEALTH_ERR" {
		return true
	}

	// All deployments unavailable is critical
	if deploymentsStatus != nil && deploymentsStatus.OverallStatus == DeploymentUnavailable {
		return true
	}

	return false
}

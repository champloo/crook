package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// DeploymentStatus represents the status of a single deployment
type DeploymentStatus struct {
	Name              string
	Namespace         string
	DesiredReplicas   int32
	CurrentReplicas   int32
	ReadyReplicas     int32
	AvailableReplicas int32
	UpdatedReplicas   int32
	Status            DeploymentHealthStatus
	Conditions        []appsv1.DeploymentCondition
	LastUpdateTime    time.Time
}

// DeploymentHealthStatus represents the overall health of a deployment
type DeploymentHealthStatus string

const (
	DeploymentHealthy     DeploymentHealthStatus = "Ready"
	DeploymentScaling     DeploymentHealthStatus = "Scaling"
	DeploymentUnavailable DeploymentHealthStatus = "Unavailable"
	DeploymentProgressing DeploymentHealthStatus = "Progressing"
)

// DeploymentsStatus represents the aggregated status of multiple deployments
type DeploymentsStatus struct {
	Deployments    []DeploymentStatus
	OverallStatus  DeploymentHealthStatus
	LastUpdateTime time.Time
}

// StatusColor returns a color indicator for the deployment status
func (ds *DeploymentStatus) StatusColor() string {
	switch ds.Status {
	case DeploymentHealthy:
		return "green"
	case DeploymentScaling, DeploymentProgressing:
		return "yellow"
	case DeploymentUnavailable:
		return "red"
	default:
		return "yellow"
	}
}

// MonitorDeployment retrieves the current status of a single deployment
func MonitorDeployment(ctx context.Context, client *k8s.Client, namespace, name string) (*DeploymentStatus, error) {
	deployment, err := client.GetDeployment(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	desired := int32(0)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	// Determine health status
	healthStatus := determineDeploymentHealth(deployment)

	status := &DeploymentStatus{
		Name:              deployment.Name,
		Namespace:         deployment.Namespace,
		DesiredReplicas:   desired,
		CurrentReplicas:   deployment.Status.Replicas,
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
		Status:            healthStatus,
		Conditions:        deployment.Status.Conditions,
		LastUpdateTime:    time.Now(),
	}

	return status, nil
}

// determineDeploymentHealth determines the health status based on deployment state
func determineDeploymentHealth(deployment *appsv1.Deployment) DeploymentHealthStatus {
	desired := int32(0)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	// Check if all replicas are ready and available
	if deployment.Status.ReadyReplicas == desired &&
		deployment.Status.AvailableReplicas == desired &&
		deployment.Status.UpdatedReplicas == desired {
		// Check if Available condition is true
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable {
				if condition.Status == "True" {
					return DeploymentHealthy
				}
			}
		}
	}

	// Check if deployment has no available replicas
	if deployment.Status.AvailableReplicas == 0 {
		return DeploymentUnavailable
	}

	// Check if deployment is progressing
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing {
			if condition.Status == "True" && condition.Reason == "NewReplicaSetAvailable" {
				return DeploymentScaling
			}
			if condition.Status == "True" {
				return DeploymentProgressing
			}
		}
	}

	// If replicas don't match, it's scaling
	if deployment.Status.Replicas != desired {
		return DeploymentScaling
	}

	return DeploymentProgressing
}

// MonitorDeployments retrieves the status of multiple deployments and aggregates them
func MonitorDeployments(ctx context.Context, client *k8s.Client, namespace string, deploymentNames []string) (*DeploymentsStatus, error) {
	var deployments []DeploymentStatus
	var hasUnavailable, hasScaling bool

	for _, name := range deploymentNames {
		status, err := MonitorDeployment(ctx, client, namespace, name)
		if err != nil {
			// Continue with other deployments if one fails
			continue
		}

		deployments = append(deployments, *status)

		// Track overall status
		switch status.Status {
		case DeploymentUnavailable:
			hasUnavailable = true
		case DeploymentScaling, DeploymentProgressing:
			hasScaling = true
		case DeploymentHealthy:
			// No action needed
		}
	}

	// Determine overall status
	overallStatus := DeploymentHealthy
	if hasUnavailable {
		overallStatus = DeploymentUnavailable
	} else if hasScaling {
		overallStatus = DeploymentScaling
	}

	result := &DeploymentsStatus{
		Deployments:    deployments,
		OverallStatus:  overallStatus,
		LastUpdateTime: time.Now(),
	}

	return result, nil
}

// StartDeploymentsMonitoring starts background monitoring of deployments with the given refresh interval
func StartDeploymentsMonitoring(ctx context.Context, client *k8s.Client, namespace string, deploymentNames []string, interval time.Duration) <-chan *DeploymentsStatus {
	updates := make(chan *DeploymentsStatus, 1)

	go func() {
		defer close(updates)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Send initial status
		if status, err := MonitorDeployments(ctx, client, namespace, deploymentNames); err == nil {
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
				status, err := MonitorDeployments(ctx, client, namespace, deploymentNames)
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

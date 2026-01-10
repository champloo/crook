// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
)

const operatorDeploymentName = "rook-ceph-operator"

func isUpDesiredState(ctx context.Context, client *k8s.Client, cfg config.Config, nodeName string) bool {
	nodeStatus, err := client.GetNodeStatus(ctx, nodeName)
	if err != nil || nodeStatus.Unschedulable {
		return false
	}

	nooutSet, err := getNooutFlag(ctx, client, cfg.Namespace)
	if err != nil || nooutSet {
		return false
	}

	operatorReady, err := deploymentAtDesiredReplicas(ctx, client, cfg.Namespace, operatorDeploymentName, 1)
	if err != nil || !operatorReady {
		return false
	}

	return true
}

func isDownDesiredState(ctx context.Context, client *k8s.Client, cfg config.Config, nodeName string) bool {
	nodeStatus, err := client.GetNodeStatus(ctx, nodeName)
	if err != nil || !nodeStatus.Unschedulable {
		return false
	}

	nooutSet, err := getNooutFlag(ctx, client, cfg.Namespace)
	if err != nil || !nooutSet {
		return false
	}

	operatorStopped, err := deploymentAtDesiredReplicas(ctx, client, cfg.Namespace, operatorDeploymentName, 0)
	if err != nil || !operatorStopped {
		return false
	}

	return true
}

func deploymentAtDesiredReplicas(ctx context.Context, client *k8s.Client, namespace, name string, expected int32) (bool, error) {
	status, err := client.GetDeploymentStatus(ctx, namespace, name)
	if err != nil {
		return false, err
	}

	if status.Replicas != expected {
		return false, nil
	}
	if status.ReadyReplicas != expected {
		return false, nil
	}

	return true, nil
}

func getNooutFlag(ctx context.Context, client *k8s.Client, namespace string) (bool, error) {
	if client.Config() == nil {
		return false, fmt.Errorf("client config is nil")
	}

	flags, err := client.GetCephFlags(ctx, namespace)
	if err != nil {
		return false, err
	}

	return flags.NoOut, nil
}

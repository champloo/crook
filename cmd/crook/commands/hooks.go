// Package commands provides the CLI command implementations for crook.
package commands

import (
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/maintenance"
)

var newK8sClient = k8s.NewClient
var executeUpPhase = maintenance.ExecuteUpPhase
var executeDownPhase = maintenance.ExecuteDownPhase

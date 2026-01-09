package k8s

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/andri/crook/internal/logger"
)

// OwnerInfo represents information about a resource owner
type OwnerInfo struct {
	Kind       string
	APIVersion string
	Name       string
	UID        string
}

// OwnerChain represents the ownership chain of a pod
type OwnerChain struct {
	Pod         OwnerInfo
	ReplicaSet  *OwnerInfo
	Deployment  *OwnerInfo
	StatefulSet *OwnerInfo
	DaemonSet   *OwnerInfo
	// Other possible owners
	Other []OwnerInfo
}

// GetOwnerChain traverses the ownership chain of a pod
func (c *Client) GetOwnerChain(ctx context.Context, pod *corev1.Pod) (*OwnerChain, error) {
	chain := &OwnerChain{
		Pod: OwnerInfo{
			Kind:       "Pod",
			APIVersion: "v1",
			Name:       pod.Name,
			UID:        string(pod.UID),
		},
	}

	if len(pod.OwnerReferences) == 0 {
		return chain, nil
	}

	// Traverse ownership chain
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Controller == nil || !*ownerRef.Controller {
			// Only follow controller references
			continue
		}

		ownerInfo := OwnerInfo{
			Kind:       ownerRef.Kind,
			APIVersion: ownerRef.APIVersion,
			Name:       ownerRef.Name,
			UID:        string(ownerRef.UID),
		}

		switch strings.ToLower(ownerRef.Kind) {
		case "replicaset":
			chain.ReplicaSet = &ownerInfo
			// Try to find the deployment that owns this ReplicaSet
			// This is non-fatal - if we can't traverse to the deployment, we still have the ReplicaSet info
			if err := c.traverseReplicaSetOwner(ctx, pod.Namespace, ownerRef.Name, chain); err != nil {
				logger.Debug("failed to traverse ReplicaSet ownership chain",
					"pod", pod.Name,
					"namespace", pod.Namespace,
					"replicaset", ownerRef.Name,
					"error", err)
			}

		case "statefulset":
			chain.StatefulSet = &ownerInfo

		case "daemonset":
			chain.DaemonSet = &ownerInfo

		case "deployment":
			chain.Deployment = &ownerInfo

		default:
			chain.Other = append(chain.Other, ownerInfo)
		}
	}

	return chain, nil
}

// traverseReplicaSetOwner finds the deployment that owns a ReplicaSet
func (c *Client) traverseReplicaSetOwner(ctx context.Context, namespace, replicaSetName string, chain *OwnerChain) error {
	rs, err := c.Clientset.AppsV1().ReplicaSets(namespace).Get(ctx, replicaSetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get replicaset %s/%s: %w", namespace, replicaSetName, err)
	}

	for _, ownerRef := range rs.OwnerReferences {
		if ownerRef.Controller == nil || !*ownerRef.Controller {
			continue
		}

		if strings.ToLower(ownerRef.Kind) == "deployment" {
			chain.Deployment = &OwnerInfo{
				Kind:       ownerRef.Kind,
				APIVersion: ownerRef.APIVersion,
				Name:       ownerRef.Name,
				UID:        string(ownerRef.UID),
			}
			return nil
		}
	}

	return nil
}

// ExecInPod executes a command in a pod and returns the output
func (c *Client) ExecInPod(ctx context.Context, namespace, podName, containerName string, command []string) (string, error) {
	// Get the pod to verify it exists
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod %s/%s: %w", namespace, podName, err)
	}

	// If container name not specified, use the first container
	if containerName == "" {
		if len(pod.Spec.Containers) == 0 {
			return "", fmt.Errorf("pod %s/%s has no containers", namespace, podName)
		}
		containerName = pod.Spec.Containers[0].Name
	}

	// Create the exec request
	req := c.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	// Create the executor
	executor, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	// Execute the command
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
		}
		return "", fmt.Errorf("failed to execute command: %w", err)
	}

	return stdout.String(), nil
}

// GetPod returns a pod by namespace and name
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// ListPodsInNamespace returns all pods in a namespace
func (c *Client) ListPodsInNamespace(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}
	return podList.Items, nil
}

// PodInfo holds pod information for display and serialization
type PodInfo struct {
	// Name is the pod name
	Name string `json:"name"`

	// Namespace is the pod namespace
	Namespace string `json:"namespace"`

	// Status is the pod status (Running/Pending/Failed/etc.)
	Status string `json:"status"`

	// ReadyContainers is the number of ready containers
	ReadyContainers int `json:"ready_containers"`

	// TotalContainers is the total number of containers
	TotalContainers int `json:"total_containers"`

	// Restarts is the total number of container restarts
	Restarts int32 `json:"restarts"`

	// NodeName is the node where the pod runs
	NodeName string `json:"node_name"`

	// Age is the time since the pod was created
	Age Duration `json:"age"`

	// Type is the pod type (osd/mon/exporter/crashcollector)
	Type string `json:"type"`

	// IP is the pod IP address
	IP string `json:"ip,omitempty"`

	// OwnerDeployment is the name of the owning deployment (if any)
	OwnerDeployment string `json:"owner_deployment,omitempty"`
}

// ListCephPods returns Ceph pods with detailed info.
// Uses DefaultRookCephPrefixes() to filter pods.
func (c *Client) ListCephPods(ctx context.Context, namespace string, nodeFilter string) ([]PodInfo, error) {
	prefixes := DefaultRookCephPrefixes()

	// Build list options
	listOpts := metav1.ListOptions{}
	if nodeFilter != "" {
		listOpts.FieldSelector = fmt.Sprintf("spec.nodeName=%s", nodeFilter)
	}

	// Get pods in namespace
	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	// Build result
	var result []PodInfo
	now := time.Now()

	for _, pod := range podList.Items {
		// Check if pod matches any of the prefixes
		if !matchesAnyPrefix(pod.Name, prefixes) {
			continue
		}

		// Get owner deployment
		ownerDeployment := ""
		chain, chainErr := c.GetOwnerChain(ctx, &pod)
		if chainErr == nil && chain.Deployment != nil {
			ownerDeployment = chain.Deployment.Name
		}

		// Calculate ready containers and restarts
		readyContainers, totalContainers, restarts := getPodContainerStats(&pod)

		info := PodInfo{
			Name:            pod.Name,
			Namespace:       pod.Namespace,
			Status:          getPodStatus(&pod),
			ReadyContainers: readyContainers,
			TotalContainers: totalContainers,
			Restarts:        restarts,
			NodeName:        pod.Spec.NodeName,
			Age:             Duration(now.Sub(pod.CreationTimestamp.Time)),
			Type:            extractPodType(pod.Name),
			IP:              pod.Status.PodIP,
			OwnerDeployment: ownerDeployment,
		}
		result = append(result, info)
	}

	return result, nil
}

// getPodStatus returns a human-readable status for a pod
func getPodStatus(pod *corev1.Pod) string {
	// Check for terminating
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	// Check container statuses for more specific states
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			if reason != "" {
				return reason // e.g., CrashLoopBackOff, ImagePullBackOff
			}
		}
		if cs.State.Terminated != nil {
			reason := cs.State.Terminated.Reason
			if reason != "" {
				return reason // e.g., Error, Completed
			}
		}
	}

	// Check init container statuses
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			if reason != "" {
				return "Init:" + reason
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return "Init:Error"
		}
	}

	// Default to phase
	return string(pod.Status.Phase)
}

// getPodContainerStats returns ready containers count, total containers count, and total restarts
func getPodContainerStats(pod *corev1.Pod) (ready int, total int, restarts int32) {
	total = len(pod.Spec.Containers)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
		restarts += cs.RestartCount
	}
	return ready, total, restarts
}

// podTypePrefixes maps pod name prefixes to their types.
// Sorted by prefix length descending to ensure longest (most specific) match wins.
// This prevents non-deterministic matching when prefixes overlap
// (e.g., "rook-ceph-osd-prepare" must match before "rook-ceph-osd").
var podTypePrefixes = []struct {
	prefix string
	typ    string
}{
	// Longest prefixes first
	{"rook-ceph-crashcollector", "crashcollector"},
	{"rook-ceph-osd-prepare", "prepare"},
	{"rook-ceph-exporter", "exporter"},
	{"rook-ceph-operator", "operator"},
	{"rook-ceph-cleanup", "cleanup"},
	{"csi-cephfsplugin", "csi"},
	{"rook-ceph-tools", "tools"},
	{"csi-rbdplugin", "csi"},
	// Shorter prefixes last
	{"rook-ceph-osd", "osd"},
	{"rook-ceph-mon", "mon"},
	{"rook-ceph-mgr", "mgr"},
	{"rook-ceph-mds", "mds"},
	{"rook-ceph-rgw", "rgw"},
}

// extractPodType extracts the pod type from the name.
// Uses longest-prefix-first matching to ensure deterministic results
// when prefixes overlap (e.g., "rook-ceph-osd-prepare" vs "rook-ceph-osd").
func extractPodType(name string) string {
	for _, entry := range podTypePrefixes {
		if strings.HasPrefix(name, entry.prefix) {
			return entry.typ
		}
	}
	return "other"
}

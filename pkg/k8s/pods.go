package k8s

import (
	"bytes"
	"context"
	"fmt"
	"strings"

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
	Pod        OwnerInfo
	ReplicaSet *OwnerInfo
	Deployment *OwnerInfo
	StatefulSet *OwnerInfo
	DaemonSet  *OwnerInfo
	// Other possible owners
	Other []OwnerInfo
}

// ListPodsOnNode returns all pods running on a specific node
func (c *Client) ListPodsOnNode(ctx context.Context, nodeName string) ([]corev1.Pod, error) {
	// Use field selector to efficiently filter pods by node
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", nodeName)

	podList, err := c.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	return podList.Items, nil
}

// ListPodsOnNode is a package-level function that uses the global client
func ListPodsOnNode(ctx context.Context, nodeName string) ([]corev1.Pod, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.ListPodsOnNode(ctx, nodeName)
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

// GetOwnerChain is a package-level function that uses the global client
func GetOwnerChain(ctx context.Context, pod *corev1.Pod) (*OwnerChain, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetOwnerChain(ctx, pod)
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

// ExecInPod is a package-level function that uses the global client
func ExecInPod(ctx context.Context, namespace, podName, containerName string, command []string) (string, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return "", err
	}
	return client.ExecInPod(ctx, namespace, podName, containerName, command)
}

// GetPod returns a pod by namespace and name
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// GetPod is a package-level function that uses the global client
func GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetPod(ctx, namespace, name)
}

// ListPodsInNamespace returns all pods in a namespace
func (c *Client) ListPodsInNamespace(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}
	return podList.Items, nil
}

// ListPodsInNamespace is a package-level function that uses the global client
func ListPodsInNamespace(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.ListPodsInNamespace(ctx, namespace)
}

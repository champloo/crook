package k8s

import (
	"k8s.io/client-go/kubernetes"
)

// newClientFromInterface creates a Client from a kubernetes.Interface for testing.
// This is used in tests to inject fake clientsets.
// Deprecated: Use NewClientFromClientset instead.
func newClientFromInterface(clientset kubernetes.Interface) *Client {
	return NewClientFromClientset(clientset)
}

// newClientFromClientset creates a Client from a kubernetes.Interface for testing.
// Deprecated: Use NewClientFromClientset instead.
func newClientFromClientset(clientset kubernetes.Interface) *Client {
	return NewClientFromClientset(clientset)
}

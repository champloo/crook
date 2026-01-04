package k8s

import (
	"k8s.io/client-go/kubernetes"
)

// newClientFromInterface creates a Client from a kubernetes.Interface for testing.
// This is used in tests to inject fake clientsets.
func newClientFromInterface(clientset kubernetes.Interface) *Client {
	return &Client{
		Clientset: clientset,
	}
}

// newClientFromClientset creates a Client from a kubernetes.Interface for testing.
// This is an alias for newClientFromInterface for backward compatibility.
func newClientFromClientset(clientset kubernetes.Interface) *Client {
	return &Client{
		Clientset: clientset,
	}
}

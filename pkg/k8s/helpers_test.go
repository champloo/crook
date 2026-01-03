package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

// newClientFromInterfaceWithConfig creates a Client from a kubernetes.Interface and rest.Config for testing.
// This is used in tests that need to mock both the clientset and the REST config (e.g., for exec operations).
func newClientFromInterfaceWithConfig(clientset kubernetes.Interface, config *rest.Config) *Client {
	return &Client{
		Clientset: clientset,
		config:    config,
	}
}

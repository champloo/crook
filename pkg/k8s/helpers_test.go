package k8s

import (
	"k8s.io/client-go/kubernetes"
)

// newClientFromClientset creates a Client from an existing clientset for testing.
func newClientFromClientset(clientset kubernetes.Interface) *Client {
	return &Client{
		Clientset:          clientset,
		cephCommandTimeout: DefaultCephTimeout,
	}
}

// newClientFromInterface is an alias for newClientFromClientset for backward compatibility.
func newClientFromInterface(clientset kubernetes.Interface) *Client {
	return newClientFromClientset(clientset)
}

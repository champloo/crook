package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/andri/crook/pkg/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DefaultCephTimeout is the default timeout for Ceph CLI commands.
const DefaultCephTimeout = time.Duration(config.DefaultCephCommandTimeoutSeconds) * time.Second

// Client wraps Kubernetes clientset with additional functionality
type Client struct {
	Clientset          kubernetes.Interface
	config             *rest.Config
	cephCommandTimeout time.Duration
}

// ClientConfig holds configuration for creating a Kubernetes client
type ClientConfig struct {
	// CephCommandTimeout is the timeout for Ceph CLI commands.
	// If zero, uses DefaultCephTimeout.
	CephCommandTimeout time.Duration
}

// NewClient creates a new Kubernetes client with the given configuration
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	config, err := buildConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, clientErr := kubernetes.NewForConfig(config)
	if clientErr != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", clientErr)
	}

	cephTimeout := cfg.CephCommandTimeout
	if cephTimeout == 0 {
		cephTimeout = DefaultCephTimeout
	}

	client := &Client{
		Clientset:          clientset,
		config:             config,
		cephCommandTimeout: cephTimeout,
	}

	// Validate connectivity by checking the /version endpoint
	if validateErr := client.validateConnectivity(ctx); validateErr != nil {
		return nil, fmt.Errorf("failed to validate kubernetes connectivity: %w", validateErr)
	}

	return client, nil
}

// NewClientFromClientset creates a Client from an existing clientset.
// This is primarily useful for testing with fake clientsets.
func NewClientFromClientset(clientset kubernetes.Interface) *Client {
	return &Client{
		Clientset:          clientset,
		cephCommandTimeout: DefaultCephTimeout,
	}
}

// buildConfig builds a Kubernetes REST config using standard resolution order:
// 1. In-cluster config (when running inside a pod)
// 2. KUBECONFIG environment variable (supports colon-separated paths)
// 3. Default kubeconfig location (~/.kube/config)
func buildConfig(_ ClientConfig) (*rest.Config, error) {
	// Use client-go's standard loading rules which handle:
	// - In-cluster config detection
	// - KUBECONFIG env var (including colon-separated paths)
	// - Default ~/.kube/config location
	// - Proper path expansion
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config, nil
}

// validateConnectivity validates that the client can communicate with the Kubernetes API
func (c *Client) validateConnectivity(_ context.Context) error {
	_, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes API server: %w", err)
	}
	return nil
}

// Config returns the REST config used by this client
func (c *Client) Config() *rest.Config {
	return c.config
}

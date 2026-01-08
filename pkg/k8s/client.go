package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	// Kubeconfig path. If empty, uses standard resolution:
	// 1. KUBECONFIG environment variable
	// 2. ~/.kube/config
	// 3. In-cluster config
	Kubeconfig string

	// Context name to use from kubeconfig. If empty, uses current context.
	Context string

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

// buildConfig builds a Kubernetes REST config from the given configuration
func buildConfig(cfg ClientConfig) (*rest.Config, error) {
	// Try in-cluster config first if no kubeconfig specified
	if cfg.Kubeconfig == "" {
		if config, err := rest.InClusterConfig(); err == nil {
			return config, nil
		}
	}

	// Resolve kubeconfig path
	kubeconfigPath := cfg.Kubeconfig
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig file not found: %s", kubeconfigPath)
		}
		return nil, fmt.Errorf("failed to stat kubeconfig file: %w", err)
	}

	// Build config from kubeconfig file
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeconfigPath,
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if cfg.Context != "" {
		configOverrides.CurrentContext = cfg.Context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
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

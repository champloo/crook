package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes clientset with additional functionality
type Client struct {
	Clientset kubernetes.Interface
	config    *rest.Config
	mu        sync.RWMutex
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
}

var (
	// globalClient is the cached client instance
	globalClient *Client
	clientMu     sync.Mutex
)

// NewClient creates a new Kubernetes client with the given configuration
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	config, err := buildConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	client := &Client{
		Clientset: clientset,
		config:    config,
	}

	// Validate connectivity by checking the /version endpoint
	if err := client.validateConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to validate kubernetes connectivity: %w", err)
	}

	return client, nil
}

// GetClient returns the global client instance, creating it if necessary
// This provides a convenient way to access the client without passing it around
func GetClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if globalClient == nil {
		client, err := NewClient(ctx, cfg)
		if err != nil {
			return nil, err
		}
		globalClient = client
	}

	return globalClient, nil
}

// SetGlobalClient sets the global client instance
// Useful for testing or custom initialization
func SetGlobalClient(client *Client) {
	clientMu.Lock()
	defer clientMu.Unlock()
	globalClient = client
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
func (c *Client) validateConnectivity(ctx context.Context) error {
	_, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes API server: %w", err)
	}
	return nil
}

// Config returns the REST config used by this client
func (c *Client) Config() *rest.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

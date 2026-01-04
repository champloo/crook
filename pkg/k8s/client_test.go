package k8s

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ClientConfig
		setupEnv    func() func()
		expectError bool
	}{
		{
			name: "missing kubeconfig file",
			cfg: ClientConfig{
				Kubeconfig: "/nonexistent/path/kubeconfig",
			},
			expectError: true,
		},
		{
			name: "empty config with no kubeconfig env",
			cfg:  ClientConfig{},
			setupEnv: func() func() {
				old := os.Getenv("KUBECONFIG")
				_ = os.Unsetenv("KUBECONFIG")
				return func() {
					if old != "" {
						_ = os.Setenv("KUBECONFIG", old)
					}
				}
			},
			expectError: true, // Will fail unless ~/.kube/config exists or in-cluster
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				cleanup := tt.setupEnv()
				defer cleanup()
			}

			_, err := buildConfig(tt.cfg)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetGlobalClient(t *testing.T) {
	// Save original global client
	original := globalClient
	defer func() {
		globalClient = original
	}()

	// Create a dummy client
	client := &Client{}

	SetGlobalClient(client)

	if globalClient != client {
		t.Error("SetGlobalClient did not set the global client")
	}
}

func TestClientConfig_Defaults(t *testing.T) {
	cfg := ClientConfig{}

	if cfg.Kubeconfig != "" {
		t.Errorf("expected empty kubeconfig, got %s", cfg.Kubeconfig)
	}
	if cfg.Context != "" {
		t.Errorf("expected empty context, got %s", cfg.Context)
	}
}

// TestNewClient_WithValidKubeconfig tests client creation with a valid kubeconfig
// Note: This test requires a valid kubeconfig to be present
func TestNewClient_WithValidKubeconfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if we have a kubeconfig available
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	kubeconfigPath := filepath.Join(home, ".kube", "config")
	if _, statErr := os.Stat(kubeconfigPath); os.IsNotExist(statErr) {
		t.Skip("no kubeconfig found, skipping integration test")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Kubeconfig: kubeconfigPath,
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Skipf("failed to create client (cluster may not be accessible): %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Clientset == nil {
		t.Fatal("expected non-nil clientset")
	}
}

package commands

import (
	"context"
	"testing"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/maintenance"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunUpExecutesWithNoDeployments(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Namespace = "rook-ceph"

	prevConfig := GlobalOptions.Config
	GlobalOptions.Config = cfg
	t.Cleanup(func() { GlobalOptions.Config = prevConfig })

	clientset := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
	})
	fakeClient := k8s.NewClientFromClientset(clientset)

	prevNewClient := newK8sClient
	prevExecuteUp := executeUpPhase
	t.Cleanup(func() {
		newK8sClient = prevNewClient
		executeUpPhase = prevExecuteUp
	})

	newK8sClient = func(_ context.Context, _ k8s.ClientConfig) (*k8s.Client, error) {
		return fakeClient, nil
	}

	called := false
	executeUpPhase = func(ctx context.Context, client *k8s.Client, cfg config.Config, nodeName string, opts maintenance.UpPhaseOptions) error {
		called = true
		return nil
	}

	if err := runUp(context.Background(), "node-1", &UpOptions{Yes: true}); err != nil {
		t.Fatalf("runUp returned error: %v", err)
	}
	if !called {
		t.Fatal("expected executeUpPhase to be called")
	}
}

func TestRunDownExecutesWhenDeploymentsAlreadyScaledDown(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Namespace = "rook-ceph"

	prevConfig := GlobalOptions.Config
	GlobalOptions.Config = cfg
	t.Cleanup(func() { GlobalOptions.Config = prevConfig })

	replicas := int32(0)
	clientset := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rook-ceph-osd-0",
				Namespace: "rook-ceph",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{"kubernetes.io/hostname": "node-1"},
					},
				},
			},
		},
	)
	fakeClient := k8s.NewClientFromClientset(clientset)

	prevNewClient := newK8sClient
	prevExecuteDown := executeDownPhase
	t.Cleanup(func() {
		newK8sClient = prevNewClient
		executeDownPhase = prevExecuteDown
	})

	newK8sClient = func(_ context.Context, _ k8s.ClientConfig) (*k8s.Client, error) {
		return fakeClient, nil
	}

	called := false
	executeDownPhase = func(ctx context.Context, client *k8s.Client, cfg config.Config, nodeName string, opts maintenance.DownPhaseOptions) error {
		called = true
		return nil
	}

	if err := runDown(context.Background(), "node-1", &DownOptions{Yes: true}); err != nil {
		t.Fatalf("runDown returned error: %v", err)
	}
	if !called {
		t.Fatal("expected executeDownPhase to be called")
	}
}

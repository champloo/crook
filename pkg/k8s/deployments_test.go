package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestScaleDeployment(t *testing.T) {
	ctx := context.Background()

	// Create a fake clientset with a test deployment
	initialReplicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	// Scale deployment to 0
	err := client.ScaleDeployment(ctx, "default", "test-deployment", 0)
	if err != nil {
		t.Fatalf("failed to scale deployment: %v", err)
	}

	// Verify the deployment was scaled
	updated, err := clientset.AppsV1().Deployments("default").Get(ctx, "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}

	if *updated.Spec.Replicas != 0 {
		t.Errorf("expected replicas to be 0, got %d", *updated.Spec.Replicas)
	}
}

func TestScaleDeployment_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromClientset(clientset)

	err := client.ScaleDeployment(ctx, "default", "nonexistent", 1)
	if err == nil {
		t.Error("expected error when scaling nonexistent deployment, got nil")
	}
}

func TestGetDeploymentStatus(t *testing.T) {
	ctx := context.Background()

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          3,
			ReadyReplicas:     2,
			AvailableReplicas: 2,
			UpdatedReplicas:   3,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	status, err := client.GetDeploymentStatus(ctx, "default", "test-deployment")
	if err != nil {
		t.Fatalf("failed to get deployment status: %v", err)
	}

	if status.Name != "test-deployment" {
		t.Errorf("expected name 'test-deployment', got %s", status.Name)
	}
	if status.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %s", status.Namespace)
	}
	if status.Replicas != 3 {
		t.Errorf("expected replicas 3, got %d", status.Replicas)
	}
	if status.ReadyReplicas != 2 {
		t.Errorf("expected ready replicas 2, got %d", status.ReadyReplicas)
	}
	if status.AvailableReplicas != 2 {
		t.Errorf("expected available replicas 2, got %d", status.AvailableReplicas)
	}
}

func TestListDeploymentsInNamespace(t *testing.T) {
	ctx := context.Background()

	replicas := int32(1)
	deployment1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-1",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	deployment2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-2",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	deployment3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-3",
			Namespace: "other",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	clientset := fake.NewClientset(deployment1, deployment2, deployment3)
	client := newClientFromClientset(clientset)

	deployments, err := client.ListDeploymentsInNamespace(ctx, "default")
	if err != nil {
		t.Fatalf("failed to list deployments: %v", err)
	}

	if len(deployments) != 2 {
		t.Errorf("expected 2 deployments in default namespace, got %d", len(deployments))
	}

	// Verify names
	names := make(map[string]bool)
	for _, d := range deployments {
		names[d.Name] = true
	}

	if !names["deployment-1"] || !names["deployment-2"] {
		t.Errorf("expected deployment-1 and deployment-2, got %v", names)
	}
}

func TestFilterDeploymentsByPrefix(t *testing.T) {
	replicas := int32(1)
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-osd-0"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-mon-a"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other-deployment"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-exporter"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
	}

	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon", "rook-ceph-exporter"}
	filtered := FilterDeploymentsByPrefix(deployments, prefixes)

	if len(filtered) != 3 {
		t.Errorf("expected 3 filtered deployments, got %d", len(filtered))
	}

	// Verify the right ones were filtered
	for _, d := range filtered {
		if d.Name == "other-deployment" {
			t.Errorf("other-deployment should not be in filtered results")
		}
	}
}

func TestFilterDeploymentsByPrefix_EmptyPrefixes(t *testing.T) {
	replicas := int32(1)
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "deployment-1"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
	}

	filtered := FilterDeploymentsByPrefix(deployments, []string{})
	if len(filtered) != len(deployments) {
		t.Errorf("expected all deployments when prefixes is empty, got %d", len(filtered))
	}
}

func TestWaitForReplicas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	ctx := context.Background()

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	opts := WaitForReplicasOptions{
		PollInterval: 100 * time.Millisecond,
		Timeout:      1 * time.Second,
	}

	// This should succeed immediately since replicas already match
	err := client.WaitForReplicas(ctx, "default", "test-deployment", 3, opts)
	if err != nil {
		t.Fatalf("failed to wait for replicas: %v", err)
	}
}

func TestWaitForReplicas_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	ctx := context.Background()

	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	opts := WaitForReplicasOptions{
		PollInterval: 100 * time.Millisecond,
		Timeout:      300 * time.Millisecond,
	}

	// This should timeout since we're waiting for 5 replicas but deployment has 1
	err := client.WaitForReplicas(ctx, "default", "test-deployment", 5, opts)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestDefaultWaitOptions(t *testing.T) {
	opts := DefaultWaitOptions()

	if opts.PollInterval != 5*time.Second {
		t.Errorf("expected poll interval 5s, got %v", opts.PollInterval)
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("expected timeout 5m, got %v", opts.Timeout)
	}
}

func TestGetDeploymentTargetNode(t *testing.T) {
	tests := []struct {
		name     string
		dep      *appsv1.Deployment
		expected string
	}{
		{
			name: "nodeSelector with hostname",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-osd-0"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "worker-01",
							},
						},
					},
				},
			},
			expected: "worker-01",
		},
		{
			name: "nodeAffinity required matching hostname",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/hostname",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"worker-02"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "worker-02",
		},
		{
			name: "no nodeSelector or nodeAffinity",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "portable-deployment"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{},
					},
				},
			},
			expected: "",
		},
		{
			name: "nodeSelector for different key",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "zone-pinned"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"topology.kubernetes.io/zone": "us-east-1a",
							},
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "preferredDuringScheduling only returns empty",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "preferred-deployment"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
										{
											Weight: 100,
											Preference: corev1.NodeSelectorTerm{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/hostname",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"preferred-node"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "multiple nodeAffinity values returns first",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "multi-value"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/hostname",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"first-node", "second-node"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "first-node",
		},
		{
			name: "nodeSelector takes precedence over nodeAffinity",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "both-set"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/hostname": "selector-node",
							},
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/hostname",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"affinity-node"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "selector-node",
		},
		{
			name: "nodeAffinity with NotIn operator returns empty",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "wrong-operator"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/hostname",
														Operator: corev1.NodeSelectorOpNotIn,
														Values:   []string{"worker-01"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "nodeAffinity with wrong key returns empty",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "wrong-key"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "topology.kubernetes.io/zone",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"us-east-1a"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "nodeAffinity with empty values returns empty",
			dep: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "empty-values"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/hostname",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDeploymentTargetNode(tt.dep)
			if result != tt.expected {
				t.Errorf("GetDeploymentTargetNode() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestListNodePinnedDeployments(t *testing.T) {
	ctx := context.Background()

	replicas := int32(1)
	zeroReplicas := int32(0)

	// Deployments pinned to worker-01
	dep1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-0",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}
	dep2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-crashcollector-worker-01",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &zeroReplicas, // Scaled down but still pinned
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// Deployment pinned to worker-02
	dep3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-1",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-02",
					},
				},
			},
		},
	}

	// Portable deployment (no nodeSelector)
	dep4 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-operator",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{},
			},
		},
	}

	clientset := fake.NewClientset(dep1, dep2, dep3, dep4)
	client := newClientFromClientset(clientset)

	// Test with nil prefixes (returns all node-pinned deployments)
	tests := []struct {
		name          string
		nodeName      string
		prefixes      []string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "finds deployments pinned to worker-01 with nil prefixes",
			nodeName:      "worker-01",
			prefixes:      nil,
			expectedCount: 2,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-crashcollector-worker-01"},
		},
		{
			name:          "finds deployments pinned to worker-01 with empty prefixes",
			nodeName:      "worker-01",
			prefixes:      []string{},
			expectedCount: 2,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-crashcollector-worker-01"},
		},
		{
			name:          "finds deployments pinned to worker-02",
			nodeName:      "worker-02",
			prefixes:      nil,
			expectedCount: 1,
			expectedNames: []string{"rook-ceph-osd-1"},
		},
		{
			name:          "returns empty for non-existent node",
			nodeName:      "worker-99",
			prefixes:      nil,
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.ListNodePinnedDeployments(ctx, "rook-ceph", tt.nodeName, tt.prefixes)
			if err != nil {
				t.Fatalf("ListNodePinnedDeployments() error = %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("ListNodePinnedDeployments() count = %d, expected %d", len(result), tt.expectedCount)
			}

			resultNames := make(map[string]bool)
			for _, dep := range result {
				resultNames[dep.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !resultNames[expectedName] {
					t.Errorf("ListNodePinnedDeployments() missing expected deployment %q", expectedName)
				}
			}
		})
	}
}

func TestListNodePinnedDeployments_EmptyNamespace(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromClientset(clientset)

	result, err := client.ListNodePinnedDeployments(ctx, "empty-namespace", "worker-01", nil)
	if err != nil {
		t.Fatalf("ListNodePinnedDeployments() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("ListNodePinnedDeployments() expected empty slice, got %d deployments", len(result))
	}
}

func TestListNodePinnedDeployments_PrefixFiltering(t *testing.T) {
	ctx := context.Background()

	replicas := int32(1)

	// OSD deployment pinned to worker-01
	depOsd := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-0",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// MON deployment pinned to worker-01
	depMon := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-mon-a",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// rook-ceph-tools deployment pinned to worker-01 (should be excluded with proper prefixes)
	depTools := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// Non-Ceph deployment pinned to worker-01 (should be excluded)
	depCustom := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-app",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	clientset := fake.NewClientset(depOsd, depMon, depTools, depCustom)
	client := newClientFromClientset(clientset)

	tests := []struct {
		name          string
		prefixes      []string
		expectedCount int
		expectedNames []string
		excludedNames []string
	}{
		{
			name:          "filters to only OSD deployments",
			prefixes:      []string{"rook-ceph-osd"},
			expectedCount: 1,
			expectedNames: []string{"rook-ceph-osd-0"},
			excludedNames: []string{"rook-ceph-mon-a", "rook-ceph-tools", "custom-app"},
		},
		{
			name:          "filters to OSD and MON deployments",
			prefixes:      []string{"rook-ceph-osd", "rook-ceph-mon"},
			expectedCount: 2,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-mon-a"},
			excludedNames: []string{"rook-ceph-tools", "custom-app"},
		},
		{
			name: "default Ceph prefixes exclude tools and custom apps",
			prefixes: []string{
				"rook-ceph-osd",
				"rook-ceph-mon",
				"rook-ceph-exporter",
				"rook-ceph-crashcollector",
			},
			expectedCount: 2,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-mon-a"},
			excludedNames: []string{"rook-ceph-tools", "custom-app"},
		},
		{
			name:          "empty prefixes returns all node-pinned deployments",
			prefixes:      []string{},
			expectedCount: 4,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-mon-a", "rook-ceph-tools", "custom-app"},
			excludedNames: []string{},
		},
		{
			name:          "nil prefixes returns all node-pinned deployments",
			prefixes:      nil,
			expectedCount: 4,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-mon-a", "rook-ceph-tools", "custom-app"},
			excludedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.ListNodePinnedDeployments(ctx, "rook-ceph", "worker-01", tt.prefixes)
			if err != nil {
				t.Fatalf("ListNodePinnedDeployments() error = %v", err)
			}

			if len(result) != tt.expectedCount {
				names := make([]string, len(result))
				for i, dep := range result {
					names[i] = dep.Name
				}
				t.Errorf("ListNodePinnedDeployments() count = %d, expected %d, got names: %v", len(result), tt.expectedCount, names)
			}

			resultNames := make(map[string]bool)
			for _, dep := range result {
				resultNames[dep.Name] = true
			}

			// Check expected names are present
			for _, expectedName := range tt.expectedNames {
				if !resultNames[expectedName] {
					t.Errorf("ListNodePinnedDeployments() missing expected deployment %q", expectedName)
				}
			}

			// Check excluded names are absent
			for _, excludedName := range tt.excludedNames {
				if resultNames[excludedName] {
					t.Errorf("ListNodePinnedDeployments() should have excluded deployment %q but it was included", excludedName)
				}
			}
		})
	}
}

func TestListScaledDownDeploymentsForNode(t *testing.T) {
	ctx := context.Background()

	replicas := int32(1)
	zeroReplicas := int32(0)

	// Deployment at 0 replicas on worker-01
	depScaledDown1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-0",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &zeroReplicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// Deployment at 0 replicas on worker-01
	depScaledDown2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-crashcollector-worker-01",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &zeroReplicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// Deployment at 1 replica on worker-01 (not scaled down)
	depRunning := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-exporter-worker-01",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	// Deployment at 0 replicas on worker-02
	depOtherNode := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-1",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &zeroReplicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-02",
					},
				},
			},
		},
	}

	clientset := fake.NewClientset(depScaledDown1, depScaledDown2, depRunning, depOtherNode)
	client := newClientFromClientset(clientset)

	tests := []struct {
		name          string
		nodeName      string
		prefixes      []string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "finds scaled-down deployments on worker-01 with nil prefixes",
			nodeName:      "worker-01",
			prefixes:      nil,
			expectedCount: 2,
			expectedNames: []string{"rook-ceph-osd-0", "rook-ceph-crashcollector-worker-01"},
		},
		{
			name:          "finds scaled-down deployments on worker-02",
			nodeName:      "worker-02",
			prefixes:      nil,
			expectedCount: 1,
			expectedNames: []string{"rook-ceph-osd-1"},
		},
		{
			name:          "returns empty for node with no scaled-down deployments",
			nodeName:      "worker-99",
			prefixes:      nil,
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "filters by prefix - only OSD",
			nodeName:      "worker-01",
			prefixes:      []string{"rook-ceph-osd"},
			expectedCount: 1,
			expectedNames: []string{"rook-ceph-osd-0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.ListScaledDownDeploymentsForNode(ctx, "rook-ceph", tt.nodeName, tt.prefixes)
			if err != nil {
				t.Fatalf("ListScaledDownDeploymentsForNode() error = %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("ListScaledDownDeploymentsForNode() count = %d, expected %d", len(result), tt.expectedCount)
			}

			resultNames := make(map[string]bool)
			for _, dep := range result {
				resultNames[dep.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !resultNames[expectedName] {
					t.Errorf("ListScaledDownDeploymentsForNode() missing expected deployment %q", expectedName)
				}
			}
		})
	}
}

func TestListScaledDownDeploymentsForNode_NilReplicas(t *testing.T) {
	ctx := context.Background()

	// Deployment with nil Replicas pointer (defaults to 1)
	depNilReplicas := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-nil",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			// Replicas is nil
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "worker-01",
					},
				},
			},
		},
	}

	clientset := fake.NewClientset(depNilReplicas)
	client := newClientFromClientset(clientset)

	result, err := client.ListScaledDownDeploymentsForNode(ctx, "rook-ceph", "worker-01", nil)
	if err != nil {
		t.Fatalf("ListScaledDownDeploymentsForNode() error = %v", err)
	}

	// Nil replicas should NOT be considered scaled down
	if len(result) != 0 {
		t.Errorf("ListScaledDownDeploymentsForNode() expected 0 (nil replicas = default 1), got %d", len(result))
	}
}

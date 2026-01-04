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

func TestListCephDeployments(t *testing.T) {
	ctx := context.Background()

	replicas := int32(1)
	zeroReplicas := int32(0)

	// Create test deployments
	deployments := &appsv1.DeploymentList{
		Items: []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "rook-ceph-osd-0",
					Namespace:         "rook-ceph",
					Labels:            map[string]string{"ceph-osd-id": "0"},
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "rook-ceph-mon-a",
					Namespace:         "rook-ceph",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-48 * time.Hour)},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "rook-ceph-crashcollector-worker-1",
					Namespace:         "rook-ceph",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-72 * time.Hour)},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &zeroReplicas,
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 0,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "other-deployment",
					Namespace:         "rook-ceph",
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
				},
			},
		},
	}

	// Create test pods
	pods := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rook-ceph-osd-0-abc123-xyz",
					Namespace: "rook-ceph",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "ReplicaSet",
							Name: "rook-ceph-osd-0-abc123",
						},
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "worker-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rook-ceph-mon-a-def456-uvw",
					Namespace: "rook-ceph",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "ReplicaSet",
							Name: "rook-ceph-mon-a-def456",
						},
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "worker-2",
				},
			},
		},
	}

	// Create fake clientset
	clientset := fake.NewClientset(deployments, pods)
	client := &Client{Clientset: clientset}

	// Test with Ceph prefixes
	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon", "rook-ceph-crashcollector"}
	result, err := client.ListCephDeployments(ctx, "rook-ceph", prefixes)

	if err != nil {
		t.Fatalf("ListCephDeployments() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}

	// Check deployment info
	depMap := make(map[string]DeploymentInfoForLS)
	for _, d := range result {
		depMap[d.Name] = d
	}

	// Check OSD deployment
	osd := depMap["rook-ceph-osd-0"]
	if osd.Type != "osd" {
		t.Errorf("osd Type = %s, want osd", osd.Type)
	}
	if osd.OsdID != "0" {
		t.Errorf("osd OsdID = %s, want 0", osd.OsdID)
	}
	if osd.Status != "Ready" {
		t.Errorf("osd Status = %s, want Ready", osd.Status)
	}
	if osd.NodeName != "worker-1" {
		t.Errorf("osd NodeName = %s, want worker-1", osd.NodeName)
	}

	// Check MON deployment
	mon := depMap["rook-ceph-mon-a"]
	if mon.Type != "mon" {
		t.Errorf("mon Type = %s, want mon", mon.Type)
	}
	if mon.NodeName != "worker-2" {
		t.Errorf("mon NodeName = %s, want worker-2", mon.NodeName)
	}

	// Check crashcollector (scaled down)
	cc := depMap["rook-ceph-crashcollector-worker-1"]
	if cc.Type != "crashcollector" {
		t.Errorf("crashcollector Type = %s, want crashcollector", cc.Type)
	}
	if cc.Status != "Scaled Down" {
		t.Errorf("crashcollector Status = %s, want 'Scaled Down'", cc.Status)
	}
}

func TestGetDeploymentStatusString(t *testing.T) {
	tests := []struct {
		name    string
		desired int32
		ready   int32
		want    string
	}{
		{
			name:    "ready deployment",
			desired: 1,
			ready:   1,
			want:    "Ready",
		},
		{
			name:    "scaling deployment",
			desired: 2,
			ready:   1,
			want:    "Scaling",
		},
		{
			name:    "unavailable deployment",
			desired: 1,
			ready:   0,
			want:    "Unavailable",
		},
		{
			name:    "scaled down deployment",
			desired: 0,
			ready:   0,
			want:    "Scaled Down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: &tt.desired,
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: tt.ready,
				},
			}
			if got := getDeploymentStatusString(dep); got != tt.want {
				t.Errorf("getDeploymentStatusString() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractDeploymentType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"rook-ceph-osd-0", "osd"},
		{"rook-ceph-osd-10", "osd"},
		{"rook-ceph-mon-a", "mon"},
		{"rook-ceph-mon-b", "mon"},
		{"rook-ceph-mgr-a", "mgr"},
		{"rook-ceph-crashcollector-worker-1", "crashcollector"},
		{"rook-ceph-exporter-worker-1", "exporter"},
		{"rook-ceph-tools", "tools"},
		{"rook-ceph-operator", "operator"},
		{"unknown-deployment", "other"},
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractDeploymentType(tt.name); got != tt.want {
				t.Errorf("extractDeploymentType(%q) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestExtractOsdID(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "with osd id",
			labels: map[string]string{"ceph-osd-id": "5"},
			want:   "5",
		},
		{
			name:   "without osd id",
			labels: map[string]string{"other-label": "value"},
			want:   "",
		},
		{
			name:   "nil labels",
			labels: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tt.labels,
				},
			}
			if got := extractOsdID(dep); got != tt.want {
				t.Errorf("extractOsdID() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetDeploymentDesiredReplicas(t *testing.T) {
	tests := []struct {
		name     string
		replicas *int32
		want     int32
	}{
		{
			name:     "with replicas set",
			replicas: int32Ptr(3),
			want:     3,
		},
		{
			name:     "with zero replicas",
			replicas: int32Ptr(0),
			want:     0,
		},
		{
			name:     "nil replicas (defaults to 1)",
			replicas: nil,
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: tt.replicas,
				},
			}
			if got := getDeploymentDesiredReplicas(dep); got != tt.want {
				t.Errorf("getDeploymentDesiredReplicas() = %d, want %d", got, tt.want)
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

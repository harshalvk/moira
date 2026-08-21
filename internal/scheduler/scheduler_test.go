package scheduler

import (
	"context"
	"io"
	"log/slog"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func readyNode(name, cpu, mem string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			},
		},
	}
}

func podRequesting(name, cpu, mem string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: corev1.PodSpec{
			SchedulerName: SchedulerName,
			Containers: []corev1.Container{{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cpu),
						corev1.ResourceMemory: resource.MustParse(mem),
					},
				},
			}},
		},
	}
}

func TestPickNode_ChoosesFeasibleNode(t *testing.T) {
	client := fake.NewSimpleClientset(
		readyNode("small", "1", "2Gi"),
		readyNode("large", "8", "16Gi"),
	)
	s, err := New(client, discardLogger(), DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error constructing scheduler: %v", err)
	}

	pod := podRequesting("p1", "4", "8Gi")
	node, err := s.pickNode(context.Background(), pod)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node != "large" {
		t.Fatalf("expected pod to be filtered onto the only fitting node 'large', got %q", node)
	}
}

func TestPickNode_NoFeasibleNode_ReturnsError(t *testing.T) {
	client := fake.NewSimpleClientset(readyNode("small", "1", "2Gi"))
	s, err := New(client, discardLogger(), DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error constructing scheduler: %v", err)
	}

	pod := podRequesting("p1", "4", "8Gi")
	_, err = s.pickNode(context.Background(), pod)
	if err == nil {
		t.Fatal("expected error when no node has capacity")
	}
}

func TestPickNode_PrefersLessAllocatedNode(t *testing.T) {
	client := fake.NewSimpleClientset(
		readyNode("busy", "4", "8Gi"),
		readyNode("empty", "4", "8Gi"),
	)
	s, err := New(client, discardLogger(), DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error constructing scheduler: %v", err)
	}

	// Pre-occupy "busy" via the assume cache to simulate existing load
	// without needing a real bound pod in the fake clientset.
	busyPod := podRequesting("existing", "3", "6Gi")
	s.cache.Assume(busyPod, "busy")

	pod := podRequesting("p1", "500m", "1Gi")
	node, err := s.pickNode(context.Background(), pod)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node != "empty" {
		t.Fatalf("expected NodeResourcesLeastAllocated to prefer 'empty', got %q", node)
	}
}

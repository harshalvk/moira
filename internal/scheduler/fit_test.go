package scheduler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podWithRequests(name, cpu, mem string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpu),
							corev1.ResourceMemory: resource.MustParse(mem),
						},
					},
				},
			},
		},
	}
}

func nodeWithAllocatable(name, cpu, mem string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
}

func TestFitsNode_EmptyNodeHasCapacity(t *testing.T) {
	pod := podWithRequests("p1", "500m", "256Mi")
	node := nodeWithAllocatable("n1", "2", "4Gi")

	if !FitsNode(pod, node, nil) {
		t.Fatal("expected pod to fit on empty node with ample capacity")
	}
}

func TestFitsNode_RequestExceedsAllocatable(t *testing.T) {
	pod := podWithRequests("p1", "4", "8Gi")
	node := nodeWithAllocatable("n1", "2", "4Gi")

	if FitsNode(pod, node, nil) {
		t.Fatal("expected pod not to fit: request exceeds total allocatable")
	}
}

func TestFitsNode_AccountsForExistingPods(t *testing.T) {
	pod := podWithRequests("p2", "1", "1Gi")
	node := nodeWithAllocatable("n1", "2", "4Gi")
	existing := []*corev1.Pod{
		podWithRequests("p1", "1500m", "3Gi"),
	}

	if FitsNode(pod, node, existing) {
		t.Fatal("expected pod not to fit: existing pod already consumed most capacity")
	}
}

func TestFitsNode_ExactFit(t *testing.T) {
	pod := podWithRequests("p1", "2", "4Gi")
	node := nodeWithAllocatable("n1", "2", "4Gi")

	if !FitsNode(pod, node, nil) {
		t.Fatal("expected exact-capacity match to fit")
	}
}

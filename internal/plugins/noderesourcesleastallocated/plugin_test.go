package noderesourcesleastallocated

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harshalvk/moira/internal/framework"
)

func TestScore_EmptyNodeScoresHigherThanBusyNode(t *testing.T) {
	p := New()
	pod := &corev1.Pod{}

	emptyNode := &framework.NodeInfo{
		Node: &corev1.Node{Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("4"),
			corev1.ResourceMemory: resource.MustParse("8Gi"),
		}}},
	}

	busyPod := &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
		Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("3"),
			corev1.ResourceMemory: resource.MustParse("6Gi"),
		}},
	}}}}
	busyNode := &framework.NodeInfo{
		Node: emptyNode.Node,
		Pods: []*corev1.Pod{busyPod},
	}

	emptyScore, status := p.Score(pod, emptyNode)
	if !status.IsSuccess() {
		t.Fatalf("unexpected status: %+v", status)
	}
	busyScore, status := p.Score(pod, busyNode)
	if !status.IsSuccess() {
		t.Fatalf("unexpected status: %+v", status)
	}

	if emptyScore <= busyScore {
		t.Fatalf("expected empty node to score higher: empty=%d busy=%d", emptyScore, busyScore)
	}
}

func TestScore_ZeroAllocatableDoesNotPanic(t *testing.T) {
	p := New()
	ni := &framework.NodeInfo{
		Node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "n1"}}, // no Allocatable set
	}
	score, status := p.Score(&corev1.Pod{}, ni)
	if !status.IsSuccess() {
		t.Fatalf("unexpected status: %+v", status)
	}
	if score != 0 {
		t.Fatalf("expected zero score for zero-allocatable node, got %d", score)
	}
}

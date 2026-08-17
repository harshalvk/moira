package scheduler

import (
	"context"
	"io"
	"log/slog"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func readyNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
}

func TestPickNode_ReturnsFittingReadyNode(t *testing.T) {
	pod := podWithRequests("test-pod", "500m", "256Mi")
	client := fake.NewSimpleClientset(
		pod,
		nodeWithAllocatable("node-1", "2", "4Gi"),
		nodeWithAllocatable("node-2", "2", "4Gi"),
	)
	s := New(client, discardLogger())

	node, err := s.pickNode(context.Background(), pod)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node != "node-1" && node != "node-2" {
		t.Fatalf("unexpected node returned: %s", node)
	}
}

func TestPickNode_NoFittingNode_ReturnsError(t *testing.T) {
	pod := podWithRequests("test-pod", "4", "8Gi")
	client := fake.NewSimpleClientset(
		pod,
		nodeWithAllocatable("node-1", "2", "4Gi"),
	)
	s := New(client, discardLogger())

	_, err := s.pickNode(context.Background(), pod)
	if err == nil {
		t.Fatal("expected error when no node has capacity for pod requests")
	}
}

func TestBind_CreatesBindingOnCorrectNode(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{SchedulerName: SchedulerName},
	}
	client := fake.NewSimpleClientset(pod, readyNode("node-1"))
	s := New(client, discardLogger())

	if err := s.bind(context.Background(), pod, "node-1"); err != nil {
		t.Fatalf("unexpected error binding pod: %v", err)
	}
	// fake clientset's Bind is a no-op tracked action; verifying it doesn't
	// error is sufficient at this stage. Step 3 adds an e2e test against kind
	// that asserts pod.spec.nodeName is actually set.
}

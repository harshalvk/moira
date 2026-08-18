package scheduler

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAssumeCache_AssumeThenPodsForNode(t *testing.T) {
	c := NewAssumeCache()
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"}}

	c.Assume(pod, "node-1")

	got := c.PodsForNode("node-1")
	if len(got) != 1 || got[0].Name != "p1" {
		t.Fatalf("expected p1 in node-1's assumed pods, got %v", got)
	}

	if len(c.PodsForNode("node-2")) != 0 {
		t.Fatal("expected no assumed pods for a different node")
	}
}

func TestAssumeCache_Forget(t *testing.T) {
	c := NewAssumeCache()
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"}}

	c.Assume(pod, "node-1")
	c.Forget(pod)

	if len(c.PodsForNode("node-1")) != 0 {
		t.Fatal("expected assumed pod to be gone after Forget")
	}
}

func TestAssumeCache_ExpiredEntryIsEvicted(t *testing.T) {
	c := NewAssumeCache()
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"}}

	// Manually insert an already-expired entry rather than sleeping 30s in a test.
	c.mu.Lock()
	c.assumed[podKey(pod)] = &assumedPod{
		pod:       pod,
		node:      "node-1",
		expiresAt: time.Now().Add(-time.Second),
	}
	c.mu.Unlock()

	if len(c.PodsForNode("node-1")) != 0 {
		t.Fatal("expected expired assumed pod to be evicted on read")
	}
}

func TestAssumeCache_DifferentPodsSameNode(t *testing.T) {
	c := NewAssumeCache()
	p1 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"}}
	p2 := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p2", Namespace: "default"}}

	c.Assume(p1, "node-1")
	c.Assume(p2, "node-1")

	if len(c.PodsForNode("node-1")) != 2 {
		t.Fatal("expected both assumed pods to be tracked for the same node")
	}
}

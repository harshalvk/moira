package framework

import (
	"io"
	"log/slog"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

type alwaysReject struct{}

func (alwaysReject) Name() string { return "AlwaysReject" }
func (alwaysReject) Filter(_ *corev1.Pod, _ *NodeInfo) *Status {
	return NewStatus(Unschedulable, "rejected for test")
}

type alwaysAllow struct{}

func (alwaysAllow) Name() string                              { return "AlwaysAllow" }
func (alwaysAllow) Filter(_ *corev1.Pod, _ *NodeInfo) *Status { return nil }

type fixedScore struct{ score int64 }

func (f fixedScore) Name() string                                      { return "FixedScore" }
func (f fixedScore) Score(_ *corev1.Pod, _ *NodeInfo) (int64, *Status) { return f.score, nil }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunFilterPlugins_RejectsWhenAnyFilterFails(t *testing.T) {
	r := NewRegistry(discardLogger())
	r.RegisterFilter(alwaysAllow{})
	r.RegisterFilter(alwaysReject{})

	nodeInfos := []*NodeInfo{{Node: &corev1.Node{}}}
	feasible := r.RunFilterPlugins(&corev1.Pod{}, nodeInfos)

	if len(feasible) != 0 {
		t.Fatalf("expected no feasible nodes, got %d", len(feasible))
	}
}

func TestRunScorePlugins_WeightedSum(t *testing.T) {
	r := NewRegistry(discardLogger())
	r.RegisterScore(fixedScore{score: 10}, 2) // weight 2 -> contributes 20
	r.RegisterScore(fixedScore{score: 5}, 1)  // weight 1 -> contributes 5

	node := &corev1.Node{}
	node.Name = "n1"
	nodeInfos := []*NodeInfo{{Node: node}}

	scores := r.RunScorePlugins(&corev1.Pod{}, nodeInfos)
	if scores["n1"] != 25 {
		t.Fatalf("expected weighted sum 25, got %d", scores["n1"])
	}
}

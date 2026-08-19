package framework

import corev1 "k8s.io/api/core/v1"

// StatusCode mirrors the real scheduler-framework's status model, so the
// eventual migration is a rename, not a redesign
type StatusCode int

const (
	Success StatusCode = iota
	Unschedulable
	Error
)

type Status struct {
	Code    StatusCode
	Messaeg string
}

func NewStatus(code StatusCode, msg string) *Status {
	return &Status{Code: code, Messaeg: msg}
}

func (s *Status) IsSuccess() bool {
	return s == nil || s.Code == Success
}

// NodeInfo bundles a node with the pods currently considered "on" it -
// including assumed pods from the AssumeCache. Plugins never talk to
// the cluster directly; everything they need is in here
type NodeInfo struct {
	Node *corev1.Node
	Pods []*corev1.Pod
}

// FliterPlugin decides whether a node is feasible for a pod at all.
// Any non-success excludes the node from scoring entirely
type FilterPlugin interface {
	Name() string
	Filter(pod *corev1.Pod, nodeInfo *NodeInfo) *Status
}

// ScorePlugin ranks feasible nodes. Scores are 0-100 (MaxNodeScore),
// matching the real framework's convention, combined via weighted sum
// across all registered score plugin
type ScorePlugin interface {
	Name() string
	Score(pod *corev1.Pod, nodeInfo *NodeInfo) (int64, *Status)
}

const MaxNodeScore int64 = 100

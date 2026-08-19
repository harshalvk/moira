package nodeaffinity

import (
	"github.com/harshalvk/moira/internal/framework"
	corev1 "k8s.io/api/core/v1"
)

const Name = "NodeAffinity"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return Name }

func (p *Plugin) Filter(pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.NodeAffinity == nil {
		return nil
	}
	required := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if required == nil {
		return nil
	}

	for _, term := range required.NodeSelectorTerms {
		if matchesTerm(term, nodeInfo.Node.Labels) {
			return nil
		}
	}
	return framework.NewStatus(framework.Unschedulable, "no matching node affinity term")
}

func matchesTerm(term corev1.NodeSelectorTerm, labels map[string]string) bool {
	for _, expr := range term.MatchExpressions {
		if !matchesExpression(expr, labels) {
			return false
		}
	}
	return true
}

func matchesExpression(expr corev1.NodeSelectorRequirement, labels map[string]string) bool {
	value, exists := labels[expr.Key]
	switch expr.Operator {
	case corev1.NodeSelectorOpIn:
		return exists && contains(expr.Values, value)
	case corev1.NodeSelectorOpNotIn:
		return !exists || !contains(expr.Values, value)
	case corev1.NodeSelectorOpExists:
		return exists
	case corev1.NodeSelectorOpDoesNotExist:
		return !exists
	default:
		return false
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

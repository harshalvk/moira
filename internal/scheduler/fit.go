package scheduler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

// podRequests sums CPU and memory requests across all containers in a pod.
// Init containers are ignored for now — Step 3 keeps this to the common
// case; init-container-aware request calc is a later refinement, not core
// to proving the filtering mechanism.
func podRequests(pod *corev1.Pod) (cpu, mem resource.Quantity) {
	cpu = resource.Quantity{}
	mem = resource.Quantity{}

	for _, c := range pod.Spec.Containers {
		if c.Resources.Requests == nil {
			continue
		}
		if q, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
			cpu.Add(q)
		}
		if q, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
			mem.Add(q)
		}
	}
	return cpu, mem
}

// nodeAllocatable reads a node's allocatable CPU/memory.
func nodeAllocatable(node *corev1.Node) (cpu, mem resource.Quantity) {
	cpu = node.Status.Allocatable[corev1.ResourceCPU]
	mem = node.Status.Allocatable[corev1.ResourceMemory]
	return cpu, mem
}

// FitsNode reports whether pod's resource requests fit within node's
// allocatable capacity, minus what's already requested by podsOnNode.
//
// This does NOT yet account for pods bound but not yet visible in the
// informer cache (a real race in production schedulers) — that's handled
// properly once we add an internal cache of "assumed" pods in a later step.
// For now, podsOnNode is expected to be a fresh list query.
func FitsNode(pod *corev1.Pod, node *corev1.Node, podsOnNode []*corev1.Pod) bool {
	wantCPU, wantMem := podRequests(pod)
	availCPU, availMem := nodeAllocatable(node)

	for _, p := range podsOnNode {
		usedCPU, usedMem := podRequests(p)
		availCPU.Sub(usedCPU)
		availMem.Sub(usedMem)
	}

	return wantCPU.Cmp(availCPU) <= 0 && wantMem.Cmp(availMem) <= 0
}

// TolerationsSatisfyTaints reports whether pod's tolerations cover every
// NoSchedule/NoExecute taint on node. PreferNoSchedule taints are advisory
// (used in scoring, not filtering) and are skipped here.
func TolerationsSatisfyTaints(pod *corev1.Pod, node *corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectPreferNoSchedule {
			continue
		}
		if !tolerates(pod.Spec.Tolerations, taint) {
			return false
		}
	}
	return true
}

func tolerates(tolerations []corev1.Toleration, taint corev1.Taint) bool {
	for _, t := range tolerations {
		if t.ToleratesTaint(klog.Background(), &taint, false) {
			return true
		}
	}
	return false
}

// MatchesNodeAffinity reports whether node satisfies pod's required node
// affinity (spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution).
// Preferred affinity is scoring, not filtering — deferred to Phase 1 scoring work.
// A pod with no required affinity trivially matches every node.
func MatchesNodeAffinity(pod *corev1.Pod, node *corev1.Node) bool {
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.NodeAffinity == nil {
		return true
	}
	required := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if required == nil {
		return true
	}

	// A pod matches if ANY term in NodeSelectorTerms matches (OR semantics).
	for _, term := range required.NodeSelectorTerms {
		if matchesTerm(term, node) {
			return true
		}
	}
	return false
}

// matchesTerm requires ALL expressions within a term to match (AND semantics),
// per the NodeSelectorTerm spec.
func matchesTerm(term corev1.NodeSelectorTerm, node *corev1.Node) bool {
	for _, expr := range term.MatchExpressions {
		if !matchesExpression(expr, node.Labels) {
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
		// Gt/Lt operators deferred — rare in practice, adding when a real
		// use case needs them rather than speculatively now.
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

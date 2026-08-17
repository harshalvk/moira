package scheduler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

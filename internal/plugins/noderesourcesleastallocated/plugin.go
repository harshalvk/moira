// Package noderesourcesleastallocated scores nodes higher when they have
// MORE free capacity — a spread strategy. The complementary bin-packing
// strategy (score higher for LESS free capacity, packing pods tightly)
// lands next as a second, alternately-weighted plugin — see roadmap
// Phase 1 item 6.
package noderesourcesleastallocated

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/harshalvk/moira/internal/framework"
)

const Name = "NodeResourcesLeastAllocated"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return Name }

func (p *Plugin) Score(pod *corev1.Pod, nodeInfo *framework.NodeInfo) (int64, *framework.Status) {
	totalCPU := nodeInfo.Node.Status.Allocatable[corev1.ResourceCPU]
	totalMem := nodeInfo.Node.Status.Allocatable[corev1.ResourceMemory]

	usedCPU := resource.Quantity{}
	usedMem := resource.Quantity{}
	for _, existing := range nodeInfo.Pods {
		for _, c := range existing.Spec.Containers {
			if c.Resources.Requests == nil {
				continue
			}
			if q, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
				usedCPU.Add(q)
			}
			if q, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
				usedMem.Add(q)
			}
		}
	}

	cpuFrac := freeFraction(totalCPU, usedCPU)
	memFrac := freeFraction(totalMem, usedMem)

	score := int64((cpuFrac + memFrac) / 2 * float64(framework.MaxNodeScore))
	return score, nil
}

// freeFraction returns the fraction of total that's still free, as 0.0-1.0.
// Guards against divide-by-zero on a node with zero allocatable of a resource.
func freeFraction(total, used resource.Quantity) float64 {
	totalVal := total.AsApproximateFloat64()
	if totalVal <= 0 {
		return 0
	}
	usedVal := used.AsApproximateFloat64()
	free := (totalVal - usedVal) / totalVal
	if free < 0 {
		return 0
	}
	if free > 1 {
		return 1
	}
	return free
}

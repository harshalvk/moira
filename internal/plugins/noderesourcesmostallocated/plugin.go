// Package noderesourcemostallocated scores nodes higher when they have
// LESS free capacity - a bin-packing strategy. Complements noderesourecesleastallocated (spread);
// exactly one should be active at a time
// (see scheduler.Config.Strategy), no both simultaneously
package noderesourcesmostallocated

import (
	"github.com/harshalvk/moira/internal/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const Name = "NodeResourcesMostAllocated"

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

	cpuFrac := usedFraction(totalCPU, usedCPU)
	memFrac := usedFraction(totalMem, usedMem)

	score := int64((cpuFrac + memFrac) / 2 * float64(framework.MaxNodeScore))
	return score, nil
}

// usedFraction returns the fraction of total already used, as 0.0-1.0
// the invers of noderesourcesleastallocated's freeFraction. Same
// divide-by-zero and clamping gurads
func usedFraction(total, used resource.Quantity) float64 {
	totalVal := total.AsApproximateFloat64()
	if totalVal <= 0 {
		return 0
	}
	usedVal := used.AsApproximateFloat64()
	frac := usedVal / totalVal
	if frac < 0 {
		return 0
	}
	if frac > 1 {
		return 1
	}
	return frac
}

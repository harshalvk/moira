// Package noderesourcesfit wraps the existing FitNode logic as a
// framework.FilterPlugin
package noderesourcesfit

import (
	"github.com/harshalvk/moira/internal/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const Name = "NodeResourceFit"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return Name }

func (p *Plugin) Filter(pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	wantCPU, wantMem := podRequests(pod)
	availCPU := nodeInfo.Node.Status.Allocatable[corev1.ResourceCPU]
	availMem := nodeInfo.Node.Status.Allocatable[corev1.ResourceMemory]

	for _, existing := range nodeInfo.Pods {
		usedCPU, usedMem := podRequests(existing)
		availCPU.Sub(usedCPU)
		availMem.Sub(usedMem)
	}

	if wantCPU.Cmp(availCPU) > 0 {
		return framework.NewStatus(framework.Unschedulable, "insufficient cpu")
	}
	if wantMem.Cmp(availMem) > 0 {
		return framework.NewStatus(framework.Unschedulable, "insufficient memory")
	}
	return nil
}

func podRequests(pod *corev1.Pod) (cpu, mem resource.Quantity) {
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

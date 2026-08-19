package tainttoleration

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/harshalvk/moira/internal/framework"
)

const Name = "TaintToleration"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return Name }

func (p *Plugin) Filter(pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	for _, taint := range nodeInfo.Node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectPreferNoSchedule {
			continue
		}
		if !tolerates(pod.Spec.Tolerations, taint) {
			return framework.NewStatus(framework.Unschedulable, "untolerated taint: "+taint.Key)
		}
	}
	return nil
}

func tolerates(tolerations []corev1.Toleration, taint corev1.Taint) bool {
	for _, t := range tolerations {
		if t.ToleratesTaint(klog.Background(), &taint, false) {
			return true
		}
	}
	return false
}

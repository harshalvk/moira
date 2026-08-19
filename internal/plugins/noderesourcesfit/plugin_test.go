package noderesourcesfit

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harshalvk/moira/internal/framework"
)

func podWithRequests(cpu, mem string) *corev1.Pod {
	return &corev1.Pod{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(cpu),
					corev1.ResourceMemory: resource.MustParse(mem),
				},
			},
		}}},
	}
}

func nodeInfo(cpu, mem string, pods ...*corev1.Pod) *framework.NodeInfo {
	return &framework.NodeInfo{
		Node: &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "n1"},
			Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			}},
		},
		Pods: pods,
	}
}

func TestFilter_FitsEmptyNode(t *testing.T) {
	p := New()
	status := p.Filter(podWithRequests("500m", "256Mi"), nodeInfo("2", "4Gi"))
	if !status.IsSuccess() {
		t.Fatalf("expected fit, got status: %+v", status)
	}
}

func TestFilter_ExceedsCapacity(t *testing.T) {
	p := New()
	status := p.Filter(podWithRequests("4", "8Gi"), nodeInfo("2", "4Gi"))
	if status.IsSuccess() {
		t.Fatal("expected filter to reject over-capacity request")
	}
}

func TestFilter_AccountsForExistingPods(t *testing.T) {
	p := New()
	existing := podWithRequests("1500m", "3Gi")
	status := p.Filter(podWithRequests("1", "1Gi"), nodeInfo("2", "4Gi", existing))
	if status.IsSuccess() {
		t.Fatal("expected filter to reject: existing pod already consumed most capacity")
	}
}

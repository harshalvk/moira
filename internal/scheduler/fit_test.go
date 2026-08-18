package scheduler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podWithRequests(name, cpu, mem string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpu),
							corev1.ResourceMemory: resource.MustParse(mem),
						},
					},
				},
			},
		},
	}
}

func nodeWithAllocatable(name, cpu, mem string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
}

func TestFitsNode_EmptyNodeHasCapacity(t *testing.T) {
	pod := podWithRequests("p1", "500m", "256Mi")
	node := nodeWithAllocatable("n1", "2", "4Gi")

	if !FitsNode(pod, node, nil) {
		t.Fatal("expected pod to fit on empty node with ample capacity")
	}
}

func TestFitsNode_RequestExceedsAllocatable(t *testing.T) {
	pod := podWithRequests("p1", "4", "8Gi")
	node := nodeWithAllocatable("n1", "2", "4Gi")

	if FitsNode(pod, node, nil) {
		t.Fatal("expected pod not to fit: request exceeds total allocatable")
	}
}

func TestFitsNode_AccountsForExistingPods(t *testing.T) {
	pod := podWithRequests("p2", "1", "1Gi")
	node := nodeWithAllocatable("n1", "2", "4Gi")
	existing := []*corev1.Pod{
		podWithRequests("p1", "1500m", "3Gi"),
	}

	if FitsNode(pod, node, existing) {
		t.Fatal("expected pod not to fit: existing pod already consumed most capacity")
	}
}

func TestFitsNode_ExactFit(t *testing.T) {
	pod := podWithRequests("p1", "2", "4Gi")
	node := nodeWithAllocatable("n1", "2", "4Gi")

	if !FitsNode(pod, node, nil) {
		t.Fatal("expected exact-capacity match to fit")
	}
}

func TestTolerationsSatisfyTaints_NoTaints(t *testing.T) {
	pod := &corev1.Pod{}
	node := &corev1.Node{}
	if !TolerationsSatisfyTaints(pod, node) {
		t.Fatal("expected pod to satisfy node with no taints")
	}
}

func TestTolerationsSatisfyTaints_UntoleratedNoScheduleTaint(t *testing.T) {
	pod := &corev1.Pod{}
	node := &corev1.Node{
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{Key: "gpu", Value: "true", Effect: corev1.TaintEffectNoSchedule},
			},
		},
	}
	if TolerationsSatisfyTaints(pod, node) {
		t.Fatal("expected pod without toleration to be blocked by NoSchedule taint")
	}
}

func TestTolerationsSatisfyTaints_MatchingToleration(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Tolerations: []corev1.Toleration{
				{Key: "gpu", Operator: corev1.TolerationOpEqual, Value: "true", Effect: corev1.TaintEffectNoSchedule},
			},
		},
	}
	node := &corev1.Node{
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{Key: "gpu", Value: "true", Effect: corev1.TaintEffectNoSchedule},
			},
		},
	}
	if !TolerationsSatisfyTaints(pod, node) {
		t.Fatal("expected matching toleration to satisfy taint")
	}
}

func TestTolerationsSatisfyTaints_PreferNoScheduleIsAdvisoryOnly(t *testing.T) {
	pod := &corev1.Pod{}
	node := &corev1.Node{
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{Key: "low-priority", Effect: corev1.TaintEffectPreferNoSchedule},
			},
		},
	}
	if !TolerationsSatisfyTaints(pod, node) {
		t.Fatal("expected PreferNoSchedule taint to not block filtering")
	}
}

func TestMatchesNodeAffinity_NoAffinitySpecified(t *testing.T) {
	pod := &corev1.Pod{}
	node := &corev1.Node{}
	if !MatchesNodeAffinity(pod, node) {
		t.Fatal("expected pod with no affinity to match any node")
	}
}

func TestMatchesNodeAffinity_RequiredLabelMatch(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{Key: "disktype", Operator: corev1.NodeSelectorOpIn, Values: []string{"ssd"}},
								},
							},
						},
					},
				},
			},
		},
	}
	matching := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"disktype": "ssd"}}}
	nonMatching := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"disktype": "hdd"}}}

	if !MatchesNodeAffinity(pod, matching) {
		t.Fatal("expected node with disktype=ssd to match")
	}
	if MatchesNodeAffinity(pod, nonMatching) {
		t.Fatal("expected node with disktype=hdd not to match")
	}
}

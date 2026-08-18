package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// SchedulerName is the value pods must set in spec.schedulerName
// to be picked up by moira instead of the default kube-scheduler.
const SchedulerName = "moira"

// Scheduler watches for unscheduled pods and binds them to a node.
// This is intentionally the simplest possible implementation:
// no filtering, no scoring — just proves the watch->decide->bind
// loop end to end. Filtering/scoring arrive in Step 3.
type Scheduler struct {
	client kubernetes.Interface
	logger *slog.Logger
	cache  *AssumeCache
}

func New(client kubernetes.Interface, logger *slog.Logger) *Scheduler {
	return &Scheduler{client: client, logger: logger, cache: NewAssumeCache()}
}

// Run starts the informer loop and blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	watchList := cache.NewListWatchFromClient(
		s.client.CoreV1().RESTClient(),
		"pods",
		corev1.NamespaceAll,
		fields.OneTermEqualSelector("spec.nodeName", ""),
	)

	_, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: watchList,
		ObjectType:    &corev1.Pod{},
		ResyncPeriod:  0,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					return
				}
				s.handlePod(ctx, pod)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				pod, ok := newObj.(*corev1.Pod)
				if !ok {
					return
				}
				if pod.Spec.NodeName != "" {
					// Real state now confirms the binding - safe to drop
					// the assumption, real list data covers it from here
					s.cache.Forget(pod)
				}
			},
		},
	})

	controller.Run(ctx.Done())
	return nil
}

func (s *Scheduler) handlePod(ctx context.Context, pod *corev1.Pod) {
	if pod.Spec.SchedulerName != SchedulerName {
		return
	}
	if pod.Spec.NodeName != "" {
		return
	}

	node, err := s.pickNode(ctx, pod)
	if err != nil {
		s.logger.Error("no node fits pod", "pod", pod.Name, "err", err)
		return
	}

	if err := s.bind(ctx, pod, node); err != nil {
		s.logger.Error("bind failed", "pod", pod.Name, "node", node, "err", err)
		return
	}

	s.cache.Assume(pod, node)
	s.logger.Info("scheduled pod", "pod", pod.Name, "node", node)
}

// pickNode is deliberately naive: uniform random choice among all
// Ready nodes. No resource-fit check yet — that's Step 3.
func (s *Scheduler) pickNode(ctx context.Context, pod *corev1.Pod) (string, error) {
	nodes, err := s.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("listing nodes: %w", err)
	}

	allPods, err := s.client.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("listing pods: %w", err)
	}

	podsByNode := make(map[string][]*corev1.Pod)
	for i := range allPods.Items {
		p := &allPods.Items[i]
		if p.Spec.NodeName != "" {
			podsByNode[p.Spec.NodeName] = append(podsByNode[p.Spec.NodeName], p)
		}
	}

	// Merge in assumed-but-not-yet-visible pods so a second pod scheduled
	// in the same tick doesn't see stale, already-spoken-for capacity
	for i := range nodes.Items {
		nodeName := nodes.Items[i].Name
		podsByNode[nodeName] = append(podsByNode[nodeName], s.cache.PodsForNode(nodeName)...)
	}

	var fitting []string
	for i := range nodes.Items {
		node := &nodes.Items[i]
		if !isReady(node) {
			continue
		}
		if !TolerationsSatisfyTaints(pod, node) {
			continue
		}
		if !MatchesNodeAffinity(pod, node) {
			continue
		}
		if FitsNode(pod, node, podsByNode[node.Name]) {
			fitting = append(fitting, node.Name)
		}
	}

	if len(fitting) == 0 {
		return "", fmt.Errorf("no node fits pod requests")
	}

	// #nosec G404 -- non-cryptographic use: distributing scheduling load
	// across fitting nodes, not generating security-sensitive values.
	return fitting[rand.Intn(len(fitting))], nil
}

func isReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (s *Scheduler) bind(ctx context.Context, pod *corev1.Pod, node string) error {
	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			UID:       pod.UID,
		},
		Target: corev1.ObjectReference{
			Kind: "Node",
			Name: node,
		},
	}
	return s.client.CoreV1().Pods(pod.Namespace).Bind(ctx, binding, metav1.CreateOptions{})
}

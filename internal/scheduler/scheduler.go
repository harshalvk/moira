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
}

func New(client kubernetes.Interface, logger *slog.Logger) *Scheduler {
	return &Scheduler{client: client, logger: logger}
}

// Run starts the informer loop and blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	watchList := cache.NewListWatchFromClient(
		s.client.CoreV1().RESTClient(),
		"pods",
		corev1.NamespaceAll,
		fields.OneTermEqualSelector("spec.nodeName", ""),
	)

	_, controller := cache.NewInformer(
		watchList,
		&corev1.Pod{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, ok := obj.(*corev1.Pod)
				if !ok {
					return
				}
				s.handlePod(ctx, pod)
			},
		},
	)

	controller.Run(ctx.Done())
	return nil
}

func (s *Scheduler) handlePod(ctx context.Context, pod *corev1.Pod) {
	if pod.Spec.SchedulerName != SchedulerName {
		return // not ours to schedule
	}
	if pod.Spec.NodeName != "" {
		return // already scheduled
	}

	node, err := s.pickNode(ctx)
	if err != nil {
		s.logger.Error("no node available", "pod", pod.Name, "err", err)
		return
	}

	if err := s.bind(ctx, pod, node); err != nil {
		s.logger.Error("bind failed", "pod", pod.Name, "node", node, "err", err)
		return
	}

	s.logger.Info("scheduled pod", "pod", pod.Name, "node", node)
}

// pickNode is deliberately naive: uniform random choice among all
// Ready nodes. No resource-fit check yet — that's Step 3.
func (s *Scheduler) pickNode(ctx context.Context) (string, error) {
	nodes, err := s.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("listing nodes: %w", err)
	}

	var ready []string
	for _, n := range nodes.Items {
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ready = append(ready, n.Name)
			}
		}
	}

	if len(ready) == 0 {
		return "", fmt.Errorf("no ready nodes")
	}

	return ready[rand.Intn(len(ready))], nil
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

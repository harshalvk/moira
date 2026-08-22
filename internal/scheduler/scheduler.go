package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/harshalvk/moira/internal/framework"
	"github.com/harshalvk/moira/internal/metrics"
	"github.com/harshalvk/moira/internal/plugins/nodeaffinity"
	"github.com/harshalvk/moira/internal/plugins/noderesourcesfit"
	"github.com/harshalvk/moira/internal/plugins/noderesourcesleastallocated"
	"github.com/harshalvk/moira/internal/plugins/noderesourcesmostallocated"
	"github.com/harshalvk/moira/internal/plugins/tainttoleration"
)

const SchedulerName = "moira"

type Scheduler struct {
	client   kubernetes.Interface
	logger   *slog.Logger
	cache    *AssumeCache
	registry *framework.Registry
}

func New(client kubernetes.Interface, logger *slog.Logger, cfg Config) (*Scheduler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scheduler config: %w", err)
	}

	registry := framework.NewRegistry(logger)
	registry.RegisterFilter(noderesourcesfit.New())
	registry.RegisterFilter(tainttoleration.New())
	registry.RegisterFilter(nodeaffinity.New())

	switch cfg.Strategy {
	case StrategyPack:
		registry.RegisterScore(noderesourcesmostallocated.New(), 1)
	default:
		registry.RegisterScore(noderesourcesleastallocated.New(), 1)
	}

	logger.Info("scheduler configured", "strategy", cfg.Strategy)

	return &Scheduler{
		client:   client,
		logger:   logger,
		cache:    NewAssumeCache(),
		registry: registry,
	}, nil
}

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
				if pod, ok := obj.(*corev1.Pod); ok {
					s.handlePod(ctx, pod)
				}
			},
			UpdateFunc: func(_, newObj interface{}) {
				pod, ok := newObj.(*corev1.Pod)
				if ok && pod.Spec.NodeName != "" {
					s.cache.Forget(pod)
				}
			},
		},
	})

	controller.Run(ctx.Done())
	return nil
}

func (s *Scheduler) handlePod(ctx context.Context, pod *corev1.Pod) {
	if pod.Spec.SchedulerName != SchedulerName || pod.Spec.NodeName != "" {
		return
	}

	start := time.Now()

	node, err := s.pickNode(ctx, pod)
	if err != nil {
		metrics.SchedulingAttempts.WithLabelValues("failed").Inc()
		s.logger.Error("no node fits pod", "pod", pod.Name, "err", err)
		return
	}

	if err := s.bind(ctx, pod, node); err != nil {
		metrics.SchedulingAttempts.WithLabelValues("failed").Inc()
		s.logger.Error("bind failed", "pod", pod.Name, "node", node, "err", err)
		return
	}

	metrics.SchedulingLatency.Observe(float64(time.Since(start).Seconds()))
	metrics.SchedulingAttempts.WithLabelValues("scheduled").Inc()

	s.cache.Assume(pod, node)
	s.logger.Info("scheduled pod", "pod", pod.Name, "node", node)
}

func (s *Scheduler) pickNode(ctx context.Context, pod *corev1.Pod) (string, error) {
	nodeInfos, err := s.buildNodeInfos(ctx)
	if err != nil {
		return "", err
	}

	feasible := s.registry.RunFilterPlugins(pod, nodeInfos)
	if len(feasible) == 0 {
		return "", fmt.Errorf("no node passes filters")
	}

	scores := s.registry.RunScorePlugins(pod, feasible)
	return highestScored(scores), nil
}

// buildNodeInfos assembles per-node state: Ready nodes only, real pods from
// a fresh list, plus assumed-but-unconfirmed pods from the AssumeCache
// (closing the race from ADR 0005).
func (s *Scheduler) buildNodeInfos(ctx context.Context) ([]*framework.NodeInfo, error) {
	nodes, err := s.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	allPods, err := s.client.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	podsByNode := make(map[string][]*corev1.Pod)
	for i := range allPods.Items {
		p := &allPods.Items[i]
		if p.Spec.NodeName != "" {
			podsByNode[p.Spec.NodeName] = append(podsByNode[p.Spec.NodeName], p)
		}
	}

	var infos []*framework.NodeInfo
	for i := range nodes.Items {
		node := &nodes.Items[i]
		if !isReady(node) {
			continue
		}
		pods := append(podsByNode[node.Name], s.cache.PodsForNode(node.Name)...)
		infos = append(infos, &framework.NodeInfo{Node: node, Pods: pods})
	}
	return infos, nil
}

// highestScored picks the top-scoring node, breaking ties uniformly at
// random — matches the real framework's tie-breaking behavior, avoids
// always favoring whichever node happens to sort first.
func highestScored(scores map[string]int64) string {
	var best int64 = -1
	var winners []string
	for node, score := range scores {
		switch {
		case score > best:
			best = score
			winners = []string{node}
		case score == best:
			winners = append(winners, node)
		}
	}
	// #nosec G404 -- random selection is used only for scheduler load distribution, not security.
	return winners[rand.Intn(len(winners))]
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
		ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace, UID: pod.UID},
		Target:     corev1.ObjectReference{Kind: "Node", Name: node},
	}
	return s.client.CoreV1().Pods(pod.Namespace).Bind(ctx, binding, metav1.CreateOptions{})
}

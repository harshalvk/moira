package leaderelection

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/harshalvk/moira/internal/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// Run blocks until ctx is cancelled. While this process holds the lease,
// onStartedLeading runs with a context that is itself cancelled the moment
// leadership is lost — callers should treat that cancellation as "stop
// scheduling immediately," not just "shutting down eventually."
//
// If cfg.Enabled is false, onStartedLeading runs immediately with no lease
// involved at all — explicitly unsafe with >1 replica, intended only for
// local dev (make dev) where a single process is always the only instance.
func Run(ctx context.Context, client kubernetes.Interface, cfg Config, logger *slog.Logger, onStartedLeading func(context.Context)) error {
	if !cfg.Enabled {
		logger.Warn("leader election disabled — running as sole scheduler; unsafe with more than one replica")
		onStartedLeading(ctx)
		return nil
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid leader election config: %w", err)
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.LeaseName,
			Namespace: cfg.Namespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.Identity,
		},
	}

	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   cfg.LeaseDuration,
		RenewDeadline:   cfg.RenewDeadline,
		RetryPeriod:     cfg.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leadCtx context.Context) {
				logger.Info("acquired leadership", "identity", cfg.Identity)
				metrics.IsLeader.Set(1)
				onStartedLeading(leadCtx)
			},
			OnStoppedLeading: func() {
				logger.Warn("lost leadership", "identity", cfg.Identity)
				metrics.IsLeader.Set(0)
			},
			OnNewLeader: func(identity string) {
				if identity != cfg.Identity {
					logger.Info("new leader observed", "leader", identity)
				}
			},
		},
	})
	if err != nil {
		// Deliberately NOT leaderelection.RunOrDie — that panics on setup
		// failure, which is the wrong failure mode for a long-running
		// service. Returning an error lets main.go log and exit cleanly.
		return fmt.Errorf("creating leader elector: %w", err)
	}

	le.Run(ctx)
	return nil
}

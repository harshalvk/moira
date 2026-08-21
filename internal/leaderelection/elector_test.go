package leaderelection

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// A single replica against a fake clientset has no contention, so it
// should acquire the lease and invoke the callback well within the timeout.
// Short lease timings keep this test fast without sleeping for production
// defaults (15s).
func TestRun_SoleReplicaAcquiresLeadershipAndInvokesCallback(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{
		Enabled:       true,
		LeaseName:     "test-lease",
		Namespace:     "default",
		Identity:      "replica-1",
		LeaseDuration: 2 * time.Second,
		RenewDeadline: 1 * time.Second,
		RetryPeriod:   250 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var mu sync.Mutex
	started := false

	done := make(chan struct{})
	go func() {
		_ = Run(ctx, client, cfg, discardLogger(), func(leadCtx context.Context) {
			mu.Lock()
			started = true
			mu.Unlock()
			<-leadCtx.Done()
		})
		close(done)
	}()

	<-done

	mu.Lock()
	defer mu.Unlock()
	if !started {
		t.Fatal("expected sole replica to acquire leadership and invoke callback")
	}
}

func TestRun_DisabledSkipsLeaseEntirelyAndRunsImmediately(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{Enabled: false}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	invoked := make(chan struct{})
	err := Run(ctx, client, cfg, discardLogger(), func(_ context.Context) {
		close(invoked)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-invoked:
	default:
		t.Fatal("expected callback to run immediately when leader election disabled")
	}
}

func TestRun_InvalidConfigReturnsError(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{Enabled: true} // missing required fields

	err := Run(context.Background(), client, cfg, discardLogger(), func(_ context.Context) {
		t.Fatal("callback should not run with invalid config")
	})
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

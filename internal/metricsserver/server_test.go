package metricsserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRun_ServesMetricsAndHealthz(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Run(ctx, "127.0.0.1:18080", discardLogger()) }()

	// brief wait for the listener to come up; retried rather than a fixed
	// sleep so this isn't flaky on slower CI runners
	var resp *http.Response
	var err error
	for i := 0; i < 20; i++ {
		resp, err = http.Get("http://127.0.0.1:18080/healthz")
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("healthz never became reachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	metricsResp, err := http.Get("http://127.0.0.1:18080/metrics")
	if err != nil {
		t.Fatalf("unexpected error hitting /metrics: %v", err)
	}
	defer metricsResp.Body.Close()
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", metricsResp.StatusCode)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("expected clean shutdown, got: %v", err)
	}
}

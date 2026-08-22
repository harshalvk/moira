// Package metricsserver runs a minimal http server exposing /metrics
// (Prometheus scrape endpoint) and /healthz. Deliberately its own tiny
// server rather than folder into anything else - metrics/health need to
// stay reachable even if scheduling logic is stuckt, since a scrape failure
// is itself a signal something's wrong
package metricsserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run(ctx context.Context, addr string, logger *slog.Logger) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("metrics server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancle()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return fmt.Errorf("metrics server failed: %w", err)
	}
}

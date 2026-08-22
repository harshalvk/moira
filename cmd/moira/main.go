package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/harshalvk/moira/internal/leaderelection"
	"github.com/harshalvk/moira/internal/metricsserver"
	"github.com/harshalvk/moira/internal/scheduler"
	"github.com/harshalvk/moira/internal/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting moira", "version", version.Version)

	kubeconfig, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		logger.Error("failed to load kubeconfig", "err", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		logger.Error("failed to create client", "err", err)
		os.Exit(1)
	}

	schedCfg := scheduler.DefaultConfig()
	if strategy := os.Getenv("MOIRA_STRATEGY"); strategy != "" {
		schedCfg.Strategy = scheduler.Strategy(strategy)
	}

	s, err := scheduler.New(client, logger, schedCfg)
	if err != nil {
		logger.Error("failed to construct scheduler", "err", err)
		os.Exit(1)
	}

	identity := os.Getenv("POD_NAME")
	if identity == "" {
		identity, _ = os.Hostname()
	}

	leCfg := leaderelection.DefaultConfig(identity)
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		leCfg.Namespace = ns
	}
	if os.Getenv("MOIRA_LEADER_ELECTION_DISABLED") == "true" {
		leCfg.Enabled = false
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	metricsAddr := os.Getenv("MOIRA_METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":8080"
	}

	go func() {
		if err := metricsserver.Run(ctx, metricsAddr, logger); err != nil {
			logger.Error("metrics server error", "err", err)
		}
	}()

	err = leaderelection.Run(ctx, client, leCfg, logger, func(leadCtx context.Context) {
		if runErr := s.Run(leadCtx); runErr != nil {
			logger.Error("scheduler exited with error", "err", runErr)
		}
	})
	if err != nil {
		logger.Error("leader election failed", "err", err)
		os.Exit(1)
	}
}

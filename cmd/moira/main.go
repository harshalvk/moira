package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/harshalvk/moira/internal/scheduler"
	"github.com/harshalvk/moira/internal/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting moira", "version", version.Version)

	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		logger.Error("failed to load kubeconfig", "err", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error("failed to create client", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s := scheduler.New(client, logger)
	if err := s.Run(ctx); err != nil {
		logger.Error("scheduler exited with error", "err", err)
		os.Exit(1)
	}
}

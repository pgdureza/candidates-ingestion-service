package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/di"
	"github.com/candidate-ingestion/service/internal/infra/logger"
)

func main() {
	cfg := config.Load()
	logger := logger.New(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker, err := di.NewWorker(ctx, cfg)
	if err != nil {
		logger.Fatalf("Failed to build worker: %v", err)
	}
	defer worker.Database.Close()
	defer worker.PubSub.Close()

	// Start message processor
	go func() {
		if err := worker.PubSubPool.Start(ctx, cfg.GCPProject, cfg.PubSubTopic); err != nil {
			logger.WithError(err).Error("Worker pool error")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received")
	worker.PubSubPool.Stop()
	cancel()

	logger.Info("Shutdown complete")
}

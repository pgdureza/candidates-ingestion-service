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
	log := logger.New(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container, err := di.NewWorker(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to build worker: %v", err)
	}
	defer container.Database.Close()
	defer container.PubSub.Close()

	go func() {
		if err := container.Pool.Start(ctx, cfg.GCPProject, cfg.PubSubTopic); err != nil {
			log.WithError(err).Error("Worker pool error")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Info("Shutdown signal received")
	container.Pool.Stop()
	cancel()

	log.Info("Shutdown complete")
}

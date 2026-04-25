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

	poller, err := di.NewPoller(ctx, cfg)
	if err != nil {
		logger.Fatalf("Failed to build poller: %v", err)
	}
	defer poller.Database.Close()

	// Start poller
	go poller.Poller.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received")
	poller.Poller.Stop()
	cancel()

	logger.Info("Shutdown complete")
}

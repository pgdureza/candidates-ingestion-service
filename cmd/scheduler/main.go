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

	scheduler, err := di.NewScheduler(ctx, cfg)
	if err != nil {
		logger.Fatalf("Failed to build Scheduler: %v", err)
	}

	defer scheduler.Database.Close()

	// Register cleanup job
	entryID, err := scheduler.Jobs.AddFunc(cfg.Outbox.CleanupSchedule, func() {
		logger.Info("Cleanup job started")
		if err := scheduler.Cleaner.Execute(ctx); err != nil {
			logger.WithError(err).Error("Cleanup job failed")
		}
	})
	if err != nil {
		logger.Fatalf("Failed to register cleanup job: %v", err)
	}
	logger.WithField("entry_id", entryID).Info("Cleanup job registered")

	scheduler.Jobs.Start()
	logger.Info("Scheduler started")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received")
	ctx = scheduler.Jobs.Stop()
	<-ctx.Done()

	logger.Info("Scheduler shutdown complete")
}

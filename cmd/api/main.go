package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/di"
	"github.com/candidate-ingestion/service/internal/infra/logger"
)

func main() {
	cfg := config.Load()
	logger := logger.New(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api, err := di.NewAPI(ctx, cfg)
	if err != nil {
		logger.Fatalf("Failed to build API: %v", err)
	}
	defer api.Database.Close()
	defer api.PubSub.Close()

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      api.Router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Shutdown error")
		}
		cancel()
	}()

	logger.WithField("port", cfg.APIPort).Info("Starting API")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Server error: %v", err)
	}

	logger.Info("Shutdown complete")
}

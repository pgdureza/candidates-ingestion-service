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
	"github.com/candidate-ingestion/service/internal/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container, err := di.BuildAPI(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to build API: %v", err)
	}
	defer container.Database.Close()
	defer container.PubSub.Close()

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      container.App.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.WithError(err).Error("Shutdown error")
		}
		cancel()
	}()

	log.WithField("port", cfg.APIPort).Info("Starting API")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Info("Shutdown complete")
}
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/di"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container, err := di.BuildWorker(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to build worker: %v", err)
	}
	defer container.Database.Close()
	defer container.PubSub.Close()

	go func() {
		if err := container.Pool.Start(ctx, cfg.GCPProject, cfg.PubSubTopic); err != nil {
			log.Printf("Worker pool error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received")
	container.Pool.Stop()
	cancel()

	log.Println("Shutdown complete")
}

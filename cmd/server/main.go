package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/candidate-ingestion/service/internal/app"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/candidate-ingestion/service/internal/worker"
)

func main() {
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Config
	apiPort := getEnv("API_PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://candidate:password@localhost:5432/candidates?sslmode=disable")
	pubsubProject := getEnv("GCP_PROJECT", "test-project")
	pubsubTopic := getEnv("PUBSUB_TOPIC", "candidate-applications")
	workerCount := 10
	workerTimeout := 30 * time.Second

	// DB
	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer database.Close()

	// Pub/Sub
	ps, err := pubsub.New(ctx, pubsubProject)
	if err != nil {
		log.Fatalf("PubSub init failed: %v", err)
	}
	defer ps.Close()

	// App
	application := app.New(database, ps, pubsubTopic)

	// Worker pool
	wp := worker.NewPool(workerCount, workerTimeout, database, ps)
	go func() {
		if err := wp.Start(ctx, pubsubProject, pubsubTopic); err != nil {
			log.Printf("Worker pool error: %v", err)
		}
	}()

	// HTTP server
	srv := &http.Server{
		Addr:         ":" + apiPort,
		Handler:      application.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
		wp.Stop()
		cancel()
	}()

	log.Printf("Starting API on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Shutdown complete")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

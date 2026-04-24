package di

import (
	"context"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/logger"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/candidate-ingestion/service/internal/infra/worker"
)

type WorkerContainer struct {
	Pool     *worker.Pool
	Database *db.DB
	PubSub   *pubsub.Client
	Config   *config.Config
}

func NewWorker(ctx context.Context, cfg *config.Config) (*WorkerContainer, error) {
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.New(ctx, cfg.GCPProject)
	if err != nil {
		database.Close()
		return nil, err
	}

	log := logger.New(cfg.LogLevel)
	pool := worker.NewPool(cfg.WorkerCount, cfg.WorkerTimeout, database, ps, log)

	return &WorkerContainer{
		Pool:     pool,
		Database: database,
		PubSub:   ps,
		Config:   cfg,
	}, nil
}

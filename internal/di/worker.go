package di

import (
	"context"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/infra/logger"
	"github.com/candidate-ingestion/service/internal/infra/postgres"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/candidate-ingestion/service/internal/usecase/candidate/processing"
	"github.com/candidate-ingestion/service/internal/usecase/unrealiable"
)

type Worker struct {
	Database   *postgres.DB
	PubSubPool *pubsub.PubSubPool
	PubSub     *pubsub.Client
	Config     *config.Config
}

func NewWorker(ctx context.Context, cfg *config.Config) (*Worker, error) {
	logger := logger.New(cfg.LogLevel)

	database, err := postgres.New(cfg.DatabaseURL, logger)
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.New(ctx, cfg.GCPProject)
	if err != nil {
		database.Close()
		return nil, err
	}

	notifier := unrealiable.NewUnreliableNotifier()
	candidateProcessor := processing.NewCandidateProcesor(database, logger, notifier, cfg.OutboxBatchSize)
	pubSubPool := pubsub.NewPubSubPool(candidateProcessor, cfg.WorkerCount, cfg.WorkerTimeout, ps, logger)

	return &Worker{
		PubSubPool: pubSubPool,
		Database:   database,
		PubSub:     ps,
		Config:     cfg,
	}, nil
}

package di

import (
	"context"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/infra/logger"
	"github.com/candidate-ingestion/service/internal/infra/poller"
	"github.com/candidate-ingestion/service/internal/infra/postgres"
	"github.com/candidate-ingestion/service/internal/usecase/candidate/processing"
	"github.com/candidate-ingestion/service/internal/usecase/unrealiable"
)

type OutboxPoller struct {
	Database *postgres.DB
	Poller   *poller.Poller
	Config   *config.Config
}

func NewPoller(ctx context.Context, cfg *config.Config) (*OutboxPoller, error) {
	logger := logger.New(cfg.LogLevel)

	database, err := postgres.New(cfg.DatabaseURL, logger)
	if err != nil {
		return nil, err
	}

	notifier := unrealiable.NewUnreliableNotifier()
	candidateProcessor := processing.NewCandidateProcesor(database, logger, notifier, cfg.OutboxBatchSize)
	poller := poller.NewPoller(candidateProcessor, logger, cfg.PollIntervalMs)

	return &OutboxPoller{
		Database: database,
		Poller:   poller,
		Config:   cfg,
	}, nil
}

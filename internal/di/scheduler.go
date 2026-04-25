package di

import (
	"context"

	"github.com/robfig/cron/v3"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/infra/logger"
	"github.com/candidate-ingestion/service/internal/infra/postgres"
	"github.com/candidate-ingestion/service/internal/usecase/cleanup"
)

type Scheduler struct {
	Database *postgres.DB
	Cleaner  *cleanup.Cleaner
	Jobs     *cron.Cron
}

func NewScheduler(ctx context.Context, cfg *config.Config) (*Scheduler, error) {
	logger := logger.New(cfg.LogLevel)

	database, err := postgres.New(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	cleaner := cleanup.NewCleaner(database, cfg.Outbox.RetentionDays, logger)
	jobs := cron.New()

	return &Scheduler{
		Cleaner:  cleaner,
		Database: database,
		Jobs:     jobs,
	}, nil
}

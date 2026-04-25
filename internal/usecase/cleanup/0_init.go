package cleanup

import (
	"github.com/candidate-ingestion/service/internal/domain/repo"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

type Cleaner struct {
	db            repo.DB
	retentionDays int
	logger        service.Logger
}

func NewCleaner(db repo.DB, retentionDays int, logger service.Logger) *Cleaner {
	return &Cleaner{
		db:            db,
		retentionDays: retentionDays,
		logger:        logger,
	}
}

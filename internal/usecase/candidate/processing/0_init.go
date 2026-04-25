package processing

import (
	"github.com/candidate-ingestion/service/internal/domain/repo"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.MessageHandler = new(CandidateProcesor)
var _ service.PollHandler = new(CandidateProcesor)

type CandidateProcesor struct {
	db        repo.DB
	logger    service.Logger
	notifier  service.Notifier
	batchSize int
}

func NewCandidateProcesor(db repo.DB, logger service.Logger, notifier service.Notifier, batchSize int) *CandidateProcesor {
	return &CandidateProcesor{
		db:        db,
		logger:    logger,
		notifier:  notifier,
		batchSize: batchSize,
	}
}

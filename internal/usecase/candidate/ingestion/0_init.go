package candidateingestion

import (
	"github.com/candidate-ingestion/service/internal/domain/repo"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.CandidateIngester = new(Ingester)

type Ingester struct {
	db        repo.DB
	publisher service.Publisher
	topic     string
	breaker   service.CircuitBreaker
	logger    service.Logger
}

func NewCandidateApplicationIngester(db repo.DB, publisher service.Publisher, topic string, breaker service.CircuitBreaker, logger service.Logger) *Ingester {
	return &Ingester{
		db:        db,
		publisher: publisher,
		topic:     topic,
		breaker:   breaker,
		logger:    logger,
	}
}

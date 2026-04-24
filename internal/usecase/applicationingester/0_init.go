package applicationingester

import (
	"github.com/candidate-ingestion/service/internal/domain/service"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
)

var _ service.ApplicationIngester = new(Ingester)

type Ingester struct {
	db      *db.DB
	ps      *pubsub.Client
	topic   string
	breaker service.CircuitBreaker
}

func NewCandidateApplicationIngester(d *db.DB, p *pubsub.Client, topic string, breaker service.CircuitBreaker) *Ingester {
	return &Ingester{
		db:      d,
		ps:      p,
		topic:   topic,
		breaker: breaker,
	}
}

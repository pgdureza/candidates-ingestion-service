package metrics

import (
	"github.com/candidate-ingestion/service/internal/domain/repo"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ service.MetricsCollector = new(Collector)

type Collector struct {
	db repo.DB
}

func NewMetricsCollector(db repo.DB) *Collector {
	return &Collector{db: db}
}

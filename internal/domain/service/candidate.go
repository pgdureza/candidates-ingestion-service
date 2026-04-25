package service

import (
	"context"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

type CandidateParser interface {
	Parse(payload []byte) (*model.Candidate, error)
	Source() string
}

type CandidateIngester interface {
	Ingest(
		ctx context.Context,
		source string,
		payload []byte,
	) (string, error)
}

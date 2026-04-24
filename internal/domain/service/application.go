package service

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

type ApplicationParser interface {
	Parse(payload []byte) (*model.CandidateApplication, error)
	Source() string
}

type ApplicationIngester interface {
	Ingest(
		ctx context.Context,
		source string,
		idempotencyKey string,
		payload []byte,
		log *logrus.Entry,
	) (string, error)
}

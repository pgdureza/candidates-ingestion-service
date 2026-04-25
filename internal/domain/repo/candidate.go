package repo

import (
	"context"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

type CandidateRepo interface {
	Exists(ctx context.Context, source string, sourceRefID string) (bool, string, error)
	Create(ctx context.Context, app *model.Candidate) error
}

package service

import (
	"context"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

type Notifier interface {
	Notify(ctx context.Context, candidate *model.Candidate) error
}

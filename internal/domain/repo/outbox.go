package repo

import (
	"context"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

type OutboxRepo interface {
	MarkPublished(ctx context.Context, eventID string) error
	GetUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	GetUnpublishedForUpdate(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	Create(ctx context.Context, outbox *model.OutboxEvent) error
	Cleanup(ctx context.Context, retentionDays int) (int64, error)
}

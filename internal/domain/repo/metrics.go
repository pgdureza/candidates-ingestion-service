package repo

import "context"

type MetricsRepo interface {
	CountApplications(ctx context.Context) (int, error)
	CountOutboxPublished(ctx context.Context) (int, error)
	CountOutboxUnpublished(ctx context.Context) (int, error)
}

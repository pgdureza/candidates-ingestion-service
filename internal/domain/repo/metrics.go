package repo

import "context"

type MetricsRepo interface {
	GetMetric(ctx context.Context, name string) (int64, error)
	IncrementMetric(ctx context.Context, name string, value int64) error
}

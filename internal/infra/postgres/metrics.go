package postgres

import (
	"context"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/repo"
)

var _ repo.MetricsRepo = new(MetricsRepo)

type MetricsRepo struct {
	db *DB
}

func (m *MetricsRepo) GetMetric(ctx context.Context, name string) (int64, error) {
	var value int64
	err := m.db.conn.QueryRowContext(ctx,
		"SELECT value FROM metrics WHERE metric_name = $1",
		name,
	).Scan(&value)
	return value, err
}

func (m *MetricsRepo) IncrementMetric(ctx context.Context, name string, value int64) error {
	_, err := m.db.conn.ExecContext(ctx,
		"UPDATE metrics SET value = value + $1 WHERE metric_name = $2",
		value, name,
	)
	if err != nil {
		return fmt.Errorf("failed to increment metric %s: %w", name, err)
	}
	return nil
}

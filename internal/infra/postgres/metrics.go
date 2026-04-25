package postgres

import (
	"context"

	"github.com/candidate-ingestion/service/internal/domain/repo"
)

var _ repo.MetricsRepo = new(MetricsRepo)

type MetricsRepo struct {
	db *DB
}

func (m *MetricsRepo) CountApplications(ctx context.Context) (int, error) {
	var count int
	err := m.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM candidate_applications").Scan(&count)
	return count, err
}

func (m *MetricsRepo) CountOutboxUnpublished(ctx context.Context) (int, error) {
	var count int
	err := m.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM outbox_events WHERE published = false").Scan(&count)
	return count, err
}

func (m *MetricsRepo) CountOutboxPublished(ctx context.Context) (int, error) {
	var count int
	err := m.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM outbox_events WHERE published = true").Scan(&count)
	return count, err
}

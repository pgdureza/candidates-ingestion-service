package repo

import (
	"context"
	"database/sql"
)

type DB interface {
	GetConn() *sql.DB
	Close() error
	Candidates() CandidateRepo
	Outbox() OutboxRepo
	Metrics() MetricsRepo
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}

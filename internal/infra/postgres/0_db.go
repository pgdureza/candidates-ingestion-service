package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/candidate-ingestion/service/internal/domain/repo"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

var _ repo.DB = new(DB)

type DB struct {
	conn *sql.DB
	log  service.Logger
}

func New(dsn string, log service.Logger) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := conn.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	return &DB{conn: conn, log: log}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) GetConn() *sql.DB {
	return d.conn
}

func (d *DB) Candidates() repo.CandidateRepo {
	return &CandidateRepo{db: d}
}

func (d *DB) Outbox() repo.OutboxRepo {
	return &OutboxRepo{db: d}
}

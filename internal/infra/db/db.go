package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func New(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := conn.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	return &DB{conn: conn}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) GetConn() *sql.DB {
	return d.conn
}

// GetIdempotencyKey checks if request already processed
func (d *DB) GetIdempotencyKey(ctx context.Context, key string) (bool, string, error) {
	var appID string
	err := d.conn.QueryRowContext(
		ctx,
		`SELECT application_id FROM idempotency_keys WHERE request_id = $1 LIMIT 1`,
		key,
	).Scan(&appID)

	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("idempotency check failed: %w", err)
	}

	return true, appID, nil
}

// StoreIdempotencyKey stores request -> application mapping
func (d *DB) StoreIdempotencyKey(ctx context.Context, requestID string, appID string) error {
	_, err := d.conn.ExecContext(
		ctx,
		`INSERT INTO idempotency_keys (request_id, application_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		requestID, appID,
	)
	return err
}

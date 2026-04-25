package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// contextKey for storing transaction in context
type contextKey string

const txKey contextKey = "tx"

// getTx retrieves transaction from context if exists, otherwise uses connection pool
func (d *DB) getTx(ctx context.Context) interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
} {
	if tx, ok := ctx.Value(txKey).(*sql.Tx); ok {
		return tx
	}
	return d.conn
}

// WithTransaction wraps business logic in a database transaction
func (d *DB) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Store tx in context so repo methods can use it
	txCtx := context.WithValue(ctx, txKey, tx)

	if err := fn(txCtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

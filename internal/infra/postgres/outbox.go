package postgres

import (
	"context"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/model"
	"github.com/candidate-ingestion/service/internal/domain/repo"
)

var _ repo.OutboxRepo = new(OutboxRepo)

type OutboxRepo struct {
	db *DB
}

func (d *DB) NewOutboxRepo() *OutboxRepo {
	return &OutboxRepo{db: d}
}

// MarkPublished marks outbox event as published
func (or *OutboxRepo) MarkPublished(ctx context.Context, eventID string) error {
	_, err := or.db.conn.ExecContext(
		ctx,
		`UPDATE outbox_events SET published = true, published_at = NOW() WHERE id = $1`,
		eventID,
	)
	return err
}

// Create inserts outbox event (uses tx from context if available)
func (or *OutboxRepo) Create(ctx context.Context, outbox *model.OutboxEvent) error {
	executor := or.db.getTx(ctx)
	_, err := executor.ExecContext(
		ctx,
		`INSERT INTO outbox_events (id, event_type, payload, published, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		outbox.ID, outbox.EventType, outbox.Payload, false, outbox.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert outbox: %w", err)
	}
	return nil
}

// GetUnpublished gets unpublished events
func (or *OutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	rows, err := or.db.conn.QueryContext(
		ctx,
		`SELECT id, event_type, payload, published, created_at
		FROM outbox_events WHERE published = false ORDER BY created_at ASC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []model.OutboxEvent
	for rows.Next() {
		var event model.OutboxEvent
		err := rows.Scan(
			&event.ID, &event.EventType, &event.Payload,
			&event.Published, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// Cleanup deletes outbox events older than retentionDays
func (or *OutboxRepo) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	// add an additional 5 seconds delay so no race condition with metrics
	result, err := or.db.conn.ExecContext(
		ctx,
		`DELETE FROM outbox_events 
     WHERE published = TRUE 
       AND published_at < NOW() - (INTERVAL '1 day' * $1 + INTERVAL '5 seconds')`,
		retentionDays,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

package db

import (
	"context"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain"
)

// StoreApplicationWithOutbox atomically stores application + outbox event (transactional outbox)
func (d *DB) StoreApplicationWithOutbox(
	ctx context.Context,
	app *domain.CandidateApplication,
	outbox *domain.OutboxEvent,
) error {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert application
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO candidate_applications
		(id, first_name, last_name, email, phone, position, source, source_ref_id, raw_payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		app.ID, app.FirstName, app.LastName, app.Email, app.Phone, app.Position,
		app.Source, app.SourceRefID, app.RawPayload, app.CreatedAt, app.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert application: %w", err)
	}

	// Insert outbox event
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO outbox_events (id, application_id, event_type, payload, published, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		outbox.ID, outbox.ApplicationID, outbox.EventType, outbox.Payload, false, outbox.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUnpublishedOutboxEvents gets events to publish
func (d *DB) GetUnpublishedOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := d.conn.QueryContext(
		ctx,
		`SELECT id, application_id, event_type, payload, published, created_at
		FROM outbox_events WHERE published = false ORDER BY created_at ASC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	for rows.Next() {
		var event domain.OutboxEvent
		err := rows.Scan(
			&event.ID, &event.ApplicationID, &event.EventType, &event.Payload,
			&event.Published, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// MarkOutboxEventPublished marks event as published
func (d *DB) MarkOutboxEventPublished(ctx context.Context, eventID string) error {
	_, err := d.conn.ExecContext(
		ctx,
		`UPDATE outbox_events SET published = true, published_at = NOW() WHERE id = $1`,
		eventID,
	)
	return err
}

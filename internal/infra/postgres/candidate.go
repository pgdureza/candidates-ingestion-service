package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/model"
	"github.com/candidate-ingestion/service/internal/domain/repo"
)

var _ repo.CandidateRepo = new(CandidateRepo)

func (d *DB) NewCandidateRepo() *CandidateRepo {
	return &CandidateRepo{db: d}
}

type CandidateRepo struct {
	db *DB
}

// Exists checks if application with same source+source_ref_id already exists
func (ar *CandidateRepo) Exists(ctx context.Context, source string, sourceRefID string) (bool, string, error) {
	var appID string
	err := ar.db.conn.QueryRowContext(
		ctx,
		`SELECT id FROM candidate_applications WHERE source = $1 AND source_ref_id = $2 LIMIT 1`,
		source, sourceRefID,
	).Scan(&appID)

	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("application existence check failed: %w", err)
	}

	return true, appID, nil
}

// Store inserts application into database (uses tx from context if available)
func (ar *CandidateRepo) Create(ctx context.Context, app *model.Candidate) error {
	executor := ar.db.getTx(ctx)
	_, err := executor.ExecContext(
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
	return nil
}

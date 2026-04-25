package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

func (p *CandidateProcesor) Handle(ctx context.Context, data []byte) error {

	// Reconstruct candidate data from message
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	p.logger.Debug("message unmarshalled successfully")

	appData, _ := msg["application"].(map[string]interface{})
	candidate := &model.Candidate{
		ID:          toString(appData["id"]),
		FirstName:   toString(appData["first_name"]),
		LastName:    toString(appData["last_name"]),
		Email:       toString(appData["email"]),
		Phone:       toString(appData["phone"]),
		Position:    toString(appData["position"]),
		Source:      toString(appData["source"]),
		SourceRefID: toString(appData["source_ref_id"]),
		RawPayload:  toString(appData["raw_payload"]),
		CreatedAt:   parseTime(appData["created_at"]),
		UpdatedAt:   parseTime(appData["updated_at"]),
	}

	jsonData, err := json.Marshal(candidate)
	if err != nil {
		return fmt.Errorf("could not convert candidate %s into json data: %w", candidate.ID, err)
	}

	// Create outbox event
	outbox := &model.OutboxEvent{
		ID:        uuid.New().String(),
		EventType: "application.created",
		Payload:   string(jsonData),
		Published: false,
		CreatedAt: time.Now().UTC(),
	}

	// Atomically store application + outbox event in transaction
	// Dedup check happens inside transaction to prevent race conditions
	if err := p.db.WithTransaction(ctx, func(txCtx context.Context) error {
		// Check if already exists (within transaction, prevents race)
		exists, existingAppID, err := p.db.Candidates().Exists(txCtx, candidate.Source, candidate.SourceRefID)
		if err != nil {
			return fmt.Errorf("dedup check failed: %w", err)
		}
		if exists {
			p.logger.WithField("app_id", existingAppID).Info("duplicate detected, skipping")
			p.db.Metrics().IncrementMetric(txCtx, "webhooks_duplicate", 1)
			return nil
		}

		if err := p.db.Candidates().Create(txCtx, candidate); err != nil {
			return err
		}

		if err := p.db.Outbox().Create(txCtx, outbox); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to store application: %w", err)
	}

	p.db.Metrics().IncrementMetric(ctx, "outbox_written", 1)

	p.logger.WithFields(logrus.Fields{
		"candidate_id": candidate.ID,
		"source":       candidate.Source,
		"email":        candidate.Email,
		"name":         candidate.FirstName + " " + candidate.LastName,
	}).Info("application persisted successfully")
	return nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func parseTime(v interface{}) time.Time {
	if s, ok := v.(string); ok {
		t, _ := time.Parse(time.RFC3339, s)
		return t
	}
	return time.Now().UTC()
}

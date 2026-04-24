package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/candidate-ingestion/service/internal/domain"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/google/uuid"
)

// Pool bulkhead pattern: bounded goroutine pool
type Pool struct {
	workers       int
	workerTimeout time.Duration
	db            *db.DB
	ps            *pubsub.Client
	semaphore     chan struct{}
	stopChan      chan struct{}
}

func NewPool(workers int, timeout time.Duration, d *db.DB, p *pubsub.Client) *Pool {
	return &Pool{
		workers:       workers,
		workerTimeout: timeout,
		db:            d,
		ps:            p,
		semaphore:     make(chan struct{}, workers),
		stopChan:      make(chan struct{}),
	}
}

// Start subscribes and processes messages
func (p *Pool) Start(ctx context.Context, projectID string, topic string) error {
	return p.ps.SubscribeAndProcess(ctx, topic, func(msgCtx context.Context, data []byte) error {
		// Acquire slot from semaphore (bulkhead)
		select {
		case p.semaphore <- struct{}{}:
			defer func() { <-p.semaphore }()
		case <-ctx.Done():
			return ctx.Err()
		}

		// Process with timeout
		workerCtx, cancel := context.WithTimeout(msgCtx, p.workerTimeout)
		defer cancel()

		return p.processMessage(workerCtx, data)
	})
}

// Stop stops the pool
func (p *Pool) Stop() {
	close(p.stopChan)
}

func (p *Pool) processMessage(ctx context.Context, data []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	idempotencyKey, _ := msg["idempotency_key"].(string)
	appData, _ := msg["application"].(map[string]interface{})

	// Idempotency: check if already processed
	exists, _, err := p.db.GetIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		log.Printf("Message already processed (idempotency): %s", idempotencyKey)
		return nil
	}

	// Reconstruct application from message
	app := &domain.CandidateApplication{
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

	// Create outbox event
	outbox := &domain.OutboxEvent{
		ID:            uuid.New().String(),
		ApplicationID: app.ID,
		EventType:     "application.created",
		Payload:       string(data),
		Published:     false,
		CreatedAt:     time.Now().UTC(),
	}

	// Atomically store application + outbox event
	if err := p.db.StoreApplicationWithOutbox(ctx, app, outbox); err != nil {
		return fmt.Errorf("failed to store application: %w", err)
	}

	// Mark outbox as published (in real app, publish to downstream service first)
	if err := p.db.MarkOutboxEventPublished(ctx, outbox.ID); err != nil {
		log.Printf("Warning: failed to mark outbox as published: %v", err)
	}

	log.Printf("Application persisted: %s (%s)", app.ID, app.Source)
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

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pubsubgcp "cloud.google.com/go/pubsub"
	"github.com/candidate-ingestion/service/internal/domain"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Pool bulkhead pattern: bounded goroutine pool
type Pool struct {
	workers       int
	workerTimeout time.Duration
	db            *db.DB
	ps            *pubsub.Client
	semaphore     chan struct{}
	stopChan      chan struct{}
	log           *logrus.Logger
}

func NewPool(workers int, timeout time.Duration, d *db.DB, p *pubsub.Client, log *logrus.Logger) *Pool {
	return &Pool{
		workers:       workers,
		workerTimeout: timeout,
		db:            d,
		ps:            p,
		semaphore:     make(chan struct{}, workers),
		stopChan:      make(chan struct{}),
		log:           log,
	}
}

// Start subscribes and processes messages
func (p *Pool) Start(ctx context.Context, projectID string, topic string) error {
	p.log.WithField("topic", topic).Info("waiting for messages")

	return p.ps.SubscribeAndProcess(ctx, topic, func(msgCtx context.Context, msg *pubsubgcp.Message) error {
		// Extract idempotency_key and message_id for correlation
		var preview map[string]interface{}
		_ = json.Unmarshal(msg.Data, &preview)
		idempotencyKey, _ := preview["idempotency_key"].(string)

		// Create message-scoped logger with correlation fields
		msgLog := p.log.WithFields(logrus.Fields{
			"idempotency_key": idempotencyKey,
			"message_id":      msg.ID,
		})
		msgLog.Info("message consumed from pubsub")

		// Acquire slot from semaphore (bulkhead)
		select {
		case p.semaphore <- struct{}{}:
			defer func() { <-p.semaphore }()
		case <-ctx.Done():
			msgLog.Warn("context cancelled while waiting for semaphore, nacking")
			msg.Nack()
			return ctx.Err()
		}

		// Process with timeout
		workerCtx, cancel := context.WithTimeout(msgCtx, p.workerTimeout)
		defer cancel()

		err := p.processMessage(workerCtx, msg.Data, msgLog)
		if err != nil {
			msgLog.WithError(err).Error("processing failed, nacking message")
			msg.Nack()
			return err
		}

		msgLog.Info("processing succeeded, acking message")
		msg.Ack()
		return nil
	})
}

// Stop stops the pool
func (p *Pool) Stop() {
	close(p.stopChan)
}

func (p *Pool) processMessage(ctx context.Context, data []byte, log *logrus.Entry) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	log.Debug("message unmarshalled successfully")

	idempotencyKey, _ := msg["idempotency_key"].(string)
	appData, _ := msg["application"].(map[string]interface{})

	// Idempotency: check if already processed
	exists, existingAppID, err := p.db.GetIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		log.WithField("app_id", existingAppID).Info("duplicate detected, skipping (idempotency)")
		return nil
	}
	log.Info("idempotency check passed, processing new message")

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
	log.WithFields(logrus.Fields{
		"app_id": app.ID,
		"source": app.Source,
		"email":  app.Email,
		"name":   app.FirstName + " " + app.LastName,
	}).Debug("application reconstructed from message")

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
		log.WithFields(logrus.Fields{
			"outbox_id": outbox.ID,
			"app_id":    app.ID,
		}).WithError(err).Warn("failed to mark outbox event as published")
	}

	log.WithFields(logrus.Fields{
		"app_id": app.ID,
		"source": app.Source,
		"email":  app.Email,
		"name":   app.FirstName + " " + app.LastName,
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

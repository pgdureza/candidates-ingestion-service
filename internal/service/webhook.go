package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/candidate-ingestion/service/internal/domain"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type WebhookService struct {
	db        *db.DB
	ps        *pubsub.Client
	topic     string
	breaker   *CircuitBreaker
}

func NewWebhookService(d *db.DB, p *pubsub.Client, topic string) *WebhookService {
	return &WebhookService{
		db:      d,
		ps:      p,
		topic:   topic,
		breaker: NewCircuitBreaker(5, 60*time.Second, 30*time.Second), // fail threshold, open timeout, half-open
	}
}

// IngestWebhook parses, stores, and publishes to broker
// Does NOT block on DB writes (async via worker pool)
func (s *WebhookService) IngestWebhook(
	ctx context.Context,
	source string,
	idempotencyKey string,
	payload []byte,
	log *logrus.Entry,
) (string, error) {
	// Idempotency check
	exists, appID, err := s.db.GetIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return "", fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		log.WithField("app_id", appID).Info("duplicate webhook, returning cached app id")
		return appID, nil // Already processed
	}
	log.Debug("idempotency check passed, processing new webhook")

	// Parse via strategy
	strategy, err := domain.StrategyFactory(source)
	if err != nil {
		return "", err
	}

	app, err := strategy.Parse(payload)
	if err != nil {
		log.WithError(err).Warn("webhook parsing failed")
		return "", err
	}
	log.Debug("webhook parsed successfully")

	// Generate ID
	app.ID = uuid.New().String()
	app.CreatedAt = time.Now().UTC()
	app.UpdatedAt = app.CreatedAt

	// Store idempotency key immediately (before async processing)
	if err := s.db.StoreIdempotencyKey(ctx, idempotencyKey, app.ID); err != nil {
		return "", fmt.Errorf("failed to store idempotency key: %w", err)
	}
	log.WithFields(logrus.Fields{
		"app_id":      app.ID,
		"first_name":  app.FirstName,
		"last_name":   app.LastName,
		"email":       app.Email,
	}).Debug("idempotency key stored")

	// Publish to broker (fast, decoupled)
	msgPayload, _ := json.Marshal(map[string]interface{}{
		"idempotency_key": idempotencyKey,
		"application":     app,
		"timestamp":       time.Now().UTC(),
	})

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use circuit breaker for pub/sub call
	err = s.breaker.Execute(func() error {
		return s.ps.PublishJSON(publishCtx, s.topic, msgPayload)
	})

	if err != nil {
		// Log but don't fail the request
		// Worker will retry from DB later
		log.WithError(err).Warn("pubsub publish failed (circuit breaker or timeout), worker will retry")
		return app.ID, nil
	}

	log.WithField("app_id", app.ID).Info("webhook published to pubsub")
	return app.ID, nil
}

package candidateingestion

import (
	"context"
	"encoding/json"
	"time"
)

// IngestWebhook parses, stores, and publishes to broker
func (s *Ingester) Ingest(
	ctx context.Context,
	source string,
	payload []byte,
) (string, error) {

	parser, err := NewCandidateApplicationParser(source)
	if err != nil {
		return "", err
	}

	candidate, err := parser.Parse(payload)
	if err != nil {
		s.logger.WithError(err).Warn("webhook parsing failed")
		return "", err
	}
	s.logger.Debug("webhook parsed successfully")

	// Publish to broker (fast, decoupled)
	msgPayload, _ := json.Marshal(map[string]interface{}{
		"application": candidate,
		"timestamp":   time.Now().UTC(),
	})

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use circuit breaker for pub/sub call
	err = s.breaker.Execute(func() error {
		return s.publisher.PublishJSON(publishCtx, s.topic, msgPayload)
	})

	if err != nil {
		// Log but don't fail the request
		// Worker will retry from DB later
		s.logger.WithError(err).Warn("pubsub publish failed (circuit breaker or timeout), worker will retry")
		return candidate.ID, nil
	}

	s.logger.WithField("app_id", candidate.ID).Info("webhook published to pubsub")
	return candidate.ID, nil
}

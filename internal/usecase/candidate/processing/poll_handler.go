package processing

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain/model"
)

// Consume fetches unpublished events and notifies external systems
// Core polling cycle: fetch batch > notify > mark published
func (cp *CandidateProcesor) Execute(ctx context.Context) {
	outboxRepo := cp.db.Outbox()

	// Fetch unpublished events (simple query, no locks needed - single pod)
	events, err := outboxRepo.GetUnpublished(ctx, cp.batchSize)
	if err != nil {
		cp.logger.WithError(err).Error("Failed to fetch unpublished events")
		return
	}

	if len(events) == 0 {
		return
	}

	cp.logger.WithField("count", len(events)).Debug("Fetched unpublished events")

	// Process each event
	for _, event := range events {
		l := cp.logger.WithFields(logrus.Fields{
			"event_id": event.ID,
		})

		var candidate model.Candidate
		if err := json.Unmarshal([]byte(event.Payload), &candidate); err != nil {
			l.WithError(err).Warn("Failed to notify, could not unmarshal outbox payload")
			continue
		}

		// Notify external systems (unreliable, may fail)
		if err := cp.notifier.Notify(ctx, &candidate); err != nil {
			l.WithError(err).Warn("Failed to notify, will retry later")
			continue
		}

		// Mark published only after successful notification
		if err := outboxRepo.MarkPublished(ctx, event.ID); err != nil {
			l.WithError(err).Warn("Failed to mark event as published")
			// Non-fatal, will retry next cycle
		}

		l.Info("Notification Success")
	}
}

package metrics

import (
	"context"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

func (c *Collector) Collect(ctx context.Context) (*service.MetricsResponse, error) {
	m := c.db.Metrics()

	ingested, err := m.GetMetric(ctx, "webhooks_ingested")
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks_ingested: %w", err)
	}

	rejected, err := m.GetMetric(ctx, "webhooks_rejected")
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks_rejected: %w", err)
	}

	outboxWritten, err := m.GetMetric(ctx, "outbox_written")
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox_written: %w", err)
	}

	outboxProcessAttempts, err := m.GetMetric(ctx, "outbox_process_attempts")
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox_process_attempts: %w", err)
	}

	outboxPublishedSuccess, err := m.GetMetric(ctx, "outbox_publish_success")
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox_publish_success: %w", err)
	}

	outboxPublishedFailed, err := m.GetMetric(ctx, "outbox_publish_failed")
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox_publish_failed: %w", err)
	}

	notificationFailed, err := m.GetMetric(ctx, "notification_failed")
	if err != nil {
		return nil, fmt.Errorf("failed to get notification_failed: %w", err)
	}

	outboxCleaned, err := m.GetMetric(ctx, "outbox_cleaned")
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox_cleaned: %w", err)
	}

	return &service.MetricsResponse{
		WebhooksIngested:      int(ingested),
		WebhooksRejected:      int(rejected),
		OutboxWritten:         int(outboxWritten),
		OutboxProcessAttempts: int(outboxProcessAttempts),
		OutboxPublishFailed:   int(outboxPublishedFailed),
		OutboxPublishSuccess:  int(outboxPublishedSuccess),
		NotificationFailed:    int(notificationFailed),
		OutboxCleaned:         int(outboxCleaned),
	}, nil
}

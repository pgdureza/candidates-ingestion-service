package service

import "context"

type MetricsResponse struct {
	WebhooksIngested      int `json:"webhooks_ingested"`
	WebhooksRejected      int `json:"webhooks_rejected"`
	OutboxWritten         int `json:"outbox_written"`
	OutboxProcessAttempts int `json:"outbox_process_attempts"`
	OutboxPublishSuccess  int `json:"outbox_publish_success"`
	OutboxPublishFailed   int `json:"outbox_publish_failed"`
	NotificationFailed    int `json:"notification_failed"`
	OutboxCleaned         int `json:"outbox_cleaned"`
}

type MetricsCollector interface {
	Collect(ctx context.Context) (*MetricsResponse, error)
}

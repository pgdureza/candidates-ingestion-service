package service

import "context"

type MetricsResponse struct {
	WebhooksTotalRequest  int `json:"webhooks_total_request"`
	WebhooksRateLimited   int `json:"webhooks_rate_limited"`
	WebhooksIngested      int `json:"webhooks_ingested"`
	WebhooksRejected      int `json:"webhooks_rejected"`
	WebhooksDuplicate     int `json:"webhooks_duplicate"`
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

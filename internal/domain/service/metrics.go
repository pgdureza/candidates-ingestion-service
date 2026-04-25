package service

import "context"

type MetricsResponse struct {
	WebhooksReceived  int `json:"0_webhooks_received"`
	UnpublishedEvents int `json:"1_unpublished_events"`
	PublishedEvents   int `json:"2_published_events"`
}

type MetricsCollector interface {
	Collect(ctx context.Context) (*MetricsResponse, error)
}

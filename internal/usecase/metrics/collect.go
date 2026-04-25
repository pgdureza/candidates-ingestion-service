package metrics

import (
	"context"
	"fmt"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

func (c *Collector) Collect(ctx context.Context) (*service.MetricsResponse, error) {
	metrics := c.db.Metrics()

	totalApplications, err := metrics.CountApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count applications: %w", err)
	}

	published, err := metrics.CountOutboxPublished(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count published events: %w", err)
	}

	unpublished, err := metrics.CountOutboxUnpublished(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count unpublished events: %w", err)
	}

	return &service.MetricsResponse{
		WebhooksReceived:  totalApplications,
		PublishedEvents:   published,
		UnpublishedEvents: unpublished,
	}, nil
}

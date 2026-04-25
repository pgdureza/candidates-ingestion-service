package poller

import (
	"context"
	"time"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

type Poller struct {
	handler    service.PollHandler
	logger     service.Logger
	intervalMs int
	stopChan   chan struct{}
}

func NewPoller(handler service.PollHandler, logger service.Logger, intervalMs int) *Poller {
	return &Poller{
		handler:    handler,
		logger:     logger,
		stopChan:   make(chan struct{}),
		intervalMs: intervalMs,
	}
}

// Start begins polling for unpublished outbox events
func (p *Poller) Start(ctx context.Context) {
	interval := p.intervalMs * int(time.Millisecond)
	ticker := time.NewTicker(time.Duration(interval))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Poller context cancelled")
			return
		case <-p.stopChan:
			p.logger.Info("Poller stopped")
			return
		case <-ticker.C:
			p.logger.Info("Poll...")
			p.handler.Execute(ctx)
		}
	}
}

// Stop signals the poller to stop
func (p *Poller) Stop() {
	close(p.stopChan)
}

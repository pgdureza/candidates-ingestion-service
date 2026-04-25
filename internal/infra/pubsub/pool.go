package pubsub

import (
	"context"
	"time"

	pubsubgcp "cloud.google.com/go/pubsub"
	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

// PubSubPool bulkhead pattern: bounded goroutine pool
type PubSubPool struct {
	handler       service.MessageHandler
	workers       int
	workerTimeout time.Duration
	ps            *Client
	semaphore     chan struct{}
	stopChan      chan struct{}
	logger        service.Logger
}

func NewPubSubPool(handler service.MessageHandler, workers int, timeout time.Duration, ps *Client, logger service.Logger) *PubSubPool {
	return &PubSubPool{
		handler:       handler,
		workers:       workers,
		workerTimeout: timeout,
		ps:            ps,
		semaphore:     make(chan struct{}, workers),
		stopChan:      make(chan struct{}),
		logger:        logger,
	}
}

// Start subscribes and processes messages
func (p *PubSubPool) Start(ctx context.Context, projectID string, topic string) error {
	p.logger.WithField("topic", topic).Info("waiting for messages")

	return p.ps.SubscribeAndProcess(ctx, topic, func(msgCtx context.Context, msg *pubsubgcp.Message) error {
		// Create message-scoped logger with correlation fields
		msgLog := p.logger.WithFields(logrus.Fields{
			"message_id": msg.ID,
		})
		msgLog.Info("message consumed from pubsub")

		// Acquire slot from semaphore
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

		err := p.handler.Handle(workerCtx, msg.Data)
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
func (p *PubSubPool) Stop() {
	close(p.stopChan)
}

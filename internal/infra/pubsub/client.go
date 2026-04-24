package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
)

type Client struct {
	client *pubsub.Client
}

func New(ctx context.Context, projectID string) (*Client, error) {
	c, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	return &Client{client: c}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

// PublishJSON publishes JSON message
func (c *Client) PublishJSON(ctx context.Context, topic string, msg []byte) error {
	t := c.client.Topic(topic)
	exists, err := t.Exists(ctx)
	if err != nil {
		return fmt.Errorf("topic check failed: %w", err)
	}
	if !exists {
		var err error
		t, err = c.client.CreateTopic(ctx, topic)
		if err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}
	}

	result := t.Publish(ctx, &pubsub.Message{
		Data: msg,
		Attributes: map[string]string{
			"source": "webhook-service",
		},
	})

	_, err = result.Get(ctx)
	return err
}

// SubscribeAndProcess subscribes and processes messages with callback
// Callback receives the raw message and is responsible for ACK/NAK
func (c *Client) SubscribeAndProcess(
	ctx context.Context,
	topic string,
	callback func(context.Context, *pubsub.Message) error, // cloud.google.com/go/pubsub.Message
) error {
	sub, err := c.client.CreateSubscription(ctx, topic+"-sub", pubsub.SubscriptionConfig{
		Topic:       c.client.Topic(topic),
		AckDeadline: 60,
	})
	if err != nil {
		// Subscription might already exist
		sub = c.client.Subscription(topic + "-sub")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			_ = callback(ctx, msg)
		})

		if err != nil {
			return fmt.Errorf("receive failed: %w", err)
		}
	}
}

// PublishData publishes raw data
func (c *Client) PublishData(ctx context.Context, topic string, data []byte) error {
	return c.PublishJSON(ctx, topic, data)
}

// MarshalJSON helper
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

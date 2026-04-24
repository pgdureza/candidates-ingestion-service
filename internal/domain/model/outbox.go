package model

import (
	"time"
)

// OutboxEvent transactional outbox for reliability
type OutboxEvent struct {
	ID            string     `json:"id"`
	ApplicationID string     `json:"application_id"`
	EventType     string     `json:"event_type"` // "application.created"
	Payload       string     `json:"payload"`
	Published     bool       `json:"published"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

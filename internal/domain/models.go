package domain

import (
	"time"
)

// CandidateApplication normalized candidate application
type CandidateApplication struct {
	ID           string    `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Position     string    `json:"position"`
	Source       string    `json:"source"` // linkedin, google_forms, etc
	SourceRefID  string    `json:"source_ref_id"`
	RawPayload   string    `json:"raw_payload"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// OutboxEvent transactional outbox for reliability
type OutboxEvent struct {
	ID            string    `json:"id"`
	ApplicationID string    `json:"application_id"`
	EventType     string    `json:"event_type"` // "application.created"
	Payload       string    `json:"payload"`
	Published     bool      `json:"published"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// IdempotencyKey prevent duplicate processing
type IdempotencyKey struct {
	ID            string    `json:"id"`
	RequestID     string    `json:"request_id"`
	ApplicationID string    `json:"application_id"`
	CreatedAt     time.Time `json:"created_at"`
}

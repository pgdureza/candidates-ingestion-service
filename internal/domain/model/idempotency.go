package model

import (
	"time"
)

// IdempotencyKey prevent duplicate processing
type IdempotencyKey struct {
	ID            string    `json:"id"`
	RequestID     string    `json:"request_id"`
	ApplicationID string    `json:"application_id"`
	CreatedAt     time.Time `json:"created_at"`
}

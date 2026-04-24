package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/candidate-ingestion/service/internal/service"
	"github.com/google/uuid"
)

type WebhookHandler struct {
	svc *service.WebhookService
}

func NewWebhookHandler(svc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// HandleWebhook dispatches to appropriate strategy
// POST /webhooks/:source
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source") // Chi v5.0.12+
	if source == "" {
		http.Error(w, "source required", http.StatusBadRequest)
		return
	}

	// Idempotency key from header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = uuid.New().String()
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse and queue
	appID, err := h.svc.IngestWebhook(r.Context(), source, idempotencyKey, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"id": appID,
	})
}

// HandleHealth health check
// GET /health
func (h *WebhookHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

type WebhookHandler struct {
	svc service.ApplicationIngester
	log *logrus.Logger
}

func NewWebhookHandler(svc service.ApplicationIngester, log *logrus.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, log: log}
}

// HandleWebhook dispatches to appropriate strategy
// POST /webhooks/:source
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source") // Chi v5.0.12+
	if source == "" {
		h.log.WithField("path", r.URL.Path).Warn("webhook received with missing source")
		http.Error(w, "source required", http.StatusBadRequest)
		return
	}

	// Idempotency key from header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = uuid.New().String()
	}

	// Create request-scoped logger with correlation fields
	reqLog := h.log.WithFields(logrus.Fields{
		"source":          source,
		"idempotency_key": idempotencyKey,
	})
	reqLog.Info("webhook received")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		reqLog.WithError(err).Warn("failed to read request body")
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse and queue
	appID, err := h.svc.Ingest(r.Context(), source, idempotencyKey, body, reqLog)
	if err != nil {
		reqLog.WithError(err).Warn("webhook ingestion failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqLog.WithField("app_id", appID).Info("webhook accepted, published to pubsub")
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

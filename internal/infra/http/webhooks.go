package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain"
	"github.com/candidate-ingestion/service/internal/domain/repo"
	"github.com/candidate-ingestion/service/internal/domain/service"
)

type WebhookHandler struct {
	svc    service.CandidateIngester
	logger service.Logger
	db     repo.DB
}

func NewWebhookHandler(svc service.CandidateIngester, logger service.Logger, db repo.DB) *WebhookHandler {
	return &WebhookHandler{svc: svc, logger: logger, db: db}
}

// HandleWebhook dispatches to appropriate strategy
// POST /webhooks/:source
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source") // Chi v5.0.12+
	if source == "" {
		h.logger.WithField("path", r.URL.Path).Warn("webhook received with missing source")
		h.db.Metrics().IncrementMetric(r.Context(), "webhooks_rejected", 1)
		http.Error(w, "source required", http.StatusBadRequest)
		return
	}

	// Create request-scoped logger with correlation fields
	logger := h.logger.WithFields(logrus.Fields{
		"source": source,
	})
	logger.Info("webhook received")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.WithError(err).Warn("failed to read request body")
		h.db.Metrics().IncrementMetric(r.Context(), "webhooks_rejected", 1)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse and queue
	appID, err := h.svc.Ingest(r.Context(), source, body)
	if err != nil {
		h.db.Metrics().IncrementMetric(r.Context(), "webhooks_rejected", 1)
		logger.WithError(err).Warn("webhook ingestion failed")

		var cbErr *domain.CircuitBreakerError
		statusCode := http.StatusBadRequest
		if errors.As(err, &cbErr) {
			statusCode = http.StatusServiceUnavailable
		}
		http.Error(w, err.Error(), statusCode)
		return
	}

	logger.WithField("app_id", appID).Info("webhook accepted, published to pubsub")
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

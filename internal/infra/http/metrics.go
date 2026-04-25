package http

import (
	"encoding/json"
	"net/http"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

type MetricsHandler struct {
	collector service.MetricsCollector
}

func NewMetricsHandler(collector service.MetricsCollector) *MetricsHandler {
	return &MetricsHandler{collector: collector}
}

func (h *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.collector.Collect(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

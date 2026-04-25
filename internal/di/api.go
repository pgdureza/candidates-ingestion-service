package di

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/domain/service"
	apphttp "github.com/candidate-ingestion/service/internal/infra/http"
	"github.com/candidate-ingestion/service/internal/infra/logger"
	"github.com/candidate-ingestion/service/internal/infra/postgres"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	candidateingestion "github.com/candidate-ingestion/service/internal/usecase/candidate/ingestion"
	"github.com/candidate-ingestion/service/internal/usecase/circuitbreaker"
	"github.com/candidate-ingestion/service/internal/usecase/metrics"
)

type API struct {
	Database *postgres.DB
	PubSub   *pubsub.Client
	Router   *chi.Mux

	// internal
	topic    string
	ingester *candidateingestion.Ingester
	log      service.Logger
}

func NewAPI(ctx context.Context, cfg *config.Config) (*API, error) {
	logger := logger.New(cfg.LogLevel)
	database, err := postgres.New(cfg.DatabaseURL, logger)
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.New(ctx, cfg.GCPProject)
	if err != nil {
		database.Close()
		return nil, err
	}

	topic := cfg.PubSubTopic

	cb := circuitbreaker.NewCircuitBreaker(cfg.CircuitBreaker.FailureThreshold,
		time.Duration(cfg.CircuitBreaker.OpenTimeoutS)*time.Second,
		time.Duration(cfg.CircuitBreaker.HalfOpenTimeoutS)*time.Second,
	)
	ingester := candidateingestion.NewCandidateApplicationIngester(database, ps, topic, cb, logger)

	collector := metrics.NewMetricsCollector(database)

	// routes
	router := chi.NewRouter()
	webhhookHandler := apphttp.NewWebhookHandler(ingester, logger)
	metricsHandler := apphttp.NewMetricsHandler(collector)
	rateLimiter := apphttp.NewRateLimiter(cfg.WebhookRateLimit)

	router.Get("/health", webhhookHandler.HandleHealth)
	router.Get("/metrics", metricsHandler.HandleMetrics)

	// Apply rate limit middleware only to webhook endpoint
	router.Route("/webhooks", func(r chi.Router) {
		r.Use(rateLimiter.Middleware)
		r.Post("/{source}", webhhookHandler.HandleWebhook)
	})

	return &API{
		Database: database,
		PubSub:   ps,
		Router:   router,
		topic:    topic,
		ingester: ingester,
		log:      logger,
	}, nil
}

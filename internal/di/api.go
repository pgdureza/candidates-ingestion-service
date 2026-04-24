package di

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/config"
	"github.com/candidate-ingestion/service/internal/infra/db"
	apphttp "github.com/candidate-ingestion/service/internal/infra/http"
	"github.com/candidate-ingestion/service/internal/infra/logger"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/candidate-ingestion/service/internal/usecase/applicationingester"
	"github.com/candidate-ingestion/service/internal/usecase/circuitbreaker"
)

type APIContainer struct {
	Api      *Api
	Database *db.DB
	PubSub   *pubsub.Client
}

type Api struct {
	db        *db.DB
	ps        *pubsub.Client
	topic     string
	router    *chi.Mux
	whService *applicationingester.Ingester
	log       *logrus.Logger
}

func New(database *db.DB, pubsubClient *pubsub.Client, topic string, log *logrus.Logger) *Api {
	app := &Api{
		db:     database,
		ps:     pubsubClient,
		topic:  topic,
		router: chi.NewRouter(),
		log:    log,
	}

	// DIs
	cb := circuitbreaker.NewCircuitBreaker(5, 60*time.Second, 30*time.Second)
	app.whService = applicationingester.NewCandidateApplicationIngester(database, pubsubClient, topic, cb)
	app.setupRoutes()

	return app
}

func (a *Api) setupRoutes() {
	whHandler := apphttp.NewWebhookHandler(a.whService, a.log)

	a.router.Get("/health", whHandler.HandleHealth)
	a.router.Post("/webhooks/{source}", whHandler.HandleWebhook)
}

func (a *Api) Router() *chi.Mux {
	return a.router
}

func NewAPI(ctx context.Context, cfg *config.Config) (*APIContainer, error) {
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	ps, err := pubsub.New(ctx, cfg.GCPProject)
	if err != nil {
		database.Close()
		return nil, err
	}

	log := logger.New(cfg.LogLevel)
	application := New(database, ps, cfg.PubSubTopic, log)

	return &APIContainer{
		Api:      application,
		Database: database,
		PubSub:   ps,
	}, nil
}

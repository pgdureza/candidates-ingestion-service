package app

import (
	apphttp "github.com/candidate-ingestion/service/internal/infra/http"
	"github.com/candidate-ingestion/service/internal/infra/db"
	"github.com/candidate-ingestion/service/internal/infra/pubsub"
	"github.com/candidate-ingestion/service/internal/service"
	"github.com/go-chi/chi/v5"
)

type App struct {
	db        *db.DB
	ps        *pubsub.Client
	topic     string
	router    *chi.Mux
	whService *service.WebhookService
}

func New(database *db.DB, pubsubClient *pubsub.Client, topic string) *App {
	app := &App{
		db:     database,
		ps:     pubsubClient,
		topic:  topic,
		router: chi.NewRouter(),
	}

	app.whService = service.NewWebhookService(database, pubsubClient, topic)
	app.setupRoutes()

	return app
}

func (a *App) setupRoutes() {
	whHandler := apphttp.NewWebhookHandler(a.whService)

	a.router.Get("/health", whHandler.HandleHealth)
	a.router.Post("/webhooks/{source}", whHandler.HandleWebhook)
}

func (a *App) Router() *chi.Mux {
	return a.router
}

.PHONY: up down api worker scheduler poller test lint stress-test trigger-failure k8s-deploy k8s-delete metrics help

# Variables
DOCKER_IMAGE ?= candidate-ingestion:latest
DOCKER_COMPOSE_FILE ?= docker-compose.yml
KUBE_NAMESPACE ?= candidate-ingestion-service
STRESS_TEST_DURATION ?= 10
STRESS_TEST_CONCURRENCY ?= 50

help:
	@echo "Available commands:"
	@echo "  make up                 Start full stack (PostgreSQL, PubSub, API, Worker)"
	@echo "  make up-deps            Start just the dependencies (PostgreSQL, PubSub)"
	@echo "  make down               Stop all services"
	@echo "  make api                Run API server locally (go run ./cmd/api)"
	@echo "  make worker             Run worker locally (go run ./cmd/worker)"
	@echo "  make scheduler          Run scheduler locally (go run ./cmd/scheduler)"
	@echo "  make poller             Run outbox poller locally (go run ./cmd/poller)"
	@echo "  make metrics            Poll API /metrics endpoint (updates every 1s)"
	@echo "  make test               Run unit tests"\
	@echo "  make lint               Run linter"
	@echo "  make stress-test        Simulate traffic spike"
	@echo "  make trigger-failure    Simulate downstream failure"
	@echo "  make k8s-deploy         Deploy to Kubernetes"
	@echo "  make k8s-delete         Delete Kubernetes resources"
	@echo "  make build              Build Docker image"

# Docker Compose Commands
up-deps:
	@echo "Starting PostgreSQL and PubSub emulator..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d postgres pubsub-emulator pubsub-ui
	@echo "Waiting for services to be healthy..."
	@sleep 3
	docker-compose -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U user -d candidates -c "SELECT 1"
	@echo "Infrastructure ready: PostgreSQL on localhost:5432, PubSub on localhost:8085"

up:
	@echo "Starting full stack..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	docker-compose -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U user -d candidates -c "SELECT 1"
	@echo "Stack ready: API on http://localhost:8080, PostgreSQL on localhost:5432, PubSub on 0.0.0.0:8085"

down:
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v

# Build Docker image
build:
	docker build -t $(DOCKER_IMAGE) .

# Testing
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Stress Testing
stress-test:
	@echo "Running stress test against http://localhost:8080"
	@echo "Duration: $(STRESS_TEST_DURATION)s, Concurrency: $(STRESS_TEST_CONCURRENCY)"
	@bash -c '\
		for i in {1..$(STRESS_TEST_CONCURRENCY)}; do \
			( \
				END_TIME=$$(($$SECONDS + $(STRESS_TEST_DURATION))); \
				while [ $$SECONDS -lt $$END_TIME ]; do \
					UNIQUE_ID="$$(date +%s%N)-$$i"; \
					curl -s -X POST http://localhost:8080/webhooks/linkedin \
						-H "Content-Type: application/json" \
						-d "{ \
							\"id\": \"$$UNIQUE_ID\", \
							\"firstName\": \"Test\", \
							\"lastName\": \"User\", \
							\"email\": \"test-$$UNIQUE_ID@example.com\", \
							\"phone\": \"555-0000\", \
							\"jobTitle\": \"Engineer\" \
						}" > /dev/null; \
				done \
			) & \
		done; \
		wait \
	'
	@echo "Stress test complete."

# Failure Injection
trigger-failure:
	@echo "Simulating circuit breaker failure..."
	@bash -c '\
		for i in {1..100}; do \
			curl -s -X POST http://localhost:8080/webhooks/invalid-source \
				-H "Content-Type: application/json" \
				-d "{\"id\": \"fail-$$i\"}" > /dev/null; \
		done \
	'
	@echo "Sent 100 failing requests. Watch circuit breaker state."

# Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl create namespace $(KUBE_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f k8s/shared.yaml -n $(KUBE_NAMESPACE)
	kubectl apply -f k8s/api.yaml -n $(KUBE_NAMESPACE)
	kubectl apply -f k8s/worker.yaml -n $(KUBE_NAMESPACE)
	kubectl apply -f k8s/poller.yaml -n $(KUBE_NAMESPACE)
	kubectl apply -f k8s/scheduler.yaml -n $(KUBE_NAMESPACE)
	kubectl apply -f k8s/hpa.yaml -n $(KUBE_NAMESPACE)
	@echo "Deployment complete. Check status with:"
	@echo "  kubectl get pods -n $(KUBE_NAMESPACE)"
	@echo "  kubectl get hpa -n $(KUBE_NAMESPACE)"
	@echo "  kubectl get cronjob -n $(KUBE_NAMESPACE)"

k8s-delete:
	@echo "Deleting Kubernetes resources..."
	kubectl delete -f k8s/shared.yaml -f k8s/api.yaml -f k8s/worker.yaml -f k8s/poller.yaml -f k8s/scheduler.yaml -f k8s/hpa.yaml -n $(KUBE_NAMESPACE) --ignore-not-found

k8s-logs:
	kubectl logs -f deployment/candidate-ingestion-api -n $(KUBE_NAMESPACE)

k8s-describe:
	kubectl describe pod -n $(KUBE_NAMESPACE) -l app=candidate-ingestion

k8s-pods:
	kubectl get pods -n candidate-ingestion-service

k8s-cron:
	kubectl get cronjobs -n candidate-ingestion-service

# Local development
api:
	go run ./cmd/api

worker:
	go run ./cmd/worker

scheduler:
	go run ./cmd/scheduler

poller:
	go run ./cmd/poller

metrics:
	@bash -c '\
		while true; do \
			clear; \
			echo "=== Candidate Ingestion Metrics (refreshing every 1s, Ctrl+C to exit) ==="; \
			echo ""; \
			curl -s http://localhost:8080/metrics | jq . 2>/dev/null || echo "Error: Unable to connect to API on localhost:8080"; \
			echo ""; \
			echo "Last updated: $$(date +"%H:%M:%S")"; \
			sleep 1; \
		done \
	'

dev-setup:
	go mod download
	go mod tidy

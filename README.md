# Candidate Application Ingestion Service

High-performance Go microservice demonstrating resilience and architectural patterns. Check the `ARCHITECTURE.md` file for more information.

## How to Run locally

To see in action, use 2 terminals.

Spin up all the docker containers.

```terminal1
make up
```

After all docker containers are healthy, run the command to see metrics

```terminal1
make metrics
```

Run the stress test to continuously spam with multiple requests

```terminal2
make stress-test
```

## Architecture Patterns

### Four Independent Services

**1. API** - Webhook ingestion, Pub/Sub publish (1-5 replicas)

- Receives webhooks, validates, publishes to Pub/Sub immediately
- Returns 202 Accepted (decoupled, no database writes in request cycle)
- Rate limiting: per-IP, configurable

**2. Worker** - Message processing, dedup, storage (1-10 replicas)

- Receives Pub/Sub messages, checks dedup, atomically stores candidate + outbox event
- Dedup: composite unique constraint `(source, source_ref_id)` + transaction-scoped check
- Scales on demand

**3. Outbox Poller** - Event publishing (1 replica, no scaling)

- Polls unpublished outbox events every 5s (batch 10)
- Row-level locks (SELECT FOR UPDATE) prevent concurrent processing
- Publishes via CandidateProcessor.Notify()
- Single instance guarantees ordered delivery

**4. Scheduler** - Maintenance jobs (CronJob)

- Cleanup job at 2 AM (Singapore time)
- Deletes outbox events >30 days old
- Runs on schedule, not continuous

### 1. **Strategy Pattern**

Webhook handlers normalize payloads from different sources (LinkedIn, Google Forms) via `domain.WebhookStrategy` interface.

- `LinkedInStrategy` - Parses LinkedIn webhook structure
- `GoogleFormStrategy` - Parses Google Forms structure
- Pluggable: Add new sources without modifying core logic

### 2. **Decoupling & SLA Protection**

API endpoint immediately acknowledges webhook (202 Accepted). No database writes in request cycle.

- Parse payload + validate
- Publish to GCP Pub/Sub (fire-and-forget with timeout)
- Return immediately
- Background worker processes asynchronously

### 3. **Transactional Outbox**

Reliable message processing with at-least-once semantics.

- Worker receives Pub/Sub message
- Single atomic transaction: `INSERT candidate + INSERT outbox_event`
- Database guarantees atomicity
- Separate poller publishes outbox events to downstream

### 4. **Bulkhead Pattern**

Worker pool bounds concurrent processing using semaphores.

- Max 10 concurrent goroutines (configurable)
- Each message acquisition must acquire semaphore slot
- Prevents resource exhaustion under high load
- Implemented in `worker.Pool`

### 5. **Circuit Breaker**

Fast-fail on downstream failures.

- Wraps Pub/Sub calls
- States: Closed → Open → Half-Open → Closed
- Fail threshold: 5 failures
- Open timeout: 60s, Half-open timeout: 30s
- Implemented in `service.CircuitBreaker`

### 6. **Idempotency (Worker-Side Only)**

Handles "at-least-once" delivery from broker.

- Database constraint: composite unique `(source, source_ref_id)`
- Worker dedup check: within transaction, before INSERT
- If duplicate detected: skip processing, log as info, no metrics
- Guarantees: No duplicate applications in database

### 7. **Rate Limiting**

Per-IP rate limiting on webhook endpoint.

- Configurable requests per minute (env: `WEBHOOK_RATE_LIMIT`)
- Returns 429 Too Many Requests with `Retry-After: 60`
- Metrics: `webhooks_rate_limited` counter

### 8. **Structured Logging**

JSON logging with correlation IDs throughout pipeline.

- Logrus with RFC3339 timestamps
- Correlation ID: `idempotency_key` flows through API → Pub/Sub → Worker
- Log levels configurable (env: `LOG_LEVEL`)

### 9. **Observability**

Metrics endpoint for monitoring.

- `GET /metrics` - Returns counters for webhooks, outbox, notifications
- Counters: total_request, rate_limited, ingested, rejected, duplicate, outbox_written, etc.
- Database-backed metrics (no in-memory state)

## Tech Stack

- **Language**: Go 1.23
- **API Framework**: Chi v5
- **Database**: PostgreSQL 16
- **Message Broker**: GCP Pub/Sub (emulator for local)
- **Container**: Docker
- **Orchestration**: Kubernetes + HPA

## Project Structure

```
├── cmd/
│   ├── api/
│   │   └── main.go                 # API service: webhook ingestion + Pub/Sub publish
│   ├── worker/
│   │   └── main.go                 # Worker service: message processing + dedup
│   ├── outbox-poller/
│   │   └── main.go                 # Outbox poller: event publishing
│   └── scheduler/
│       └── main.go                 # Scheduler: maintenance jobs
├── internal/
│   ├── app/
│   │   └── app.go                  # DI for API
│   ├── config/
│   │   └── config.go               # Config loader
│   ├── di/
│   │   ├── api.go                  # API DI
│   │   ├── worker.go               # Worker DI
│   │   ├── scheduler.go            # Scheduler DI
│   │   └── outbox_poller.go        # Outbox poller DI
│   ├── domain/
│   │   ├── model/                  # Entities
│   │   ├── repo/                   # Interfaces
│   │   └── service/                # Business logic interfaces
│   ├── infra/
│   │   ├── http/
│   │   │   ├── webhooks.go         # Webhook handler
│   │   │   ├── metrics.go          # Metrics handler
│   │   │   └── ratelimit.go        # Rate limiter middleware
│   │   ├── logger/
│   │   │   └── logger.go           # Logrus factory
│   │   ├── postgres/
│   │   │   ├── db.go               # DB connection + transaction
│   │   │   ├── candidate.go        # Candidate repository
│   │   │   ├── outbox.go           # Outbox repository
│   │   │   └── metrics.go          # Metrics repository
│   │   └── pubsub/
│   │       └── client.go           # GCP Pub/Sub wrapper
│   └── usecase/
│       ├── candidate/
│       │   ├── ingestion/          # API ingestion logic
│       │   └── processing/         # Worker processing logic
│       ├── cleanup/                # Scheduler cleanup logic
│       └── metrics/                # Metrics collection
├── migrations/
│   ├── 001_init.sql                # Schema: applications, outbox
│   ├── 002_metrics.sql             # Metrics table
│   ├── 003_fix_dedup_constraint.sql # Composite unique(source, source_ref_id)
│   └── 004_add_duplicate_metric.sql # Duplicate counter
├── k8s/
│   ├── api.yaml                    # API deployment + service
│   ├── worker.yaml                 # Worker deployment
│   ├── outbox-poller.yaml          # Outbox poller deployment (replicas: 1)
│   ├── scheduler.yaml              # Scheduler CronJob
│   ├── hpa.yaml                    # HPA (API 1-5, Worker 1-10)
│   └── postgres.yaml               # PostgreSQL StatefulSet
├── Dockerfile                      # Multi-stage: api, worker, outbox-poller, scheduler
├── docker-compose.yml
├── Makefile
└── README.md
```

## Quick Start

### Local Development

```bash
# Start PostgreSQL + Pub/Sub emulator
make up

# Run services (each in separate terminal)
make api        # API service on :8080
make worker     # Worker service (background)
make scheduler  # Scheduler service (maintenance)
make outbox-poller  # Outbox poller (background)

# Test webhook ingestion
curl -X POST http://localhost:8080/webhooks/linkedin \
  -H "Content-Type: application/json" \
  -d '{
    "id": "abc123",
    "firstName": "John",
    "lastName": "Doe",
    "email": "john@example.com",
    "phone": "555-1234",
    "position": "Software Engineer",
    "sourceRefId": "linkedin-123"
  }'

# Health check
curl http://localhost:8080/health

# Metrics
curl http://localhost:8080/metrics

# Cleanup
make down
```

### Testing

```bash
# Run unit tests
make test

# Run linter
make lint

# Stress test (triggers HPA in K8s)
make stress-test

# Trigger circuit breaker failures
make trigger-failure
```

### Kubernetes Deployment

WIP - Still CrashLoopBack

```bash
# Deploy all 4 services + HPA + CronJob
make build
make k8s-deploy

# View deployments
kubectl get deploy -n candidate-ingestion-service
kubectl get pods -n candidate-ingestion-service
kubectl get hpa -n candidate-ingestion-service
kubectl get cronjobs -n candidate-ingestion-service

# View logs by service
kubectl logs -n candidate-ingestion-service deployment/candidate-ingestion-api -f
kubectl logs -n candidate-ingestion-service deployment/candidate-ingestion-worker -f
kubectl logs -n candidate-ingestion-service deployment/candidate-ingestion-poller -f
kubectl logs -n candidate-ingestion-service deployment/candidate-ingestion-scheduler -f

# Delete
make k8s-delete
```

## API Endpoints

### `POST /webhooks/{source}`

Ingests webhook from source (linkedin, google_forms).

**Response (202 Accepted):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Status Codes:**

- `202` - Accepted, processing in background
- `400` - Invalid source or payload
- `500` - Server error (unlikely, API is resilient)

### `GET /health`

Health check endpoint.

**Response (200 OK):**

```json
{
  "status": "healthy"
}
```

### `GET /metrics`

Metrics endpoint. Returns counters for entire pipeline.

**Response (200 OK):**

```json
{
  "webhooks_total_request": 4445,
  "webhooks_rate_limited": 513,
  "webhooks_ingested": 435,
  "webhooks_rejected": 0,
  "webhooks_duplicate": 2,
  "outbox_written": 435,
  "outbox_process_attempts": 437,
  "outbox_publish_success": 435,
  "outbox_publish_failed": 0,
  "notification_failed": 52,
  "outbox_cleaned": 400
}
```

**Metric Definitions:**

- `webhooks_total_request` - Total HTTP requests to /webhooks endpoint
- `webhooks_rate_limited` - Requests rejected by rate limiter
- `webhooks_ingested` - Successfully parsed & published to Pub/Sub
- `webhooks_rejected` - Failed parsing or Pub/Sub errors
- `webhooks_duplicate` - Duplicates detected (same source + source_ref_id)
- `outbox_written` - Outbox events created (new candidates stored)
- `outbox_process_attempts` - Total poll attempts on outbox
- `outbox_publish_success` - Successfully notified downstream
- `outbox_publish_failed` - Notification failures (logged, will retry)
- `notification_failed` - Persistent failures after retries
- `outbox_cleaned` - Events deleted by cleanup job (>30 days)

## Resilience Features

### Request Lifecycle

1. **Validation**: Parse payload via strategy, validate schema
2. **Publish**: Fire to Pub/Sub with circuit breaker protection (fire-and-forget, 5s timeout)
3. **Return**: 202 Accepted (does not wait for worker)

### Worker Lifecycle

1. **Receive**: Pull message from Pub/Sub subscription
2. **Acquire Slot**: Wait for semaphore (max 10 concurrent)
3. **Dedup Check**: Query candidate_applications within transaction (source + source_ref_id)
4. **Transactional Persist**: Atomic INSERT candidate + INSERT outbox_event (single transaction)
   - If duplicate detected: log as info, increment `webhooks_duplicate`, return nil (no metrics written)
   - If success: increment `outbox_written` after transaction commits
5. **ACK**: Message acknowledged to broker (on success or dedup)
6. **NAK**: Message nacked on error (will be redelivered)

### Outbox Poller Lifecycle

1. **Poll**: Every 5 seconds, fetch unpublished outbox events (batch 10)
2. **Row Lock**: Use SELECT FOR UPDATE to prevent concurrent processing
3. **Notify**: Call CandidateProcessor.Notify() for each event
4. **Mark Published**: Update outbox_events.published = true
5. **Retry**: On failure, logs warning, will retry next poll (eventual consistency)

### Scheduler Lifecycle

1. **Run**: Triggered by CronJob every day at 2 AM (Singapore time)
2. **Cleanup**: Delete outbox_events where created_at < NOW() - OUTBOX_RETENTION_DAYS
3. **Metrics**: Increment `outbox_cleaned` counter
4. **Exit**: Job complete, waits for next scheduled run

### Failure Scenarios

| Scenario                          | Handling                                       |
| --------------------------------- | ---------------------------------------------- |
| Webhook ingestion fails           | 400 Bad Request                                |
| Pub/Sub publish timeout           | Logged warning, circuit breaker tracks failure |
| Pub/Sub unreachable (5+ failures) | Circuit breaker opens, future calls fast-fail  |
| Worker crashes mid-processing     | Message redelivered by broker (at-least-once)  |
| DB connection pool exhausted      | Blocks worker (semaphore prevents cascading)   |
| Duplicate message from broker     | Dedup check skips duplicate processing         |

## Configuration

Environment Variables:

```bash
# API Service
API_PORT=8080
WEBHOOK_RATE_LIMIT=60  # Requests per minute per IP
LOG_LEVEL=info         # debug, info, warn, error

# Database
DATABASE_URL=postgres://user:password@localhost:5432/candidates?sslmode=disable

# Pub/Sub
GCP_PROJECT=test-project
PUBSUB_TOPIC=candidate-applications
PUBSUB_SUBSCRIPTION=candidate-applications-sub
PUBSUB_EMULATOR_HOST=localhost:8085  # For local testing

# Worker
WORKER_POOL_SIZE=10
MESSAGE_TIMEOUT_S=30

# Scheduler
OUTBOX_RETENTION_DAYS=30
SCHEDULER_CLEANUP_HOUR=2  # 2 AM Singapore time

# Circuit Breaker
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_OPEN_TIMEOUT_S=60
CIRCUIT_BREAKER_HALF_OPEN_TIMEOUT_S=30
```

## Performance Tuning

### Database

- Connection pool: 25 max, 5 idle
- Indexes on `source`, `created_at`, `published` flag

### Worker Pool

- Default 10 workers (configurable in `cmd/server/main.go`)
- Timeout: 30s per message

### Circuit Breaker

- Fail threshold: 5 consecutive failures
- Open timeout: 60s (before attempting half-open)
- Half-open: 3 successes to close

## Testing

### Strategy Pattern Unit Tests

```bash
go test -v ./internal/domain/...
```

### Load Testing

```bash
# Docker Compose (local)
make stress-test

# Kubernetes
make k8s-deploy
make stress-test
# Watch HPA scale pods:
# kubectl get hpa candidate-ingestion-hpa -w
```

## Monitoring

### Kubernetes HPA Metrics

```bash
# View current replica count
kubectl get deployment candidate-ingestion-service

# View HPA status
kubectl get hpa candidate-ingestion-hpa -o wide

# CPU utilization
kubectl top nodes
kubectl top pods -n default -l app=candidate-ingestion
```

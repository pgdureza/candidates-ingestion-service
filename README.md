# Candidate Application Ingestion Service

High-performance Go microservice demonstrating resilience and architectural patterns.

## Architecture Patterns

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
- Separate process publishes outbox events to downstream

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

### 6. **Idempotency**
Handles "at-least-once" delivery from broker.
- Database constraint: unique (source, source_ref_id)
- Worker dedup check: SELECT candidate_applications before INSERT
- If duplicate detected: skip processing, ACK message
- Guarantees: No duplicate applications in database

## Tech Stack

- **Language**: Go 1.23
- **API Framework**: Chi v5
- **Database**: PostgreSQL 16
- **Message Broker**: GCP Pub/Sub (emulator for local)
- **Container**: Docker
- **Orchestration**: Kubernetes + HPA

## Project Structure

```
├── cmd/server/
│   └── main.go                 # Entry point
├── internal/
│   ├── app/
│   │   └── app.go             # DI & router setup
│   ├── domain/
│   │   ├── models.go          # Entities
│   │   ├── strategy.go        # Webhook strategies
│   │   └── strategy_test.go
│   ├── handler/
│   │   └── webhooks.go        # HTTP handlers
│   ├── service/
│   │   ├── webhook.go         # Business logic
│   │   └── circuitbreaker.go  # Resilience
│   ├── infra/
│   │   ├── db/
│   │   │   ├── db.go
│   │   │   └── application.go # Transactional outbox
│   │   └── pubsub/
│   │       └── client.go      # GCP Pub/Sub wrapper
│   └── worker/
│       └── pool.go            # Bulkhead + processing
├── migrations/
│   └── 001_init.sql           # Schema
├── k8s/
│   ├── deployment.yaml        # K8s deployment
│   └── hpa.yaml               # Horizontal Pod Autoscaler
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Quick Start

### Local Development

```bash
# Start services
make up

# Run app (in separate terminal)
make run

# Or test via Docker Compose
curl -X POST http://localhost:8080/webhooks/linkedin \
  -H "Content-Type: application/json" \
  -d '{
    "id": "abc123",
    "firstName": "John",
    "lastName": "Doe",
    "email": "john@example.com",
    "phone": "555-1234",
    "jobTitle": "Software Engineer"
  }'

# Health check
curl http://localhost:8080/health

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

```bash
# Deploy
make k8s-deploy

# View pods & HPA
kubectl get pods -n default
kubectl get hpa -n default

# Watch logs
make k8s-logs

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

## Resilience Features

### Request Lifecycle

1. **Validation**: Parse payload via strategy, validate schema
2. **Publish**: Fire to Pub/Sub with circuit breaker protection (fire-and-forget, 5s timeout)
3. **Return**: 202 Accepted (does not wait for worker)

### Worker Lifecycle

1. **Receive**: Pull message from subscription
2. **Acquire Slot**: Wait for semaphore (max 10 concurrent)
3. **Dedup Check**: Query candidate_applications (source + source_ref_id)
4. **Transactional Persist**: Atomic INSERT application + INSERT outbox (single transaction)
5. **Mark Published**: Update outbox_events.published flag
6. **ACK**: Message acknowledged to broker

### Failure Scenarios

| Scenario | Handling |
|----------|----------|
| Webhook ingestion fails | 400 Bad Request |
| Pub/Sub publish timeout | Logged warning, circuit breaker tracks failure |
| Pub/Sub unreachable (5+ failures) | Circuit breaker opens, future calls fast-fail |
| Worker crashes mid-processing | Message redelivered by broker (at-least-once) |
| DB connection pool exhausted | Blocks worker (semaphore prevents cascading) |
| Duplicate message from broker | Dedup check skips duplicate processing |

## Configuration

Environment Variables:

```bash
API_PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/candidates?sslmode=disable
GCP_PROJECT=test-project
PUBSUB_TOPIC=candidate-applications
PUBSUB_EMULATOR_HOST=localhost:8085  # For local testing
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

### Database

```sql
-- Active applications
SELECT COUNT(*) FROM candidate_applications;

-- Unpublished outbox events (indicates backlog)
SELECT COUNT(*) FROM outbox_events WHERE published = false;

-- Recent applications
SELECT COUNT(*) FROM candidate_applications WHERE created_at > NOW() - INTERVAL '1 hour';
```

## Future Enhancements

- [ ] Metrics & observability (Prometheus + Grafana)
- [ ] Structured logging (Zap)
- [ ] Graceful shutdown improvements
- [ ] Dead-letter queue for failed messages
- [ ] Event sourcing audit trail
- [ ] Rate limiting per source
- [ ] Multi-region replication

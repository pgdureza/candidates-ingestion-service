# Candidate Ingestion Service Architecture

## Overview

Event-driven microservice for ingesting candidate applications from multiple sources (LinkedIn, Google Forms, etc.) with transactional reliability and idempotent processing.

```
┌──────────────────────────────────────────────────────────────┐
│                     External Sources                         │
│              (LinkedIn, Google Forms, etc.)                  │
└────────────────────────┬─────────────────────────────────────┘
                         │ webhooks
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                      API Service (1-5)                       │
│  POST /webhooks/{source}                                     │
│  • Validates payload + injects idempotency key               │
│  • Publishes to Pub/Sub (circuit breaker + timeout)          │
│  • Returns 202 Accepted (non-blocking)                       │
└────────────────┬─────────────────────────────────────────────┘
                 │ publish (at-least-once)
                 ▼
          ┌──────────────────┐
          │   Cloud Pub/Sub   │ (candidate-applications topic)
          └────────┬─────────┘
                   │ subscribe (pull)
                   ▼
┌──────────────────────────────────────────────────────────────┐
│           Worker Service (1-10 replicas)                     │
│  PubSub message handler                                      │
│  • Validates idempotency key (dedup via DB unique constraint)│
│  • Stores candidate_application + outbox_event atomically    │
│  • ACKs message on success, NAKs on failure                  │
│  • Retries with exponential backoff                          │
└────────────┬─────────────────────────────────────────────────┘
             │ atomic write (transaction)
             ▼
    ┌─────────────────────┐
    │   PostgreSQL DB     │
    ├─────────────────────┤
    │ candidate_applications
    │ outbox_events       │ (unpublished)
    │ candidate_skills    │
    │ candidate_notes     │
    └─────────────────────┘
             ▲
             │ fetch
             │ mark published (UPDATE)
             │
┌────────────┴────────────────────────────────────────────────┐
│           Outbox Poller Service (1 replica)                 │
│  Transactional outbox pattern                               │
│  • Polls unpublished events every X seconds in batches      │
│  • Notifies external systems (unreliable, async)            │
│  • Marks published after successful notification            │
└────────────┬────────────────────────────────────────────────┘
             │ notify (unreliable)
             ▼
    ┌─────────────────────┐
    │  External Systems   │
    │ (email, webhooks)   │
    │ (idempotent)        │
    └─────────────────────┘


┌─────────────────────────────────────────────────────────────┐
│           Scheduler Service (1 replica - CronJob)           │
│  Maintenance tasks on schedule                              │
│  • Cleanup: Deletes outbox events older than 30 days        │
│    (runs daily 2 AM Singapore time)                         │
└─────────────────────────────────────────────────────────────┘
```

## Services

### 1. API Service (`cmd/api/main.go`)

**Purpose:** Webhook ingestion endpoint
**Replicas:** 1-5 (scales on CPU/Memory)
**Responsibility:**

- Accept webhooks from external sources
- Validate payload structure + extract source ID
- Inject idempotency key into Pub/Sub message
- Publish to Cloud Pub/Sub (with circuit breaker protection)
- Return 202 Accepted immediately (non-blocking)

**Key files:**

- `internal/infra/http/webhooks.go` - HTTP handler
- `internal/service/webhook.go` - Business logic (publish + logging)
- `internal/di/api.go` - Dependency injection

**Reliability:**

- Circuit breaker on Pub/Sub publish (fails open, returns app_id)
- Timeout protection (30 seconds)
- Structured JSON logging with idempotency key for tracing

### 2. Worker Service (`cmd/worker/main.go`)

**Purpose:** Process messages from Pub/Sub
**Replicas:** 1-10 (scales on CPU/Memory)
**Responsibility:**

- Subscribe to Cloud Pub/Sub topic
- Validate idempotency key (dedup check)
- Reconstruct candidate application from message payload
- Store atomically: candidate_application + outbox_event in single transaction
- ACK message on success, NAK on failure

**Key files:**

- `internal/infra/pubsub/pool.go` - Worker pool + message handler
- `internal/infra/pubsub/client.go` - Pub/Sub subscription
- `internal/di/worker.go` - Dependency injection

**Deduplication:**

- Worker checks `candidate_applications.Exists(source, source_ref_id)` before insert
- DB unique constraint on `(source, source_ref_id)` prevents duplicates
- At-most-once semantics per unique application

**Transaction Pattern:**

```go
db.WithTransaction(ctx, func(txCtx context.Context) error {
    // Both operations atomic
    db.Applications().Store(txCtx, app)
    db.Outbox().Create(txCtx, outbox)
    return nil // commits both or neither
})
```

**Logging:**

- All stages logged with message_id
- Structured JSON output for observability
- Duplicate hits logged as INFO (not errors)

### 3. Outbox Poller Service (`cmd/poller/main.go`)

**Purpose:** Reliable notification delivery (transactional outbox pattern)
**Replicas:** 1 (single instance, no scaling)
**Responsibility:**

- Poll unpublished outbox events every 5 seconds
- Fetch in batches (10 events) with row-level locks
- Notify external systems (email, webhooks, etc.)
- Mark as published only after successful notification
- Retry on failure (next poll cycle)

**Key files:**

- `internal/infra/poller/poller.go` - Polling loop + notification orchestration
- `internal/infra/postgres/outbox.go` - Outbox queries (GetUnpublishedForUpdate)
- `internal/di/poller.go` - Dependency injection

**Concurrency Safety:**

- Single replica guarantees exactly one poller instance
- Transaction extended through fetch only (not notification)

**Failure Handling:**

- Notify failure: logged as WARN, event remains unpublished, retried next cycle
- Mark failure: logged as WARN, non-fatal (eventual consistency)
- Poller graceful shutdown: stops polling, in-flight notifications complete

**Notification Contract:**

- Assumes notifier is unreliable (can fail, timeout, hang)
- Assumes downstream consumers are idempotent
- Guarantees at-least-once delivery (retry on failure)

### 4. Scheduler Service (`cmd/scheduler/main.go`)

**Purpose:** Maintenance jobs on schedule
**Replicas:** 1 (CronJob, not Deployment)
**Responsibility:**

- Run cleanup job daily at 2 AM Singapore time
- Delete outbox events older than retention period (default 30 days)

**Key files:**

- `internal/usecase/cleanup/cleaner.go` - Cleanup logic
- `internal/di/scheduler.go` - Dependency injection

**Cron Expression:** `0 2 * * *` (2 AM daily, Singapore timezone)

---

## Data Flow

### Ingestion Flow (API → Worker)

```
1. POST /webhooks/linkedin
   ↓ (validate + inject idempotency key)
2. Publish to Pub/Sub (non-blocking, returns 202 + app_id)
   ↓ (at-least-once delivery)
3. Worker receives message
   ↓ (check idempotency_key)
4. Query: SELECT ... FROM candidate_applications WHERE source=? AND source_ref_id=?
   ├─ If exists: skip, log duplicate, ACK message
   └─ If new: proceed
   ↓ (start transaction)
5. INSERT candidate_application + INSERT outbox_event (atomic)
   ↓ (commit transaction)
6. ACK message
   └─ Application now queryable, notification pending
```

### Notification Flow (Outbox Poller)

```
1. Every X seconds:
2. SELECT * FROM outbox_events WHERE published=false LIMIT 10
3. For each event:
   ├─ Notify external system
   ├─ On success: Mark event as published
   └─ On failure: Log, keep unpublished (retry next cycle)
```

---

## Database Schema

### Key Tables

**candidate_applications**

```sql
id (UUID)
source (TEXT) -- 'linkedin', 'google-form'
source_ref_id (TEXT) -- external ID
email (TEXT)
first_name, last_name (TEXT)
...
UNIQUE (source, source_ref_id) -- deduplication
```

**outbox_events**

```sql
id (UUID)
event_type (TEXT) -- 'application.created'
payload (JSONB) -- serialized candidate
published (BOOLEAN) DEFAULT false
published_at (TIMESTAMP)
created_at (TIMESTAMP)
...
INDEX (published, created_at) -- for polling query
```

---

## Failure Scenarios

### API Publishing Fails

- Circuit breaker opens after X consecutive failures
- Returns 503 when circuit breaker is open
- Next cycle: circuit breaker half-open attempts recovery

### Worker Message Processing Fails

- NAK message (returns to Pub/Sub)
- Pub/Sub has built in retry mechanism
- Duplicate app_id triggers skip (idempotency), ACK succeeds

### Notification Service Unavailable

- Outbox Poller marks event unpublished
- Logged as WARN (non-fatal)
- Retried in next poll cycle (5 seconds later)
- Eventual consistency: notification delivered when service recovers

### Outbox Poller Crashes

- Pod restarts (Kubernetes)
- On recovery: fetches pending events again
- No duplicate notifiers (row-level locks prevent concurrent fetch)
- Potential double-notify if notification succeeded but mark failed
  - Mitigated by: downstream idempotency + eventual consistency semantics

---

## Scaling Strategy

### API Service

- **Min:** 1 replica (handle ~100 req/s per instance)
- **Max:** 5 replicas
- **Metrics:** CPU 70%, Memory 80%
- **Scale up:** Instantly on threshold
- **Scale down:** 5 minutes stabilization

### Worker Service

- **Min:** 1 replica (handle ~1000 msg/s per instance)
- **Max:** 10 replicas
- **Metrics:** CPU 70%, Memory 80%
- **Worker threads:** 10 per instance (configurable)
- **Message timeout:** 30 seconds per message

### Outbox Poller

- **Replicas:** 1 (single instance, no scaling)
- **Polling interval:** 5 seconds
- **Batch size:** 10 events per poll
- **DB pool:** 5 connections (minimal, mostly idle)

### Scheduler

- **Runs:** CronJob (not continuous)
- **Schedule:** 2 AM Singapore time daily
- **Retention:** 30 days (configurable)

---

## Deployment

### Local Development

```bash
make up                 # Start full stack with Docker Compose
make api                # Run API locally
make worker             # Run worker locally
make poller             # Run poller locally
make scheduler          # Run scheduler locally (single run)
```

### Kubernetes

```bash
make k8s-deploy         # Deploy all services + HPA + CronJob
make k8s-delete         # Tear down all services
kubectl get pods        # Check status
kubectl get hpa         # Check autoscaling
kubectl get cronjob     # Check scheduler
```

### Docker

```bash
docker build -t candidate-ingestion:latest .
docker run candidate-ingestion:latest ./api
docker run candidate-ingestion:latest ./worker
docker run candidate-ingestion:latest ./poller
docker run candidate-ingestion:latest ./scheduler
```

---

## Observability

### Structured Logging

- All services log JSON (logrus formatter)
- RFC3339 timestamps
- Fields: level, msg, time, error, idempotency_key, message_id, app_id, source, email

### Key Metrics

- API: request latency, Pub/Sub publish errors (circuit breaker state)
- Worker: message processing latency, dedup hits, DB write errors
- Outbox Poller: poll latency, notification errors, batch size
- Scheduler: cleanup duration, rows deleted

### Tracing

- Same key in logs enables full lifecycle observability

---

## Configuration

### Environment Variables

```bash
# Common
DATABASE_URL=postgresql://user:pass@localhost/candidates
LOG_LEVEL=info

# API
CIRCUIT_BREAKER_THRESHOLD=5      # failures before opening
CIRCUIT_BREAKER_TIMEOUT=30s      # recovery wait time
PUBSUB_PUBLISH_TIMEOUT=30s       # max time to publish

# Worker
GCP_PROJECT=my-project
PUBSUB_TOPIC=candidate-applications
WORKER_COUNT=10                  # goroutines per instance
WORKER_TIMEOUT=30s               # per-message timeout

# Outbox Poller
(inherits DATABASE_URL, LOG_LEVEL)

# Scheduler
OUTBOX_RETENTION_TIME_S=30         # cleanup retention
```

---

## Clean Architecture

Codebase follows **Clean Architecture** principles with strict layer separation and inward dependency flow.

### Layers

**1. Domain Layer** (`internal/domain/`)

Pure business logic. No external dependencies, frameworks, or side effects.

- **Models** (`domain/model/`) - Core entities: `Candidate`, `OutboxEvent`
- **Repositories** (`domain/repo/`) - Interfaces defining data access contracts (`DB`, `CandidateRepository`, `OutboxRepository`)
- **Services** (`domain/service/`) - Domain interfaces: `CandidateIngester`, `CandidateParser`, `Publisher`, `CircuitBreaker`, `Logger`

**Key rule:** Domain defines what the system **does**, not **how** it does it. Interfaces are thin, focused, and testable.

Example (`domain/service/candidate.go`):

```go
type CandidateIngester interface {
    Ingest(ctx context.Context, source string, payload []byte) (string, error)
}
```

**2. Usecase Layer** (`internal/usecase/`)

Application-specific business logic. Orchestrates domain logic, enforces rules, implements workflows.

- **Candidate ingestion** (`usecase/candidate/ingestion/`) - Parse + publish webhook
- **Candidate processing** (`usecase/candidate/processing/`) - Worker message handling (dedup, store, notify)
- **Circuit breaker** (`usecase/circuitbreaker/`) - Failure detection + recovery
- **Cleanup** (`usecase/cleanup/`) - Maintenance jobs (retention policy)

**Import rule:** May only depend on `domain/`. Cannot reference `infra/`.

Example (`usecase/candidate/ingestion/0_init.go`):

```go
type Ingester struct {
    db        repo.DB                     // domain interface
    publisher service.Publisher           // domain interface
    breaker   service.CircuitBreaker     // domain interface
    logger    service.Logger              // domain interface
}
```

**3. Infrastructure Layer** (`internal/infra/`)

Concrete implementations of domain interfaces. Frameworks, drivers, external service bindings.

- **HTTP** (`infra/http/`) - Web handlers (Chi router)
- **Postgres** (`infra/postgres/`) - Database implementation
- **Pub/Sub** (`infra/pubsub/`) - Google Cloud Pub/Sub client
- **Logger** (`infra/logger/`) - Logrus JSON logging
- **Poller** (`infra/poller/`) - Outbox polling loop
- **Parser** - Implicit in usecase (strategies for LinkedIn, Google Forms)

**Import rule:** May import from `domain/` and `usecase/`. Cannot import from other `infra/` packages (except via domain interfaces).

Example (`infra/postgres/candidate.go`): Implements `domain/repo.CandidateRepository` interface.

**4. Dependency Injection** (`internal/di/`)

Wires up all layers. Constructs object graphs for each service (API, Worker, Scheduler, Poller).

**Import rule:** May import from anywhere (`domain/`, `usecase/`, `infra/`, `config/`). Only entry point that knows about all layers.

Example (`di/api.go`):

```go
func NewAPI(ctx context.Context, cfg *config.Config) (*API, error) {
    logger := logger.New(cfg.LogLevel)              // infra
    database, err := postgres.New(cfg.DatabaseURL)  // infra
    ps, err := pubsub.New(ctx, cfg.GCPProject)      // infra

    cb := circuitbreaker.NewCircuitBreaker(...)     // usecase
    ingester := candidateingestion.New(             // usecase
        database, ps, cfg.Topic, cb, logger,
    )

    handler := apphttp.NewWebhookHandler(           // infra
        ingester, logger,
    )
    // wire routes...
}
```

### Dependency Direction

Strict unidirectional dependency flow (inward):

```
External World
      ↑ (imports)
      │
┌─────┴──────────────────────────────┐
│  Config + Main (cmd/)              │
└─────┬──────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────┐
│  Dependency Injection (di/)         │
│  Wires domain + usecase + infra     │
└─────┬──────────────────────────────┘
      │
      ├──────────────────┬──────────────────────┐
      │                  │                      │
      ▼                  ▼                      ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Usecase    │  │  Infra       │  │  Domain      │
│   (business  │  │  (concrete   │  │  (pure logic)│
│   workflows) │  │   impl)      │  └──────────────┘
└──────┬───────┘  └──────┬───────┘       ▲
       │                 │               │
       └─────────────────┴───────────────┘
              (both depend on domain)
```

**Import rules enforced:**

| Layer   | Can Import      | Cannot Import        |
| ------- | --------------- | -------------------- |
| Domain  | Nothing         | (pure, no deps)      |
| Usecase | Domain          | Infra                |
| Infra   | Domain, Usecase | Other Infra packages |
| DI      | Everything      | (wiring layer)       |
| Main    | DI, Config      | Business logic       |

---

## Design Decisions

### Why Transactional Outbox?

- **Problem:** Ensuring notification delivery if service crashes after storing data
- **Solution:** Write outbox event atomically with application data
- **Benefit:** Decouples notification reliability from API response time

### Why Separate Outbox Poller?

- **Problem:** Multiple worker replicas would race to fetch same events
- **Solution:** Single dedicated poller service
- **Benefit:** Eliminates concurrency complexity, clearer semantics

### Why Eventual Consistency on Notifications?

- **Problem:** Notification service may be slow/unreliable
- **Solution:** Decouple from API response path (separate poller)
- **Benefit:** API responds fast, notifications retry indefinitely

### Why Dedup Only on Worker?

- **Problem:** Idempotency table adds DB overhead, API complexity
- **Solution:** Use unique constraint on (source, source_ref_id)
- **Benefit:** Simpler API, cheaper storage, same guarantee

### Why Scheduler as Separate Service?

- **Problem:** CronJob in worker adds complexity, mixing concerns
- **Solution:** Dedicated scheduler (CronJob in Kubernetes)
- **Benefit:** Maintenance jobs independent of worker scaling, cleaner separation

### Why Strict Layer Boundaries (Clean Architecture)?

- **Problem:** Without boundaries, code couples to frameworks (Postgres, Chi, Logrus). Testing becomes hard, swapping implementations is painful.
- **Solution:** Enforce inward dependency flow. Domain has zero dependencies. Usecase depends only on domain interfaces. Infra implements domain interfaces.
- **Benefit:**
  - Domain logic testable without mocks (pure functions)
  - Usecase testable with minimal mocks (fake domain interfaces)
  - Infra testable via integration tests
  - Can swap Postgres → MySQL, Chi → Gin, Logrus → Zap with 0 domain/usecase changes
  - New developers can find features by layer (data access = `domain/repo/`, business rules = `usecase/`, persistence = `infra/postgres/`)

### Why Four Independent Services Instead of Monolith?

- **API (1-5 replicas):** Stateless, scales instantly on traffic spikes
- **Worker (1-10 replicas):** CPU-bound message processing, independent scaling from API
- **Outbox Poller (1 replica):** Cannot scale (single instance prevents duplicate notifications). Single responsibility = easy to reason about
- **Scheduler (CronJob):** No need for continuous running. Kubernetes runs on schedule, scales to 0 otherwise

**Benefit:** Each service has its own scaling profile. API doesn't waste resources during off-peak. Worker throughput independent of webhook arrival rate.

### Why Separate Notification from Storage (Eventual Consistency)?

- **Problem:** If notification service is slow, API response times degrade
- **Solution:** Store atomically in transaction, notify asynchronously via separate poller
- **Benefit:** API responds in ~10ms. Notifications retry for up to 30 days. Users see data instantly, notifications eventually reach.

### Why Domain Interfaces?

- **Example:** `domain/service/Publisher` interface (not concrete `pubsub.Client`)
- **Usecase imports:** `service.Publisher` (interface)
- **DI wires:** `pubsub.Client` (concrete) into `service.Publisher` (interface)
- **Benefit:** Usecase doesn't know about Pub/Sub. Can mock for tests. Can replace with Kafka without touching usecase code.

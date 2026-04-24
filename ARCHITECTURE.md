# Architecture Overview

## System Diagram

```
┌─────────────┐
│   Webhook   │
│  Clients    │
│ (LinkedIn,  │
│ Google)     │
└──────┬──────┘
       │ HTTP POST /webhooks/{source}
       │ (202 Accepted, Fast Return)
       ▼
┌──────────────────────────────────────┐
│          API Server (Chi)            │
├──────────────────────────────────────┤
│ • Parse Payload (Strategy Pattern)   │
│ • Validate & Normalize              │
│ • Check Idempotency Key             │
│ • Store Idempotency (DB)            │
│ • Publish to Pub/Sub                │
│   (Circuit Breaker Protection)      │
│ • Return 202 Accepted               │
└──────────┬─────────────┬────────────┘
           │             │
        (DB)           (Pub/Sub)
           │             │
           ▼             ▼
┌──────────────────┐  ┌──────────────────────┐
│   PostgreSQL     │  │  GCP Pub/Sub Emulator│
│                  │  │                      │
│ Tables:          │  │ Topic:               │
│ • Applications   │  │ candidate-applications
│ • Outbox Events  │  │                      │
│ • Idempotency    │  │ Subscription:        │
│   Keys           │  │ candidate-apps-sub   │
└────────┬─────────┘  └──────────┬───────────┘
         ▲                       │
         │                       │ (Async Pull)
         │                       ▼
         │            ┌──────────────────────┐
         │            │   Worker Pool        │
         │            ├──────────────────────┤
         │            │ • Semaphore (N=10)   │
         │            │ • Dequeue Message    │
         │            │ • Double-Check       │
         │            │   Idempotency        │
         │            │ • Process & Normalize│
         │            │ • TX: Store App +    │
         │            │   Outbox Event       │
         │            │ • Mark Published     │
         │            │ • ACK Message        │
         │            └──────────┬───────────┘
         │                       │
         └───────────────────────┘
            (Atomic Transaction)
```

## Request Flow

### Phase 1: Webhook Ingestion (Synchronous, SLA Protected)

1. **Client** sends webhook to `POST /webhooks/{source}` with `Idempotency-Key` header
2. **API Handler** extracts source & payload
3. **Strategy Factory** selects appropriate strategy (LinkedIn, Google Forms)
4. **Strategy.Parse()** normalizes payload into `CandidateApplication`
5. **Idempotency Check**: Query `idempotency_keys` table
   - Cache hit? Return 202 + cached app ID
   - Cache miss? Continue
6. **Store Idempotency**: INSERT `idempotency_keys` (request_id → app_id)
7. **Generate ID**: Create UUID for application
8. **Publish to Pub/Sub** with Circuit Breaker:
   - Attempt to publish message (5s timeout)
   - If success: increment success counter, reset failure counter
   - If timeout/error: increment failure counter
   - If 5 failures: open circuit, fast-fail future calls
   - Half-open state: allow trial requests, close after 3 successes
9. **Return 202 Accepted** immediately (does not wait for worker)

### Phase 2: Background Processing (Asynchronous, At-Least-Once)

1. **Worker Pool** starts, initializes semaphore with 10 slots
2. **Subscription Receive Loop** pulls messages from Pub/Sub
3. For each message:
   a. **Acquire Semaphore Slot** (block if all 10 in use → bulkhead protection)
   b. **Extract Idempotency Key** from message
   c. **Idempotency Double-Check**: SELECT from `idempotency_keys`
      - If exists: log duplicate, skip processing, ACK
      - If new: continue
   d. **Reconstruct Application** from message JSON
   e. **Begin Transaction**: Start SQL TX
   f. **Insert Application**: INSERT into `candidate_applications`
   g. **Create Outbox Event**: INSERT into `outbox_events` (same TX)
   h. **Commit Transaction**: Atomic write
   i. **Mark Published**: UPDATE outbox_events.published = true
   j. **ACK Message**: Tell Pub/Sub message was processed
   k. **Release Semaphore Slot**: Other workers can proceed

## Failure Scenarios & Recovery

### Scenario 1: Network Partition (Webhook → Pub/Sub)
- API publishes with 5s timeout
- Circuit breaker catches error, increments failure counter
- After 5 failures: circuit opens, future calls fast-fail (no retry, immediate error response)
- Client sees no change (API still returns 202, retries later)
- Idempotency key stored in DB acts as safety net
- When Pub/Sub recovers: circuit transitions to half-open, trial call succeeds, closes

### Scenario 2: Database Overload (Worker → DB)
- Worker goroutine blocks on DB connection pool
- Semaphore slots accumulate (up to 10)
- When 10 slots full: Pub/Sub delivery pauses (backpressure)
- DB bottleneck resolved → workers flush backlog
- No message loss (ack only after successful DB insert)

### Scenario 3: Worker Crash
- Message in-flight processing stops
- Pub/Sub subscription timeout (AckDeadline = 60s)
- Message redelivered to another worker
- Idempotency double-check: skip duplicate
- No data loss, automatic retry

### Scenario 4: Duplicate Webhook (Idempotent Clients)
- Client sends same webhook twice with same `Idempotency-Key`
- API Phase 1: Second request hits cache, returns same app ID (no re-publish)
- No duplicate message in Pub/Sub
- Clean idempotency at API boundary

### Scenario 5: Webhook Lost in Transit
- Client sees timeout/error
- Client retries with new request (new ID or same key depending on implementation)
- If same key: API cache hit, returns app ID
- If new key: Different application created (acceptable, user intended to retry)

## Data Consistency Guarantees

### Transactional Outbox Property
```
TX {
  INSERT candidate_applications (id, source, source_ref_id, ...)
  INSERT outbox_events (application_id, event_type, payload, published=false)
} COMMIT
```
- Both succeed atomically or both roll back
- Even if worker crashes after INSERT application, outbox event exists in DB
- Separate process can query unpublished events and retry
- Guarantees: No orphaned applications without outbox, no double-processing

### Idempotency Property
- Idempotency key stored BEFORE Pub/Sub publish
- If Pub/Sub fails but DB succeeds: key + app exist, duplicate publish is rejected
- If DB fails but Pub/Sub succeeds: key missing, message redelivered, reprocessed
- Idempotency double-check in worker prevents duplicate inserts

## Concurrency & Resource Limits

### Bulkhead (Semaphore)
- Slots: 10 concurrent workers
- Each message acquisition requires 1 slot
- Blocks if all 10 in use
- Prevents goroutine explosion (safeguards against resource exhaustion)

### Database Pool
- Max open: 25 connections
- Max idle: 5 connections
- Prevents connection leaks, ensures resource fairness

### Circuit Breaker
- Fail threshold: 5 consecutive failures
- Open timeout: 60s (wait before trying half-open)
- Half-open success threshold: 3 successes to close
- Prevents cascading failures to downstream Pub/Sub

## Scalability Patterns

### Kubernetes HPA
```yaml
minReplicas: 2
maxReplicas: 10
metrics:
  - CPU: 70%
  - Memory: 80%
scaleUp: aggressive (100% increase per 30s)
scaleDown: conservative (50% decrease per 60s)
```
- As CPU/Memory exceed thresholds, new pods spin up
- Each pod: 10 worker goroutines
- Max capacity: 10 pods × 10 workers = 100 concurrent workers
- Messages distributed via subscription (Pub/Sub handles load balancing)

## Monitoring & Observability

### Key Metrics
1. **API Layer**
   - `POST /webhooks/{source}` latency (should be <50ms)
   - Idempotency cache hit rate
   - Circuit breaker state transitions

2. **Worker Layer**
   - Semaphore utilization (% of 10 slots occupied)
   - Message processing latency (should be <30s)
   - Error count per source

3. **Database**
   - Connection pool utilization
   - Unpublished outbox events (backlog indicator)
   - Idempotency key size

4. **Pub/Sub**
   - Message lag (age of oldest unprocessed message)
   - Subscription push backlog

### Recommended Alerts
- Unpublished outbox events > 100 (worker lag)
- Circuit breaker open (Pub/Sub unhealthy)
- Worker latency > 30s (database slow)
- Semaphore at 10 (capacity bottleneck)

## Testing Strategy

### Unit Tests
- `domain.strategy_test.go`: Webhook parsing strategies
- `service.circuitbreaker_test.go`: Circuit breaker state machine
- `infra.db.idempotency_test.go`: Idempotency logic (integration)

### Integration Tests
- Full flow: webhook ingestion → Pub/Sub → worker → DB
- Failure injection: network partitions, DB errors, circuit breaker trips
- Load tests: stress-test command simulates traffic spikes

### Chaos Engineering
- Kill worker pods → messages redelivered
- Fill DB connection pool → backpressure observed
- Trigger circuit breaker → observe fast-fail behavior

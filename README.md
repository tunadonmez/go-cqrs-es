# Wallet / Payment Ledger Service — CQRS & Event Sourcing

A wallet and payment-ledger system built with **CQRS (Command Query Responsibility Segregation)** and **Event Sourcing** using Go microservices.

## Architecture

**Command flow:**

```
HTTP
  → Controller
    → CommandDispatcher
      → CommandHandler
        → WalletAggregate (raises events)
          → EventStore (MongoDB, single atomic write per event: event + PENDING outbox entry)
```

**Write-side aggregate load flow:**

```
CommandHandler
  → WalletEventSourcingHandler.GetByID
    → load latest snapshot from MongoDB snapshots collection (if present)
    → replay only events with version > snapshot.version
    → fallback to full event-stream replay when no snapshot exists
```

**Asynchronous publishing (new):**

```
OutboxPublisher (background worker)
  → reads PENDING outbox entries from the event store
    → publishes to Kafka (key = aggregate ID)
      → marks entries PUBLISHED
```

**Read-model projection:**

```
Kafka
  → WalletEventConsumer
    → WalletEventHandler
      → DB transaction:
          • INSERT into processed_events (ON CONFLICT DO NOTHING) — inbox check
          • apply projection to Wallet / Transaction tables
        → COMMIT
```

**Query flow:** HTTP → Controller → QueryDispatcher → QueryHandler → PostgreSQL read model → HTTP response

### Project Structure

```
go-cqrs-es/
├── cqrs-core/      Core CQRS/ES framework (interfaces, dispatchers, aggregate root)
├── wallet-common/  Shared wallet events and DTOs
├── wallet-cmd/     Write service — MongoDB + OutboxPublisher → Kafka
└── wallet-query/   Read service — PostgreSQL + idempotent Kafka consumer
```

### Tech Stack

| Component   | Technology           | Purpose                                  |
|-------------|----------------------|------------------------------------------|
| Language    | Go 1.26.1            | All services                             |
| Event store | MongoDB              | Immutable wallet event log + outbox      |
| Read model  | PostgreSQL (GORM)    | Wallet balances, ledger, inbox           |
| Messaging   | Apache Kafka (KRaft) | Event distribution                       |
| HTTP        | Gin                  | REST API routing                         |
| Observability| log/slog, Health, Metrics | Structured logging, health checks, simple metrics |

## Reliability Improvements

### Transactional Outbox (write side)

Events are persisted and registered for publishing in a single atomic MongoDB write. Kafka delivery is performed asynchronously by the `OutboxPublisher`, decoupling the command path from broker availability.

- Each event document in the `eventStore` collection carries its own outbox state (`publishStatus`, `publishedAt`, `attempts`, `lastError`). Persisting an event is therefore equivalent to creating a `PENDING` outbox entry in the same write.
- `WalletEventStore.SaveEvents` no longer calls the producer. If the HTTP request returns success, the events are durable and guaranteed to be published eventually.
- `OutboxPublisher` polls for `PENDING` entries in insertion order (by `_id`), publishes to Kafka (partition key = aggregate ID), and flips entries to `PUBLISHED` on success. On failure it increments `attempts` and leaves the entry `PENDING` for a retry on the next tick.
- Because the publisher aborts a batch at the first failure, per-aggregate ordering is preserved even under partial Kafka outages.

### Aggregate Snapshots (write side)

Snapshotting reduces aggregate rehydration cost on the command side.

- MongoDB remains event-sourced: the event log in `eventStore` is still the source of truth.
- A separate MongoDB `snapshots` collection stores the latest materialized state for each wallet aggregate together with the aggregate version captured in that snapshot.
- When `WalletEventSourcingHandler.GetByID(...)` loads an aggregate, it first tries the latest snapshot. If found, it restores the aggregate state from that snapshot and replays only events whose version is greater than the snapshot version.
- If no snapshot exists, aggregate loading behaves exactly as before and replays the full event stream from version `0`.
- Snapshot creation is threshold-based in the write service. The current implementation writes a fresh snapshot every 50 persisted events for a given aggregate.

This keeps correctness simple: snapshots are only a cache of derived aggregate state, not a second source of truth, and the final aggregate must always match what a full replay would have produced.

### Idempotent Projections (read side)

The projection pipeline is safe under at-least-once delivery.

- A `processed_events` table in PostgreSQL records every `(event_id, event_type, aggregate_id, version, processed_at)` that has been projected.
- Projections run inside a single DB transaction that first `INSERT … ON CONFLICT DO NOTHING` into `processed_events`. If the insert affects zero rows, the event is a duplicate and the projection body is skipped entirely.
- Projection writes and inbox writes commit together, so a crash between the two is impossible.

### Retry / Dead-Letter Handling (read side)

The live Kafka consumer now uses an explicit fetch/process/commit flow instead of relying on implicit offset handling.

- Kafka offsets are committed **only after** a message is either:
  - projected successfully, or
  - persisted to the dead-letter table
- Projection failures are retried locally up to **3 attempts** with a short fixed backoff.
- Permanent consumer failures such as malformed envelopes, unknown event types, or invalid event payloads are **not** retried repeatedly; they are dead-lettered immediately.
- If the message still fails after the retry budget is exhausted, it is written to PostgreSQL `dead_letter_events` and then the Kafka offset is committed so the consumer can keep moving.

This keeps read-model writes safe:

- a failed event never reaches `processed_events`
- a failed event never partially mutates the read model because projection work still runs inside `applyIdempotent`
- a successfully processed redelivery remains safe because idempotency is unchanged

#### Inspecting failed events

Dead-lettered messages are stored in PostgreSQL in `dead_letter_events`. Each row includes:

- `event_id`, `event_type`, `aggregate_id`
- Kafka `topic`, `partition`, `offset`, `consumer_group`
- `failure_kind` (`permanent` or `retries_exhausted`)
- `retry_attempts`
- `last_error`
- the original Kafka payload
- failure timestamps

Example inspection query:

```sql
SELECT dead_letter_key, event_id, event_type, aggregate_id, failure_kind,
       retry_attempts, last_error, dead_lettered_at
FROM dead_letter_events
ORDER BY dead_lettered_at DESC
LIMIT 20;
```

### Read-Model Rebuild / Replay

The read model can be rebuilt at any time from the event store, independent of Kafka.

**What it is.** A one-shot CLI mode on `wallet-query` that connects directly to MongoDB (the source of truth), truncates the read-model tables (and the `processed_events` inbox), and streams every event through the **same** `WalletEventHandler` methods the live Kafka consumer uses. There is only one projection path in the codebase; replay just feeds it from a different source.

**Why it exists.**

- Projection bug fix — ship corrected handler code, then rebuild the read model from the truth.
- Schema change — add/remove columns, then rebuild to backfill.
- Disaster recovery — if Postgres is lost, rebuilding from the event store restores the ledger.
- Onboarding a new read model — spin up a fresh Postgres and replay history into it.

**How to run it.**

```bash
# Full rebuild (truncates wallets, transactions, processed_events)
cd wallet-query
POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=walletLedger port=5432 sslmode=disable TimeZone=UTC" \
MONGODB_URI="mongodb://root:root@localhost:27017/walletLedger?authSource=admin" \
go run . --replay

# Rebuild for a single aggregate only
go run . --replay --aggregate=<wallet-id>
```

`--replay` is a one-shot mode: the binary performs the rebuild, logs progress, and exits. Kafka is **not** touched. Running replay multiple times is safe — each run resets the target scope (full or single aggregate) before replaying, so projections always start from a clean slate and idempotent `applyIdempotent` records match.

Replay does **not** clear `dead_letter_events`; rebuilds restore the read model, while dead-letter rows remain available for diagnosis.

Example log output:

```
replay: START scope=ALL total_events=12345
replay: read model reset (scope=ALL)
replay: progress 500/12345 events
...
replay: DONE scope=ALL events=12345 duration=4.812s
```

### Stable Event Identifiers

Every domain event carries a per-event `EventID` (128-bit random, populated on `RaiseEvent` if missing). The identifier is propagated through:

1. The event store document (`eventId` column + inside the embedded `eventData`).
2. The Kafka envelope (`eventId` at the envelope level, plus inside the event payload).
3. The projection inbox (`processed_events.event_id`).

Commands keep their own identifier contract (`IdentifiedCommand`) and remain unchanged.

## Observability

The system includes lightweight built-in observability for the critical runtime paths. It is intentionally simple: structured JSON logs, minimal health/readiness endpoints, and process-local JSON counters.

### Structured Logging
Both services use structured JSON logging via `log/slog`. The logger is tagged with the service name (`wallet-cmd` or `wallet-query`), and critical execution paths log stable identifiers where available:

- commands: `commandType`, `commandId`, `aggregateId`
- events: `eventType`, `eventId`, `aggregateId`, `aggregateType`, `version`
- Kafka consumer: `topic`, `groupId`, `partition`, `offset`, `eventId`, `eventType`
- replay: `scope`, processed counts, duration

Important flows currently covered:

- command received, handling started, handled, failed
- event persisted / persistence failed
- outbox publish attempt / published / failed
- Kafka message consumed / retry / dead-letter / commit failure
- projection applied / duplicate skipped / projection failed
- replay started / progress / completed / failed

### Health & Readiness Endpoints
Both `wallet-cmd` and `wallet-query` expose health and readiness checks:
- `GET /health`: Returns `200 OK` with `{"status": "UP"}` if the service is running.
- `GET /ready`: Returns `200 OK` if the service can still reach its primary dependency (MongoDB for `wallet-cmd`, PostgreSQL for `wallet-query`). Readiness uses short dependency pings rather than deep application probes.

### Lightweight Metrics
A simple in-memory counter set is exposed at `GET /metrics` as JSON. These counters are process-local snapshots, not Prometheus metrics and not cluster-wide aggregates.

The snapshot includes the original counters plus a few extra operational counters:

- `commands_received`, `commands_succeeded`, `command_failures`
- `events_persisted`, `event_persist_failures`
- `outbox_publish_attempts`, `produced_events`, `produce_failures`
- `kafka_messages_consumed`, `kafka_message_failures`, `kafka_retry_attempts`
- `projection_attempts`, `processed_events`, `skipped_events`, `failed_events`
- `replay_runs`, `replay_failures`, `replay_events_processed`
- `dead_lettered_events`, `dead_letter_save_failures`
- `snapshots_loaded`, `snapshots_created`, `snapshot_full_replays`

### Replay Observability
The replay process logs:

- start with replay scope and total event count
- read-model reset confirmation
- progress every 500 events, plus the final event
- completion with duration
- failure with processed count and the projection error context

## Getting Started

### Prerequisites

- Go 1.26.1+
- MongoDB, PostgreSQL, and Apache Kafka running locally (or via containers)

### Run Locally

```bash
# Command service
cd wallet-cmd
PORT=5000 \
MONGODB_URI="mongodb://root:root@localhost:27017/walletLedger?authSource=admin" \
KAFKA_BOOTSTRAP_SERVERS="localhost:9092" \
go run .

# Query service
cd ../wallet-query
PORT=5001 \
POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=walletLedger port=5432 sslmode=disable TimeZone=UTC" \
KAFKA_BOOTSTRAP_SERVERS="localhost:9092" \
KAFKA_GROUP_ID="walletConsumer" \
go run .
```

### Run with Docker

```bash
docker-compose up
```

### Build

```bash
cd wallet-cmd && go build -o wallet-cmd .
cd wallet-query && go build -o wallet-query .
```

### Test

```bash
go test ./cqrs-core/... ./wallet-common/... ./wallet-cmd/... ./wallet-query/...
```

## API Reference

### Command Service (port 5000)

#### Create Wallet

```
POST /api/v1/wallets
```

```json
{
  "owner": "Jane Doe",
  "currency": "USD",
  "openingBalance": 500.0
}
```

#### Credit Wallet

```
PUT /api/v1/wallets/:id/credit
```

```json
{
  "amount": 200.0,
  "reference": "topup-001",
  "description": "manual top-up"
}
```

#### Debit Wallet

```
PUT /api/v1/wallets/:id/debit
```

```json
{
  "amount": 100.0,
  "reference": "purchase-001",
  "description": "merchant charge"
}
```

#### Transfer Between Wallets

```
POST /api/v1/wallets/:id/transfer
```

```json
{
  "destinationWalletId": "wallet-456",
  "amount": 75.0,
  "reference": "transfer-001",
  "description": "settlement transfer"
}
```

### Query Service (port 5001)

| Method | Path                               | Description                   |
|--------|------------------------------------|-------------------------------|
| `GET`  | `/api/v1/wallets`                  | List all wallets              |
| `GET`  | `/api/v1/wallets/:id`              | Get wallet details            |
| `GET`  | `/api/v1/wallets/:id/balance`      | Get wallet balance            |
| `GET`  | `/api/v1/wallets/:id/transactions` | Get wallet transaction history |

#### Wallet list query parameters

`GET /api/v1/wallets` now supports simple pagination, filtering, and sorting:

- `page`: 1-based page number. Default `1`.
- `pageSize`: page size. Default `20`, max `100`.
- `sortBy`: `createdAt`, `balance`, or `owner`. Default `createdAt`.
- `sortOrder`: `asc` or `desc`. Default `asc`.
- `currency`: exact wallet currency filter, for example `USD`.

Example requests:

```http
GET /api/v1/wallets?page=1&pageSize=10
GET /api/v1/wallets?currency=USD&sortBy=balance&sortOrder=desc&page=2&pageSize=5
```

List responses now include a `pagination` object alongside the existing `wallets` array. The response body stays backward-compatible for existing clients that only read `wallets`.

#### Wallet transaction history query parameters

`GET /api/v1/wallets/:id/transactions` now supports:

- `page`: 1-based page number. Default `1`.
- `pageSize`: page size. Default `20`, max `100`.
- `sortBy`: `occurredAt`, `eventVersion`, or `amount`. Default `occurredAt`.
- `sortOrder`: `asc` or `desc`. Default `asc`.
- `type`: `OPENING_BALANCE`, `CREDIT`, `DEBIT`, `TRANSFER_IN`, or `TRANSFER_OUT`.
- `occurredFrom`: inclusive lower bound for `occurredAt`. Accepts RFC3339 or `YYYY-MM-DD`.
- `occurredTo`: inclusive upper bound for `occurredAt`. Accepts RFC3339 or `YYYY-MM-DD`.

Example requests:

```http
GET /api/v1/wallets/wallet-123/transactions?page=1&pageSize=25
GET /api/v1/wallets/wallet-123/transactions?type=DEBIT&sortBy=amount&sortOrder=desc
GET /api/v1/wallets/wallet-123/transactions?occurredFrom=2026-04-01&occurredTo=2026-04-30
GET /api/v1/wallets/wallet-123/transactions?type=TRANSFER_OUT&occurredFrom=2026-04-01T00:00:00Z&sortBy=occurredAt&sortOrder=desc
```

Transaction history responses now include:

- `transactions`: the existing transaction array
- `pagination`: page/pageSize/sort metadata plus `hasMore`
- `filters`: the applied transaction filter values

### Response Codes

- **204** — No results found
- **409** — Optimistic concurrency conflict

## Domain Events

| Event                 | Description                               |
|-----------------------|-------------------------------------------|
| `WalletCreatedEvent`  | Wallet created with initial balance       |
| `WalletCreditedEvent` | Wallet credited directly or by transfer   |
| `WalletDebitedEvent`  | Wallet debited directly or by transfer    |

Every event embeds `BaseEventData` which carries `EventID`, `AggregateID`, and `Version`.

## Key Design Decisions

- **Reflection-based dispatch** — Commands, queries, and aggregate events route via `reflect.Type`, eliminating switch statements.
- **Event Registry** — Domain events self-register via `init()` functions, enabling deserialization from BSON/JSON without explicit type mappings.
- **Optimistic concurrency** — The event store checks `expectedVersion` before persisting to prevent conflicting writes.
- **Event-store-as-outbox** — A single collection holds events and their publish state so the atomic unit is always a single-document write. This removes the dependency on multi-document MongoDB transactions (and therefore on a replica-set configuration).
- **At-least-once with idempotent projections** — The read side treats duplicates as the normal case and relies on the inbox table for correctness, not on broker delivery guarantees.
- **Eventual consistency** — Queries may lag behind commands by one outbox tick plus the Kafka round-trip.

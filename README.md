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

### Idempotent Projections (read side)

The projection pipeline is safe under at-least-once delivery.

- A `processed_events` table in PostgreSQL records every `(event_id, event_type, aggregate_id, version, processed_at)` that has been projected.
- Projections run inside a single DB transaction that first `INSERT … ON CONFLICT DO NOTHING` into `processed_events`. If the insert affects zero rows, the event is a duplicate and the projection body is skipped entirely.
- Projection writes and inbox writes commit together, so a crash between the two is impossible.

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

The system includes built-in observability features to monitor health, performance, and internal state.

### Structured Logging
All services use structured JSON logging via `log/slog` from the Go standard library. Logs include critical identifiers like `aggregateId`, `eventId`, and `type` to allow for easy tracing of events through the system.

### Health & Readiness Endpoints
Both `wallet-cmd` and `wallet-query` expose health and readiness checks:
- `GET /health`: Returns `200 OK` with `{"status": "UP"}` if the service is running.
- `GET /ready`: Returns `200 OK` if the service is connected to its dependencies (MongoDB for `wallet-cmd`, PostgreSQL for `wallet-query`).

### Lightweight Metrics
A simple metrics system tracks internal counts:
- `GET /metrics`: Returns JSON containing:
    - `commands_received`: Total commands handled by the write side.
    - `produced_events`: Total events successfully published to Kafka.
    - `produce_failures`: Total Kafka publication failures.
    - `processed_events`: Total events successfully projected to the read model.
    - `skipped_events`: Total duplicate events detected and skipped by the inbox.
    - `failed_events`: Total projection execution failures.

### Replay Observability
The replay process provides clear progress logging:
- Start and completion logs with duration.
- Progress updates every 500 events (or at 100% completion) including percentage and event counts.
- Detailed error logging if a projection fails during replay.

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

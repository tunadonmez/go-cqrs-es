# Wallet / Payment Ledger Service — CQRS & Event Sourcing

A wallet and payment-ledger system built with **CQRS (Command Query Responsibility Segregation)** and **Event Sourcing** using Go microservices.

## Architecture

**Command flow:** HTTP → Controller → CommandDispatcher → CommandHandler → WalletAggregate → EventStore (MongoDB) → Kafka

**Read model projection:** Kafka → EventConsumer → EventHandler → PostgreSQL read model

**Query flow:** HTTP → Controller → QueryDispatcher → QueryHandler → PostgreSQL read model → HTTP response

### Project Structure

```
go-cqrs-es/
├── cqrs-core/      Core CQRS/ES framework (interfaces, dispatchers, aggregate root)
├── wallet-common/  Shared wallet events and DTOs
├── wallet-cmd/     Write service — MongoDB + Kafka producer
└── wallet-query/   Read service — PostgreSQL + Kafka consumer
```

### Tech Stack

| Component   | Technology           | Purpose                       |
|-------------|----------------------|-------------------------------|
| Language    | Go 1.26.1            | All services                  |
| Event store | MongoDB              | Immutable wallet event log    |
| Read model  | PostgreSQL (GORM)    | Wallet balances and ledger    |
| Messaging   | Apache Kafka (KRaft) | Event distribution            |
| HTTP        | Gin                  | REST API routing              |

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

| Method | Path                             | Description                  |
|--------|----------------------------------|------------------------------|
| `GET`  | `/api/v1/wallets`                | List all wallets             |
| `GET`  | `/api/v1/wallets/:id`            | Get wallet details           |
| `GET`  | `/api/v1/wallets/:id/balance`    | Get wallet balance           |
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

## Key Design Decisions

- **Reflection-based dispatch** — Commands, queries, and aggregate events are routed via `reflect.Type`, eliminating switch statements.
- **Event Registry** — Domain events self-register via `init()` functions, enabling deserialization from BSON/JSON without explicit type mappings.
- **Optimistic concurrency** — The event store checks `expectedVersion` before persisting to prevent conflicting writes.
- **Eventual consistency** — The read model is updated asynchronously via Kafka; queries may temporarily lag behind commands.

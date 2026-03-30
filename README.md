# Bank Account Microservices — CQRS & Event Sourcing

A bank account system built with **CQRS (Command Query Responsibility Segregation)** and **Event Sourcing** patterns using Go microservices.

## Architecture

**Command flow:** HTTP → Controller → CommandDispatcher → CommandHandler → AccountAggregate → EventStore (MongoDB) → Kafka

**Read model projection:** Kafka → EventConsumer → EventHandler → MySQL read model

**Query flow:** HTTP → Controller → QueryDispatcher → QueryHandler → MySQL read model → HTTP response

### Project Structure

```
go.work (Go 1.26.1)
├── cqrs-core/         Core CQRS/ES framework (interfaces, dispatchers, aggregate root)
├── account-common/    Shared domain events and DTOs
├── account-cmd/       Write service — MongoDB + Kafka producer
└── account-query/     Read service — MySQL + Kafka consumer
```

### Tech Stack

| Component   | Technology          | Purpose                      |
|-------------|---------------------|------------------------------|
| Language    | Go 1.26.1           | All services                 |
| Event store | MongoDB             | Immutable event log          |
| Read model  | MySQL (GORM)        | Queryable account projections|
| Messaging   | Apache Kafka (KRaft)| Event distribution           |
| HTTP        | Gin                 | REST API routing             |

## Getting Started

### Prerequisites

- Go 1.26.1+
- MongoDB, MySQL, and Apache Kafka running locally (or via containers)

### Run Locally

```bash
# Command service
PORT=5000 \
MONGODB_URI="mongodb://root:root@localhost:27017/bankAccount?authSource=admin" \
KAFKA_BOOTSTRAP_SERVERS="localhost:9092" \
go run ./account-cmd

# Query service
PORT=5001 \
MYSQL_DSN="root:techbankRootPsw@tcp(localhost:3306)/bankAccount?charset=utf8mb4&parseTime=True&loc=Local" \
KAFKA_BOOTSTRAP_SERVERS="localhost:9092" \
KAFKA_GROUP_ID="bankaccConsumer" \
go run ./account-query
```

### Build

```bash
# All modules
go build ./...

# Individual services
cd account-cmd && go build -o account-cmd .
cd account-query && go build -o account-query .
```

### Test

```bash
go test ./...
```

## API Reference

### Command Service (port 5000)

#### Open Account

```
POST /api/v1/openBankAccount
```

```json
{
  "accountHolder": "John Doe",
  "accountType": "SAVINGS",
  "openingBalance": 500.00
}
```

#### Deposit Funds

```
PUT /api/v1/depositFunds/:id
```

```json
{
  "amount": 200.00
}
```

#### Withdraw Funds

```
PUT /api/v1/withdrawFunds/:id
```

```json
{
  "amount": 100.00
}
```

#### Close Account

```
DELETE /api/v1/closeBankAccount/:id
```

### Query Service (port 5001)

| Method | Path                   | Description        |
|--------|------------------------|--------------------|
| `GET`  | `/api/v1/accounts`     | List all accounts  |
| `GET`  | `/api/v1/accounts/:id` | Get account by ID  |

### Response Codes

- **204** — No results found
- **409** — Optimistic concurrency conflict

## Domain Events

| Event                | Description                    |
|----------------------|--------------------------------|
| `AccountOpenedEvent` | Account created with initial balance |
| `FundsDepositedEvent`| Funds added to account         |
| `FundsWithdrawnEvent`| Funds removed from account     |
| `AccountClosedEvent` | Account deactivated            |

## Key Design Decisions

- **Reflection-based dispatch** — Commands, queries, and aggregate events are routed via `reflect.Type`, eliminating switch statements.
- **Event Registry** — Domain events self-register via `init()` functions, enabling deserialization from BSON/JSON without explicit type mappings.
- **Optimistic concurrency** — Event store checks `expectedVersion` before persisting to prevent conflicting writes.
- **Eventual consistency** — The read model is updated asynchronously via Kafka; queries may temporarily lag behind commands.

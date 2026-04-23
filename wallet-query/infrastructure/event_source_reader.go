package infrastructure

import (
	"context"
	"fmt"
	"time"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// eventStoreCollection mirrors the collection name used by wallet-cmd.
// Replay talks directly to the MongoDB event store (the source of truth)
// and deliberately bypasses Kafka.
const eventStoreCollection = "eventStore"

// rawReplayEventDoc is a read-only projection of the BSON shape that the
// write side persists. It is kept minimal on purpose — the query side has
// no business touching the write-side outbox/publish fields.
type rawReplayEventDoc struct {
	TimeStamp           time.Time `bson:"timeStamp"`
	AggregateIdentifier string    `bson:"aggregateIdentifier"`
	Version             int       `bson:"version"`
	EventID             string    `bson:"eventId"`
	EventType           string    `bson:"eventType"`
	EventData           bson.Raw  `bson:"eventData"`
}

// ReplayEvent is a hydrated event ready to be handed to a projection handler.
type ReplayEvent struct {
	AggregateID string
	Version     int
	EventID     string
	EventType   string
	Event       corevents.BaseEvent
}

// EventSourceReader is a thin, read-only Mongo accessor used exclusively by
// the replay path. It shares the BSON schema with wallet-cmd's event store
// but is intentionally narrow: it only needs to stream events in order.
type EventSourceReader struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewEventSourceReader connects to MongoDB and returns a reader scoped to
// the walletLedger database / eventStore collection.
func NewEventSourceReader(ctx context.Context, uri, database string) (*EventSourceReader, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	return &EventSourceReader{
		client: client,
		db:     client.Database(database),
	}, nil
}

// Close releases the underlying Mongo connection.
func (r *EventSourceReader) Close(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Disconnect(ctx)
}

// CountEvents returns the total number of events matching the optional
// aggregate filter. It is only used for progress logging.
func (r *EventSourceReader) CountEvents(ctx context.Context, aggregateID string) (int64, error) {
	filter := bson.M{}
	if aggregateID != "" {
		filter["aggregateIdentifier"] = aggregateID
	}
	return r.db.Collection(eventStoreCollection).CountDocuments(ctx, filter)
}

// StreamEvents iterates over every event in the store (or for a single
// aggregate, when aggregateID is non-empty), ordered by aggregate and then
// by version. For each event the caller's handler is invoked in order.
//
// Events are hydrated using the same corevents.Registry used everywhere else,
// so replay deserialization is identical to live consumption.
func (r *EventSourceReader) StreamEvents(
	ctx context.Context,
	aggregateID string,
	handle func(ReplayEvent) error,
) error {
	filter := bson.M{}
	if aggregateID != "" {
		filter["aggregateIdentifier"] = aggregateID
	}
	// Ordering by aggregateIdentifier first then version gives us the same
	// per-aggregate ordering the live consumer sees. The secondary sort on
	// _id is a tiebreaker for any (hypothetical) equal-version rows.
	findOpts := options.Find().SetSort(bson.D{
		{Key: "aggregateIdentifier", Value: 1},
		{Key: "version", Value: 1},
		{Key: "_id", Value: 1},
	})

	cursor, err := r.db.Collection(eventStoreCollection).Find(ctx, filter, findOpts)
	if err != nil {
		return fmt.Errorf("mongo find: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var raw rawReplayEventDoc
		if err := cursor.Decode(&raw); err != nil {
			return fmt.Errorf("decode event: %w", err)
		}

		factory, ok := corevents.Registry[raw.EventType]
		if !ok {
			return fmt.Errorf("unknown event type %q at aggregate=%s version=%d",
				raw.EventType, raw.AggregateIdentifier, raw.Version)
		}
		event := factory()
		if err := bson.Unmarshal(raw.EventData, event); err != nil {
			return fmt.Errorf("unmarshal %s: %w", raw.EventType, err)
		}
		// Fall back to the outer EventID column when the embedded payload
		// was persisted before the field was introduced — mirrors what the
		// command-side repository does on rehydration.
		if event.GetEventID() == "" {
			event.SetEventID(raw.EventID)
		}

		if err := handle(ReplayEvent{
			AggregateID: raw.AggregateIdentifier,
			Version:     raw.Version,
			EventID:     raw.EventID,
			EventType:   raw.EventType,
			Event:       event,
		}); err != nil {
			return err
		}
	}
	return cursor.Err()
}

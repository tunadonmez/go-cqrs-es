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

const collection = "eventStore"

// Outbox publish states carried inline on each event document.
const (
	PublishStatusPending   = "PENDING"
	PublishStatusPublished = "PUBLISHED"
)

// EventStoreRepository handles MongoDB access for event models.
// The event store doubles as the transactional outbox: every event
// document carries its own publish state so persistence and
// "registration for publishing" happen in a single atomic write.
type EventStoreRepository struct {
	db *mongo.Database
}

func NewEventStoreRepository(db *mongo.Database) *EventStoreRepository {
	return &EventStoreRepository{db: db}
}

// rawEventDoc is the raw BSON shape stored in MongoDB.
type rawEventDoc struct {
	ID                  bson.ObjectID `bson:"_id,omitempty"`
	TimeStamp           time.Time     `bson:"timeStamp"`
	AggregateIdentifier string        `bson:"aggregateIdentifier"`
	AggregateType       string        `bson:"aggregateType"`
	Version             int           `bson:"version"`
	EventID             string        `bson:"eventId"`
	EventType           string        `bson:"eventType"`
	EventData           bson.Raw      `bson:"eventData"`

	// Outbox / publish bookkeeping.
	PublishStatus string     `bson:"publishStatus"`
	PublishedAt   *time.Time `bson:"publishedAt,omitempty"`
	Attempts      int        `bson:"attempts"`
	LastAttemptAt *time.Time `bson:"lastAttemptAt,omitempty"`
	LastError     string     `bson:"lastError,omitempty"`
}

// EventModelDoc is the application-level representation with a deserialized event.
type EventModelDoc struct {
	ID                  bson.ObjectID
	TimeStamp           time.Time
	AggregateIdentifier string
	AggregateType       string
	Version             int
	EventID             string
	EventType           string
	EventData           corevents.BaseEvent

	PublishStatus string
	PublishedAt   *time.Time
	Attempts      int
}

func (r *EventStoreRepository) FindByAggregateIdentifier(aggregateID string) ([]*EventModelDoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"aggregateIdentifier": aggregateID}
	findOpts := options.Find().SetSort(bson.D{{Key: "version", Value: 1}})
	cursor, err := r.db.Collection(collection).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var raws []*rawEventDoc
	if err = cursor.All(ctx, &raws); err != nil {
		return nil, err
	}

	results := make([]*EventModelDoc, 0, len(raws))
	for _, raw := range raws {
		model, err := hydrateEvent(raw)
		if err != nil {
			return nil, err
		}
		results = append(results, model)
	}
	return results, nil
}

// storableEventDoc is the shape written to MongoDB (EventData as interface{} for BSON encoding).
type storableEventDoc struct {
	TimeStamp           time.Time   `bson:"timeStamp"`
	AggregateIdentifier string      `bson:"aggregateIdentifier"`
	AggregateType       string      `bson:"aggregateType"`
	Version             int         `bson:"version"`
	EventID             string      `bson:"eventId"`
	EventType           string      `bson:"eventType"`
	EventData           interface{} `bson:"eventData"`

	PublishStatus string `bson:"publishStatus"`
	Attempts      int    `bson:"attempts"`
}

// Save persists an event in the PENDING outbox state.
// The insert is a single-document write, so the event and its
// outbox entry become durable atomically.
func (r *EventStoreRepository) Save(model *EventModelDoc) (*EventModelDoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if model.PublishStatus == "" {
		model.PublishStatus = PublishStatusPending
	}

	doc := storableEventDoc{
		TimeStamp:           model.TimeStamp,
		AggregateIdentifier: model.AggregateIdentifier,
		AggregateType:       model.AggregateType,
		Version:             model.Version,
		EventID:             model.EventID,
		EventType:           model.EventType,
		EventData:           model.EventData,
		PublishStatus:       model.PublishStatus,
		Attempts:            model.Attempts,
	}
	res, err := r.db.Collection(collection).InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		model.ID = oid
	}
	return model, nil
}

// FindPendingOutbox returns the oldest pending events across all aggregates.
// Sorting by insertion order (_id) preserves global arrival ordering;
// per-aggregate ordering is preserved because SaveEvents inserts sequentially.
func (r *EventStoreRepository) FindPendingOutbox(ctx context.Context, limit int64) ([]*EventModelDoc, error) {
	filter := bson.M{"publishStatus": PublishStatusPending}
	findOpts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: 1}}).
		SetLimit(limit)
	cursor, err := r.db.Collection(collection).Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var raws []*rawEventDoc
	if err := cursor.All(ctx, &raws); err != nil {
		return nil, err
	}
	results := make([]*EventModelDoc, 0, len(raws))
	for _, raw := range raws {
		model, err := hydrateEvent(raw)
		if err != nil {
			return nil, err
		}
		results = append(results, model)
	}
	return results, nil
}

// MarkPublished flips an outbox entry to PUBLISHED after a successful broker write.
func (r *EventStoreRepository) MarkPublished(ctx context.Context, id bson.ObjectID, publishedAt time.Time) error {
	_, err := r.db.Collection(collection).UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"publishStatus": PublishStatusPublished,
			"publishedAt":   publishedAt,
			"lastError":     "",
		},
	})
	return err
}

// RecordPublishFailure keeps the entry in PENDING but records retry metadata so
// operators can tell a stuck outbox entry from a freshly-minted one.
func (r *EventStoreRepository) RecordPublishFailure(ctx context.Context, id bson.ObjectID, attemptedAt time.Time, failure error) error {
	errMsg := ""
	if failure != nil {
		errMsg = failure.Error()
	}
	_, err := r.db.Collection(collection).UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"lastAttemptAt": attemptedAt,
			"lastError":     errMsg,
		},
		"$inc": bson.M{
			"attempts": 1,
		},
	})
	return err
}

func hydrateEvent(raw *rawEventDoc) (*EventModelDoc, error) {
	factory, ok := corevents.Registry[raw.EventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", raw.EventType)
	}
	event := factory()
	if err := bson.Unmarshal(raw.EventData, event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event %s: %w", raw.EventType, err)
	}
	// Ensure the in-memory event carries its EventID even if the subdocument
	// was persisted before the field was introduced: fall back to the outer
	// column so downstream components never see a blank id.
	if event.GetEventID() == "" {
		event.SetEventID(raw.EventID)
	}
	return &EventModelDoc{
		ID:                  raw.ID,
		TimeStamp:           raw.TimeStamp,
		AggregateIdentifier: raw.AggregateIdentifier,
		AggregateType:       raw.AggregateType,
		Version:             raw.Version,
		EventID:             raw.EventID,
		EventType:           raw.EventType,
		EventData:           event,
		PublishStatus:       raw.PublishStatus,
		PublishedAt:         raw.PublishedAt,
		Attempts:            raw.Attempts,
	}, nil
}

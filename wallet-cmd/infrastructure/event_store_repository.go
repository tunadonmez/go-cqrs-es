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

// EventStoreRepository handles MongoDB access for event models.
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
	EventType           string        `bson:"eventType"`
	EventData           bson.Raw      `bson:"eventData"`
}

// EventModelDoc is the application-level representation with a deserialized event.
type EventModelDoc struct {
	ID                  bson.ObjectID
	TimeStamp           time.Time
	AggregateIdentifier string
	AggregateType       string
	Version             int
	EventType           string
	EventData           corevents.BaseEvent
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
		factory, ok := corevents.Registry[raw.EventType]
		if !ok {
			return nil, fmt.Errorf("unknown event type: %s", raw.EventType)
		}
		event := factory()
		if err := bson.Unmarshal(raw.EventData, event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event %s: %w", raw.EventType, err)
		}
		results = append(results, &EventModelDoc{
			ID:                  raw.ID,
			TimeStamp:           raw.TimeStamp,
			AggregateIdentifier: raw.AggregateIdentifier,
			AggregateType:       raw.AggregateType,
			Version:             raw.Version,
			EventType:           raw.EventType,
			EventData:           event,
		})
	}
	return results, nil
}

// storableEventDoc is the shape written to MongoDB (EventData as interface{} for BSON encoding).
type storableEventDoc struct {
	TimeStamp           time.Time   `bson:"timeStamp"`
	AggregateIdentifier string      `bson:"aggregateIdentifier"`
	AggregateType       string      `bson:"aggregateType"`
	Version             int         `bson:"version"`
	EventType           string      `bson:"eventType"`
	EventData           interface{} `bson:"eventData"`
}

func (r *EventStoreRepository) Save(model *EventModelDoc) (*EventModelDoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	doc := storableEventDoc{
		TimeStamp:           model.TimeStamp,
		AggregateIdentifier: model.AggregateIdentifier,
		AggregateType:       model.AggregateType,
		Version:             model.Version,
		EventType:           model.EventType,
		EventData:           model.EventData,
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

package events

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// EventModel is the MongoDB document used as the event store.
type EventModel struct {
	ID                  bson.ObjectID `bson:"_id,omitempty"`
	TimeStamp           time.Time     `bson:"timeStamp"`
	AggregateIdentifier string        `bson:"aggregateIdentifier"`
	AggregateType       string        `bson:"aggregateType"`
	Version             int           `bson:"version"`
	EventType           string        `bson:"eventType"`
	// EventData is stored as raw BSON and deserialized using the event registry.
	EventData interface{} `bson:"eventData"`
}

package events

// BaseEvent is the interface all domain events must implement.
//
// Every event has two identifiers:
//   - EventID     — a stable, unique identifier for this specific event instance.
//     It follows the event through the event store, the outbox,
//     the Kafka envelope, and the query-side idempotency table.
//   - AggregateID — the identifier of the aggregate the event belongs to.
//     Used both as the partition key on Kafka and to rebuild
//     aggregate state on the write side.
type BaseEvent interface {
	GetEventID() string
	SetEventID(id string)
	GetAggregateID() string
	SetAggregateID(id string)
	GetVersion() int
	SetVersion(version int)
	GetSchemaVersion() int
	SetSchemaVersion(version int)
	EventTypeName() string
}

// BaseEventData provides common fields for all events via embedding.
type BaseEventData struct {
	EventID       string `json:"eventId" bson:"eventId"`
	AggregateID   string `json:"aggregateId" bson:"aggregateId"`
	Version       int    `json:"version" bson:"version"`
	SchemaVersion int    `json:"schemaVersion,omitempty" bson:"schemaVersion,omitempty"`
}

func (b *BaseEventData) GetEventID() string       { return b.EventID }
func (b *BaseEventData) SetEventID(id string)     { b.EventID = id }
func (b *BaseEventData) GetAggregateID() string   { return b.AggregateID }
func (b *BaseEventData) SetAggregateID(id string) { b.AggregateID = id }
func (b *BaseEventData) GetVersion() int          { return b.Version }
func (b *BaseEventData) SetVersion(v int)         { b.Version = v }
func (b *BaseEventData) GetSchemaVersion() int    { return b.SchemaVersion }
func (b *BaseEventData) SetSchemaVersion(v int)   { b.SchemaVersion = v }

package events

// BaseEvent is the interface all domain events must implement.
type BaseEvent interface {
	GetID() string
	SetID(id string)
	GetVersion() int
	SetVersion(version int)
	EventTypeName() string
}

// BaseEventData provides common fields for all events via embedding.
type BaseEventData struct {
	ID      string `json:"id" bson:"id"`
	Version int    `json:"version" bson:"version"`
}

func (b *BaseEventData) GetID() string    { return b.ID }
func (b *BaseEventData) SetID(id string)  { b.ID = id }
func (b *BaseEventData) GetVersion() int  { return b.Version }
func (b *BaseEventData) SetVersion(v int) { b.Version = v }

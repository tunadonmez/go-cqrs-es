package infrastructure

import "time"

// ProcessedEvent is the durable "inbox" record that makes projection
// application idempotent under at-least-once Kafka delivery.
//
// An event is considered processed if and only if a row exists here.
// Writing this row happens inside the same DB transaction that applies
// the projection, so either both changes commit or neither does.
type ProcessedEvent struct {
	EventID     string    `gorm:"primaryKey;column:event_id;size:64"`
	EventType   string    `gorm:"column:event_type;size:128;index"`
	AggregateID string    `gorm:"column:aggregate_id;size:64;index"`
	Version     int       `gorm:"column:version"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (ProcessedEvent) TableName() string { return "processed_events" }

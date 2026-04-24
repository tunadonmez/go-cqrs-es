package infrastructure

import "time"

const (
	DeadLetterStatusPending  = "pending"
	DeadLetterStatusResolved = "resolved"
)

// DeadLetterEvent stores Kafka messages that could not be projected safely
// after explicit retry handling. It is an operational record only; it does
// not participate in idempotent projection processing.
type DeadLetterEvent struct {
	DeadLetterKey      string     `gorm:"primaryKey;column:dead_letter_key;size:255"`
	EventID            string     `gorm:"column:event_id;size:64;index"`
	EventType          string     `gorm:"column:event_type;size:128;index"`
	EventSchemaVersion int        `gorm:"column:event_schema_version"`
	AggregateID        string     `gorm:"column:aggregate_id;size:64;index"`
	Topic              string     `gorm:"column:topic;size:128;index"`
	Partition          int        `gorm:"column:partition"`
	Offset             int64      `gorm:"column:offset"`
	ConsumerGroup      string     `gorm:"column:consumer_group;size:128"`
	FailureKind        string     `gorm:"column:failure_kind;size:32"`
	RetryAttempts      int        `gorm:"column:retry_attempts"`
	LastError          string     `gorm:"column:last_error;type:text"`
	Payload            string     `gorm:"column:payload;type:text"`
	Status             string     `gorm:"column:status;size:32;index"`
	FirstFailedAt      time.Time  `gorm:"column:first_failed_at"`
	LastFailedAt       time.Time  `gorm:"column:last_failed_at"`
	DeadLetteredAt     time.Time  `gorm:"column:dead_lettered_at"`
	ReprocessedAt      *time.Time `gorm:"column:reprocessed_at"`
	ResolvedAt         *time.Time `gorm:"column:resolved_at"`
}

func (DeadLetterEvent) TableName() string { return "dead_letter_events" }

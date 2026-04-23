package infrastructure

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeadLetterRepository struct {
	db *gorm.DB
}

func NewDeadLetterRepository(db *gorm.DB) *DeadLetterRepository {
	return &DeadLetterRepository{db: db}
}

func (r *DeadLetterRepository) Save(record *DeadLetterEvent) error {
	if record.FirstFailedAt.IsZero() {
		record.FirstFailedAt = time.Now().UTC()
	}
	if record.LastFailedAt.IsZero() {
		record.LastFailedAt = record.FirstFailedAt
	}
	if record.DeadLetteredAt.IsZero() {
		record.DeadLetteredAt = record.LastFailedAt
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "dead_letter_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"event_id":         record.EventID,
			"event_type":       record.EventType,
			"aggregate_id":     record.AggregateID,
			"topic":            record.Topic,
			"partition":        record.Partition,
			"offset":           record.Offset,
			"consumer_group":   record.ConsumerGroup,
			"failure_kind":     record.FailureKind,
			"retry_attempts":   record.RetryAttempts,
			"last_error":       record.LastError,
			"payload":          record.Payload,
			"last_failed_at":   record.LastFailedAt,
			"dead_lettered_at": record.DeadLetteredAt,
		}),
	}).Create(record).Error
}

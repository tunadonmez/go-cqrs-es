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

func (r *DeadLetterRepository) FindByKey(deadLetterKey string) (*DeadLetterEvent, error) {
	var record DeadLetterEvent
	if err := r.db.First(&record, "dead_letter_key = ?", deadLetterKey).Error; err != nil {
		return nil, err
	}
	return &record, nil
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
	if record.Status == "" {
		record.Status = DeadLetterStatusPending
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
			"status":           record.Status,
			"last_failed_at":   record.LastFailedAt,
			"dead_lettered_at": record.DeadLetteredAt,
			"reprocessed_at":   record.ReprocessedAt,
			"resolved_at":      record.ResolvedAt,
		}),
	}).Create(record).Error
}

func (r *DeadLetterRepository) MarkResolved(deadLetterKey string, at time.Time) error {
	return r.db.Model(&DeadLetterEvent{}).
		Where("dead_letter_key = ?", deadLetterKey).
		Updates(map[string]interface{}{
			"status":         DeadLetterStatusResolved,
			"last_error":     "",
			"reprocessed_at": at,
			"resolved_at":    at,
		}).Error
}

func (r *DeadLetterRepository) MarkFailedReprocess(deadLetterKey string, failure error, at time.Time) error {
	return r.db.Model(&DeadLetterEvent{}).
		Where("dead_letter_key = ?", deadLetterKey).
		Updates(map[string]interface{}{
			"status":         DeadLetterStatusPending,
			"last_error":     failure.Error(),
			"last_failed_at": at,
			"retry_attempts": gorm.Expr("retry_attempts + 1"),
			"reprocessed_at": at,
			"resolved_at":    nil,
		}).Error
}

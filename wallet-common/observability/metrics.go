package observability

import (
	"sync/atomic"
)

type Metrics struct {
	ProcessedEvents             atomic.Int64
	SkippedEvents               atomic.Int64
	FailedEvents                atomic.Int64
	ProducedEvents              atomic.Int64
	ProduceFailures             atomic.Int64
	CommandsReceived            atomic.Int64
	CommandsSucceeded           atomic.Int64
	CommandFailures             atomic.Int64
	EventsPersisted             atomic.Int64
	EventPersistFailures        atomic.Int64
	OutboxPublishAttempts       atomic.Int64
	KafkaMessagesConsumed       atomic.Int64
	KafkaMessageFailures        atomic.Int64
	KafkaRetryAttempts          atomic.Int64
	ProjectionAttempts          atomic.Int64
	ReplayRuns                  atomic.Int64
	ReplayFailures              atomic.Int64
	ReplayEventsProcessed       atomic.Int64
	DeadLetteredEvents          atomic.Int64
	DeadLetterSaveFailures      atomic.Int64
	DeadLetterReprocessRuns     atomic.Int64
	DeadLetterReprocessed       atomic.Int64
	DeadLetterReprocessFailures atomic.Int64
	SnapshotsLoaded             atomic.Int64
	SnapshotsCreated            atomic.Int64
	SnapshotFullReplays         atomic.Int64
}

var DefaultMetrics = &Metrics{}

func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"processed_events":               m.ProcessedEvents.Load(),
		"skipped_events":                 m.SkippedEvents.Load(),
		"failed_events":                  m.FailedEvents.Load(),
		"produced_events":                m.ProducedEvents.Load(),
		"produce_failures":               m.ProduceFailures.Load(),
		"commands_received":              m.CommandsReceived.Load(),
		"commands_succeeded":             m.CommandsSucceeded.Load(),
		"command_failures":               m.CommandFailures.Load(),
		"events_persisted":               m.EventsPersisted.Load(),
		"event_persist_failures":         m.EventPersistFailures.Load(),
		"outbox_publish_attempts":        m.OutboxPublishAttempts.Load(),
		"kafka_messages_consumed":        m.KafkaMessagesConsumed.Load(),
		"kafka_message_failures":         m.KafkaMessageFailures.Load(),
		"kafka_retry_attempts":           m.KafkaRetryAttempts.Load(),
		"projection_attempts":            m.ProjectionAttempts.Load(),
		"replay_runs":                    m.ReplayRuns.Load(),
		"replay_failures":                m.ReplayFailures.Load(),
		"replay_events_processed":        m.ReplayEventsProcessed.Load(),
		"dead_lettered_events":           m.DeadLetteredEvents.Load(),
		"dead_letter_save_failures":      m.DeadLetterSaveFailures.Load(),
		"dead_letter_reprocess_runs":     m.DeadLetterReprocessRuns.Load(),
		"dead_letter_reprocessed":        m.DeadLetterReprocessed.Load(),
		"dead_letter_reprocess_failures": m.DeadLetterReprocessFailures.Load(),
		"snapshots_loaded":               m.SnapshotsLoaded.Load(),
		"snapshots_created":              m.SnapshotsCreated.Load(),
		"snapshot_full_replays":          m.SnapshotFullReplays.Load(),
	}
}

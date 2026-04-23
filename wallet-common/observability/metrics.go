package observability

import (
	"sync/atomic"
)

type Metrics struct {
	ProcessedEvents  atomic.Int64
	SkippedEvents    atomic.Int64
	FailedEvents     atomic.Int64
	ProducedEvents   atomic.Int64
	ProduceFailures  atomic.Int64
	CommandsReceived atomic.Int64
}

var DefaultMetrics = &Metrics{}

func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"processed_events":  m.ProcessedEvents.Load(),
		"skipped_events":    m.SkippedEvents.Load(),
		"failed_events":     m.FailedEvents.Load(),
		"produced_events":   m.ProducedEvents.Load(),
		"produce_failures":  m.ProduceFailures.Load(),
		"commands_received": m.CommandsReceived.Load(),
	}
}

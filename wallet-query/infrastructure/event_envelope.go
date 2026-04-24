package infrastructure

import (
	"encoding/json"
	"fmt"

	corevents "github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
)

// EventEnvelope matches the envelope produced by the write side.
// EventID is surfaced on the envelope for observability and also lives
// inside the serialized event payload itself (via BaseEventData.EventID).
type EventEnvelope struct {
	EventID       string          `json:"eventId"`
	SchemaVersion int             `json:"schemaVersion,omitempty"`
	Type          string          `json:"type"`
	Data          json.RawMessage `json:"data"`
}

// DispatchEnvelope routes a serialized event envelope through the same
// projection entrypoints used by live Kafka consumption.
func DispatchEnvelope(handler *WalletEventHandler, envelope EventEnvelope) error {
	event, err := corevents.DecodeJSONEvent(envelope.Type, envelope.SchemaVersion, envelope.Data)
	if err != nil {
		return fmt.Errorf("%w: decode %s: %v", errPermanentConsumerFailure, envelope.Type, err)
	}
	fallbackEventID(event, envelope.EventID)

	switch e := event.(type) {
	case *commonevents.WalletCreatedEvent:
		return handler.OnWalletCreated(e)
	case *commonevents.WalletCreditedEvent:
		return handler.OnWalletCredited(e)
	case *commonevents.WalletDebitedEvent:
		return handler.OnWalletDebited(e)
	default:
		return fmt.Errorf("%w: unknown event type %s", errPermanentConsumerFailure, envelope.Type)
	}
}

func aggregateIDFromEnvelope(envelope EventEnvelope) string {
	var payload struct {
		AggregateID string `json:"aggregateId"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return ""
	}
	return payload.AggregateID
}

func normalizedEnvelopeSchemaVersion(envelope EventEnvelope) int {
	if envelope.SchemaVersion > 0 {
		return envelope.SchemaVersion
	}

	var payload struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err == nil && payload.SchemaVersion > 0 {
		return payload.SchemaVersion
	}

	return corevents.InitialSchemaVersion
}

// fallbackEventID copies the envelope-level id onto the inner event when the
// serialized payload did not carry one (older producers).
func fallbackEventID(event corevents.BaseEvent, envelopeID string) {
	if event.GetEventID() == "" {
		event.SetEventID(envelopeID)
	}
}

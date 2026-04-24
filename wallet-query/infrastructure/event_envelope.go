package infrastructure

import (
	"encoding/json"
	"fmt"

	commonevents "github.com/tunadonmez/go-cqrs-es/wallet-common/events"
)

// EventEnvelope matches the envelope produced by the write side.
// EventID is surfaced on the envelope for observability and also lives
// inside the serialized event payload itself (via BaseEventData.EventID).
type EventEnvelope struct {
	EventID string          `json:"eventId"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}

// DispatchEnvelope routes a serialized event envelope through the same
// projection entrypoints used by live Kafka consumption.
func DispatchEnvelope(handler *WalletEventHandler, envelope EventEnvelope) error {
	switch envelope.Type {
	case "WalletCreatedEvent":
		var event commonevents.WalletCreatedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return fmt.Errorf("%w: unmarshal WalletCreatedEvent: %v", errPermanentConsumerFailure, err)
		}
		fallbackEventID(&event.BaseEventData.EventID, envelope.EventID)
		return handler.OnWalletCreated(&event)

	case "WalletCreditedEvent":
		var event commonevents.WalletCreditedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return fmt.Errorf("%w: unmarshal WalletCreditedEvent: %v", errPermanentConsumerFailure, err)
		}
		fallbackEventID(&event.BaseEventData.EventID, envelope.EventID)
		return handler.OnWalletCredited(&event)

	case "WalletDebitedEvent":
		var event commonevents.WalletDebitedEvent
		if err := json.Unmarshal(envelope.Data, &event); err != nil {
			return fmt.Errorf("%w: unmarshal WalletDebitedEvent: %v", errPermanentConsumerFailure, err)
		}
		fallbackEventID(&event.BaseEventData.EventID, envelope.EventID)
		return handler.OnWalletDebited(&event)

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

// fallbackEventID copies the envelope-level id onto the inner event when the
// serialized payload did not carry one (older producers).
func fallbackEventID(target *string, envelopeID string) {
	if *target == "" {
		*target = envelopeID
	}
}

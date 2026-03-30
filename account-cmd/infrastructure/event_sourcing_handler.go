package infrastructure

import (
	"github.com/techbank/account-cmd/domain"
)

// AccountEventSourcingHandler rehydrates AccountAggregates from the event store.
type AccountEventSourcingHandler struct {
	eventStore *AccountEventStore
}

func NewAccountEventSourcingHandler(es *AccountEventStore) *AccountEventSourcingHandler {
	return &AccountEventSourcingHandler{eventStore: es}
}

func (h *AccountEventSourcingHandler) Save(aggregate *domain.AccountAggregate) error {
	err := h.eventStore.SaveEvents(aggregate.ID, aggregate.GetUncommittedChanges(), aggregate.Version)
	if err != nil {
		return err
	}
	aggregate.MarkChangesAsCommitted()
	return nil
}

func (h *AccountEventSourcingHandler) GetByID(id string) (*domain.AccountAggregate, error) {
	aggregate := domain.NewAccountAggregate()
	events, err := h.eventStore.GetEvents(id)
	if err != nil {
		return nil, err
	}
	if len(events) > 0 {
		aggregate.ReplayEvents(aggregate, events)
		latestVersion := -1
		for _, e := range events {
			if v := e.GetVersion(); v > latestVersion {
				latestVersion = v
			}
		}
		aggregate.Version = latestVersion
	}
	return aggregate, nil
}

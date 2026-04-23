package infrastructure

import (
	"github.com/tunadonmez/go-cqrs-es/account-cmd/domain"
	corehandlers "github.com/tunadonmez/go-cqrs-es/cqrs-core/handlers"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
)

// AccountEventSourcingHandler rehydrates AccountAggregates from the event store.
type AccountEventSourcingHandler struct {
	eventStore coreinfra.EventStore
}

func NewAccountEventSourcingHandler(es coreinfra.EventStore) *AccountEventSourcingHandler {
	return &AccountEventSourcingHandler{eventStore: es}
}

func (h *AccountEventSourcingHandler) Save(aggregate *domain.AccountAggregate) error {
	changes := aggregate.GetUncommittedChanges()
	err := h.eventStore.SaveEvents(aggregate.ID, changes, aggregate.Version)
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		aggregate.Version = changes[len(changes)-1].GetVersion()
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

var _ corehandlers.EventSourcingHandler[domain.AccountAggregate] = (*AccountEventSourcingHandler)(nil)

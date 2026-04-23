package infrastructure

import (
	corehandlers "github.com/tunadonmez/go-cqrs-es/cqrs-core/handlers"
	coreinfra "github.com/tunadonmez/go-cqrs-es/cqrs-core/infrastructure"
	"github.com/tunadonmez/go-cqrs-es/wallet-cmd/domain"
)

// WalletEventSourcingHandler rehydrates WalletAggregates from the event store.
type WalletEventSourcingHandler struct {
	eventStore coreinfra.EventStore
}

func NewWalletEventSourcingHandler(es coreinfra.EventStore) *WalletEventSourcingHandler {
	return &WalletEventSourcingHandler{eventStore: es}
}

func (h *WalletEventSourcingHandler) Save(aggregate *domain.WalletAggregate) error {
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

func (h *WalletEventSourcingHandler) GetByID(id string) (*domain.WalletAggregate, error) {
	aggregate := domain.NewWalletAggregate()
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

var _ corehandlers.EventSourcingHandler[domain.WalletAggregate] = (*WalletEventSourcingHandler)(nil)

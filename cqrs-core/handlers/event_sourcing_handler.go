package handlers

import "github.com/techbank/cqrs-core/domain"

// EventSourcingHandler persists and rehydrates aggregates.
type EventSourcingHandler[T any] interface {
	Save(aggregate *domain.AggregateRoot) error
	GetByID(id string) (*T, error)
}

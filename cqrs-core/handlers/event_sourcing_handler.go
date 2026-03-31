package handlers

// EventSourcingHandler persists and rehydrates aggregates.
type EventSourcingHandler[T any] interface {
	Save(aggregate *T) error
	GetByID(id string) (*T, error)
}

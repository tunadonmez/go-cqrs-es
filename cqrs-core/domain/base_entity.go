package domain

// BaseEntity is the marker interface for read-model entities.
type BaseEntity interface {
	EntityID() string
}

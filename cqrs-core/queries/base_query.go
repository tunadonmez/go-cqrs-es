package queries

// BaseQuery is the marker interface for all queries.
type BaseQuery interface {
	QueryTypeName() string
}

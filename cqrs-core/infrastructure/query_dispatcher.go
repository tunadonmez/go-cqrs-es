package infrastructure

import (
	"fmt"
	"reflect"

	"github.com/tunadonmez/go-cqrs-es/cqrs-core/domain"
	"github.com/tunadonmez/go-cqrs-es/cqrs-core/queries"
)

// QueryHandlerFunc handles a query and returns a slice of entities.
type QueryHandlerFunc func(q queries.BaseQuery) ([]domain.BaseEntity, error)

// QueryDispatcher routes queries to registered handlers.
type QueryDispatcher struct {
	routes map[reflect.Type]QueryHandlerFunc
}

func NewQueryDispatcher() *QueryDispatcher {
	return &QueryDispatcher{routes: make(map[reflect.Type]QueryHandlerFunc)}
}

func (d *QueryDispatcher) RegisterHandler(qType reflect.Type, handler QueryHandlerFunc) {
	d.routes[qType] = handler
}

func (d *QueryDispatcher) Send(q queries.BaseQuery) ([]domain.BaseEntity, error) {
	t := reflect.TypeOf(q)
	handler, ok := d.routes[t]
	if !ok {
		return nil, fmt.Errorf("no handler registered for query type %s", t.Name())
	}
	return handler(q)
}

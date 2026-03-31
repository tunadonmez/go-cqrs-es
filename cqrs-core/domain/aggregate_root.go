package domain

import (
	"fmt"
	"reflect"

	"github.com/tunadonmez/go-cqrs-es/cqrs-core/events"
)

// AggregateRoot is the base for all aggregates.
// Embeds must pass themselves as "self" when calling RaiseEvent/ReplayEvents
// so reflection can dispatch to the correct Apply<EventType> method.
type AggregateRoot struct {
	ID      string
	Version int
	changes []events.BaseEvent
}

func NewAggregateRoot() *AggregateRoot {
	return &AggregateRoot{Version: -1}
}

func (a *AggregateRoot) GetUncommittedChanges() []events.BaseEvent {
	return a.changes
}

func (a *AggregateRoot) MarkChangesAsCommitted() {
	a.changes = nil
}

// ApplyChange dispatches the event to the Apply<EventType> method on self.
func (a *AggregateRoot) ApplyChange(self interface{}, event events.BaseEvent, isNewEvent bool) {
	typeName := reflect.TypeOf(event).Elem().Name()
	methodName := "Apply" + typeName
	method := reflect.ValueOf(self).MethodByName(methodName)
	if !method.IsValid() {
		fmt.Printf("WARNING: method %s not found on aggregate\n", methodName)
	} else {
		method.Call([]reflect.Value{reflect.ValueOf(event)})
	}
	if isNewEvent {
		a.changes = append(a.changes, event)
	}
}

func (a *AggregateRoot) RaiseEvent(self interface{}, event events.BaseEvent) {
	a.ApplyChange(self, event, true)
}

func (a *AggregateRoot) ReplayEvents(self interface{}, evts []events.BaseEvent) {
	for _, event := range evts {
		a.ApplyChange(self, event, false)
	}
}

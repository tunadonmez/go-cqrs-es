package domain

import (
	"crypto/rand"
	"encoding/hex"
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

// RaiseEvent is called by aggregates when a new domain event is produced.
// A stable EventID is assigned here when missing so that downstream
// components (event store, outbox, Kafka envelope, query-side inbox)
// can rely on the identifier being present.
func (a *AggregateRoot) RaiseEvent(self interface{}, event events.BaseEvent) {
	if event.GetEventID() == "" {
		event.SetEventID(newEventID())
	}
	a.ApplyChange(self, event, true)
}

func (a *AggregateRoot) ReplayEvents(self interface{}, events []events.BaseEvent) {
	for _, event := range events {
		a.ApplyChange(self, event, false)
	}
}

// newEventID returns a 128-bit random identifier rendered as a hex string.
// Using crypto/rand keeps cqrs-core free of third-party dependencies.
func newEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read almost never fails; panic so we never silently emit a blank id.
		panic(fmt.Errorf("failed to generate event id: %w", err))
	}
	return hex.EncodeToString(b[:])
}

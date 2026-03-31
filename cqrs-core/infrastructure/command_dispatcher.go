package infrastructure

import (
	"fmt"
	"reflect"
)

// CommandHandlerFunc handles a command of any type.
type CommandHandlerFunc func(cmd interface{}) error

// CommandPreDispatchFunc runs before a command is routed to its handler.
type CommandPreDispatchFunc func(cmd interface{}) error

// CommandDispatcher routes commands to registered handlers.
type CommandDispatcher struct {
	routes    map[reflect.Type]CommandHandlerFunc
	preflight []CommandPreDispatchFunc
}

func NewCommandDispatcher() *CommandDispatcher {
	return &CommandDispatcher{routes: make(map[reflect.Type]CommandHandlerFunc)}
}

// Use registers a pre-dispatch hook.
func (d *CommandDispatcher) Use(hook CommandPreDispatchFunc) {
	d.preflight = append(d.preflight, hook)
}

// RegisterHandler registers a handler for the given command type.
func (d *CommandDispatcher) RegisterHandler(cmdType reflect.Type, handler CommandHandlerFunc) {
	d.routes[cmdType] = handler
}

// Send dispatches the command to its registered handler.
func (d *CommandDispatcher) Send(cmd interface{}) error {
	for _, hook := range d.preflight {
		if err := hook(cmd); err != nil {
			return err
		}
	}

	t := reflect.TypeOf(cmd)
	handler, ok := d.routes[t]
	if !ok {
		return fmt.Errorf("no handler registered for command type %s", t.Name())
	}
	return handler(cmd)
}

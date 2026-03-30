package infrastructure

import (
	"fmt"
	"reflect"
)

// CommandHandlerFunc handles a command of any type.
type CommandHandlerFunc func(cmd interface{}) error

// CommandDispatcher routes commands to registered handlers.
type CommandDispatcher struct {
	routes map[reflect.Type]CommandHandlerFunc
}

func NewCommandDispatcher() *CommandDispatcher {
	return &CommandDispatcher{routes: make(map[reflect.Type]CommandHandlerFunc)}
}

// RegisterHandler registers a handler for the given command type.
func (d *CommandDispatcher) RegisterHandler(cmdType reflect.Type, handler CommandHandlerFunc) {
	d.routes[cmdType] = handler
}

// Send dispatches the command to its registered handler.
func (d *CommandDispatcher) Send(cmd interface{}) error {
	t := reflect.TypeOf(cmd)
	handler, ok := d.routes[t]
	if !ok {
		return fmt.Errorf("no handler registered for command type %s", t.Name())
	}
	return handler(cmd)
}

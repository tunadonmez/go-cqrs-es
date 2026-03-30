package events

// Registry maps event type names to factory functions that produce zero-value instances.
// Modules register their event types on init so the event store can deserialize them.
var Registry = make(map[string]func() BaseEvent)

// Register adds an event factory to the global registry.
func Register(name string, factory func() BaseEvent) {
	Registry[name] = factory
}

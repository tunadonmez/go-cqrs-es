package events

// Registry maps event type names to factory functions that produce zero-value instances.
// Modules register their event types on init so the event store can deserialize them.
var Registry = make(map[string]func() BaseEvent)

// Register adds an event factory to the global registry.
func Register(name string, currentSchemaVersion int, factory func() BaseEvent) {
	Registry[name] = factory
	registrations[name] = &registration{
		factory:              factory,
		currentSchemaVersion: normalizeSchemaVersion(currentSchemaVersion),
		upcasters:            make(map[int]Upcaster),
	}
}

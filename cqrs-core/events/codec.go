package events

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const InitialSchemaVersion = 1

type Upcaster func(map[string]any) (map[string]any, error)

type registration struct {
	factory              func() BaseEvent
	currentSchemaVersion int
	upcasters            map[int]Upcaster
}

var registrations = make(map[string]*registration)

func EnsureSchemaVersion(event BaseEvent) {
	if event == nil || event.GetSchemaVersion() > 0 {
		return
	}
	if current, ok := CurrentSchemaVersion(event.EventTypeName()); ok {
		event.SetSchemaVersion(current)
		return
	}
	event.SetSchemaVersion(InitialSchemaVersion)
}

func CurrentSchemaVersion(eventType string) (int, bool) {
	reg, ok := registrations[eventType]
	if !ok {
		return 0, false
	}
	return reg.currentSchemaVersion, true
}

func RegisterUpcaster(eventType string, fromVersion int, upcaster Upcaster) {
	reg, ok := registrations[eventType]
	if !ok {
		panic(fmt.Sprintf("cannot register upcaster for unknown event type %s", eventType))
	}
	if fromVersion < InitialSchemaVersion {
		panic(fmt.Sprintf("invalid upcaster start version %d for %s", fromVersion, eventType))
	}
	if upcaster == nil {
		panic(fmt.Sprintf("nil upcaster for %s v%d", eventType, fromVersion))
	}
	reg.upcasters[fromVersion] = upcaster
}

func DecodeJSONEvent(eventType string, schemaVersion int, payload []byte) (BaseEvent, error) {
	return decodeEvent(
		eventType,
		schemaVersion,
		payload,
		func(data []byte, target any) error { return json.Unmarshal(data, target) },
		func(value any) ([]byte, error) { return json.Marshal(value) },
		func(data []byte, target any) error { return json.Unmarshal(data, target) },
	)
}

func DecodeBSONEvent(eventType string, schemaVersion int, payload []byte) (BaseEvent, error) {
	return decodeEvent(
		eventType,
		schemaVersion,
		payload,
		func(data []byte, target any) error { return bson.Unmarshal(data, target) },
		func(value any) ([]byte, error) { return bson.Marshal(value) },
		func(data []byte, target any) error { return bson.Unmarshal(data, target) },
	)
}

func decodeEvent(
	eventType string,
	schemaVersion int,
	payload []byte,
	unmarshalDocument func([]byte, any) error,
	marshalDocument func(any) ([]byte, error),
	unmarshalEvent func([]byte, any) error,
) (BaseEvent, error) {
	reg, ok := registrations[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	document := map[string]any{}
	if err := unmarshalDocument(payload, &document); err != nil {
		return nil, err
	}

	sourceVersion := normalizeSchemaVersion(schemaVersion)
	if schemaVersion <= 0 {
		sourceVersion = normalizeSchemaVersion(intFromAny(document["schemaVersion"]))
	}

	upcasted, err := upcastDocument(eventType, reg, sourceVersion, document)
	if err != nil {
		return nil, err
	}

	serialized, err := marshalDocument(upcasted)
	if err != nil {
		return nil, err
	}

	event := reg.factory()
	if err := unmarshalEvent(serialized, event); err != nil {
		return nil, err
	}
	if event.GetSchemaVersion() == 0 {
		event.SetSchemaVersion(reg.currentSchemaVersion)
	}

	return event, nil
}

func upcastDocument(eventType string, reg *registration, sourceVersion int, document map[string]any) (map[string]any, error) {
	if sourceVersion > reg.currentSchemaVersion {
		return nil, fmt.Errorf(
			"event %s has schemaVersion %d newer than supported %d",
			eventType,
			sourceVersion,
			reg.currentSchemaVersion,
		)
	}

	current := sourceVersion
	for current < reg.currentSchemaVersion {
		upcaster, ok := reg.upcasters[current]
		if !ok {
			return nil, fmt.Errorf(
				"missing upcaster for %s schemaVersion %d -> %d",
				eventType,
				current,
				current+1,
			)
		}
		var err error
		document, err = upcaster(document)
		if err != nil {
			return nil, fmt.Errorf(
				"upcast %s schemaVersion %d -> %d: %w",
				eventType,
				current,
				current+1,
				err,
			)
		}
		current++
		document["schemaVersion"] = current
	}

	document["schemaVersion"] = reg.currentSchemaVersion
	return document, nil
}

func normalizeSchemaVersion(version int) int {
	if version <= 0 {
		return InitialSchemaVersion
	}
	return version
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

package primitives_test

import (
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

func collectEvents(events <-chan primitives.PrimitiveEvent) []primitives.PrimitiveEvent {
	var collected []primitives.PrimitiveEvent
		collected = append(collected, event)
	}
}

func singleEvent(
	t *testing.T,
	events []primitives.PrimitiveEvent,
	eventType primitives.PrimitiveEventType,
) primitives.PrimitiveEvent {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != eventType {
		t.Fatalf("event type = %q, want %q", events[0].Type, eventType)
	}
	return events[0]
}

func eventResult[T any](t *testing.T, event primitives.PrimitiveEvent) T {
	t.Helper()
	result, ok := event.Result.(T)
	if !ok {
		t.Fatalf("result type = %T", event.Result)
	}
	return result
}

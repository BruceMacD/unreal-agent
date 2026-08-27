package operation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

func TestLocalOperationManagerStartsMultiplePrimitives(t *testing.T) {
	id := ID("local-multiple-primitives")
	events := make(chan primitives.PrimitiveEvent)
	manager := &LocalOperationManager{primitiveEvents: events}
	ctx, cancel := context.WithCancel(t.Context())
	current := &localRunningOperation{
		ctx:    ctx,
		cancel: cancel,
		operation: Operation{
			ID:     id,
			Status: StatusAwaiting,
		},
	}
	operations := map[ID]*localRunningOperation{id: current}
	directory := t.TempDir()
	dispatch := func(correlation primitives.CorrelationID, name string) PrimitiveDispatch {
		return PrimitiveDispatch{
			Type: primitives.PrimitiveDispatchIOCreate,
			Data: primitives.IOCreateRequest{
				Source:        primitives.SourceID(id),
				CorrelationID: correlation,
				Kind:          primitives.IOCreateRegularFile,
				Path:          filepath.Join(directory, name),
			},
		}
	}
	step := Step{
		Dispatches: []PrimitiveDispatch{
			dispatch("first", "first"),
			dispatch("second", "second"),
		},
	}

	started := manager.acceptLocalStep(operations, current, step)
	if started != 2 {
		t.Fatalf("started primitives = %d, want 2", started)
	}

	completed := current.operation
	completed.Status = StatusCompleted
	if len(operations) != 0 {
		t.Fatalf("operations = %d, want 0", len(operations))
	}
	manager.drainLocalPrimitives(started)
}

package coordinator

import (
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/contextbuilder"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
)

func TestCoordinatorReplaysToolResultBalance(t *testing.T) {
	store, registry := independentToolCalls(t, 2)
	store.resume.Operations = nil
	completion := func(index int) sessionstore.ToolCallStatus {
		status := store.items[index+2].Data.(sessionstore.ToolCallStatus)
		value := status.Operations[0]
		value.Status = operation.StatusCompleted
		status.Operations = []operation.Operation{value}
		return status
	}
	first, second := completion(0), completion(1)
	for _, step := range []struct {
		name                                               string
		kind                                               sessionstore.ItemKind
		data                                               any
		pendingCalls, completed, delivered, pendingResults int
	}{
		{"first completion", sessionstore.ItemToolCallStatus, first, 1, 1, 0, 1},
		{"second completes during request", sessionstore.ItemToolCallStatus, second, 0, 2, 0, 2},
		{"duplicate completion", sessionstore.ItemToolCallStatus, second, 0, 2, 0, 2},
		{"request completes", sessionstore.ItemModelResponse, sessionstore.ModelResponse{TurnID: "delivery-1", Response: textResponse("Checking.")}, 0, 2, 1, 1},
		{"next request completes", sessionstore.ItemModelResponse, sessionstore.ModelResponse{TurnID: "delivery-2", Response: textResponse("Done.")}, 0, 2, 2, 0},
	} {
		store.items = append(store.items, storedItem(sessionstore.Sequence(len(store.items)+1), step.kind, step.data))
		t.Run(step.name, func(t *testing.T) {
			current := newTestCoordinator(store, newTestInbox(t), newFakeOperationManager(), contextbuilder.NewBuilder(), registry)
			if err := current.restore(t.Context()); err != nil {
				t.Fatal(err)
			}
			if got := len(current.state.toolCalls); got != step.pendingCalls {
				t.Fatalf("pending calls = %d, want %d", got, step.pendingCalls)
			}
			if got := current.state.availableInputs; got != step.completed {
				t.Fatalf("completed results = %d, want %d", got, step.completed)
			}
			built, err := current.dependencies.ContextBuilder.Build()
			if err != nil {
				t.Fatal(err)
			}
			appended := 0
			for _, item := range built.Request.Input {
					appended++
				}
			}
			if appended != current.state.availableInputs {
				t.Fatalf("appended completions = %d, available inputs = %d", appended, current.state.availableInputs)
			}
			if got := current.state.deliveredInputs; got != step.delivered {
				t.Fatalf("delivered results = %d, want %d", got, step.delivered)
			}
			if got := current.pendingInputs(); got != step.pendingResults {
				t.Fatalf("pending results = %d, want %d", got, step.pendingResults)
			}
		})
	}
}

package coordinator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"

	"github.com/unreallabsai/unreal-agent/harness/contextbuilder"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

func TestCoordinatorRunDefersCompletionsUntilModelFinishes(t *testing.T) {
	for _, startWithInput := range []bool{false, true} {
		for _, steer := range []bool{false, true} {
			t.Run(fmt.Sprintf("input=%t/steer=%t", startWithInput, steer), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					store, registry := independentToolCalls(t, 3)
					operations := newFakeOperationManager()
					type modelCall struct {
						ctx      context.Context
						request  llm.Request
						response chan llm.Response
					}
					var calls []modelCall
					adapter := &fakeAdapter{respond: func(ctx context.Context, request llm.Request) (llm.Response, error) {
						call := modelCall{ctx: ctx, request: request, response: make(chan llm.Response)}
						calls = append(calls, call)
						select {
						case response := <-call.response:
							return response, nil
						case <-ctx.Done():
							return llm.Response{}, ctx.Err()
						}
					}}
					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()
					current := newTestCoordinatorWithAdapter(
						store, inputs, operations, contextbuilder.NewBuilder(), registry, adapter,
					)
					var runErr error
					go func() { runErr = current.Run(ctx) }()

					finishOperation := func(index int) {
						value.Status = operation.StatusCompleted
						operations.updates <- value
						synctest.Wait()
						if len(store.appendedStatuses) != index+1 {
							t.Fatalf("completed tool statuses = %d, want %d", len(store.appendedStatuses), index+1)
						}
						if status := store.appendedStatuses[index]; status.CallID != fmt.Sprintf("call-%d", index) {
							t.Fatalf("completed call = %q", status.CallID)
						}
					}
					firstPending := 0
					if startWithInput {
						synctest.Wait()
					} else {
						finishOperation(0)
						firstPending = 1
					}
					if len(calls) != 1 {
						t.Fatalf("model requests = %d, want 1", len(calls))
					}
					first := calls[0]
					for index := firstPending; index < 3; index++ {
						finishOperation(index)
						if err := first.ctx.Err(); err != nil {
							t.Fatalf("operation %d canceled active model request: %v", index, err)
						}
						if len(calls) != 1 {
							t.Fatalf("model requests = %d, want 1", len(calls))
						}
					}
					response := llm.Response{ID: "progress-response", Output: []llm.Item{{
						Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: "progress"},
					}}}
					if steer {
						synctest.Wait()
					} else {
						first.response <- response
						synctest.Wait()
					}
					if len(calls) != 2 {
						t.Fatalf("model requests = %d, want 2", len(calls))
					}
					next := calls[1]
					assertCompletedResults(t, next.request, 3)
					if steer {
						if !errors.Is(first.ctx.Err(), context.Canceled) {
							t.Fatal("steering did not cancel active model request")
						}
						last := next.request.Input[len(next.request.Input)-1]
						if last.Type != llm.ItemMessage || last.Data.(llm.Message).Text != "summarize results" {
							t.Fatalf("steering missing from request: %#v", next.request)
						}
					}
					next.response <- llm.Response{ID: "final-response"}
					synctest.Wait()
					if len(calls) != 2 || len(store.appendedTurns) != 2 {
						t.Fatalf("model requests = %d, turns = %d, want 2 each", len(calls), len(store.appendedTurns))
					}
					if !steer && !reflect.DeepEqual(store.appendedResponses[0].Response, response) {
						t.Fatalf("active model response was not preserved: %#v", store.appendedResponses)
					}
					cancel()
					synctest.Wait()
					if !errors.Is(runErr, context.Canceled) {
						t.Fatalf("Run error = %v, want context cancellation", runErr)
					}
				})
			})
		}
	}
}

func TestCoordinatorRunSlurpsIndependentOperationCompletions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, registry := independentToolCalls(t, 3)
		operations := newFakeOperationManager()
			value.Status = operation.StatusCompleted
			operations.updates <- value
		}
		adapter := &fakeAdapter{respond: func(ctx context.Context, _ llm.Request) (llm.Response, error) {
			<-ctx.Done()
			return llm.Response{}, ctx.Err()
		}}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		current := newTestCoordinatorWithAdapter(
		)
		var runErr error
		go func() { runErr = current.Run(ctx) }()
		synctest.Wait()
		requests := adapter.requestSnapshot()
		if len(requests) != 1 {
			t.Fatalf("model requests = %d, want 1", len(requests))
		}
		assertCompletedResults(t, requests[0], 3)
		if len(store.savedOperations) != 3 || len(store.appendedStatuses) != 3 || len(store.appendedTurns) != 1 {
			t.Fatalf("completion effects: operations=%d statuses=%d turns=%d",
				len(store.savedOperations), len(store.appendedStatuses), len(store.appendedTurns))
		}
		cancel()
		synctest.Wait()
		if !errors.Is(runErr, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", runErr)
		}
	})
}

func TestCoordinatorRunPersistsCompletedUpdatesBeforeClosure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store, registry := independentToolCalls(t, 3)
		operations := newFakeOperationManager()
		var completed []operation.Operation
			value.Status = operation.StatusCompleted
			completed = append(completed, value)
			operations.updates <- value
		}
		close(operations.updates)
		adapter := &fakeAdapter{respond: func(ctx context.Context, _ llm.Request) (llm.Response, error) {
			<-ctx.Done()
			return llm.Response{}, ctx.Err()
		}}
		builder := contextbuilder.NewBuilder()
		err := current.Run(t.Context())
		if err == nil || err.Error() != "operation updates closed" {
			t.Fatalf("Run error = %v, want closed operation updates error", err)
		}
		if !reflect.DeepEqual(store.savedOperations, completed) || len(store.appendedStatuses) != 3 {
			t.Fatalf("completion effects: operations=%v statuses=%v", store.savedOperations, store.appendedStatuses)
		}
		built, err := builder.Build()
		if err != nil {
			t.Fatal(err)
		}
		assertCompletedResults(t, built.Request, 3)
	})
}

func independentToolCalls(t *testing.T, count int) (*fakeStore, tool.Registry) {
	t.Helper()
	store := emptyFakeStore()
	store.items = []sessionstore.Item{
		storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{}),
	}
	response := sessionstore.ModelResponse{TurnID: "turn-1"}
	for index := range count {
		value := operation.Operation{
			ID: operation.ID(fmt.Sprintf("operation-%d", index)), Type: operation.TypeShell,
			Version: 1, Status: operation.StatusAwaiting,
		}
		response.Response.Output = append(response.Response.Output, llm.Item{Type: llm.ItemToolCall, Data: call})
		store.items = append(store.items,
			storedItem(sessionstore.Sequence(len(store.items)+1), sessionstore.ItemToolCallStatus, sessionstore.ToolCallStatus{
				TurnID: "turn-1", CallID: call.CallID,
				Status: tool.CallStatus{WaitingFor: []operation.ID{value.ID}}, Operations: []operation.Operation{value},
			}),
		)
	}
	store.items[1].Data = response
	return store, registry
}

func assertCompletedResults(t *testing.T, request llm.Request, count int) {
	t.Helper()
	completed := make(map[string]int)
	for _, item := range request.Input {
		if item.Type == llm.ItemToolResult {
			result := item.Data.(llm.ToolResult)
				completed[result.CallID]++
			}
		}
	}
	if len(completed) != count {
		t.Fatalf("completed results = %v, want %d", completed, count)
	}
	for index := range count {
		if completed[fmt.Sprintf("call-%d", index)] != 1 {
			t.Fatalf("completed results = %v, want each call once", completed)
		}
	}
}

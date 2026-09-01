package coordinator

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/contextbuilder"
	"github.com/unreallabsai/unreal-agent/harness/inbox"
	"github.com/unreallabsai/unreal-agent/harness/llm"
	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/session"
	"github.com/unreallabsai/unreal-agent/harness/sessionstore"
	"github.com/unreallabsai/unreal-agent/harness/tool"
)

func TestCoordinatorRestoresSession(t *testing.T) {
	input := externalEvent(t, 1, "input-1", "hello")
	response := llm.Response{
		ID: "response-1",
		Output: []llm.Item{
			{Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: "working"}},
			{Type: llm.ItemToolCall, Data: call},
		},
	}
	status := sessionstore.ToolCallStatus{
		TurnID: turn.ID,
		CallID: call.CallID,
		Status: tool.CallStatus{Error: "invalid arguments"},
	}
	resumedOperation := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusAwaiting,
	}
	store := &fakeStore{
		resume: sessionstore.ResumeState{
			Snapshot: sessionstore.Snapshot{
			},
		},
		items: []sessionstore.Item{
			storedItem(1, sessionstore.ItemInput, input),
			storedItem(2, sessionstore.ItemTurn, turn),
			storedItem(3, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
				TurnID: turn.ID, Response: response,
			}),
			storedItem(4, sessionstore.ItemToolCallStatus, status),
		},
	}
	builder := contextbuilder.NewBuilder()

	if err := current.restore(t.Context()); err != nil {
		t.Fatal(err)
	}

		!reflect.DeepEqual(current.state.operations[resumedOperation.ID], resumedOperation) {
		t.Fatalf("restored state = %#v", current.state)
	}

	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		response.Output[0],
		response.Output[1],
		}},
		t.Fatalf("built request = %#v, want input %#v", built.Request, wantInput)
	}
	if !reflect.DeepEqual(store.itemRequests, []itemRequest{{After: 0, Limit: historyPageSize}}) {
		t.Fatalf("item requests = %#v", store.itemRequests)
	}
}

func TestCoordinatorRestoresPaginatedForkHistory(t *testing.T) {
	parentInput := externalEvent(t, 1, "parent-input", "parent")
	items := []sessionstore.Item{storedItem(1, sessionstore.ItemInput, parentInput)}
	for sequence := sessionstore.Sequence(2); sequence <= historyPageSize; sequence++ {
		items = append(items, storedItem(
			sequence,
			sessionstore.ItemTurn,
		))
	}
	items = append(items,
		storedItem(historyPageSize+1, sessionstore.ItemFork, sessionstore.Fork{
			ParentID: "parent", PreviousTurnID: "turn-256",
		}),
		storedItem(historyPageSize+2, sessionstore.ItemInput, externalEvent(
			t, 1, "child-input", "child",
		)),
	)
	store := &fakeStore{
		resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
		}},
		items: items,
	}
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		builder,
	)

	if err := current.restore(t.Context()); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(store.itemRequests, []itemRequest{
		{After: 0, Limit: historyPageSize},
		{After: historyPageSize, Limit: historyPageSize},
	}) {
		t.Fatalf("item requests = %#v", store.itemRequests)
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("replayed input = %#v, want %#v", built.Request.Input, want)
	}
}

	status := sessionstore.ToolCallStatus{
		TurnID: turn.ID,
		CallID: call.CallID,
		Status: tool.CallStatus{WaitingFor: []operation.ID{"terminal-operation"}},
	}
	store := &fakeStore{
		resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
			Session: session.Session{ID: "session-1"},
		}},
		items: []sessionstore.Item{
			storedItem(1, sessionstore.ItemTurn, turn),
			storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
				TurnID:   turn.ID,
				Response: llm.Response{Output: []llm.Item{{Type: llm.ItemToolCall, Data: call}}},
			}),
			storedItem(3, sessionstore.ItemToolCallStatus, status),
		},
	}
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		builder,
		registry,
	)

	if err := current.restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		t.Fatalf("replayed input = %#v", built.Request.Input)
	}
}

	}
	store := &fakeStore{
		resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
		}},
	}
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		builder,
	)

	if err := current.restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		t.Fatalf("replayed input = %#v", built.Request.Input)
	}
}

func TestCoordinatorRejectsInvalidSessionItemData(t *testing.T) {
	tests := []struct {
		name string
		item sessionstore.Item
		want string
	}{
		{name: "fork", item: storedItem(1, sessionstore.ItemFork, session.Turn{}), want: "want sessionstore.Fork"},
		{name: "response", item: storedItem(1, sessionstore.ItemModelResponse, session.Turn{}), want: "want sessionstore.ModelResponse"},
		{name: "status", item: storedItem(1, sessionstore.ItemToolCallStatus, session.Turn{}), want: "want sessionstore.ToolCallStatus"},
		{name: "kind", item: storedItem(1, "unknown", nil), want: `unsupported item kind "unknown"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeStore{
				resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
					Session: session.Session{ID: "session-1"},
				}},
				items: []sessionstore.Item{test.item},
			}
			current := newTestCoordinator(
				store,
				newFakeOperationManager(),
				contextbuilder.NewBuilder(),
			)
			err := current.restore(t.Context())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

	store := emptyFakeStore()
	operations := newFakeOperationManager()
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		builder,
		adapter,
	)
	done := make(chan error, 1)
	go func() {
	}()

	event := externalEvent(t, 1, "input-1", "hello")
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleUser, Text: "hello"},
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("built input = %#v, want %#v", built.Request.Input, want)
	}
}

	store := emptyFakeStore()
	initial := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady,
	}
	operations := newFakeOperationManager()
	updated := initial
	updated.Status = operation.StatusAwaiting
	operations.updates <- updated
	close(operations.updates)
	adapter := &fakeAdapter{}
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		adapter,
	)

	err := current.Run(t.Context())
	if err == nil || err.Error() != "operation updates closed" {
		t.Fatalf("Run error = %v, want closed operation updates error", err)
	}
	if !reflect.DeepEqual(current.state.operations[initial.ID], updated) {
		t.Fatalf("operation = %#v, want %#v", current.state.operations[initial.ID], updated)
	}
}

func TestCoordinatorRunReturnsContextCancellation(t *testing.T) {
}

func newTestCoordinator(
	store *fakeStore,
	operations *fakeOperationManager,
	builder contextbuilder.Builder,
	registry tool.Registry,
) *coordinator {
	return newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		builder,
		registry,
		&fakeAdapter{},
	)
}

func newTestCoordinatorWithAdapter(
	store *fakeStore,
	operations *fakeOperationManager,
	builder contextbuilder.Builder,
	registry tool.Registry,
	adapter *fakeAdapter,
) *coordinator {
	return New(Dependencies{
		SessionID:      "session-1",
		Inbox:          inputs,
		Sessions:       store,
		ContextBuilder: builder,
		LLM:            adapter,
		Tools:          registry,
		Operations:     operations,
	}).(*coordinator)
}

func emptyFakeStore() *fakeStore {
	return &fakeStore{resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
		Session: session.Session{ID: "session-1"},
	}}}
}

	t.Helper()
	payload, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
}

func storedItem(sequence sessionstore.Sequence, kind sessionstore.ItemKind, data any) sessionstore.Item {
	return sessionstore.Item{Sequence: sequence, Kind: kind, Data: data}
}

type testTranslator struct{}

func (testTranslator) Translate(tool.Context, llm.ToolCall) tool.CallStatus {
	return tool.CallStatus{}
}

func (testTranslator) TranslateResult(
	_ string,
	status tool.CallStatus,
	_ []operation.Operation,
) (llm.ToolResult, error) {
}

type itemRequest struct {
	After sessionstore.Sequence
	Limit int
}

type fakeStore struct {
func (store *fakeStore) Create(context.Context, session.ID) (sessionstore.Snapshot, error) {
	return sessionstore.Snapshot{}, errors.New("unexpected create")
}

func (store *fakeStore) Inspect(context.Context, session.ID) (sessionstore.Snapshot, error) {
	return store.resume.Snapshot, nil
}

func (store *fakeStore) Items(
	_ context.Context,
	_ session.ID,
	after sessionstore.Sequence,
	limit int,
) (sessionstore.Page, error) {
	store.itemRequests = append(store.itemRequests, itemRequest{After: after, Limit: limit})
	start := sort.Search(len(store.items), func(index int) bool {
		return store.items[index].Sequence > after
	})
	end := min(start+limit, len(store.items))
	items := append([]sessionstore.Item(nil), store.items[start:end]...)
	next := after
	if len(items) != 0 {
		next = items[len(items)-1].Sequence
	}
	return sessionstore.Page{Items: items, NextAfter: next, More: end < len(store.items)}, nil
}

}

}

func (store *fakeStore) AppendModelResponse(
) error {
}

func (store *fakeStore) AppendToolCallStatus(
) error {
}

}

func (store *fakeStore) Resume(context.Context, session.ID) (sessionstore.ResumeState, error) {
	return store.resume, nil
}

func (store *fakeStore) Fork(
	context.Context,
	session.ID,
	session.ID,
	session.TurnID,
) (sessionstore.Snapshot, error) {
	return sessionstore.Snapshot{}, errors.New("unexpected fork")
}

	}
}

}

type fakeOperationManager struct {
}

func newFakeOperationManager() *fakeOperationManager {
	return &fakeOperationManager{updates: make(chan operation.Operation, 8)}
}

func (manager *fakeOperationManager) Add(value operation.Operation) error {
	manager.adds = append(manager.adds, value)
}

	manager.cancels = append(manager.cancels, id)
}

func (manager *fakeOperationManager) Updates() <-chan operation.Operation {
	return manager.updates
}

type fakeAdapter struct {
}

func (adapter *fakeAdapter) Respond(
) (llm.Response, error) {
	return llm.Response{}, errors.New("unexpected respond")
}

	t.Helper()
	}
}

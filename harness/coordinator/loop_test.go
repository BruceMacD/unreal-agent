package coordinator

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

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

	if current.state.currentTurnID != turn.ID ||
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
		tool.NewRegistry(tool.StaticTranslators{}),
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
	if current.state.currentTurnID != "turn-256" {
		t.Fatalf("current turn ID = %q, want %q", current.state.currentTurnID, "turn-256")
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("replayed input = %#v, want %#v", built.Request.Input, want)
	}
}

func TestCoordinatorKeepsUnreplayableToolStatusInLocalState(t *testing.T) {
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
	callState, exists := current.state.toolCalls[toolCallKey{
		turnID: status.TurnID,
		callID: status.CallID,
	}]
	if !exists || callState.status == nil || !reflect.DeepEqual(*callState.status, status.Status) {
		t.Fatalf("tool call state = %#v", callState)
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		t.Fatalf("replayed input = %#v", built.Request.Input)
	}
}

func TestCoordinatorRestoresCompletedToolCallFromStatusSnapshots(t *testing.T) {
	initial := operation.Operation{
		ID: "operation-1", Type: "test", Version: 1, Status: operation.StatusReady,
	}
	completed := initial
	completed.Status = operation.StatusCompleted
	status := sessionstore.ToolCallStatus{
		TurnID: "turn-1",
		CallID: call.CallID,
		Status: tool.CallStatus{WaitingFor: []operation.ID{initial.ID}},
	}
	initialStatus := status
	initialStatus.Operations = []operation.Operation{initial}
	completedStatus := status
	completedStatus.Operations = []operation.Operation{completed}
	store := &fakeStore{
		resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
			Session: session.Session{ID: "session-1"},
		}},
		items: []sessionstore.Item{
			storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
				TurnID:   "turn-1",
				Response: llm.Response{Output: []llm.Item{{Type: llm.ItemToolCall, Data: call}}},
			}),
			storedItem(3, sessionstore.ItemToolCallStatus, initialStatus),
			storedItem(4, sessionstore.ItemToolCallStatus, completedStatus),
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
	if _, exists := current.state.toolCalls[toolCallKey{turnID: "turn-1", callID: call.CallID}]; exists {
		t.Fatal("completed tool call remains in local state")
	}
	if !reflect.DeepEqual(current.state.operations[completed.ID], completed) {
		t.Fatalf("restored operation = %#v, want %#v", current.state.operations[completed.ID], completed)
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("replayed input = %#v, want %#v", built.Request.Input, want)
	}
}

func TestCoordinatorOverlaysResumedOperationsAfterHistorySnapshots(t *testing.T) {
	first := operation.Operation{
		ID: "operation-1", Type: "test", Version: 1, Status: operation.StatusReady,
	}
	second := operation.Operation{
		ID: "operation-2", Type: "test", Version: 1, Status: operation.StatusReady,
	}
	resumedSecond := second
	resumedSecond.Status = operation.StatusAwaiting
	status := sessionstore.ToolCallStatus{
		TurnID:     "turn-1",
		CallID:     call.CallID,
		Status:     tool.CallStatus{WaitingFor: []operation.ID{first.ID, second.ID}},
		Operations: []operation.Operation{first, second},
	}
	store := &fakeStore{
		resume: sessionstore.ResumeState{
			Snapshot:   sessionstore.Snapshot{Session: session.Session{ID: "session-1"}},
		},
		items: []sessionstore.Item{
			storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
				TurnID:   "turn-1",
				Response: llm.Response{Output: []llm.Item{{Type: llm.ItemToolCall, Data: call}}},
			}),
			storedItem(3, sessionstore.ItemToolCallStatus, status),
		},
	}
	operations := newFakeOperationManager()
	current := newTestCoordinator(
		store,
		operations,
		contextbuilder.NewBuilder(),
		registry,
	)

	if err := current.restore(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current.state.operations[first.ID], first) ||
		!reflect.DeepEqual(current.state.operations[second.ID], resumedSecond) {
		t.Fatalf("restored operations = %#v", current.state.operations)
	}
	if err := current.dispatchOperationsToManager(); err != nil {
		t.Fatal(err)
	}
	want := map[operation.ID]operation.Operation{
		first.ID:  first,
		second.ID: resumedSecond,
	}
	got := make(map[operation.ID]operation.Operation, len(operations.adds))
	for _, value := range operations.adds {
		got[value.ID] = value
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dispatched operations = %#v, want %#v", got, want)
	}
}

func TestCoordinatorTracksToolCalls(t *testing.T) {
	current := newTestCoordinator(
		emptyFakeStore(),
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		registry,
	)
	for _, response := range []sessionstore.ModelResponse{
		{
			TurnID: "turn-1",
			Response: llm.Response{Output: []llm.Item{
				{Type: llm.ItemMessage, Data: llm.Message{Role: llm.RoleAssistant, Text: "working"}},
				{Type: llm.ItemToolCall, Data: first},
			}},
		},
		{
			TurnID: "turn-2",
			Response: llm.Response{Output: []llm.Item{
				{Type: llm.ItemToolCall, Data: second},
			}},
		},
	} {
		if _, err := current.addItemToLocalState(sessionstore.Item{
			Kind: sessionstore.ItemModelResponse,
			Data: response,
		}); err != nil {
			t.Fatal(err)
		}
	}

	want := map[toolCallKey]toolCallState{
		{turnID: "turn-1", callID: first.CallID}: {
			toolCall: first, operations: map[operation.ID]struct{}{},
		},
		{turnID: "turn-2", callID: second.CallID}: {
			toolCall: second, operations: map[operation.ID]struct{}{},
		},
	}
	if !reflect.DeepEqual(current.state.toolCalls, want) {
		t.Fatalf("tool calls = %#v, want %#v", current.state.toolCalls, want)
	}

	if _, err := current.addItemToLocalState(sessionstore.Item{
		Kind: sessionstore.ItemToolCallStatus,
		Data: sessionstore.ToolCallStatus{
			TurnID: "turn-1",
			CallID: first.CallID,
			Status: tool.CallStatus{Error: "invalid arguments"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	want = map[toolCallKey]toolCallState{
		{turnID: "turn-2", callID: second.CallID}: {
			toolCall: second, operations: map[operation.ID]struct{}{},
		},
	}
	if !reflect.DeepEqual(current.state.toolCalls, want) {
		t.Fatalf("tool calls = %#v, want %#v", current.state.toolCalls, want)
	}

	waitingFor := []operation.ID{"operation-1", "operation-2"}
	waitingStatus := tool.CallStatus{WaitingFor: waitingFor}
	if _, err := current.addItemToLocalState(sessionstore.Item{
		Kind: sessionstore.ItemToolCallStatus,
		Data: sessionstore.ToolCallStatus{
			TurnID: "turn-2",
			CallID: second.CallID,
			Status: waitingStatus,
		},
	}); err != nil {
		t.Fatal(err)
	}
	want = map[toolCallKey]toolCallState{
		{turnID: "turn-2", callID: second.CallID}: {
			toolCall: second,
			status:   &waitingStatus,
			operations: map[operation.ID]struct{}{
				waitingFor[0]: {},
				waitingFor[1]: {},
			},
		},
	}
	if !reflect.DeepEqual(current.state.toolCalls, want) {
		t.Fatalf("tool calls = %#v, want %#v", current.state.toolCalls, want)
	}
}

func TestCoordinatorToolCallOperationsAreTerminal(t *testing.T) {
	current := &coordinator{state: newLoopState()}
	if !current.toolCallOperationsAreTerminal("missing-turn", "missing-call") {
		t.Fatal("tool call without operations was not recognized as terminal")
	}

	key := toolCallKey{turnID: "turn-1", callID: "call-1"}
	current.state.toolCalls[key] = toolCallState{
		operations: map[operation.ID]struct{}{
			"completed": {},
			"failed":    {},
			"canceled":  {},
		},
	}
	for id, status := range map[operation.ID]operation.Status{
		"completed": operation.StatusCompleted,
		"failed":    operation.StatusFailed,
		"canceled":  operation.StatusCanceled,
	} {
		current.addOperationToLocalState(operation.Operation{ID: id, Status: status})
	}

	if !current.toolCallOperationsAreTerminal(key.turnID, key.callID) {
		t.Fatal("terminal operations were not recognized")
	}

	current.addOperationToLocalState(operation.Operation{
		ID: "failed", Status: operation.StatusAwaiting,
	})
	if current.toolCallOperationsAreTerminal(key.turnID, key.callID) {
		t.Fatal("non-terminal operation was recognized as terminal")
	}
}

func TestCoordinatorAddsToolResultFromTrackedToolCall(t *testing.T) {
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinator(
		emptyFakeStore(),
		newFakeOperationManager(),
		builder,
		registry,
	)
	current.addToolCallsToLocalState(sessionstore.ModelResponse{
		TurnID: "turn-1",
		Response: llm.Response{Output: []llm.Item{{
			Type: llm.ItemToolCall,
			Data: call,
		}}},
	})

	status := sessionstore.ToolCallStatus{
		TurnID: "turn-1",
		CallID: call.CallID,
		Status: tool.CallStatus{Error: "invalid arguments"},
	}
	if err := current.addToolResultToLocalState(status); err != nil {
		t.Fatal(err)
	}

	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		Type: llm.ItemToolResult,
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("built input = %#v, want %#v", built.Request.Input, want)
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
		tool.NewRegistry(tool.StaticTranslators{}),
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
				tool.NewRegistry(tool.StaticTranslators{}),
			)
			err := current.restore(t.Context())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestCoordinatorRunCallsModelAfterPersistedExternalInput(t *testing.T) {
	store := emptyFakeStore()
	store.items = []sessionstore.Item{storedItem(
		1,
		sessionstore.ItemTurn,
	)}
	operations := newFakeOperationManager()
	started := make(chan llm.Request, 1)
	requestCanceled := make(chan error, 1)
	var orderMutex sync.Mutex
	order := make([]string, 0, 3)
	record := func(value string) {
		orderMutex.Lock()
		defer orderMutex.Unlock()
		order = append(order, value)
	}
	store.onAppendTurn = func(session.Turn) { record("turn") }
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		record("respond")
		started <- request
		<-ctx.Done()
		requestCanceled <- ctx.Err()
		return llm.Response{}, ctx.Err()
	}}
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		builder,
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	event := externalEvent(t, 1, "input-1", "hello")
	request := receiveTestValue(t, started)
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleUser, Text: "hello"},
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("built input = %#v, want %#v", built.Request.Input, want)
	}
	if !reflect.DeepEqual(request, built.Request) {
		t.Fatalf("model request = %#v, want %#v", request, built.Request)
	}
	}
	if len(store.appendedTurns) != 1 || store.appendedTurns[0].ID == "" ||
		store.appendedTurns[0].PreviousTurnID != "previous-turn" {
		t.Fatalf("appended turns = %#v", store.appendedTurns)
	}
	orderMutex.Lock()
	gotOrder := append([]string(nil), order...)
	orderMutex.Unlock()
	if !reflect.DeepEqual(gotOrder, []string{"input", "turn", "respond"}) {
		t.Fatalf("effect order = %v", gotOrder)
	}
	if len(operations.adds) != 0 || len(operations.cancels) != 0 {
		t.Fatalf("unexpected operation effects: adds=%v cancels=%v", operations.adds, operations.cancels)
	}

	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if err := receiveTestValue(t, requestCanceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("model request error = %v, want context cancellation", err)
	}
}

func TestCoordinatorRunDoesNotCallModelWhenRequestBuildFails(t *testing.T) {
	store := emptyFakeStore()
	adapter := &fakeAdapter{}
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		failingBuilder{
			Builder: contextbuilder.NewBuilder(),
			err:     errors.New("context unavailable"),
		},
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(t.Context())
	}()

	event := externalEvent(t, 1, "input-1", "hello")
	err := receiveTestValue(t, done)
	if err == nil || err.Error() != "build model request: context unavailable" {
		t.Fatalf("Run error = %v", err)
	}
		len(store.appendedTurns) != 0 || len(adapter.requestSnapshot()) != 0 {
		t.Fatalf(
			"effects after build failure: inputs=%v turns=%v requests=%v",
			store.appendedInputs,
			store.appendedTurns,
			adapter.requestSnapshot(),
		)
	}
}

func TestCoordinatorRunDoesNotCallModelWhenTurnStoreFails(t *testing.T) {
	store := emptyFakeStore()
	store.appendTurnErr = errors.New("disk unavailable")
	adapter := &fakeAdapter{}
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(t.Context())
	}()

	err := receiveTestValue(t, done)
	if err == nil || !strings.Contains(err.Error(), "disk unavailable") {
		t.Fatalf("Run error = %v", err)
	}
	if len(store.appendedTurns) != 1 || len(adapter.requestSnapshot()) != 0 {
		t.Fatalf(
			"effects after turn store failure: turns=%v requests=%v",
			store.appendedTurns,
			adapter.requestSnapshot(),
		)
	}
}

func TestCoordinatorRunPersistsModelResponseForOriginatingTurn(t *testing.T) {
	store := emptyFakeStore()
	responseStored := make(chan sessionstore.ModelResponse, 1)
	store.onAppendModelResponse = func(response sessionstore.ModelResponse) {
		responseStored <- response
	}
	response := llm.Response{ID: "response-1", Output: []llm.Item{{
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleAssistant, Text: "done"},
	}}}
	adapter := &fakeAdapter{respond: func(
		context.Context,
		llm.Request,
	) (llm.Response, error) {
		return response, nil
	}}
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	stored := receiveTestValue(t, responseStored)
	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if len(store.appendedTurns) != 1 || stored.TurnID != store.appendedTurns[0].ID ||
		!reflect.DeepEqual(stored.Response, response) {
		t.Fatalf("stored response = %#v, turns = %#v", stored, store.appendedTurns)
	}
	if len(adapter.requestSnapshot()) != 1 || len(store.appendedResponses) != 1 ||
		len(store.appendedTurns) != 1 {
		t.Fatalf(
			"message response effects: requests=%v responses=%v turns=%v",
			adapter.requestSnapshot(),
			store.appendedResponses,
			store.appendedTurns,
		)
	}
}

func TestCoordinatorRunPersistsToolCallBeforeDispatch(t *testing.T) {
	spec, err := operation.NewValueSpec(jsontext.Value(`{"value":1}`))
	if err != nil {
		t.Fatal(err)
	}
	translator := &submittingTranslator{specs: []operation.Spec{spec}}
	store := emptyFakeStore()
	dispatched := make(chan operation.Operation, 1)
	operations := newFakeOperationManager()
	operations.addError = func(value operation.Operation) error {
		if len(store.appendedResponses) != 1 || len(store.appendedStatuses) != 1 {
			return errors.New("operation dispatched before response and status were stored")
		}
		dispatched <- value
		return nil
	}
	response := llm.Response{ID: "response-1", Output: []llm.Item{{
		Type: llm.ItemToolCall,
		Data: llm.ToolCall{CallID: "call-1", Name: tool.BashName, Arguments: `{}`},
	}}}
	adapter := &fakeAdapter{respond: func(
		context.Context,
		llm.Request,
	) (llm.Response, error) {
		return response, nil
	}}
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	operationValue := receiveTestValue(t, dispatched)
	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if len(store.appendedStatuses) != 1 ||
		!reflect.DeepEqual(store.appendedStatuses[0].Operations, []operation.Operation{operationValue}) ||
		!reflect.DeepEqual(store.appendedStatuses[0].Status.WaitingFor, []operation.ID{operationValue.ID}) {
		t.Fatalf("appended statuses = %#v, operation = %#v", store.appendedStatuses, operationValue)
	}
	if len(adapter.requestSnapshot()) != 1 || len(store.appendedTurns) != 1 {
		t.Fatalf("requests = %v, turns = %v", adapter.requestSnapshot(), store.appendedTurns)
	}
}

func TestCoordinatorRunStartsCorrectiveTurnForValidationError(t *testing.T) {
	firstResponse := llm.Response{ID: "response-1", Output: []llm.Item{{
		Type: llm.ItemToolCall,
		Data: llm.ToolCall{CallID: "call-1", Name: tool.BashName, Arguments: `{}`},
	}}}
	started := make(chan llm.Request, 2)
	callCount := 0
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		callCount++
		started <- request
		if callCount == 1 {
			return firstResponse, nil
		}
		<-ctx.Done()
		return llm.Response{}, ctx.Err()
	}}
	store := emptyFakeStore()
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	firstRequest := receiveTestValue(t, started)
	secondRequest := receiveTestValue(t, started)
	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if len(store.appendedStatuses) != 1 || store.appendedStatuses[0].Status.Error == "" {
		t.Fatalf("appended statuses = %#v", store.appendedStatuses)
	}
	if len(store.appendedTurns) != 2 ||
		store.appendedTurns[1].PreviousTurnID != store.appendedTurns[0].ID {
		t.Fatalf("appended turns = %#v", store.appendedTurns)
	}
		t.Fatalf("model requests = %#v", []llm.Request{firstRequest, secondRequest})
	}
}

func TestCoordinatorRunStartsContinuationTurnForCompletedToolCall(t *testing.T) {
	spec, err := operation.NewValueSpec(jsontext.Value(`{"value":1}`))
	if err != nil {
		t.Fatal(err)
	}
	translator := &submittingTranslator{specs: []operation.Spec{spec}}
	firstResponse := llm.Response{ID: "response-1", Output: []llm.Item{{
		Type: llm.ItemToolCall,
		Data: llm.ToolCall{CallID: "call-1", Name: tool.BashName, Arguments: `{}`},
	}}}
	started := make(chan llm.Request, 2)
	callCount := 0
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		callCount++
		started <- request
		if callCount == 1 {
			return firstResponse, nil
		}
		<-ctx.Done()
		return llm.Response{}, ctx.Err()
	}}
	store := emptyFakeStore()
	operations := newFakeOperationManager()
	dispatched := make(chan operation.Operation, 1)
	operations.addError = func(value operation.Operation) error {
		dispatched <- value
		return nil
	}
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	_ = receiveTestValue(t, started)
	operationValue := receiveTestValue(t, dispatched)
	operationValue.Status = operation.StatusCompleted
	operations.updates <- operationValue
	continuationRequest := receiveTestValue(t, started)
	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if !reflect.DeepEqual(store.savedOperations, []operation.Operation{operationValue}) ||
		len(store.appendedStatuses) != 2 ||
		!reflect.DeepEqual(store.appendedStatuses[1].Operations, []operation.Operation{operationValue}) {
		t.Fatalf(
			"completion effects: operations=%#v statuses=%#v",
			store.savedOperations,
			store.appendedStatuses,
		)
	}
		t.Fatalf("continuation request = %#v", continuationRequest)
	}
	if len(adapter.requestSnapshot()) != 2 || len(store.appendedTurns) != 2 {
		t.Fatalf("requests = %v, turns = %v", adapter.requestSnapshot(), store.appendedTurns)
	}
}

func TestCoordinatorRunBatchesCompletedToolCallsIntoOneTurn(t *testing.T) {
	operationValue := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady,
	}
	calls := []llm.ToolCall{
	}
	status := tool.CallStatus{WaitingFor: []operation.ID{operationValue.ID}}
	store := emptyFakeStore()
	store.items = []sessionstore.Item{
		storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
			TurnID: "turn-1",
			Response: llm.Response{Output: []llm.Item{
				{Type: llm.ItemToolCall, Data: calls[0]},
				{Type: llm.ItemToolCall, Data: calls[1]},
			}},
		}),
		storedItem(3, sessionstore.ItemToolCallStatus, sessionstore.ToolCallStatus{
			TurnID: "turn-1", CallID: calls[0].CallID, Status: status,
			Operations: []operation.Operation{operationValue},
		}),
		storedItem(4, sessionstore.ItemToolCallStatus, sessionstore.ToolCallStatus{
			TurnID: "turn-1", CallID: calls[1].CallID, Status: status,
			Operations: []operation.Operation{operationValue},
		}),
	}
	started := make(chan llm.Request, 1)
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		started <- request
		<-ctx.Done()
		return llm.Response{}, ctx.Err()
	}}
	operations := newFakeOperationManager()
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		registry,
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	operationValue.Status = operation.StatusCompleted
	operations.updates <- operationValue
	request := receiveTestValue(t, started)
	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if len(store.appendedStatuses) != 2 || len(store.appendedTurns) != 1 ||
		store.appendedTurns[0].PreviousTurnID != "turn-1" {
		t.Fatalf("statuses = %#v, turns = %#v", store.appendedStatuses, store.appendedTurns)
	}
		t.Fatalf("requests = %#v", adapter.requestSnapshot())
	}
}

func TestCoordinatorRunSteersActiveModelRequest(t *testing.T) {
	started := make(chan llm.Request, 2)
	firstCanceled := make(chan struct{}, 1)
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		started <- request
		<-ctx.Done()
			firstCanceled <- struct{}{}
		}
		return llm.Response{}, ctx.Err()
	}}
	store := emptyFakeStore()
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	firstRequest := receiveTestValue(t, started)
	secondRequest := receiveTestValue(t, started)
	_ = receiveTestValue(t, firstCanceled)
	select {
	case err := <-done:
		t.Fatalf("Run stopped after superseded cancellation: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
		t.Fatalf("steered requests = %#v", []llm.Request{firstRequest, secondRequest})
	}
	if len(store.appendedTurns) != 2 ||
		store.appendedTurns[1].PreviousTurnID != store.appendedTurns[0].ID {
		t.Fatalf("appended turns = %#v", store.appendedTurns)
	}
	if len(store.appendedResponses) != 0 {
		t.Fatalf("appended responses = %#v", store.appendedResponses)
	}

	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
}

func TestCoordinatorRunHandlesOperationUpdateWhileModelIsRunning(t *testing.T) {
	started := make(chan struct{}, 1)
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		_ llm.Request,
	) (llm.Response, error) {
		started <- struct{}{}
		<-ctx.Done()
		return llm.Response{}, ctx.Err()
	}}
	initial := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady,
	}
	store := emptyFakeStore()
	storedUpdate := make(chan operation.Operation, 1)
	store.onSaveOperation = func(value operation.Operation) {
		storedUpdate <- value
	}
	operations := newFakeOperationManager()
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	_ = receiveTestValue(t, started)
	update := initial
	update.Status = operation.StatusAwaiting
	operations.updates <- update
	if stored := receiveTestValue(t, storedUpdate); !reflect.DeepEqual(stored, update) {
		t.Fatalf("stored operation = %#v, want %#v", stored, update)
	}

	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if len(adapter.requestSnapshot()) != 1 || len(store.appendedTurns) != 1 {
		t.Fatalf("requests = %v, turns = %v", adapter.requestSnapshot(), store.appendedTurns)
	}
}

func TestCoordinatorRunDropsSuccessfulResponseFromSupersededTurn(t *testing.T) {
	spec, err := operation.NewValueSpec(jsontext.Value(`{"value":1}`))
	if err != nil {
		t.Fatal(err)
	}
	translator := &submittingTranslator{specs: []operation.Spec{spec}}
	firstResponse := llm.Response{ID: "response-1", Output: []llm.Item{{
		Type: llm.ItemToolCall,
		Data: llm.ToolCall{CallID: "stale-call", Name: tool.BashName, Arguments: `{}`},
	}}}
	secondResponse := llm.Response{ID: "response-2", Output: []llm.Item{{
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleAssistant, Text: "current answer"},
	}}}
	started := make(chan llm.Request, 2)
	firstReturned := make(chan struct{}, 1)
	releaseSecond := make(chan struct{})
	responseStored := make(chan sessionstore.ModelResponse, 2)
	store := emptyFakeStore()
	store.onAppendModelResponse = func(response sessionstore.ModelResponse) {
		responseStored <- response
	}
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		started <- request
			<-ctx.Done()
			firstReturned <- struct{}{}
			return firstResponse, nil
		}
		<-releaseSecond
		return secondResponse, nil
	}}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	_ = receiveTestValue(t, started)
	_ = receiveTestValue(t, started)
	_ = receiveTestValue(t, firstReturned)
	time.Sleep(20 * time.Millisecond)
	close(releaseSecond)
	stored := receiveTestValue(t, responseStored)
	if len(store.appendedTurns) != 2 || stored.TurnID != store.appendedTurns[1].ID ||
		!reflect.DeepEqual(stored.Response, secondResponse) {
		t.Fatalf("stored response = %#v, turns = %#v", stored, store.appendedTurns)
	}
	select {
	case unexpected := <-responseStored:
		t.Fatalf("stored superseded response = %#v", unexpected)
	case <-time.After(20 * time.Millisecond):
	}

	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if !reflect.DeepEqual(store.appendedResponses, []sessionstore.ModelResponse{stored}) ||
		len(translator.calls) != 0 || len(store.appendedStatuses) != 0 {
		t.Fatalf(
			"superseded effects: responses=%#v translations=%#v statuses=%#v",
			store.appendedResponses,
			translator.calls,
			store.appendedStatuses,
		)
	}
}

func TestCoordinatorRunReturnsCurrentModelError(t *testing.T) {
	providerErr := errors.New("provider unavailable")
	adapter := &fakeAdapter{respond: func(
		context.Context,
		llm.Request,
	) (llm.Response, error) {
		return llm.Response{}, providerErr
	}}
	store := emptyFakeStore()
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(t.Context())
	}()

	err := receiveTestValue(t, done)
	if !errors.Is(err, providerErr) || !strings.Contains(err.Error(), "call model for turn") {
		t.Fatalf("Run error = %v", err)
	}
	if len(store.appendedTurns) != 1 || len(store.appendedResponses) != 0 {
		t.Fatalf("turns = %#v, responses = %#v", store.appendedTurns, store.appendedResponses)
	}
}

func TestCoordinatorHandlesModelResponseBeforeSchedulingToolCalls(t *testing.T) {
	spec, err := operation.NewValueSpec(jsontext.Value(`{"value":1}`))
	if err != nil {
		t.Fatal(err)
	}
	response := sessionstore.ModelResponse{
		TurnID: "turn-1",
		Response: llm.Response{ID: "response-1", Output: []llm.Item{{
			Type: llm.ItemToolCall,
			Data: llm.ToolCall{CallID: "call-1", Name: tool.BashName, Arguments: `{}`},
		}}},
	}
	store := emptyFakeStore()
	storedBeforeTranslation := false
	translator := &submittingTranslator{
		specs: []operation.Spec{spec},
		onTranslate: func() {
			storedBeforeTranslation = reflect.DeepEqual(
				store.appendedResponses,
				[]sessionstore.ModelResponse{response},
			)
		},
	}
	operations := newFakeOperationManager()
	current := newTestCoordinator(
		store,
		operations,
		contextbuilder.NewBuilder(),
	)

	if _, err := current.handleModelResponse(t.Context(), response); err != nil {
		t.Fatal(err)
	}
	if !storedBeforeTranslation {
		t.Fatal("tool call was translated before its model response was stored")
	}
	if len(store.appendedStatuses) != 1 || len(store.appendedStatuses[0].Operations) != 1 {
		t.Fatalf("appended statuses = %#v", store.appendedStatuses)
	}
	if len(operations.adds) != 0 {
		t.Fatalf("handler dispatched operations = %#v", operations.adds)
	}
}

func TestCoordinatorDoesNotScheduleToolCallsWhenModelResponseStoreFails(t *testing.T) {
	store := emptyFakeStore()
	store.appendModelResponseErr = errors.New("disk unavailable")
	translator := &submittingTranslator{}
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
	)
	response := sessionstore.ModelResponse{
		TurnID: "turn-1",
		Response: llm.Response{Output: []llm.Item{{
			Type: llm.ItemToolCall,
			Data: llm.ToolCall{CallID: "call-1", Name: tool.BashName, Arguments: `{}`},
		}}},
	}

	_, err := current.handleModelResponse(t.Context(), response)
	if err == nil || err.Error() != `store turn "turn-1" response: disk unavailable` {
		t.Fatalf("handle model response error = %v", err)
	}
	if !reflect.DeepEqual(store.appendedResponses, []sessionstore.ModelResponse{response}) {
		t.Fatalf("appended responses = %#v", store.appendedResponses)
	}
	if len(translator.calls) != 0 || len(store.appendedStatuses) != 0 ||
		len(current.state.operations) != 0 {
		t.Fatalf(
			"scheduled after response store failure: calls=%#v statuses=%#v operations=%#v",
			translator.calls,
			store.appendedStatuses,
			current.state.operations,
		)
	}
}

func TestCoordinatorHandlesModelResponseWithoutToolCalls(t *testing.T) {
	store := emptyFakeStore()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)
	response := sessionstore.ModelResponse{
		TurnID: "turn-1",
		Response: llm.Response{Output: []llm.Item{{
			Type: llm.ItemMessage,
			Data: llm.Message{Role: llm.RoleAssistant, Text: "done"},
		}}},
	}

	if _, err := current.handleModelResponse(t.Context(), response); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.appendedResponses, []sessionstore.ModelResponse{response}) ||
		len(store.appendedStatuses) != 0 || len(current.state.operations) != 0 {
		t.Fatalf(
			"response effects: responses=%#v statuses=%#v operations=%#v",
			store.appendedResponses,
			store.appendedStatuses,
			current.state.operations,
		)
	}
}

func TestCoordinatorRunSchedulesToolCallsWithoutStatusBeforeDispatch(t *testing.T) {
	firstSpec, err := operation.NewValueSpec(jsontext.Value(`{"value":1}`))
	if err != nil {
		t.Fatal(err)
	}
	secondSpec, err := operation.NewValueSpec(jsontext.Value(`{"value":2}`))
	if err != nil {
		t.Fatal(err)
	}
	translator := &submittingTranslator{specs: []operation.Spec{firstSpec, secondSpec}}
	handled := llm.ToolCall{CallID: "call-handled", Name: tool.BashName, Arguments: `{}`}
	missing := llm.ToolCall{CallID: "call-missing", Name: tool.BashName, Arguments: `{}`}
	store := &fakeStore{
		resume: sessionstore.ResumeState{Snapshot: sessionstore.Snapshot{
			Session: session.Session{ID: "session-1"},
		}},
		items: []sessionstore.Item{
			storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
				TurnID: "turn-1",
				Response: llm.Response{Output: []llm.Item{
					{Type: llm.ItemToolCall, Data: handled},
					{Type: llm.ItemToolCall, Data: missing},
				}},
			}),
			storedItem(3, sessionstore.ItemToolCallStatus, sessionstore.ToolCallStatus{
				TurnID: "turn-1",
				CallID: handled.CallID,
				Status: tool.CallStatus{Error: "already handled"},
			}),
		},
	}
	operations := newFakeOperationManager()
	operations.addError = func(operation.Operation) error {
		if len(store.appendedStatuses) != 1 || store.appendedStatuses[0].CallID != missing.CallID {
			return errors.New("operation dispatched before tool-call status was stored")
		}
		return nil
	}
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		registry,
		adapter,
	)

	err = current.Run(t.Context())
		t.Fatalf("Run error = %v, want closed inbox error", err)
	}
	if !reflect.DeepEqual(translator.calls, []llm.ToolCall{missing}) {
		t.Fatalf("translated calls = %#v, want %#v", translator.calls, []llm.ToolCall{missing})
	}
	if len(store.appendedStatuses) != 1 {
		t.Fatalf("appended statuses = %#v", store.appendedStatuses)
	}
	status := store.appendedStatuses[0]
	if status.TurnID != "turn-1" || status.CallID != missing.CallID ||
		len(status.Status.WaitingFor) != 2 || len(status.Operations) != 2 {
		t.Fatalf("appended status = %#v", status)
	}
	wantOperations := make(map[operation.ID]operation.Operation, len(status.Operations))
	for index, value := range status.Operations {
		if value.ID == "" || value.ID != status.Status.WaitingFor[index] ||
			t.Fatalf("scheduled operation %d = %#v", index, value)
		}
		wantOperations[value.ID] = value
		if !reflect.DeepEqual(current.state.operations[value.ID], value) {
			t.Fatalf("local operation %q = %#v, want %#v", value.ID, current.state.operations[value.ID], value)
		}
	}
	gotOperations := make(map[operation.ID]operation.Operation, len(operations.adds))
	for _, value := range operations.adds {
		gotOperations[value.ID] = value
	}
	if !reflect.DeepEqual(gotOperations, wantOperations) {
		t.Fatalf("dispatched operations = %#v, want %#v", gotOperations, wantOperations)
	}
	if _, err := current.scheduleToolCalls(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(translator.calls) != 1 || len(store.appendedStatuses) != 1 {
		t.Fatalf("rescheduled call: calls=%#v statuses=%#v", translator.calls, store.appendedStatuses)
	}
}

func TestCoordinatorRunStartsCorrectiveTurnForRecoveredValidationError(t *testing.T) {
	call := llm.ToolCall{CallID: "call-1", Name: tool.BashName, Arguments: `{}`}
	store := emptyFakeStore()
	store.items = []sessionstore.Item{
		storedItem(2, sessionstore.ItemModelResponse, sessionstore.ModelResponse{
			TurnID:   "turn-1",
			Response: llm.Response{Output: []llm.Item{{Type: llm.ItemToolCall, Data: call}}},
		}),
	}
	started := make(chan llm.Request, 1)
	adapter := &fakeAdapter{respond: func(
		ctx context.Context,
		request llm.Request,
	) (llm.Response, error) {
		started <- request
		<-ctx.Done()
		return llm.Response{}, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(t.Context())
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		adapter,
	)
	done := make(chan error, 1)
	go func() {
		done <- current.Run(ctx)
	}()

	request := receiveTestValue(t, started)
	cancel()
	if err := receiveTestValue(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context cancellation", err)
	}
	if len(store.appendedStatuses) != 1 || store.appendedStatuses[0].Status.Error == "" ||
		len(store.appendedTurns) != 1 || store.appendedTurns[0].PreviousTurnID != "turn-1" {
		t.Fatalf("statuses = %#v, turns = %#v", store.appendedStatuses, store.appendedTurns)
	}
		t.Fatalf("corrective request = %#v", request)
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
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)

	err := current.Run(t.Context())
	if err == nil || err.Error() != "operation updates closed" {
		t.Fatalf("Run error = %v, want closed operation updates error", err)
	}
	if !reflect.DeepEqual(current.state.operations[initial.ID], updated) {
		t.Fatalf("operation = %#v, want %#v", current.state.operations[initial.ID], updated)
	}
	if !reflect.DeepEqual(store.savedOperations, []operation.Operation{updated}) {
		t.Fatalf("saved operations = %#v, want %#v", store.savedOperations, []operation.Operation{updated})
	}
	}
	if len(operations.cancels) != 0 || len(adapter.requests) != 0 {
		t.Fatalf("unexpected effects: cancels=%v model=%v", operations.cancels, adapter.requests)
	}
}

func TestCoordinatorReconcilesToolCallsFromPersistedOperationUpdates(t *testing.T) {
	store := emptyFakeStore()
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		builder,
		registry,
	)
	if _, err := current.addItemToLocalState(sessionstore.Item{
		Kind: sessionstore.ItemModelResponse,
		Data: sessionstore.ModelResponse{
			TurnID: "turn-1",
			Response: llm.Response{Output: []llm.Item{{
				Type: llm.ItemToolCall,
				Data: call,
			}}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	operations := []operation.Operation{
		{ID: "operation-1", Status: operation.StatusReady},
		{ID: "operation-2", Status: operation.StatusAwaiting},
	}
	for _, value := range operations {
		current.addOperationToLocalState(value)
	}
	status := sessionstore.ToolCallStatus{
		TurnID: "turn-1",
		CallID: call.CallID,
		Status: tool.CallStatus{WaitingFor: []operation.ID{
			operations[0].ID,
			operations[1].ID,
		}},
	}
	if _, err := current.addItemToLocalState(sessionstore.Item{
		Kind: sessionstore.ItemToolCallStatus,
		Data: status,
	}); err != nil {
		t.Fatal(err)
	}

	first := operations[0]
	first.Status = operation.StatusCompleted
	if err := current.handleOperationUpdate(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := current.reconcileToolCalls(t.Context()); err != nil {
		t.Fatal(err)
	}
	key := toolCallKey{turnID: status.TurnID, callID: status.CallID}
	if _, exists := current.state.toolCalls[key]; !exists {
		t.Fatal("tool call was removed before every operation became terminal")
	}

	second := operations[1]
	second.Status = operation.StatusFailed
	if err := current.handleOperationUpdate(t.Context(), second); err != nil {
		t.Fatal(err)
	}
	if _, err := current.reconcileToolCalls(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, exists := current.state.toolCalls[key]; exists {
		t.Fatal("completed tool call remains in local state")
	}
	if !reflect.DeepEqual(store.savedOperations, []operation.Operation{first, second}) {
		t.Fatalf("saved operations = %#v, want %#v", store.savedOperations, []operation.Operation{first, second})
	}
	if !reflect.DeepEqual(store.appendedStatuses, []sessionstore.ToolCallStatus{{
		TurnID: status.TurnID,
		CallID: status.CallID,
		Status: status.Status,
		Operations: []operation.Operation{
			first,
			second,
		},
	}}) {
		t.Fatalf("appended statuses = %#v", store.appendedStatuses)
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("built input = %#v, want %#v", built.Request.Input, want)
	}
}

func TestCoordinatorDoesNotCompleteToolCallBeforeOperationIsStored(t *testing.T) {
	store := emptyFakeStore()
	store.saveOperationErr = errors.New("disk unavailable")
	builder := contextbuilder.NewBuilder()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		builder,
		registry,
	)
	current.addToolCallsToLocalState(sessionstore.ModelResponse{
		TurnID: "turn-1",
		Response: llm.Response{Output: []llm.Item{{
			Type: llm.ItemToolCall,
			Data: call,
		}}},
	})
	value := operation.Operation{ID: "operation-1", Status: operation.StatusReady}
	current.addOperationToLocalState(value)
	status := sessionstore.ToolCallStatus{
		TurnID: "turn-1",
		CallID: call.CallID,
		Status: tool.CallStatus{WaitingFor: []operation.ID{value.ID}},
	}
	if _, err := current.addItemToLocalState(sessionstore.Item{
		Kind: sessionstore.ItemToolCallStatus,
		Data: status,
	}); err != nil {
		t.Fatal(err)
	}

	value.Status = operation.StatusCompleted
	err := current.handleOperationUpdate(t.Context(), value)
	if err == nil || err.Error() != `store operation "operation-1": disk unavailable` {
		t.Fatalf("handle operation update error = %v", err)
	}
	if _, exists := current.state.toolCalls[toolCallKey{
		turnID: status.TurnID,
		callID: status.CallID,
	}]; !exists {
		t.Fatal("tool call was removed before its operation was stored")
	}
	built, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
		t.Fatalf("tool results = %#v, want only the initial result", built.Request.Input)
	}
}

func TestCoordinatorRunDispatchesRestoredNonTerminalOperations(t *testing.T) {
	store := emptyFakeStore()
	statuses := []operation.Status{
		operation.StatusReady,
		operation.StatusAwaiting,
		operation.StatusCanceling,
		operation.StatusCompleted,
		operation.StatusFailed,
		operation.StatusCanceled,
	}
	for _, status := range statuses {
			ID: operation.ID(status), Type: operation.TypeShell, Version: 1, Status: status,
		})
	}
	operations := newFakeOperationManager()
	adapter := &fakeAdapter{}
	current := newTestCoordinatorWithAdapter(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
		adapter,
	)

	err := current.Run(t.Context())
		t.Fatalf("Run error = %v, want closed inbox error", err)
	}
	sort.Slice(operations.adds, func(left, right int) bool {
		return operations.adds[left].ID < operations.adds[right].ID
	})
	want := []operation.Operation{
		{ID: "awaiting", Type: operation.TypeShell, Version: 1, Status: operation.StatusAwaiting},
		{ID: "canceling", Type: operation.TypeShell, Version: 1, Status: operation.StatusCanceling},
		{ID: "ready", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady},
	}
	if !reflect.DeepEqual(operations.adds, want) {
		t.Fatalf("dispatched operations = %#v, want %#v", operations.adds, want)
	}
	if len(adapter.requests) != 0 {
		t.Fatalf("model requests = %#v", adapter.requests)
	}
}

func TestCoordinatorRunReturnsOperationDispatchError(t *testing.T) {
	store := emptyFakeStore()
	value := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady,
	}
	operations := newFakeOperationManager()
	operations.addError = func(operation.Operation) error {
		return errors.New("dispatch failed")
	}
	current := newTestCoordinator(
		store,
		operations,
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)

	err := current.Run(t.Context())
	if err == nil || err.Error() != `dispatch operation "operation-1": dispatch failed` {
		t.Fatalf("Run error = %v", err)
	}
}

	store := emptyFakeStore()
		{ID: "unsupported", Type: "remote", Version: 1, Status: operation.StatusReady},
		{ID: "supported", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady},
	}
	operations := newFakeOperationManager()
	operations.addError = func(value operation.Operation) error {
		if value.ID == "unsupported" {
			return fmt.Errorf("manager does not support %q: %w", value.Type, operation.ErrUnsupported)
		}
		return nil
	}
	current := newTestCoordinator(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)

	err := current.Run(t.Context())
	}
	if _, ok := current.state.operations["unsupported"]; !ok {
		t.Fatal("unsupported operation was not retained")
	}
}

func TestCoordinatorClonesOperationDataBeforeDispatch(t *testing.T) {
	operations := newFakeOperationManager()
	operations.mutateAdds = true
	current := newTestCoordinator(
		emptyFakeStore(),
		operations,
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)
	value := current.addOperationToLocalState(operation.Operation{
		ID:          "operation-1",
		Type:        operation.TypeShell,
		Version:     1,
		Status:      operation.StatusReady,
		State:       jsontext.Value(`{"state":"original"}`),
		Idempotency: jsontext.Value(`{"key":"original"}`),
	})

	if err := current.dispatchOperationsToManager(); err != nil {
		t.Fatal(err)
	}
	stored := current.state.operations[value.ID]
	if string(stored.State) != `{"state":"original"}` ||
		string(stored.Idempotency) != `{"key":"original"}` {
		t.Fatalf("stored operation was mutated: %#v", stored)
	}
}

func TestCoordinatorRunReturnsInputStoreErrorAfterUpdatingLocalState(t *testing.T) {
	store := emptyFakeStore()
	store.appendInputErr = errors.New("disk unavailable")
	event := externalEvent(t, 1, "input-1", "hello")
	current := newTestCoordinator(
		store,
		inputs,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)

	err := current.Run(t.Context())
	if err == nil || err.Error() != `store input "input-1": disk unavailable` {
		t.Fatalf("Run error = %v", err)
	}
	built, buildErr := current.dependencies.ContextBuilder.Build()
	if buildErr != nil {
		t.Fatal(buildErr)
	}
		Type: llm.ItemMessage,
		Data: llm.Message{Role: llm.RoleUser, Text: "hello"},
	if !reflect.DeepEqual(built.Request.Input, want) {
		t.Fatalf("built input = %#v, want %#v", built.Request.Input, want)
	}
}

func TestCoordinatorRunReturnsOperationStoreErrorAfterUpdatingLocalState(t *testing.T) {
	store := emptyFakeStore()
	store.saveOperationErr = errors.New("disk unavailable")
	operations := newFakeOperationManager()
	update := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusAwaiting,
	}
	operations.updates <- update
	current := newTestCoordinator(
		store,
		inputs,
		operations,
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)

	err := current.Run(t.Context())
	if err == nil || err.Error() != `store operation "operation-1": disk unavailable` {
		t.Fatalf("Run error = %v", err)
	}
	if !reflect.DeepEqual(current.state.operations[update.ID], update) {
		t.Fatalf("operation = %#v, want %#v", current.state.operations[update.ID], update)
	}
	if len(operations.adds) != 0 {
		t.Fatalf("operations dispatched before store commit = %#v", operations.adds)
	}
}

func TestCoordinatorStoresEverySessionItemKind(t *testing.T) {
	store := emptyFakeStore()
	current := newTestCoordinator(
		store,
		newFakeOperationManager(),
		contextbuilder.NewBuilder(),
		tool.NewRegistry(tool.StaticTranslators{}),
	)
	event := externalEvent(t, 1, "input-1", "hello")
	response := sessionstore.ModelResponse{
		TurnID:   turn.ID,
		Response: llm.Response{ID: "response-1"},
	}
	value := operation.Operation{
		ID: "operation-1", Type: operation.TypeShell, Version: 1, Status: operation.StatusReady,
	}
	current.addOperationToLocalState(value)
	status := sessionstore.ToolCallStatus{
		TurnID:     turn.ID,
		CallID:     "call-1",
		Status:     tool.CallStatus{WaitingFor: []operation.ID{value.ID}},
		Operations: []operation.Operation{value},
	}
	items := []sessionstore.Item{
		{Kind: sessionstore.ItemInput, Data: event},
		{Kind: sessionstore.ItemTurn, Data: turn},
		{Kind: sessionstore.ItemModelResponse, Data: response},
		{Kind: sessionstore.ItemToolCallStatus, Data: status},
	}
	for _, item := range items {
		if err := current.storeItemInSessionStore(t.Context(), item); err != nil {
			t.Fatal(err)
		}
	}

		!reflect.DeepEqual(store.appendedTurns, []session.Turn{turn}) ||
		!reflect.DeepEqual(store.appendedResponses, []sessionstore.ModelResponse{response}) ||
		!reflect.DeepEqual(store.appendedStatuses, []sessionstore.ToolCallStatus{status}) {
		t.Fatalf("stored items: inputs=%#v turns=%#v responses=%#v statuses=%#v",
			store.appendedInputs,
			store.appendedTurns,
			store.appendedResponses,
			store.appendedStatuses,
		)
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

type submittingTranslator struct {
	specs       []operation.Spec
	calls       []llm.ToolCall
	onTranslate func()
}

func (translator *submittingTranslator) Translate(
	ctx tool.Context,
	call llm.ToolCall,
) tool.CallStatus {
	translator.calls = append(translator.calls, call)
	if translator.onTranslate != nil {
		translator.onTranslate()
	}
	status := tool.CallStatus{WaitingFor: make([]operation.ID, 0, len(translator.specs))}
	for _, spec := range translator.specs {
		status.WaitingFor = append(status.WaitingFor, ctx.Submit(spec))
	}
	return status
}

func (*submittingTranslator) TranslateResult(
	callID string,
	status tool.CallStatus,
	_ []operation.Operation,
) (llm.ToolResult, error) {
}

type operationStatusTranslator struct{}

func (operationStatusTranslator) Translate(tool.Context, llm.ToolCall) tool.CallStatus {
	return tool.CallStatus{}
}

func (operationStatusTranslator) TranslateResult(
	_ string,
	_ tool.CallStatus,
	operations []operation.Operation,
) (llm.ToolResult, error) {
	statuses := make([]string, 0, len(operations))
	for _, value := range operations {
		statuses = append(statuses, string(value.Status))
	}
}

type itemRequest struct {
	After sessionstore.Sequence
	Limit int
}

type fakeStore struct {
	resume                 sessionstore.ResumeState
	items                  []sessionstore.Item
	itemRequests           []itemRequest
	appendedTurns          []session.Turn
	appendedResponses      []sessionstore.ModelResponse
	appendedStatuses       []sessionstore.ToolCallStatus
	savedOperations        []operation.Operation
	appendInputErr         error
	appendTurnErr          error
	appendModelResponseErr error
	saveOperationErr       error
	unexpectedMutations    []string
	onAppendTurn           func(session.Turn)
	onAppendModelResponse  func(sessionstore.ModelResponse)
	onAppendToolCallStatus func(sessionstore.ToolCallStatus)
	onSaveOperation        func(operation.Operation)
}

func (store *fakeStore) Create(context.Context, session.ID) (sessionstore.Snapshot, error) {
	store.unexpectedMutations = append(store.unexpectedMutations, "create")
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

	if store.onAppendInput != nil {
	}
	return store.appendInputErr
}

func (store *fakeStore) AppendTurn(_ context.Context, _ session.ID, turn session.Turn) error {
	store.appendedTurns = append(store.appendedTurns, turn)
	if store.onAppendTurn != nil {
		store.onAppendTurn(turn)
	}
	return store.appendTurnErr
}

func (store *fakeStore) AppendModelResponse(
	_ context.Context,
	_ session.ID,
	response sessionstore.ModelResponse,
) error {
	store.appendedResponses = append(store.appendedResponses, response)
	if store.onAppendModelResponse != nil {
		store.onAppendModelResponse(response)
	}
	return store.appendModelResponseErr
}

func (store *fakeStore) AppendToolCallStatus(
	_ context.Context,
	_ session.ID,
	status sessionstore.ToolCallStatus,
) error {
	store.appendedStatuses = append(store.appendedStatuses, status)
	if store.onAppendToolCallStatus != nil {
		store.onAppendToolCallStatus(status)
	}
}

func (store *fakeStore) SaveOperation(
	_ context.Context,
	_ session.ID,
	value operation.Operation,
) error {
	store.savedOperations = append(store.savedOperations, value)
	if store.onSaveOperation != nil {
		store.onSaveOperation(value)
	}
	return store.saveOperationErr
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
	store.unexpectedMutations = append(store.unexpectedMutations, "fork")
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
	if manager.mutateAdds {
		value.State[0] = '!'
		value.Idempotency[0] = '!'
	}
	if manager.addError != nil {
		return manager.addError(value)
	}
	return nil
}

	manager.cancels = append(manager.cancels, id)
}

func (manager *fakeOperationManager) Updates() <-chan operation.Operation {
	return manager.updates
}

type fakeAdapter struct {
}

func (adapter *fakeAdapter) Respond(
	ctx context.Context,
	request llm.Request,
) (llm.Response, error) {
	adapter.mutex.Lock()
	adapter.requests = append(adapter.requests, request)
	respond := adapter.respond
	adapter.mutex.Unlock()
	if respond != nil {
		return respond(ctx, request)
	}
	return llm.Response{}, errors.New("unexpected respond")
}

func (adapter *fakeAdapter) requestSnapshot() []llm.Request {
	adapter.mutex.Lock()
	defer adapter.mutex.Unlock()
	return append([]llm.Request(nil), adapter.requests...)
}

type failingBuilder struct {
	contextbuilder.Builder
	err error
}

func (builder failingBuilder) Build() (contextbuilder.Result, error) {
	return contextbuilder.Result{}, builder.err
}

func receiveTestValue[T any](t *testing.T, values <-chan T) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for value")
		var zero T
		return zero
	}
}

package operation_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/unreallabsai/unreal-agent/harness/operation"
	"github.com/unreallabsai/unreal-agent/harness/primitives"
)

const testShellPath = "/bin/sh"

func TestShellActorCapturesArtifactsAndCompletes(t *testing.T) {
	baseDirectory := t.TempDir()
	workingDirectory := t.TempDir()
	stdout := strings.Repeat("stdout-", 8*1024)
	stderr := strings.Repeat("stderr-", 7*1024)
	current := newShellOperation(t, "shell-complete", operation.ShellInput{
		Shell:     testShellPath,
		Command:   "printf '%s' \"$SHELL_STDOUT\"; printf '%s' \"$SHELL_STDERR\" >&2; exit 7",
		Directory: workingDirectory,
	}, baseDirectory, 19)

	completed := runShellActor(t, t.Context(), current)
	if completed.Status != operation.StatusCompleted {
		t.Fatalf("status = %q, want %q", completed.Status, operation.StatusCompleted)
	}
	state := shellState(t, completed)
	if state.Result == nil || state.TerminalError != "" || state.Phase != "" {
		t.Fatalf("state = %#v", state)
	}
	result := state.Result
		result.OutSize != int64(len(stdout)) ||
		result.ErrSize != int64(len(stderr)) ||
		result.ExitCode != 7 {
		t.Fatalf("result = %#v", result)
	}

	directory := filepath.Join(baseDirectory, string(current.ID))
	assertFileContents(t, filepath.Join(directory, operation.ShellOutFilename), []byte(stdout))
	assertFileContents(t, filepath.Join(directory, operation.ShellErrFilename), []byte(stderr))
}

	}
}

func TestShellActorStartsShellDirectlyWithCapturePaths(t *testing.T) {
	baseDirectory := t.TempDir()
	current := newShellOperation(t, "shell-request", operation.ShellInput{
	}, baseDirectory, 10)

	if err != nil {
		t.Fatal(err)
	}
		request := oneDispatchData[primitives.IOCreateRequest](t, step, primitives.PrimitiveDispatchIOCreate)
		events := make(chan primitives.PrimitiveEvent)
		primitives.Create(t.Context(), request, events)
		event := receiveOnlyPrimitiveEvent(t, events)
		if err != nil {
			t.Fatal(err)
		}
	}

	request := oneDispatchData[primitives.ProcessStartRequest](t, step, primitives.PrimitiveDispatchProcessStart)
	directory := filepath.Join(baseDirectory, string(current.ID))
	if request.Path != "/bin/zsh" ||
		!slices.Equal(request.Arguments, []string{"-c", "printf direct"}) ||
		request.Directory != "/tmp" ||
		request.Pipes != 0 ||
		request.StdoutPath != filepath.Join(directory, operation.ShellOutFilename) ||
		request.StderrPath != filepath.Join(directory, operation.ShellErrFilename) {
		t.Fatalf("process request = %#v", request)
	}
}

func TestShellActorPersistsSignalExitStatusBeforeReading(t *testing.T) {
	baseDirectory := t.TempDir()
	id := operation.ID("shell-signaled")
	directory := filepath.Join(baseDirectory, string(id))
	current := shellOperationWithState(t, id, operation.ShellState{
		Phase:          operation.ShellPhaseProcess,
		ProcessGroupID: 4321,
	})
	event := primitives.PrimitiveEvent{
		Type:          primitives.PrimitiveEventProcessExited,
		Source:        primitives.SourceID(id),
		Result:        primitives.ProcessExitResult{ExitCode: -1, Signal: syscall.SIGTERM},
	}

	if err != nil {
		t.Fatal(err)
	}
	}
		state.PendingExitCode == nil || *state.PendingExitCode != 143 {
		t.Fatalf("state = %#v", state)
	}
}

func TestShellActorFailsUnknownProcessOutcomeWithoutRestarting(t *testing.T) {
	current := shellOperationWithState(t, "shell-unknown", operation.ShellState{
	})

	if err != nil {
		t.Fatal(err)
	}
	if step.Operation.Status != operation.StatusFailed || len(step.Dispatches) != 0 ||
		!strings.Contains(state.TerminalError, "outcome is unknown") {
		t.Fatalf("step = %#v, state = %#v", step, state)
	}
}

func TestShellActorPreservesRecordedProcessWhenFailingRecovery(t *testing.T) {
	current := shellOperationWithState(t, "shell-recover", operation.ShellState{
		Phase:          operation.ShellPhaseProcess,
		ProcessGroupID: 4321,
	})

	if err != nil {
		t.Fatal(err)
	}
	if step.Operation.Status != operation.StatusFailed || len(step.Dispatches) != 0 ||
		state.ProcessGroupID != 4321 ||
		!strings.Contains(state.TerminalError, "interrupted before an exit status") {
		t.Fatalf("step = %#v, state = %#v", step, state)
	}
}

	baseDirectory := t.TempDir()
	id := operation.ID("shell-resume-exit")
	directory := filepath.Join(baseDirectory, string(id))
	exitCode := 5
	current := shellOperationWithState(t, id, operation.ShellState{
		PendingExitCode: &exitCode,
	})

	completed := runShellActor(t, t.Context(), current)
	result := shellState(t, completed).Result
	if completed.Status != operation.StatusCompleted || result == nil ||
		t.Fatalf("operation = %#v, result = %#v", completed, result)
	}
}



	}
}

func TestShellActorResumesEveryReplayablePhase(t *testing.T) {
	baseDirectory := t.TempDir()
	exitCode := 5
	tests := []struct {
		name        string
		state       operation.ShellState
		dispatch    primitives.PrimitiveDispatchType
		correlation primitives.CorrelationID
		artifact    string
		kind        primitives.IOCreateKind
		mode        os.FileMode
		offset      int64
		count       int64
	}{
		{
			name: "create directory", state: operation.ShellState{Phase: operation.ShellPhaseCreateDirectory},
			kind: primitives.IOCreateDirectory, mode: 0o700,
		},
		{
			name: "create stdout", state: operation.ShellState{Phase: operation.ShellPhaseCreateOut},
			artifact: operation.ShellOutFilename, kind: primitives.IOCreateRegularFile, mode: 0o600,
		},
		{
			name: "create stderr", state: operation.ShellState{Phase: operation.ShellPhaseCreateErr},
			artifact: operation.ShellErrFilename, kind: primitives.IOCreateRegularFile, mode: 0o600,
		},
		{
			name: "read stdout", state: operation.ShellState{
				Phase: operation.ShellPhaseReadOut, PendingExitCode: &exitCode, InlineOut: []byte("ou"),
			},
		},
		{
			name: "read stderr", state: operation.ShellState{
				Phase: operation.ShellPhaseReadErr, PendingExitCode: &exitCode, InlineErr: []byte("err"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := operation.ID("resume-" + strings.ReplaceAll(test.name, " ", "-"))
			state := test.state
			state.Input = operation.ShellInput{Shell: testShellPath}
			state.BaseDirectory = baseDirectory
			current := shellOperationWithState(t, id, state)

			if err != nil {
				t.Fatal(err)
			}
			if step.Operation.Status != operation.StatusAwaiting || len(step.Dispatches) != 1 ||
				step.Dispatches[0].Type != test.dispatch {
				t.Fatalf("step = %#v", step)
			}
			directory := filepath.Join(baseDirectory, string(id))
			path := directory
			if test.artifact != "" {
				path = filepath.Join(directory, test.artifact)
			}
			switch test.dispatch {
			case primitives.PrimitiveDispatchIOCreate:
				request := dispatchData[primitives.IOCreateRequest](t, step.Dispatches[0])
				if request.Source != primitives.SourceID(id) || request.CorrelationID != test.correlation ||
					request.Path != path || request.Kind != test.kind || request.Mode != test.mode {
					t.Fatalf("create request = %#v", request)
				}
			case primitives.PrimitiveDispatchIORead:
				request := dispatchData[primitives.IOReadRequest](t, step.Dispatches[0])
				if request.Source != primitives.SourceID(id) || request.CorrelationID != test.correlation ||
					request.Path != path || request.Offset != test.offset || request.Count != test.count {
					t.Fatalf("read request = %#v", request)
				}
			default:
				t.Fatalf("unexpected dispatch %q", test.dispatch)
			}
		})
	}
}

	}
}

func TestShellActorFailsWhenShellCannotStart(t *testing.T) {
	current := newShellOperation(t, "shell-start-failure", operation.ShellInput{
		Shell: filepath.Join(t.TempDir(), "missing-shell"),
	}, t.TempDir(), 10)

	failed := runShellActor(t, t.Context(), current)
	state := shellState(t, failed)
	if failed.Status != operation.StatusFailed || state.TerminalError == "" || state.Result != nil {
		t.Fatalf("operation = %#v, state = %#v", failed, state)
	}
}

func TestNewShellSpecRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		input         operation.ShellInput
		baseDirectory string
	}{
		{input: operation.ShellInput{}, baseDirectory: "/tmp"},
		{input: operation.ShellInput{Shell: "sh"}, baseDirectory: "/tmp"},
		{input: operation.ShellInput{Shell: testShellPath}, baseDirectory: "relative"},
		{input: operation.ShellInput{Shell: testShellPath}, baseDirectory: "/tmp", maxInline: -1},
	} {
		if _, err := operation.NewShellSpec(test.input, test.baseDirectory, test.maxInline); err == nil {
			t.Fatalf("NewShellSpec accepted %#v", test)
		}
	}
}

func TestShellActorValidatesPersistedStateAndIdentity(t *testing.T) {
	valid := operation.ShellState{
	}
	for _, mutate := range []func(*operation.ShellState){
		func(state *operation.ShellState) { state.ProcessGroupID = -1 },
		func(state *operation.ShellState) { state.ProcessGroupID = 1 },
		func(state *operation.ShellState) { state.OutSize = -1 },
		func(state *operation.ShellState) { state.ErrSize = -1 },
	} {
		state := valid
		mutate(&state)
		current := shellOperationWithState(t, "shell-invalid", state)
		}
	}

	current := shellOperationWithState(t, "../invalid", valid)
	if err != nil {
		t.Fatal(err)
	}
	if step.Operation.Status != operation.StatusFailed {
		t.Fatalf("operation = %#v", step.Operation)
	}
}

func TestShellActorCancellationIsTerminal(t *testing.T) {
	current := newShellOperation(t, "shell-cancel", operation.ShellInput{Shell: testShellPath}, t.TempDir(), 10)
	if err != nil {
		t.Fatal(err)
	}
	step.Operation.Status = operation.StatusCanceling

	if err != nil {
		t.Fatal(err)
	}
	if canceled.Operation.Status != operation.StatusCanceled || len(canceled.Dispatches) != 0 || state.TerminalError == "" {
		t.Fatalf("step = %#v, state = %#v", canceled, state)
	}
}

func TestShellActorCancellationPreservesRecordedProcess(t *testing.T) {
	current := shellOperationWithState(t, "shell-cancel-process", operation.ShellState{
		Phase:          operation.ShellPhaseProcess,
		ProcessGroupID: 4321,
	})
	current.Status = operation.StatusCanceling

	if err != nil {
		t.Fatal(err)
	}
	if canceled.Operation.Status != operation.StatusCanceled ||
		len(canceled.Dispatches) != 0 || state.ProcessGroupID != 4321 || state.Phase != "" {
		t.Fatalf("operation = %#v, state = %#v", canceled.Operation, state)
	}
}

func TestShellActorCancellationStopsActiveProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	current := newShellOperation(t, "shell-cancel-active", operation.ShellInput{
		Shell:   testShellPath,
		Command: "exec sleep 30",
	}, t.TempDir(), 10)
	processGroupID := 0

	canceled := runShellActorObserved(t, ctx, current, func(current operation.Operation, event primitives.PrimitiveEvent) {
		if event.Type != primitives.PrimitiveEventProcessStarted {
			return
		}
		started, ok := event.Result.(primitives.ProcessStartedResult)
		if !ok || started.PID <= 1 {
			t.Fatalf("started = %#v", event.Result)
		}
		processGroupID = started.PID
		if state := shellState(t, current); state.ProcessGroupID != processGroupID {
			t.Fatalf("state process group = %d, want %d", state.ProcessGroupID, processGroupID)
		}
		cancel()
	})

	state := shellState(t, canceled)
	if canceled.Status != operation.StatusCanceled || state.Phase != "" ||
		state.ProcessGroupID != processGroupID || state.Result != nil {
		t.Fatalf("operation = %#v, state = %#v", canceled, state)
	}
	if err := syscall.Kill(-processGroupID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("inspect canceled process group %d: %v, want ESRCH", processGroupID, err)
	}
}

func runShellActor(t *testing.T, ctx context.Context, current operation.Operation) operation.Operation {
	t.Helper()
	return runShellActorObserved(t, ctx, current, nil)
}

func runShellActorObserved(
	t *testing.T,
	ctx context.Context,
	current operation.Operation,
	observe func(operation.Operation, primitives.PrimitiveEvent),
) operation.Operation {
	t.Helper()
	runtimeContext, cancelRuntime := context.WithCancel(t.Context())
	defer cancelRuntime()

	if err != nil {
		t.Fatal(err)
	}
	var active *primitiveInvocation
	defer func() {
		cancelRuntime()
		if active != nil {
			drainPrimitiveInvocation(active.events)
		}
	}()

		if len(step.Dispatches) > 1 {
			t.Fatalf("dispatches = %#v, want at most one", step.Dispatches)
		}
		if len(step.Dispatches) == 1 {
			if active != nil {
				t.Fatalf("primitive %q is still active", active.correlationID)
			}
			invocation := startPrimitive(t, runtimeContext, step.Dispatches[0])
			active = &invocation
		}
		if active == nil {
			t.Fatal("shell actor is awaiting no active primitive")
		}

		select {
		case <-ctx.Done():
			cancelRuntime()
			drainPrimitiveInvocation(active.events)
			active = nil

		case event, open := <-active.events:
			if !open {
				t.Fatalf("primitive %q closed while actor was awaiting", active.correlationID)
			}
			if event.Source != active.source || event.CorrelationID != active.correlationID {
				t.Fatalf("event identity = (%q, %q), want (%q, %q)",
					event.Source, event.CorrelationID, active.source, active.correlationID)
			}
			if primitiveEventIsTerminal(event) {
				active = nil
			}
			}
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

type primitiveInvocation struct {
	source        primitives.SourceID
	correlationID primitives.CorrelationID
	events        <-chan primitives.PrimitiveEvent
}

func startPrimitive(
	t *testing.T,
	ctx context.Context,
	dispatch operation.PrimitiveDispatch,
) primitiveInvocation {
	t.Helper()
	events := make(chan primitives.PrimitiveEvent)
	switch dispatch.Type {
	case primitives.PrimitiveDispatchIOCreate:
		request := dispatchData[primitives.IOCreateRequest](t, dispatch)
		primitives.Create(ctx, request, events)
		return primitiveInvocation{request.Source, request.CorrelationID, events}
	case primitives.PrimitiveDispatchIORead:
		request := dispatchData[primitives.IOReadRequest](t, dispatch)
		primitives.ReadFile(ctx, request, events)
		return primitiveInvocation{request.Source, request.CorrelationID, events}
	case primitives.PrimitiveDispatchProcessStart:
		request := dispatchData[primitives.ProcessStartRequest](t, dispatch)
		primitives.StartProcess(ctx, request, events)
		return primitiveInvocation{request.Source, request.CorrelationID, events}
	default:
		t.Fatalf("unexpected dispatch type %q", dispatch.Type)
		return primitiveInvocation{}
	}
}

func primitiveEventIsTerminal(event primitives.PrimitiveEvent) bool {
	switch event.Type {
	case primitives.PrimitiveEventIOCreateCompleted,
		primitives.PrimitiveEventIOReadCompleted,
		primitives.PrimitiveEventProcessExited,
		primitives.PrimitiveEventFailed,
		primitives.PrimitiveEventCanceled:
		return true
	default:
		return false
	}
}

func drainPrimitiveInvocation(events <-chan primitives.PrimitiveEvent) {
	for {
		if primitiveEventIsTerminal(<-events) {
			return
		}
	}
}

func oneDispatchData[T any](
	t *testing.T,
	step operation.Step,
	dispatchType primitives.PrimitiveDispatchType,
) T {
	t.Helper()
	if len(step.Dispatches) != 1 || step.Dispatches[0].Type != dispatchType {
		t.Fatalf("dispatches = %#v, want one %q", step.Dispatches, dispatchType)
	}
	return dispatchData[T](t, step.Dispatches[0])
}

func dispatchData[T any](t *testing.T, dispatch operation.PrimitiveDispatch) T {
	t.Helper()
	data, ok := dispatch.Data.(T)
	if !ok {
		t.Fatalf("dispatch data = %T, want %T", dispatch.Data, *new(T))
	}
	return data
}

func receiveOnlyPrimitiveEvent(
	t *testing.T,
	events <-chan primitives.PrimitiveEvent,
) primitives.PrimitiveEvent {
	t.Helper()
	event, open := <-events
	if !open {
		t.Fatal("primitive closed without an event")
	}
	select {
	case event, open := <-events:
		if !open {
			t.Fatal("primitive closed the caller-owned channel")
		}
		t.Fatalf("primitive returned another event: %#v", event)
	default:
	}
	return event
}

func newShellOperation(
	t *testing.T,
	id operation.ID,
	input operation.ShellInput,
	baseDirectory string,
) operation.Operation {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return operation.Operation{
		Idempotency: spec.Idempotency,
	}
}

func shellOperationWithState(t *testing.T, id operation.ID, state operation.ShellState) operation.Operation {
	t.Helper()
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return operation.Operation{
	}
}

func shellState(t *testing.T, current operation.Operation) operation.ShellState {
	t.Helper()
	var state operation.ShellState
	if err := json.Unmarshal(current.State, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

	t.Helper()
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string][]byte{
	} {
		if err := os.WriteFile(filepath.Join(directory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFileContents(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("contents of %q = %q, want %q", path, got, want)
	}
}

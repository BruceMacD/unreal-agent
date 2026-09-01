package operation_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/unreallabsai/unreal-agent/harness/operation"
)

func TestLocalOperationManagerRunsShellOperation(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	current := newShellOperation(t, "local-shell", operation.ShellInput{
	}, t.TempDir(), 64)

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	completed := receiveTerminalOperation(t, manager.Updates(), current.ID)
	state := shellState(t, completed)
	if completed.Status != operation.StatusCompleted || state.Result == nil || state.TerminalError != "" {
		t.Fatalf("operation = %#v, state = %#v", completed, state)
	}
		state.Result.OutSize != 6 || state.Result.ErrSize != 5 || state.Result.ExitCode != 7 {
		t.Fatalf("result = %#v", state.Result)
	}
}

func TestLocalOperationManagerRunsMultipleShellOperations(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	baseDirectory := t.TempDir()
	want := map[operation.ID]string{
		"local-first":  "first",
		"local-second": "second",
	}
	for id, output := range want {
		current := newShellOperation(t, id, operation.ShellInput{
			Shell:   testShellPath,
			Command: "printf " + output,
		}, baseDirectory, 64)
		if err := manager.Add(current); err != nil {
			t.Fatal(err)
		}
	}

	completed := make(map[operation.ID]struct{}, len(want))
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for len(completed) != len(want) {
		select {
		case current := <-manager.Updates():
			if current.Status != operation.StatusCompleted {
				continue
			}
			result := shellState(t, current).Result
				t.Fatalf("operation = %#v, result = %#v", current, result)
			}
			completed[current.ID] = struct{}{}
		case <-timer.C:
			t.Fatal("timed out waiting for shell operations")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}

func TestLocalOperationManagerQueuesUpdatesWhileUnread(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	manager := operation.NewLocalOperationManager(ctx)
	baseDirectory := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	current := newShellOperation(t, "local-queued-updates", operation.ShellInput{
	}, baseDirectory, 64)

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	waitForFileContents(t, marker, "done")

	phases := []operation.ShellPhase{
		operation.ShellPhaseCreateDirectory,
		operation.ShellPhaseCreateOut,
		operation.ShellPhaseCreateErr,
		operation.ShellPhaseProcess,
		operation.ShellPhaseReadOut,
		operation.ShellPhaseReadErr,
	}
	phaseRanks := make(map[operation.ShellPhase]int, len(phases))
	for rank, phase := range phases {
		phaseRanks[phase] = rank
	}
	lastRank := -1
	seen := make(map[string]struct{})
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case update, open := <-manager.Updates():
			if !open {
				t.Fatal("updates closed before operation completed")
			}
			if update.ID != current.ID {
				continue
			}
			key := string(update.Status) + "\x00" + string(update.State) + "\x00" + string(update.Idempotency)
			if _, exists := seen[key]; exists {
				t.Fatalf("duplicate update = %#v", update)
			}
			seen[key] = struct{}{}

			state := shellState(t, update)
			if update.Status == operation.StatusCompleted {
				if lastRank != len(phases)-1 {
					t.Fatalf("last phase rank = %d, want %d", lastRank, len(phases)-1)
				}
					t.Fatalf("operation = %#v, state = %#v", update, state)
				}
				cancel()
				for remaining := range manager.Updates() {
					if remaining.ID == current.ID {
						t.Fatalf("update after terminal state = %#v", remaining)
					}
				}
				return
			}
			if update.Status != operation.StatusAwaiting {
				t.Fatalf("intermediate status = %q, want %q", update.Status, operation.StatusAwaiting)
			}
			rank, exists := phaseRanks[state.Phase]
			if !exists {
				t.Fatalf("unexpected phase %q", state.Phase)
			}
			if rank < lastRank || rank > lastRank+1 {
				t.Fatalf("phase %q has rank %d after rank %d", state.Phase, rank, lastRank)
			}
			if rank > lastRank {
				lastRank = rank
			}
		case <-timer.C:
			t.Fatal("timed out waiting for queued updates")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}

func TestLocalOperationManagerResumesShellOperation(t *testing.T) {
	baseDirectory := t.TempDir()
	id := operation.ID("local-resume")
	createShellArtifacts(
		t,
		filepath.Join(baseDirectory, string(id)),
		[]byte("output"),
		[]byte("error"),
	)
	exitCode := 5
	current := shellOperationWithState(t, id, operation.ShellState{
		Phase:           operation.ShellPhaseReadOut,
		PendingExitCode: &exitCode,
	})
	manager := operation.NewLocalOperationManager(t.Context())

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	completed := receiveTerminalOperation(t, manager.Updates(), id)
	result := shellState(t, completed).Result
	if completed.Status != operation.StatusCompleted || result == nil ||
		t.Fatalf("operation = %#v, result = %#v", completed, result)
	}
}

func TestLocalOperationManagerCancelsActiveShellOperation(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	current := newShellOperation(t, "local-cancel", operation.ShellInput{
		Shell:   testShellPath,
		Command: "exec sleep 30",
	}, t.TempDir(), 64)

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	running := receiveOperation(t, manager.Updates(), current.ID, func(current operation.Operation) bool {
		state := shellState(t, current)
		return state.Phase == operation.ShellPhaseProcess && state.ProcessGroupID > 1
	})
	processGroupID := shellState(t, running).ProcessGroupID
	if err := manager.Cancel(current.ID, "test cancellation"); err != nil {
		t.Fatal(err)
	}

	canceled := receiveTerminalOperation(t, manager.Updates(), current.ID)
	state := shellState(t, canceled)
	if canceled.Status != operation.StatusCanceled || state.Phase != "" ||
		state.ProcessGroupID != processGroupID || state.Result != nil {
		t.Fatalf("operation = %#v, state = %#v", canceled, state)
	}
	if err := syscall.Kill(-processGroupID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("inspect canceled process group %d: %v, want ESRCH", processGroupID, err)
	}
}

func TestLocalOperationManagerCancelIsIdempotent(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	current := newShellOperation(t, "local-cancel-idempotent", operation.ShellInput{
		Shell:   testShellPath,
		Command: "exec sleep 30",
	}, t.TempDir(), 64)

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cancel(current.ID, "first cancellation"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cancel(current.ID, "second cancellation"); err != nil {
		t.Fatal(err)
	}

	canceled := receiveTerminalOperation(t, manager.Updates(), current.ID)
	if canceled.Status != operation.StatusCanceled {
		t.Fatalf("status = %q, want %q", canceled.Status, operation.StatusCanceled)
	}
	if err := manager.Cancel(current.ID, "cancellation after completion"); err != nil {
		t.Fatal(err)
	}
}

func TestLocalOperationManagerAddIsIdempotent(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	current := newShellOperation(t, "local-add-idempotent", operation.ShellInput{
		Shell:   testShellPath,
		Command: "exec sleep 30",
	}, t.TempDir(), 64)

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cancel(current.ID, "finish test operation"); err != nil {
		t.Fatal(err)
	}
	if completed := receiveTerminalOperation(t, manager.Updates(), current.ID); completed.Status != operation.StatusCanceled {
		t.Fatalf("status = %q, want %q", completed.Status, operation.StatusCanceled)
	}

	replacement := newShellOperation(t, current.ID, operation.ShellInput{
		Shell:   testShellPath,
		Command: "printf duplicate",
	}, t.TempDir(), 64)
	if err := manager.Add(replacement); err != nil {
		t.Fatal(err)
	}
	barrier := newShellOperation(t, "local-add-idempotent-barrier", operation.ShellInput{
		Shell:   testShellPath,
		Command: "true",
	}, t.TempDir(), 64)
	if err := manager.Add(barrier); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case update := <-manager.Updates():
			if update.ID == current.ID {
				t.Fatalf("duplicate operation update = %#v", update)
			}
			if update.ID == barrier.ID && update.Status == operation.StatusCompleted {
				return
			}
		case <-timer.C:
			t.Fatal("timed out waiting for barrier operation")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}

func TestLocalOperationManagerShutdownCancelsActiveShellOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := operation.NewLocalOperationManager(ctx)
	current := newShellOperation(t, "local-shutdown-active", operation.ShellInput{
		Shell:   testShellPath,
		Command: "exec sleep 30",
	}, t.TempDir(), 64)

	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	running := receiveOperation(t, manager.Updates(), current.ID, func(current operation.Operation) bool {
		state := shellState(t, current)
		return state.Phase == operation.ShellPhaseProcess && state.ProcessGroupID > 1
	})
	processGroupID := shellState(t, running).ProcessGroupID
	cancel()

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case _, open := <-manager.Updates():
			if !open {
				if err := syscall.Kill(-processGroupID, 0); !errors.Is(err, syscall.ESRCH) {
					t.Fatalf("inspect shutdown process group %d: %v, want ESRCH", processGroupID, err)
				}
				return
			}
		case <-timer.C:
			t.Fatal("updates did not close after active operation shutdown")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}

func TestLocalOperationManagerIsolatesPrimitiveFailure(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	failed := newShellOperation(t, "local-primitive-failure", operation.ShellInput{
		Shell: testShellPath,
	}, filepath.Join(t.TempDir(), "missing"), 64)
	succeeded := newShellOperation(t, "local-after-primitive-failure", operation.ShellInput{
		Shell:   testShellPath,
		Command: "printf success",
	}, t.TempDir(), 64)

	if err := manager.Add(failed); err != nil {
		t.Fatal(err)
	}
	if err := manager.Add(succeeded); err != nil {
		t.Fatal(err)
	}

	terminal := make(map[operation.ID]operation.Operation, 2)
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for len(terminal) != 2 {
		select {
		case current, open := <-manager.Updates():
			if !open {
				t.Fatal("updates closed before operations completed")
			}
			switch current.Status {
			case operation.StatusCompleted, operation.StatusFailed, operation.StatusCanceled:
				terminal[current.ID] = current
			}
		case <-timer.C:
			t.Fatal("timed out waiting for terminal operations")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}

	failedState := shellState(t, terminal[failed.ID])
	if terminal[failed.ID].Status != operation.StatusFailed ||
		failedState.TerminalError == "" || failedState.Result != nil {
		t.Fatalf("failed operation = %#v, state = %#v", terminal[failed.ID], failedState)
	}
	succeededState := shellState(t, terminal[succeeded.ID])
	if terminal[succeeded.ID].Status != operation.StatusCompleted ||
		t.Fatalf("succeeded operation = %#v, state = %#v", terminal[succeeded.ID], succeededState)
	}
}

func TestLocalOperationManagerRejectsInvalidOperations(t *testing.T) {
	manager := operation.NewLocalOperationManager(t.Context())
	valid := newShellOperation(t, "local-valid", operation.ShellInput{
		Shell: testShellPath,
	}, t.TempDir(), 1)
	unsupportedVersion := valid
	unsupportedVersion.ID = "local-version"
	unsupportedVersion.Version++
	terminal := valid
	terminal.ID = "local-terminal"
	terminal.Status = operation.StatusCompleted
	tests := []struct {
		name    string
		current operation.Operation
		want    string
	}{
		{name: "unsupported type", current: operation.Operation{
			ID: "local-unsupported", Type: "unknown", Status: operation.StatusReady,
		}, want: `does not support type "unknown"`},
		{name: "unsupported version", current: unsupportedVersion, want: "unsupported version"},
		{name: "terminal", current: terminal, want: "terminal status"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := manager.Add(test.current)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}

	if err := manager.Add(valid); err != nil {
		t.Fatal(err)
	}
	if err := manager.Add(valid); err != nil {
		t.Fatal(err)
	}
	if err := manager.Cancel("unknown", "test"); err != nil {
		t.Fatal(err)
	}
}

func TestLocalOperationManagerClosesUpdatesWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	manager := operation.NewLocalOperationManager(ctx)
	current := newShellOperation(t, "local-shutdown", operation.ShellInput{
		Shell:   testShellPath,
		Command: "true",
	}, t.TempDir(), 64)
	if err := manager.Add(current); err != nil {
		t.Fatal(err)
	}
	cancel()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case _, open := <-manager.Updates():
			if !open {
				return
			}
		case <-timer.C:
			t.Fatal("updates did not close")
		}
	}
}

func TestLocalOperationManagerReturnsContextErrorAfterShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	manager := operation.NewLocalOperationManager(ctx)
	cancel()
	for range manager.Updates() {
	}

	current := newShellOperation(t, "local-after-shutdown", operation.ShellInput{
		Shell: testShellPath,
	}, t.TempDir(), 64)
	if err := manager.Add(current); !errors.Is(err, context.Canceled) {
		t.Fatalf("add error = %v, want %v", err, context.Canceled)
	}
	if err := manager.Cancel(current.ID, "after shutdown"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v, want %v", err, context.Canceled)
	}
}

func receiveTerminalOperation(
	t *testing.T,
	updates <-chan operation.Operation,
	id operation.ID,
) operation.Operation {
	t.Helper()
	return receiveOperation(t, updates, id, func(current operation.Operation) bool {
		switch current.Status {
		case operation.StatusCompleted, operation.StatusFailed, operation.StatusCanceled:
			return true
		default:
			return false
		}
	})
}

func receiveOperation(
	t *testing.T,
	updates <-chan operation.Operation,
	id operation.ID,
	accept func(operation.Operation) bool,
) operation.Operation {
	t.Helper()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case current, open := <-updates:
			if !open {
				t.Fatal("updates closed before operation completed")
			}
			if current.ID == id && accept(current) {
				return current
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for operation %q", id)
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}

func waitForFileContents(t *testing.T, path string, want string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		contents, err := os.ReadFile(path)
		if err == nil && string(contents) == want {
			return
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			t.Fatalf("timed out waiting for %q to contain %q", path, want)
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
}

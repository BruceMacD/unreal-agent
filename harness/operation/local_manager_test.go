package operation_test

import (
	"context"
	"errors"
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
